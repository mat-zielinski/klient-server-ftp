<script lang="ts">
  // Strona linku publicznego /s/<id> (bez logowania): nazwa pliku lub folderu, rozmiar, data wygaśnięcia, pole
  // hasła gdy link go wymaga (oko + Caps Lock). Plik: duży przycisk Pobierz. Folder: okruszki, wchodzenie
  // w podfoldery, pobieranie pojedynczych plików i „Pobierz wszystko (ZIP)”. Po poprawnym haśle serwer ustawia
  // ciasteczko linku, więc pobieranie to zwykłe odnośniki. Wygasły link (410) albo nieistniejący (404) = komunikat.
  import { onMount } from 'svelte'
  import { BladApi, trescBledu } from '../api/klient'
  import type { Wpis } from '../api/typy'
  import { IKONA_RODZAJU } from '../ikony/ikony'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import { maDate } from '../komponenty/admin/narzedzia'
  import { adresPobraniaLinku, adresZipLinku, infoLinku, okruszki } from '../komponenty/admin/publiczny'
  import { KOD_BRAK_DOSTEPU, KOD_NIE_ZNALEZIONO, KOD_WYGASLO, KOD_ZAKAZ } from '../komponenty/admin/stale'
  import type { InfoPubliczne } from '../komponenty/admin/typy'
  import Ikona from '../komponenty/wspolne/Ikona.svelte'
  import PoleHasla from '../komponenty/wspolne/PoleHasla.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import { data, rozmiar } from '../stan/format'
  import { nawigacja } from '../stan/nawigacja.svelte'

  type Stan = 'wczytywanie' | 'haslo' | 'gotowe' | 'wygasl' | 'brak' | 'blad'

  const id = nawigacja.publiczny ?? ''
  let stan = $state<Stan>('wczytywanie')
  let info = $state<InfoPubliczne | null>(null)
  let nazwaKorzenia = $state('')
  let sciezka = $state('')
  let haslo = $state('')
  let bladHasla = $state('')
  let blad = $state('')
  let trwa = $state(false)

  const pliki = $derived<Wpis[]>(
    [...(info?.wpisy ?? [])].sort((a, b) => +b.folder - +a.folder || a.nazwa.localeCompare(b.nazwa, 'pl')),
  )
  const sciezkaOkruszkow = $derived(okruszki(nazwaKorzenia || info?.nazwa || '', sciezka))

  /** Wczytuje korzeń albo podfolder; `h` puste = poleganie na ciasteczku linku. */
  async function wczytaj(h: string, s: string) {
    trwa = true
    try {
      const nowe = await infoLinku(id, h, s || undefined)
      info = nowe
      sciezka = s
      if (!s) nazwaKorzenia = nowe.nazwa
      haslo = ''
      bladHasla = ''
      stan = 'gotowe'
    } catch (b) {
      obsluzBlad(b, h)
    } finally {
      trwa = false
    }
  }

  function obsluzBlad(b: unknown, h: string) {
    const kod = b instanceof BladApi ? b.kod : -1
    if (kod === KOD_BRAK_DOSTEPU || kod === KOD_ZAKAZ) {
      bladHasla = h ? (b instanceof BladApi && b.komunikat) || 'Złe hasło.' : ''
      stan = 'haslo'
    } else if (kod === KOD_WYGASLO) {
      stan = 'wygasl'
    } else if (kod === KOD_NIE_ZNALEZIONO) {
      stan = 'brak'
    } else {
      blad = trescBledu(b) || 'Nie udało się otworzyć linku.'
      stan = 'blad'
    }
  }

  function wyslijHaslo(e: Event) {
    e.preventDefault()
    if (!haslo) return
    void wczytaj(haslo, sciezka)
  }

  function otworz(s: string) {
    if (!trwa) void wczytaj('', s)
  }

  onMount(() => {
    if (!id) stan = 'brak'
    else void wczytaj('', '')
  })
</script>

<EkranSrodkowy podtytul={info?.folder ? 'Udostępniony folder' : 'Udostępniony plik'} szeroka={info?.folder ?? false}>
  {#if stan === 'wczytywanie'}
    <Pusto ikona="klepsydra" tytul="Otwieranie linku…" />
  {:else if stan === 'wygasl'}
    <Pusto ikona="zegar" tytul="Link wygasł"
      opis="Ten link stracił ważność albo wyczerpał limit pobrań. Poproś autora o nowy." />
  {:else if stan === 'brak'}
    <Pusto ikona="link" tytul="Nie ma takiego linku" opis="Sprawdź, czy adres jest pełny, albo poproś autora o nowy." />
  {:else if stan === 'blad'}
    <Pusto ikona="blad" tytul="Nie udało się otworzyć linku" opis={blad}>
      <Przycisk ikona="odswiez" onclick={() => wczytaj('', sciezka)}>Spróbuj ponownie</Przycisk>
    </Pusto>
  {:else if stan === 'haslo'}
    <form class="formularz" onsubmit={wyslijHaslo} novalidate>
      <div class="naglowek">
        <span class="duza-ikona"><Ikona nazwa="klodka" rozmiar="l" /></span>
        <div>
          <h2>Link chroniony hasłem</h2>
          <p class="opis">Hasło przekazał Ci autor linku.</p>
        </div>
      </div>
      <PoleHasla etykieta="Hasło linku" bind:value={haslo} ikona="klucz" autocomplete="off" autofocus
        blad={bladHasla || undefined} />
      <Przycisk rodzaj="glowny" typ="submit" ikona="klodka_otwarta" rozmiar="duzy" wylaczony={!haslo || trwa}>
        {trwa ? 'Sprawdzanie…' : 'Otwórz'}
      </Przycisk>
    </form>
  {:else if info}
    <div class="link">
      <div class="naglowek">
        <span class="duza-ikona"><Ikona nazwa={info.folder ? 'folder' : 'plik'} rozmiar="l" /></span>
        <div class="tytul">
          <h2 class="skroc" title={nazwaKorzenia || info.nazwa}>{nazwaKorzenia || info.nazwa}</h2>
          <p class="opis">
            {info.folder ? `Folder, ${pliki.length} pozycji` : rozmiar(info.rozmiar)}
            {#if maDate(info.wygasa)} · ważny do {data(info.wygasa)}{/if}
            {#if info.limit_pobran} · pobrania {info.pobrano ?? 0} z {info.limit_pobran}{/if}
          </p>
        </div>
      </div>

      {#if !info.folder}
        <a class="przycisk glowny duzy" href={adresPobraniaLinku(id)} download={info.nazwa}>
          <Ikona nazwa="pobierz" />Pobierz ({rozmiar(info.rozmiar)})
        </a>
      {:else}
        <div class="pasek-folderu">
          <nav class="okruszki" aria-label="Ścieżka folderu">
            {#each sciezkaOkruszkow as o, i (o.sciezka)}
              {#if i > 0}<Ikona nazwa="w_prawo" rozmiar="s" />{/if}
              {#if i === sciezkaOkruszkow.length - 1}
                <span class="okruszek biezacy skroc" aria-current="page">{o.nazwa}</span>
              {:else}
                <button type="button" class="okruszek skroc" onclick={() => otworz(o.sciezka)}>{o.nazwa}</button>
              {/if}
            {/each}
          </nav>
          <a class="przycisk zwykly" href={adresZipLinku(id, sciezka || undefined)} download>
            <Ikona nazwa="archiwum" rozmiar="s" />Pobierz wszystko (ZIP)
          </a>
        </div>
        {#if pliki.length === 0}
          <Pusto ikona="folder_otwarty" tytul="Folder jest pusty" />
        {:else}
          <ul class="lista" aria-busy={trwa}>
            {#each pliki as w (w.sciezka)}
              <li>
                <span class="ikona-wpisu" class:folder={w.folder}>
                  <Ikona nazwa={IKONA_RODZAJU[w.rodzaj] ?? (w.folder ? 'folder' : 'plik')} />
                </span>
                {#if w.folder}
                  <button type="button" class="nazwa wejdz skroc" title="Otwórz {w.nazwa}" onclick={() => otworz(w.sciezka)}>
                    {w.nazwa}
                  </button>
                  <Ikona nazwa="w_prawo" rozmiar="s" />
                {:else}
                  <span class="nazwa skroc" title={w.nazwa}>{w.nazwa}</span>
                  <span class="opis mono">{rozmiar(w.rozmiar)}</span>
                  <a class="przycisk duch maly sama-ikona" href={adresPobraniaLinku(id, w.sciezka)} download={w.nazwa}
                    title="Pobierz {w.nazwa}" aria-label="Pobierz {w.nazwa}">
                    <Ikona nazwa="pobierz" rozmiar="s" />
                  </a>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </div>
  {/if}
</EkranSrodkowy>

<style>
  .formularz,
  .link {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-xl);
    min-width: 0;
  }
  .tytul {
    min-width: 0;
  }
  h2 {
    margin: 0;
  }
  .duza-ikona {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-naglowek-strony);
    height: var(--wysokosc-naglowek-strony);
    border-radius: var(--promien-kafel);
    background: var(--kolor-tlo-info);
    color: var(--kolor-niebieski);
  }
  /* odnośniki w wyglądzie Przycisk (pobieranie zwykłym linkiem, z ciasteczkiem linku) */
  .przycisk {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--odstep-s);
    height: var(--wysokosc-przycisk);
    padding: 0 var(--odstep-xl);
    border-radius: var(--promien-przycisk);
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    white-space: nowrap;
    text-decoration: none;
    transition: background var(--czas-szybki), filter var(--czas-szybki);
  }
  .przycisk:hover {
    background: var(--kolor-przycisk-hover);
  }
  .przycisk:focus-visible {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
    outline-offset: var(--odstep-pozycje);
  }
  .glowny {
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .glowny:hover {
    background: var(--kolor-niebieski-tlo);
    filter: brightness(1.1);
  }
  .duzy {
    height: var(--wysokosc-przycisk-duzy);
    font-size: var(--czcionka-tekst);
  }
  .duch {
    background: transparent;
    border: var(--ramka);
  }
  .maly {
    height: var(--wysokosc-przycisk-maly);
  }
  .sama-ikona {
    flex: none;
    width: var(--wysokosc-przycisk-maly);
    padding: 0;
  }
  .pasek-folderu {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--odstep-m);
  }
  .okruszki {
    display: flex;
    align-items: center;
    gap: var(--odstep-xs);
    min-width: 0;
    flex: 1 1 auto;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .okruszek {
    max-width: var(--kolumna-l);
    padding: var(--odstep-pozycje) var(--odstep-xs);
    border: 0;
    border-radius: var(--promien-segment);
    background: none;
    color: var(--kolor-niebieski);
    font-size: var(--czcionka-maly);
    cursor: pointer;
  }
  .okruszek:hover {
    background: var(--kolor-przycisk);
  }
  .okruszek.biezacy {
    color: var(--kolor-tekst);
    font-weight: var(--czcionka-polgruby);
    cursor: default;
  }
  .lista {
    list-style: none;
    margin: 0;
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    max-height: var(--uklad-okno-maks-h);
    overflow: auto;
  }
  .lista[aria-busy='true'] {
    opacity: var(--uklad-suwak-przygaszenie);
  }
  .lista li {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-height: var(--wysokosc-wiersz-tabeli);
    padding: var(--odstep-xs) var(--odstep-l);
    color: var(--kolor-opis);
  }
  .lista li + li {
    border-top: var(--ramka);
  }
  .lista li:hover {
    background: var(--kolor-karta);
  }
  .ikona-wpisu {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  .ikona-wpisu.folder {
    color: var(--kolor-niebieski);
  }
  .nazwa {
    flex: 1 1 auto;
    color: var(--kolor-tekst);
  }
  .wejdz {
    padding: 0;
    border: 0;
    background: none;
    text-align: left;
    font-size: inherit;
    cursor: pointer;
  }
  .wejdz:hover {
    color: var(--kolor-niebieski);
    text-decoration: underline;
  }
</style>
