<script lang="ts">
  // Pasek narzędzi plików: jeden rząd o stałej wysokości, który nigdy się nie zawija.
  // Bez zaznaczenia: szukaj, sortowanie, widok lista / kafelki, tryb „Zaznacz”, odśwież, nowy folder, wgraj.
  // Z zaznaczeniem pasek zmienia się w pasek zaznaczenia: licznik, Pobierz ZIP, Usuń, Odznacz.
  // Gdy brakuje miejsca, kolejno: szukajka się zwęża, przyciski tracą podpisy (zostaje ikona z dymkiem
  // i etykietą dostępności), a na końcu rzadziej używane akcje przechodzą do menu „…”.
  import { flushSync, onMount } from 'svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Wybor from '../wspolne/Wybor.svelte'
  import MenuKontekstowe, { type PozycjaMenu } from '../wspolne/MenuKontekstowe.svelte'

  let {
    szukaj = $bindable(''),
    widok = $bindable('lista'),
    sortowanie = $bindable('nazwa:0'),
    wieleNaraz = $bindable(false),
    zapis,
    zaznaczono,
    wczytywanie,
    nowyFolder,
    wgrajPliki,
    odswiez,
    pobierzZip,
    usunZaznaczone,
    odznacz,
  }: {
    szukaj: string
    widok: 'lista' | 'kafelki'
    sortowanie: string
    wieleNaraz: boolean
    zapis: boolean
    zaznaczono: number
    wczytywanie: boolean
    nowyFolder: () => void
    wgrajPliki: () => void
    odswiez: () => void
    pobierzZip: () => void
    usunZaznaczone: () => void
    odznacz: () => void
  } = $props()

  const OPCJE_SORTU = [
    { wartosc: 'nazwa:0', etykieta: 'Nazwa A do Z' },
    { wartosc: 'nazwa:1', etykieta: 'Nazwa Z do A' },
    { wartosc: 'zmieniono:1', etykieta: 'Najnowsze' },
    { wartosc: 'zmieniono:0', etykieta: 'Najstarsze' },
    { wartosc: 'rozmiar:1', etykieta: 'Największe' },
    { wartosc: 'rozmiar:0', etykieta: 'Najmniejsze' },
  ]
  const BEZ_ZAPISU = 'Brak prawa zapisu w tym udziale'
  /** Etapy zwężania: 0 = pełne podpisy, 1 = same ikony, 2 = rzadsze akcje w menu „…” (w pasku zaznaczenia: sam licznik). */
  const OSTATNI_ETAP = 2

  let pasek = $state<HTMLDivElement | null>(null)
  let poleSzukania = $state<HTMLDivElement | null>(null)
  let przyciskWiecej = $state<HTMLDivElement | null>(null)
  let etap = $state(0)
  let menu = $state<{ x: number; y: number } | null>(null)

  const trybZaznaczenia = $derived(zaznaczono > 0)
  const napisZaznaczania = $derived(wieleNaraz ? 'Koniec zaznaczania' : 'Zaznacz')

  /** Dobiera najmniej ściśnięty etap, w którym zawartość mieści się w jednym rzędzie. */
  function dopasuj() {
    if (!pasek) return
    for (let e = 0; e <= OSTATNI_ETAP; e++) {
      etap = e
      flushSync()
      if (pasek.scrollWidth <= pasek.clientWidth) return
    }
  }

  onMount(() => {
    if (!pasek) return
    const obserwator = new ResizeObserver(() => dopasuj())
    obserwator.observe(pasek)
    return () => obserwator.disconnect()
  })

  // zmiana zawartości paska (tryb zaznaczenia, napis „Zaznacz”, prawo zapisu) = ponowne dopasowanie
  $effect(() => {
    void [trybZaznaczenia, wieleNaraz, zapis]
    queueMicrotask(dopasuj)
  })

  /** Przenosi fokus do pola szukania (skrót „/”, paleta poleceń); w trybie zaznaczenia najpierw odznacza. */
  export function skupSzukaj() {
    if (trybZaznaczenia) {
      zakonczZaznaczanie()
      flushSync()
      dopasuj()
    }
    const el = poleSzukania?.querySelector('input')
    el?.focus()
    el?.select()
  }

  function zakonczZaznaczanie() {
    odznacz()
    wieleNaraz = false
  }

  function otworzMenu() {
    const r = przyciskWiecej?.getBoundingClientRect()
    if (r) menu = { x: r.left, y: r.bottom }
  }

  const pozycjeMenu = $derived<PozycjaMenu[]>([
    { etykieta: 'Widok: lista', ikona: widok === 'lista' ? 'ok' : 'lista', onclick: () => (widok = 'lista') },
    { etykieta: 'Widok: kafelki', ikona: widok === 'kafelki' ? 'ok' : 'kafelki', onclick: () => (widok = 'kafelki') },
    ...OPCJE_SORTU.map((o, i) => ({
      etykieta: `Sortuj: ${o.etykieta}`,
      ikona: sortowanie === o.wartosc ? ('ok' as const) : ('sortuj' as const),
      separator: i === 0,
      onclick: () => (sortowanie = o.wartosc),
    })),
    { etykieta: napisZaznaczania, ikona: 'ok_kolo', separator: true, onclick: () => (wieleNaraz = !wieleNaraz) },
    { etykieta: 'Odśwież', ikona: 'odswiez', wylaczona: wczytywanie, onclick: odswiez },
    {
      etykieta: 'Nowy folder',
      ikona: 'folder_plus',
      wylaczona: !zapis,
      opis: zapis ? undefined : BEZ_ZAPISU,
      onclick: nowyFolder,
    },
  ])
</script>

<div class="pasek" class:ikony={etap >= 1} class:wasko={etap >= 2} bind:this={pasek} role="toolbar" aria-label="Narzędzia plików">
  {#if trybZaznaczenia}
    <span class="licznik" title="Zaznaczono: {zaznaczono}" aria-live="polite">
      <span class="napis-licznika">Zaznaczono:</span>
      <span class="liczba">{zaznaczono}</span>
    </span>
    <span class="odstep"></span>
    <Przycisk ikona="archiwum" tytul="Pobierz zaznaczone jako ZIP" onclick={pobierzZip}>Pobierz ZIP</Przycisk>
    <Przycisk rodzaj="niebezpieczny" ikona="kosz" wylaczony={!zapis} tytul={zapis ? 'Usuń zaznaczone' : BEZ_ZAPISU}
      onclick={usunZaznaczone}>Usuń</Przycisk>
    <Przycisk rodzaj="duch" ikona="zamknij" tytul="Odznacz wszystko (Esc)" onclick={zakonczZaznaczanie}>Odznacz</Przycisk>
  {:else}
    <div class="szukaj" bind:this={poleSzukania}>
      <Pole etykieta="Szukaj" ukryjEtykiete typ="search" ikona="szukaj" bind:value={szukaj}
        podpowiedz="Szukaj w tym folderze i podfolderach (/)" />
    </div>

    {#if etap < 2}
      <div class="sort">
        <Wybor etykieta="Sortowanie" ukryjEtykiete opcje={OPCJE_SORTU} bind:value={sortowanie} />
      </div>

      <div class="segment" role="group" aria-label="Widok">
        <span class="opcja" class:aktywny={widok === 'lista'}>
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="lista" tytul="Lista" onclick={() => (widok = 'lista')} />
        </span>
        <span class="opcja" class:aktywny={widok === 'kafelki'}>
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="kafelki" tytul="Kafelki" onclick={() => (widok = 'kafelki')} />
        </span>
      </div>

      <Przycisk rodzaj="duch" ikona="ok_kolo" tytul="Zaznaczanie wielu (Ctrl+A zaznacza wszystko)"
        onclick={() => (wieleNaraz = !wieleNaraz)}>{napisZaznaczania}</Przycisk>
    {/if}

    <span class="odstep"></span>

    {#if etap < 2}
      <Przycisk rodzaj="duch" ikona="odswiez" tytul="Odśwież" wylaczony={wczytywanie} onclick={odswiez} />
      <Przycisk ikona="folder_plus" wylaczony={!zapis} tytul={zapis ? 'Nowy folder' : BEZ_ZAPISU} onclick={nowyFolder}>
        Nowy folder
      </Przycisk>
    {/if}
    <Przycisk rodzaj="glowny" ikona="wgraj" wylaczony={!zapis} tytul={zapis ? 'Wgraj pliki' : BEZ_ZAPISU} onclick={wgrajPliki}>
      Wgraj
    </Przycisk>
    {#if etap >= 2}
      <div class="wiecej" bind:this={przyciskWiecej}>
        <Przycisk rodzaj="duch" ikona="wiecej" tytul="Więcej: widok, sortowanie, zaznaczanie, odśwież, nowy folder"
          onclick={otworzMenu} />
      </div>
    {/if}
  {/if}
</div>

{#if menu}
  <MenuKontekstowe x={menu.x} y={menu.y} pozycje={pozycjeMenu} zamknij={() => (menu = null)} />
{/if}

<style>
  .pasek {
    display: flex;
    align-items: center;
    flex-wrap: nowrap;
    gap: var(--odstep-m);
    height: var(--wysokosc-przycisk);
    min-width: 0;
    /* nadmiar przycięty, ale obwódka fokusu (outline z odsunięciem) zostaje widoczna */
    overflow: clip;
    overflow-clip-margin: calc(var(--odstep-ramka) * 3);
  }
  .szukaj {
    flex: 1 1 var(--uklad-pole-w);
    min-width: var(--kolumna-m);
    max-width: calc(var(--uklad-pole-w) * 1.5);
  }
  .sort {
    flex: none;
    width: calc(var(--kolumna-l) * 0.8);
  }
  .segment {
    display: inline-flex;
    flex: none;
    padding: var(--odstep-pozycje);
    gap: var(--odstep-pozycje);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .opcja,
  .wiecej {
    display: inline-flex;
    flex: none;
  }
  .opcja :global(.przycisk) {
    border-color: transparent;
  }
  .opcja.aktywny :global(.przycisk) {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
  }
  .odstep {
    flex: 1 1 0;
    min-width: 0;
  }
  .licznik {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-s);
    flex: none;
    height: var(--wysokosc-przycisk);
    padding: 0 var(--odstep-xl);
    box-sizing: border-box;
    border-radius: var(--promien-przycisk);
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    white-space: nowrap;
  }
  .napis-licznika {
    font-family: var(--czcionka-ui);
  }
  /* etap „same ikony”: podpis zostaje dla czytnika ekranu, dymek daje title przycisku */
  .ikony :global(.przycisk:has(.napis)) {
    width: var(--wysokosc-przycisk);
    padding: 0;
  }
  .ikony :global(.przycisk .napis),
  .wasko .napis-licznika {
    position: absolute;
    width: var(--odstep-ramka);
    height: var(--odstep-ramka);
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
</style>
