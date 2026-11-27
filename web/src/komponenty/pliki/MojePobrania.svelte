<script lang="ts">
  // Pływający panel „Moje pobrania” (prawy dół, zwijany): transfery użytkownika na żywo z SSE.
  // Każdy: nazwa, protokół, pasek postępu, prędkość, pozostały czas, mini wykres prędkości z 60 s.
  import { onMount } from 'svelte'
  import type { StanTransferu, ZdarzenieTransferu } from '../../api/typy'
  import { sluchajTransferow } from '../../api/transfery'
  import { sesja } from '../../stan/sesja.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import PasekPostepu from '../wspolne/PasekPostepu.svelte'
  import { czasTrwania, predkosc, rozmiar } from '../../stan/format'
  import MiniWykres from './MiniWykres.svelte'
  import { nazwaZeSciezki } from './narzedzia'
  import { DLUGOSC_HISTORII, KLUCZ_PANELU, ZNIKANIE_ZAKONCZONYCH_MS } from './stale'

  interface Pozycja {
    stan: StanTransferu
    historia: number[]
    wynik?: 'ok' | 'przerwane' | 'blad'
  }

  let pozycje = $state<Pozycja[]>([])
  let zwiniety = $state(localStorage.getItem(KLUCZ_PANELU) === '1')

  const aktywne = $derived(pozycje.filter((p) => !p.wynik))
  const laczna = $derived(aktywne.reduce((s, p) => s + p.stan.predkosc_bps, 0))

  const WYNIKI = {
    ok: { stan: 'ok', tekst: 'Zakończono' },
    przerwane: { stan: 'ostrz', tekst: 'Przerwano' },
    blad: { stan: 'blad', tekst: 'Błąd' },
  } as const

  function naZdarzenie(z: ZdarzenieTransferu) {
    // tylko własne transfery (admin dostaje z SSE wszystkie)
    if (z.stan.kto !== sesja.stan?.uzytkownik?.login) return
    const i = pozycje.findIndex((p) => p.stan.id === z.stan.id)
    const poprzednia = i >= 0 ? pozycje[i] : null
    // serwer może podać własną historię; inaczej zbieramy próbki po stronie klienta
    const historia = z.stan.historia?.length
      ? z.stan.historia.slice(-DLUGOSC_HISTORII)
      : [...(poprzednia?.historia ?? []), z.stan.predkosc_bps].slice(-DLUGOSC_HISTORII)
    const nowa: Pozycja = { stan: z.stan, historia }
    if (z.typ === 'koniec') {
      nowa.wynik = z.wynik ?? 'ok'
      const id = z.stan.id
      setTimeout(() => (pozycje = pozycje.filter((p) => p.stan.id !== id || !p.wynik)), ZNIKANIE_ZAKONCZONYCH_MS)
    }
    if (i >= 0) pozycje[i] = nowa
    else pozycje.push(nowa)
  }

  onMount(() => sluchajTransferow(naZdarzenie, () => {}))

  function przelacz() {
    zwiniety = !zwiniety
    localStorage.setItem(KLUCZ_PANELU, zwiniety ? '1' : '0')
  }

  function stanPaska(p: Pozycja): 'ok' | 'info' | 'blad' | 'ostrz' {
    if (!p.wynik) return 'info'
    return WYNIKI[p.wynik].stan
  }
</script>

{#if pozycje.length}
  <section class="panel" class:zwiniety aria-label="Moje pobrania">
    <button class="naglowek" onclick={przelacz} aria-expanded={!zwiniety}>
      <Ikona nazwa="pobierz" rozmiar="s" />
      <span class="tytul">Moje pobrania</span>
      <span class="licznik">{aktywne.length}</span>
      {#if aktywne.length}
        <span class="opis">{predkosc(laczna)}</span>
      {/if}
      <span class="odstep"></span>
      <Ikona nazwa={zwiniety ? 'strzalka_gora' : 'strzalka_dol'} rozmiar="s" />
    </button>
    {#if !zwiniety}
      <ul class="lista">
        {#each pozycje as p (p.stan.id)}
          {@const u = p.stan.rozmiar > 0 ? p.stan.przeslano / p.stan.rozmiar : null}
          <li class="pozycja">
            <div class="wiersz">
              <Ikona nazwa={p.stan.kierunek === 'wgranie' ? 'wgraj' : 'pobierz'} rozmiar="s" />
              <span class="nazwa skroc" title={p.stan.sciezka}>{nazwaZeSciezki(p.stan.sciezka)}</span>
              <Pigulka stan="info">{p.stan.protokol.toUpperCase()}</Pigulka>
            </div>
            <PasekPostepu wartosc={p.wynik === 'ok' ? 1 : u} stan={stanPaska(p)} etykieta="Postęp transferu" />
            <div class="wiersz dane">
              {#if p.wynik}
                <Pigulka stan={WYNIKI[p.wynik].stan}>{WYNIKI[p.wynik].tekst}</Pigulka>
                <span class="opis">{rozmiar(p.stan.przeslano)}</span>
              {:else}
                <span class="opis">{rozmiar(p.stan.przeslano)}{p.stan.rozmiar ? ` z ${rozmiar(p.stan.rozmiar)}` : ''}</span>
                <span class="odstep"></span>
                <span class="mono">{predkosc(p.stan.predkosc_bps)}</span>
                <span class="opis">{p.stan.pozostalo_s > 0 ? `zostało ${czasTrwania(p.stan.pozostalo_s)}` : ''}</span>
              {/if}
            </div>
            <MiniWykres probki={p.historia} />
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}

<style>
  .panel {
    position: fixed;
    right: var(--odstep-sekcja);
    bottom: var(--odstep-sekcja);
    z-index: var(--warstwa-menu);
    width: var(--uklad-toast-w);
    max-width: calc(100vw - var(--odstep-sekcja) * 2);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
    box-shadow: 0 var(--odstep-xs) var(--odstep-sekcja) var(--kolor-cien);
    overflow: hidden;
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    width: 100%;
    height: var(--wysokosc-naglowek-karty);
    padding: 0 var(--odstep-l);
    border: 0;
    background: var(--kolor-karta);
    text-align: left;
  }
  .panel:not(.zwiniety) .naglowek {
    border-bottom: var(--ramka);
  }
  .tytul {
    font-weight: var(--czcionka-gruby);
  }
  .licznik {
    padding: 0 var(--odstep-s);
  }
  .odstep {
    flex: 1;
  }
  .lista {
    list-style: none;
    margin: 0;
    padding: var(--odstep-m);
    display: flex;
    flex-direction: column;
    gap: var(--odstep-m);
    max-height: 50vh;
    overflow-y: auto;
  }
  .pozycja {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    padding: var(--odstep-m);
    background: var(--kolor-karta);
    border-radius: var(--promien-kafel);
  }
  .nazwa {
    flex: 1;
    font-weight: var(--czcionka-polgruby);
  }
  .dane {
    font-size: var(--czcionka-maly);
    gap: var(--odstep-s);
  }
</style>
