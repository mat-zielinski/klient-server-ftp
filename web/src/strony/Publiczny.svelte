<script lang="ts">
  // Strona linku publicznego /s/<id> (bez logowania): nazwa pliku lub folderu, rozmiar, data wygaśnięcia, pole
  // hasła gdy link go wymaga, przycisk Pobierz; dla folderu lista plików z pobieraniem pojedynczych plików.
  // Wygasły link (410) albo nieistniejący (404) = jasny komunikat zamiast formularza.
  import { onMount } from 'svelte'
  import { BladApi, trescBledu } from '../api/klient'
  import type { Wpis } from '../api/typy'
  import { IKONA_RODZAJU } from '../ikony/ikony'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import { maDate } from '../komponenty/admin/narzedzia'
  import { adresPobraniaLinku, infoLinku, pobierzZHaslem } from '../komponenty/admin/publiczny'
  import { KOD_BRAK_DOSTEPU, KOD_NIE_ZNALEZIONO, KOD_WYGASLO, KOD_ZAKAZ } from '../komponenty/admin/stale'
  import type { InfoPubliczne } from '../komponenty/admin/typy'
  import Ikona from '../komponenty/wspolne/Ikona.svelte'
  import Pole from '../komponenty/wspolne/Pole.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import { data, rozmiar } from '../stan/format'
  import { nawigacja } from '../stan/nawigacja.svelte'

  type Stan = 'wczytywanie' | 'haslo' | 'gotowe' | 'wygasl' | 'brak' | 'blad'

  const id = nawigacja.publiczny ?? ''
  let stan = $state<Stan>('wczytywanie')
  let info = $state<InfoPubliczne | null>(null)
  let haslo = $state('')
  let zatwierdzoneHaslo = $state('')
  let bladHasla = $state('')
  let blad = $state('')
  let trwa = $state(false)
  let pobierane = $state('')

  const pliki = $derived<Wpis[]>(
    [...(info?.wpisy ?? [])].sort((a, b) => +b.folder - +a.folder || a.nazwa.localeCompare(b.nazwa, 'pl')),
  )

  async function wczytaj(h: string) {
    trwa = true
    try {
      info = await infoLinku(id, h)
      zatwierdzoneHaslo = h
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
    void wczytaj(haslo)
  }

  async function pobierz(nazwa: string, sciezka?: string) {
    if (!zatwierdzoneHaslo) {
      location.href = adresPobraniaLinku(id, sciezka)
      return
    }
    pobierane = sciezka ?? nazwa
    try {
      await pobierzZHaslem(id, zatwierdzoneHaslo, nazwa, sciezka)
    } catch (b) {
      obsluzBlad(b, zatwierdzoneHaslo)
    } finally {
      pobierane = ''
    }
  }

  onMount(() => {
    if (!id) stan = 'brak'
    else void wczytaj('')
  })
</script>

<EkranSrodkowy podtytul={info?.folder ? "Udostępniony folder" : "Udostępniony plik"} szeroka={info?.folder ?? false}>
  {#if stan === 'wczytywanie'}
    <Pusto ikona="klepsydra" tytul="Otwieranie linku…" />
  {:else if stan === 'wygasl'}
    <Pusto ikona="zegar" tytul="Link wygasł"
      opis="Ten link stracił ważność albo wyczerpał limit pobrań. Poproś autora o nowy." />
  {:else if stan === 'brak'}
    <Pusto ikona="link" tytul="Nie ma takiego linku" opis="Sprawdź, czy adres jest pełny, albo poproś autora o nowy." />
  {:else if stan === 'blad'}
    <Pusto ikona="blad" tytul="Nie udało się otworzyć linku" opis={blad}>
      <Przycisk ikona="odswiez" onclick={() => wczytaj(zatwierdzoneHaslo)}>Spróbuj ponownie</Przycisk>
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
      <Pole etykieta="Hasło" typ="password" bind:value={haslo} ikona="klucz" autocomplete="off" autofocus
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
          <h2 class="skroc" title={info.nazwa}>{info.nazwa}</h2>
          <p class="opis">
            {info.folder ? `Folder, ${pliki.length} pozycji` : rozmiar(info.rozmiar)}
            {#if maDate(info.wygasa)} · ważny do {data(info.wygasa)}{/if}
            {#if info.limit_pobran} · pobrania {info.pobrano ?? 0} z {info.limit_pobran}{/if}
          </p>
        </div>
      </div>

      {#if !info.folder}
        <Przycisk rodzaj="glowny" ikona="pobierz" rozmiar="duzy" wylaczony={!!pobierane} onclick={() => pobierz(info!.nazwa)}>
          {pobierane ? 'Pobieranie…' : `Pobierz (${rozmiar(info.rozmiar)})`}
        </Przycisk>
        {#if zatwierdzoneHaslo}
          <Komunikat rodzaj="info">Plik pobiera się w tle; przeglądarka zapisze go po pobraniu całości.</Komunikat>
        {/if}
      {:else if pliki.length === 0}
        <Pusto ikona="folder_otwarty" tytul="Folder jest pusty" />
      {:else}
        <ul class="lista">
          {#each pliki as w (w.sciezka)}
            <li>
              <span class="ikona-wpisu"><Ikona nazwa={IKONA_RODZAJU[w.rodzaj] ?? 'plik'} /></span>
              <span class="nazwa skroc" title={w.nazwa}>{w.nazwa}</span>
              {#if w.folder}
                <span class="opis">Folder</span>
              {:else}
                <span class="opis mono">{rozmiar(w.rozmiar)}</span>
                <Przycisk rodzaj="duch" rozmiar="maly" ikona="pobierz" tytul="Pobierz {w.nazwa}"
                  wylaczony={pobierane === w.sciezka} onclick={() => pobierz(w.nazwa, w.sciezka)} />
              {/if}
            </li>
          {/each}
        </ul>
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
  .lista {
    list-style: none;
    margin: 0;
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    max-height: var(--uklad-okno-maks-h);
    overflow: auto;
  }
  .lista li {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-height: var(--wysokosc-wiersz-tabeli);
    padding: var(--odstep-xs) var(--odstep-l);
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
  .nazwa {
    flex: 1 1 auto;
  }
</style>
