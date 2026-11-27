<script lang="ts">
  // Podgląd pliku w oknie i galeria: obraz (przybliżanie, pełny ekran, wstępne ładowanie sąsiednich), wideo,
  // dźwięk, PDF (iframe), tekst. Strzałki ← → i licznik „3 / 120” po plikach z podglądem.
  import type { Wpis } from '../../api/typy'
  import { adresPobrania } from '../../api/klient'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import { rozmiar } from '../../stan/format'
  import { pobierzAdres } from './narzedzia'
  import { adresMiniatury } from './operacje'
  import { maPodglad } from './rodzaje'
  import { LIMIT_PODGLADU_TEKSTU } from './stale'
  import PodgladObrazu from './PodgladObrazu.svelte'

  let {
    otwarte = $bindable(false),
    wpis = $bindable(null),
    udzial,
    pliki,
  }: { otwarte: boolean; wpis: Wpis | null; udzial: string; pliki: Wpis[] } = $props()

  const adres = $derived(wpis ? adresPobrania(udzial, wpis.sciezka, true) : '')
  const zPodgladem = $derived(pliki.filter(maPodglad))
  const indeks = $derived(wpis ? zPodgladem.findIndex((p) => p.sciezka === wpis!.sciezka) : -1)

  let tekst = $state<string | null>(null)
  let bladTekstu = $state('')
  let scena = $state<HTMLDivElement | null>(null)
  let pelnyEkran = $state(false)

  $effect(() => {
    tekst = null
    bladTekstu = ''
    if (!otwarte || !wpis || wpis.rodzaj !== 'tekst') return
    if (wpis.rozmiar > LIMIT_PODGLADU_TEKSTU) {
      bladTekstu = `Plik jest za duży na podgląd (${rozmiar(wpis.rozmiar)}). Pobierz go.`
      return
    }
    const kontroler = new AbortController()
    fetch(adres, { signal: kontroler.signal, credentials: 'same-origin' })
      .then(async (r) => {
        if (!r.ok) throw new Error(`Nie udało się wczytać pliku (${r.status}).`)
        tekst = await r.text()
      })
      .catch((e: Error) => {
        if (e.name !== 'AbortError') bladTekstu = e.message
      })
    return () => kontroler.abort()
  })

  // wstępne ładowanie sąsiednich obrazów galerii
  $effect(() => {
    if (!otwarte || indeks < 0 || zPodgladem.length < 2) return
    for (const krok of [1, -1]) {
      const s = zPodgladem[(indeks + krok + zPodgladem.length) % zPodgladem.length]
      if (s.rodzaj === 'obraz') new Image().src = adresPobrania(udzial, s.sciezka, true)
    }
  })

  $effect(() => {
    const naZmiane = () => (pelnyEkran = !!document.fullscreenElement)
    document.addEventListener('fullscreenchange', naZmiane)
    return () => document.removeEventListener('fullscreenchange', naZmiane)
  })

  $effect(() => {
    if (!otwarte && document.fullscreenElement) void document.exitFullscreen()
  })

  function przejdz(krok: number) {
    if (indeks < 0 || zPodgladem.length < 2) return
    wpis = zPodgladem[(indeks + krok + zPodgladem.length) % zPodgladem.length]
  }

  function przelaczPelnyEkran() {
    if (document.fullscreenElement) void document.exitFullscreen()
    else void scena?.requestFullscreen?.()
  }

  function klawisz(e: KeyboardEvent) {
    if (!otwarte) return
    const cel = e.target as HTMLElement
    if (cel.tagName === 'VIDEO' || cel.tagName === 'AUDIO') return
    if (e.key === 'ArrowRight') przejdz(1)
    else if (e.key === 'ArrowLeft') przejdz(-1)
    else if (e.key === 'f' && wpis?.rodzaj === 'obraz') przelaczPelnyEkran()
  }
</script>

<svelte:window onkeydown={klawisz} />

<Okno tytul={wpis?.nazwa ?? 'Podgląd'} bind:otwarte szerokie>
  {#if wpis}
    <div class="podglad" class:pelny={pelnyEkran} bind:this={scena}>
      {#key wpis.sciezka}
        {#if wpis.rodzaj === 'obraz'}
          <PodgladObrazu {adres} miniatura={adresMiniatury(udzial, wpis.sciezka, 'm')} nazwa={wpis.nazwa} />
        {:else if wpis.rodzaj === 'wideo'}
          <!-- svelte-ignore a11y_media_has_caption -->
          <video src={adres} controls autoplay preload="metadata"></video>
        {:else if wpis.rodzaj === 'dzwiek'}
          <audio src={adres} controls autoplay preload="metadata"></audio>
        {:else if wpis.rodzaj === 'pdf'}
          <iframe src={adres} title={wpis.nazwa}></iframe>
        {:else if wpis.rodzaj === 'tekst'}
          {#if bladTekstu}
            <Pusto ikona="tekst" tytul="Brak podglądu" opis={bladTekstu} />
          {:else if tekst === null}
            <p class="opis">Wczytywanie…</p>
          {:else}
            <pre class="tekst">{tekst}</pre>
          {/if}
        {:else}
          <Pusto ikona="plik" tytul="Brak podglądu" opis="Ten rodzaj pliku można tylko pobrać." />
        {/if}
      {/key}
      {#if pelnyEkran && zPodgladem.length > 1}
        <button class="strzalka lewa" aria-label="Poprzedni" onclick={() => przejdz(-1)}><Ikona nazwa="strzalka_lewo" /></button>
        <button class="strzalka prawa" aria-label="Następny" onclick={() => przejdz(1)}><Ikona nazwa="strzalka" /></button>
        <span class="licznik-pelny">{indeks + 1} / {zPodgladem.length}</span>
      {/if}
    </div>
  {/if}
  {#snippet stopka()}
    {#if zPodgladem.length > 1 && indeks >= 0}
      <Przycisk rodzaj="duch" ikona="strzalka_lewo" tytul="Poprzedni (strzałka w lewo)" onclick={() => przejdz(-1)} />
      <span class="opis licznik-pozycji">{indeks + 1} / {zPodgladem.length}</span>
      <Przycisk rodzaj="duch" ikona="strzalka" tytul="Następny (strzałka w prawo)" onclick={() => przejdz(1)} />
    {/if}
    <span class="odstep"></span>
    {#if wpis?.rodzaj === 'obraz'}
      <Przycisk rodzaj="duch" ikona="zewnetrzny" tytul="Pełny ekran (F)" onclick={przelaczPelnyEkran}>Pełny ekran</Przycisk>
    {:else if wpis}
      <Przycisk rodzaj="duch" ikona="zewnetrzny" onclick={() => window.open(adres, '_blank', 'noopener')}>
        Otwórz w karcie
      </Przycisk>
    {/if}
    {#if wpis}
      <Przycisk rodzaj="glowny" ikona="pobierz" onclick={() => wpis && pobierzAdres(adresPobrania(udzial, wpis.sciezka))}>
        Pobierz
      </Przycisk>
    {/if}
  {/snippet}
</Okno>

<style>
  .podglad {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(var(--wysokosc-naglowek-strony) * 4);
  }
  .podglad.pelny {
    background: var(--kolor-tlo);
  }
  video {
    max-width: 100%;
    max-height: 70vh;
    border-radius: var(--promien-segment);
    background: var(--kolor-mono);
  }
  audio {
    width: 100%;
  }
  iframe {
    width: 100%;
    height: 70vh;
    border: var(--ramka);
    border-radius: var(--promien-segment);
    background: var(--kolor-na-ciemnym);
  }
  .tekst {
    width: 100%;
    max-height: 70vh;
    overflow: auto;
    margin: 0;
    padding: var(--odstep-l);
    background: var(--kolor-mono);
    border: var(--ramka);
    border-radius: var(--promien-segment);
    font-size: var(--czcionka-maly);
    white-space: pre-wrap;
    word-break: break-word;
  }
  .odstep {
    flex: 1;
  }
  .licznik-pozycji,
  .licznik-pelny {
    font-family: var(--czcionka-mono);
  }
  .strzalka {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: var(--wysokosc-przycisk-duzy);
    height: var(--wysokosc-przycisk-duzy);
    border: 0;
    border-radius: var(--promien-kolo);
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
    font-size: var(--czcionka-duzy);
    line-height: 1;
  }
  .lewa {
    left: var(--odstep-sekcja);
  }
  .prawa {
    right: var(--odstep-sekcja);
  }
  .licznik-pelny {
    position: absolute;
    bottom: var(--odstep-sekcja);
    left: 50%;
    transform: translateX(-50%);
    padding: var(--odstep-xs) var(--odstep-m);
    border-radius: var(--promien-pigulka);
    background: var(--kolor-panel);
  }
</style>
