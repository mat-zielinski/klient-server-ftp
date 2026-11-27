<script lang="ts">
  // Pasek narzędzi plików: szukaj, widok lista / kafelki, sortowanie, nowy folder, wgraj, odśwież,
  // tryb zaznaczania wielu oraz akcje dla zaznaczonych (ZIP, usuń).
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Wybor from '../wspolne/Wybor.svelte'

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

  let poleSzukania = $state<HTMLDivElement | null>(null)

  /** Przenosi fokus do pola szukania (skrót „/”, paleta poleceń). */
  export function skupSzukaj() {
    const el = poleSzukania?.querySelector('input')
    el?.focus()
    el?.select()
  }
</script>

<div class="pasek">
  <div class="szukaj" bind:this={poleSzukania}>
    <Pole etykieta="Szukaj" ukryjEtykiete typ="search" ikona="szukaj" bind:value={szukaj}
      podpowiedz="Szukaj w tym folderze i podfolderach (/)" />
  </div>

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
    onclick={() => (wieleNaraz = !wieleNaraz)}>
    {wieleNaraz ? 'Koniec zaznaczania' : 'Zaznacz'}
  </Przycisk>

  {#if zaznaczono > 0}
    <span class="licznik" title="Zaznaczone">{zaznaczono}</span>
    <Przycisk ikona="archiwum" onclick={pobierzZip}>Pobierz ZIP</Przycisk>
    <Przycisk rodzaj="niebezpieczny" ikona="kosz" wylaczony={!zapis} tytul={zapis ? 'Usuń zaznaczone' : BEZ_ZAPISU}
      onclick={usunZaznaczone}>Usuń</Przycisk>
  {/if}

  <span class="odstep"></span>

  <Przycisk rodzaj="duch" ikona="odswiez" tytul="Odśwież" wylaczony={wczytywanie} onclick={odswiez} />
  <Przycisk ikona="folder_plus" wylaczony={!zapis} tytul={zapis ? 'Nowy folder' : BEZ_ZAPISU} onclick={nowyFolder}>
    Nowy folder
  </Przycisk>
  <Przycisk rodzaj="glowny" ikona="wgraj" wylaczony={!zapis} tytul={zapis ? 'Wgraj pliki' : BEZ_ZAPISU} onclick={wgrajPliki}>
    Wgraj
  </Przycisk>
</div>

<style>
  .pasek {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--odstep-m);
  }
  .szukaj {
    flex: 1 1 var(--uklad-pole-w);
    max-width: calc(var(--uklad-pole-w) * 1.5);
  }
  .sort {
    width: calc(var(--kolumna-l) * 0.8);
  }
  .segment {
    display: inline-flex;
    padding: var(--odstep-pozycje);
    gap: var(--odstep-pozycje);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .opcja {
    display: inline-flex;
  }
  .opcja :global(.przycisk) {
    border-color: transparent;
  }
  .opcja.aktywny :global(.przycisk) {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
  }
  .odstep {
    flex: 1;
  }
</style>
