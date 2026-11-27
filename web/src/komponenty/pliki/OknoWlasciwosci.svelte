<script lang="ts">
  // Okno „Właściwości” wpisu: nazwa, rodzaj, rozmiar, data zmiany, udział, ścieżka i suma SHA-256 na żądanie.
  import type { Wpis } from '../../api/typy'
  import { trescBledu } from '../../api/klient'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import { data, rozmiar } from '../../stan/format'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import { kopiuj } from './narzedzia'
  import { suma } from './operacje'
  import { NAZWY_RODZAJOW } from './rodzaje'

  let { otwarte = $bindable(false), wpis, udzial }: { otwarte: boolean; wpis: Wpis | null; udzial: string } = $props()

  let sha = $state('')
  let liczenie = $state(false)
  let blad = $state('')

  $effect(() => {
    void wpis?.sciezka
    sha = ''
    blad = ''
    liczenie = false
  })

  async function oblicz() {
    if (!wpis) return
    const sciezka = wpis.sciezka
    liczenie = true
    blad = ''
    try {
      const wynik = await suma(udzial, sciezka)
      if (wpis?.sciezka === sciezka) sha = wynik
    } catch (e) {
      blad = trescBledu(e)
    } finally {
      liczenie = false
    }
  }

  async function kopiujSume() {
    const ok = await kopiuj(sha)
    powiadom(ok ? 'Skopiowano sumę SHA-256.' : 'Nie udało się skopiować.', ok ? 'ok' : 'blad')
  }
</script>

<Okno tytul="Właściwości" bind:otwarte>
  {#if wpis}
    <dl class="dane">
      <dt class="etykieta">Nazwa</dt>
      <dd class="skroc" title={wpis.nazwa}>{wpis.nazwa}</dd>
      <dt class="etykieta">Rodzaj</dt>
      <dd>{NAZWY_RODZAJOW[wpis.rodzaj]}</dd>
      {#if !wpis.folder}
        <dt class="etykieta">Rozmiar</dt>
        <dd>{rozmiar(wpis.rozmiar)} <span class="opis">({wpis.rozmiar.toLocaleString('pl-PL')} B)</span></dd>
      {/if}
      <dt class="etykieta">Zmieniono</dt>
      <dd>{data(wpis.zmieniono)}</dd>
      <dt class="etykieta">Udział</dt>
      <dd>{udzial}</dd>
      <dt class="etykieta">Ścieżka</dt>
      <dd class="mono lamany">{wpis.sciezka}</dd>
      {#if !wpis.folder}
        <dt class="etykieta">SHA-256</dt>
        <dd class="suma">
          {#if sha}
            <code class="mono lamany">{sha}</code>
            <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" onclick={kopiujSume}>Kopiuj</Przycisk>
          {:else}
            <Przycisk rozmiar="maly" ikona="klucz" wylaczony={liczenie} onclick={oblicz}>
              {liczenie ? 'Liczenie…' : 'Oblicz SHA-256'}
            </Przycisk>
            {#if blad}<span class="blad">{blad}</span>{/if}
          {/if}
        </dd>
      {/if}
    </dl>
  {/if}
  {#snippet stopka()}
    <Przycisk onclick={() => (otwarte = false)}>Zamknij</Przycisk>
  {/snippet}
</Okno>

<style>
  .dane {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: var(--odstep-s) var(--odstep-l);
    margin: 0;
    align-items: baseline;
  }
  .dane dd {
    margin: 0;
    min-width: 0;
  }
  .lamany {
    word-break: break-all;
  }
  .suma {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--odstep-s);
  }
  .suma code {
    font-size: var(--czcionka-drobny);
  }
  .blad {
    color: var(--kolor-czerwony);
    font-size: var(--czcionka-maly);
  }
</style>
