<script lang="ts">
  // Strona Pliki (wzór FileBrowser Quantum): udziały, okruszki, pasek narzędzi, wirtualizowana lista / kafelki,
  // podgląd i galeria, akcje z menu kontekstowego, wgrywanie przeciągnij i upuść z wznawianiem, „Moje pobrania”.
  // Położenie w hashu: #/pliki/<udzial>/<sciezka>. Logika w komponenty/pliki/*.
  import { onMount } from 'svelte'
  import { BladApi, adresPobrania, trescBledu } from '../api/klient'
  import type { UdzialInfo, Wpis } from '../api/typy'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import Szkielet from '../komponenty/wspolne/Szkielet.svelte'
  import Wybor from '../komponenty/wspolne/Wybor.svelte'
  import ListaUdzialow from '../komponenty/pliki/ListaUdzialow.svelte'
  import Okruszki from '../komponenty/pliki/Okruszki.svelte'
  import PasekNarzedzi from '../komponenty/pliki/PasekNarzedzi.svelte'
  import TabelaPlikow from '../komponenty/pliki/TabelaPlikow.svelte'
  import KafelkiPlikow from '../komponenty/pliki/KafelkiPlikow.svelte'
  import Podglad from '../komponenty/pliki/Podglad.svelte'
  import AkcjePlikow from '../komponenty/pliki/AkcjePlikow.svelte'
  import ListaWgrywania from '../komponenty/pliki/ListaWgrywania.svelte'
  import MojePobrania from '../komponenty/pliki/MojePobrania.svelte'
  import StrefaUpuszczania from '../komponenty/pliki/StrefaUpuszczania.svelte'
  import * as op from '../komponenty/pliki/operacje'
  import { normalizuj, pobierzAdres, rodzic } from '../komponenty/pliki/narzedzia'
  import { maPodglad } from '../komponenty/pliki/rodzaje'
  import { KolejkaWgrywania } from '../komponenty/pliki/wgrywanie.svelte'
  import { toPliki, zListy, zUpuszczenia, type PlikDoWgrania } from '../komponenty/pliki/upuszczanie'
  import type { ZdarzeniaWpisow } from '../komponenty/pliki/zdarzenia'
  import { obslugaKlawiszy } from '../komponenty/pliki/klawiatura'
  import { Zaznaczenie } from '../komponenty/pliki/zaznaczenie.svelte'
  import { zarejestrujPoleceniaPlikow } from '../komponenty/pliki/polecenia'
  import { KLUCZ_WIDOKU, OPOZNIENIE_SZUKANIA_MS } from '../komponenty/pliki/stale'
  import { nawigacja } from '../stan/nawigacja.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { uklad } from '../stan/uklad.svelte'

  // --- stan ---
  let udzialy = $state<UdzialInfo[] | null>(null)
  let bladUdzialow = $state('')
  let wpisy = $state<Wpis[]>([])
  let wczytano = $state(false)
  let wczytywanie = $state(false)
  let blad = $state<{ kod: number; tekst: string } | null>(null)
  let szukaj = $state('')
  let wyniki = $state<Wpis[] | null>(null)
  let szukanie = $state(false)
  let widok = $state<'lista' | 'kafelki'>(localStorage.getItem(KLUCZ_WIDOKU) === 'kafelki' ? 'kafelki' : 'lista')
  let sortowanie = $state('nazwa:0')
  let wieleNaraz = $state(false)
  let przeciaganie = $state(0)
  let obszar = $state<HTMLDivElement | null>(null)
  let poleWgrywania = $state<HTMLInputElement | null>(null)
  let podglad = $state<{ otwarte: boolean; wpis: Wpis | null }>({ otwarte: false, wpis: null })

  let akcje = $state<ReturnType<typeof AkcjePlikow> | null>(null)
  let pasek = $state<ReturnType<typeof PasekNarzedzi> | null>(null)
  let lista = $state<ReturnType<typeof TabelaPlikow> | null>(null)
  let kafelki = $state<ReturnType<typeof KafelkiPlikow> | null>(null)

  const kolejka = new KolejkaWgrywania()

  // --- pochodne ---
  const udzial = $derived(nawigacja.parametry.udzial ?? '')
  const sciezka = $derived(normalizuj(nawigacja.parametry.sciezka ?? '/'))
  const infoUdzialu = $derived(udzialy?.find((u) => u.nazwa === udzial))
  const zapis = $derived(!!infoUdzialu && infoUdzialu.prawo === 'zapis' && !infoUdzialu.tylko_odczyt)
  const sortKlucz = $derived(sortowanie.split(':')[0] as op.KluczSortu)
  const malejaco = $derived(sortowanie.endsWith(':1'))
  const widoczne = $derived(wyniki ?? wpisy)
  const zapytanie = $derived(wyniki ? szukaj.trim() : '')
  const opcjeUdzialow = $derived((udzialy ?? []).map((u) => ({ wartosc: u.nazwa, etykieta: u.nazwa })))

  const zazn = new Zaznaczenie(() => widoczne)

  // --- wczytywanie ---
  onMount(() => {
    void wczytajUdzialy()
    kolejka.naGotowe = (z) => {
      if (z.udzial === udzial && op.wFolderze(z, sciezka)) void wczytaj()
    }
    return zarejestrujPoleceniaPlikow({
      zapis: () => zapis,
      nowyFolder: () => akcje?.nowyFolder(),
      wgraj: () => poleWgrywania?.click(),
      szukaj: () => pasek?.skupSzukaj(),
      widok: (w) => (widok = w),
      zaznaczWszystko: () => zazn.wszystko(),
    })
  })

  async function wczytajUdzialy() {
    try {
      udzialy = await op.udzialy()
      bladUdzialow = ''
    } catch (e) {
      bladUdzialow = trescBledu(e)
    }
  }

  // brak udziału w hashu (albo nieznany) = pierwszy dostępny
  $effect(() => {
    if (!udzialy?.length || udzialy.some((u) => u.nazwa === udzial)) return
    nawigacja.zastap('pliki', { udzial: udzialy[0].nazwa, sciezka: '/' })
  })

  let numerWczytania = 0
  async function wczytaj() {
    if (!udzial) return
    const numer = ++numerWczytania
    wczytywanie = true
    try {
      const nowe = await op.lista(udzial, sciezka, sortKlucz, malejaco)
      if (numer !== numerWczytania) return
      wpisy = nowe
      blad = null
    } catch (e) {
      if (numer !== numerWczytania) return
      wpisy = []
      blad = { kod: e instanceof BladApi ? e.kod : 0, tekst: trescBledu(e) }
    } finally {
      if (numer === numerWczytania) {
        wczytywanie = false
        wczytano = true
      }
    }
  }

  // zmiana udziału, folderu albo sortowania = nowa lista
  $effect(() => {
    void [udzial, sciezka, sortKlucz, malejaco]
    if (infoUdzialu) void wczytaj()
  })

  // nowy folder = czyste zaznaczenie, wyszukiwanie i szkielet zamiast starej listy
  $effect(() => {
    void [udzial, sciezka]
    zazn.wyczysc()
    szukaj = ''
    wczytano = false
  })

  $effect(() => localStorage.setItem(KLUCZ_WIDOKU, widok))

  // wyszukiwanie z opóźnieniem (indeks, a gdy niedostępny zwykłe szukanie)
  $effect(() => {
    const q = szukaj.trim()
    if (!q) {
      wyniki = null
      szukanie = false
      return
    }
    const u = udzial
    const s = sciezka
    szukanie = true
    const zegar = setTimeout(async () => {
      try {
        const r = await op.szukaj(u, s, q)
        if (szukaj.trim() === q) wyniki = r
      } catch (e) {
        powiadom(trescBledu(e), 'blad')
      } finally {
        if (szukaj.trim() === q) szukanie = false
      }
    }, OPOZNIENIE_SZUKANIA_MS)
    return () => clearTimeout(zegar)
  })

  // --- nawigacja i otwieranie ---
  function idzDo(s: string, u = udzial) {
    nawigacja.idz('pliki', { udzial: u, sciezka: s })
  }

  function otworz(w: Wpis) {
    if (w.folder) idzDo(w.sciezka)
    else if (maPodglad(w)) podglad = { otwarte: true, wpis: w }
    else pobierzAdres(adresPobrania(udzial, w.sciezka))
  }

  const zdarzenia: ZdarzeniaWpisow = {
    klik: (_w, i, e) => zazn.klik(i, e, wieleNaraz),
    przelacz: (_w, i) => zazn.przelacz(i),
    otworz,
    menu(_w, i, e) {
      e.preventDefault()
      e.stopPropagation()
      void akcje?.otworzMenu(e.clientX, e.clientY, zazn.doMenu(i))
    },
    sortuj: (k, m) => (sortowanie = `${k}:${m ? 1 : 0}`),
  }

  // --- klawiatura ---
  function przesun(krok: number, rozszerz: boolean) {
    const i = zazn.przesun(krok, rozszerz)
    if (i >= 0) (widok === 'kafelki' ? kafelki : lista)?.przewinDo(i)
  }

  const klawisz = obslugaKlawiszy({
    kafelki: () => widok === 'kafelki',
    liczba: () => widoczne.length,
    wKolumnie: () => (widok === 'kafelki' ? (kafelki?.wWierszu() ?? 1) : 1),
    przesun,
    otworz: () => zazn.biezacy && otworz(zazn.biezacy),
    usun: () => akcje?.usun(zazn.wybrane),
    zmienNazwe: () => zazn.biezacy && akcje?.zmienNazwe(zazn.biezacy),
    wyzej: () => sciezka !== '/' && idzDo(rodzic(sciezka)),
    odznacz: () => zazn.odznacz(),
    zaznaczWszystko: () => zazn.wszystko(),
    menu: () => {
      const w = zazn.biezacy
      if (!w) return
      const r = obszar?.querySelector(`[data-indeks="${zazn.kursor}"]`)?.getBoundingClientRect()
      void akcje?.otworzMenu(r ? r.left + r.width / 3 : 0, r ? r.bottom : 0, zazn.doMenu(zazn.kursor))
    },
  })

  function menuTla(e: MouseEvent) {
    if ((e.target as HTMLElement).closest('[data-indeks]')) return
    e.preventDefault()
    void akcje?.otworzMenu(e.clientX, e.clientY, null)
  }

  // --- wgrywanie ---
  function wgraj(pliki: PlikDoWgrania[]) {
    if (!pliki.length) return
    if (!zapis) powiadom('Brak prawa zapisu w tym udziale.', 'ostrz')
    else kolejka.dodaj(pliki, udzial, sciezka)
  }

  function wybranoPliki(e: Event) {
    const el = e.currentTarget as HTMLInputElement
    wgraj(zListy(el.files))
    el.value = ''
  }

  function naDragEnter(e: DragEvent) {
    if (!toPliki(e)) return
    e.preventDefault()
    przeciaganie++
  }
  function naDragOver(e: DragEvent) {
    if (!toPliki(e)) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = zapis ? 'copy' : 'none'
  }
  async function naDrop(e: DragEvent) {
    if (!toPliki(e) || !e.dataTransfer) return
    e.preventDefault()
    przeciaganie = 0
    try {
      wgraj(await zUpuszczenia(e.dataTransfer))
    } catch (b) {
      powiadom(`Nie udało się odczytać upuszczonych plików: ${trescBledu(b)}`, 'blad')
    }
  }

  function poZmianie() {
    zazn.odznacz()
    void wczytaj()
  }
</script>

<div class="strona" class:waska={uklad.ponizej('waski')}>
  {#if !uklad.ponizej('waski')}
    <aside class="kolumna">
      {#if udzialy}
        <ListaUdzialow {udzialy} wybrany={udzial} wybierz={(u) => idzDo('/', u)} />
      {:else if !bladUdzialow}
        <Szkielet linie={3} />
      {/if}
    </aside>
  {/if}

  <section
    class="glowna"
    aria-label="Pliki"
    ondragenter={naDragEnter}
    ondragover={naDragOver}
    ondragleave={() => (przeciaganie = Math.max(0, przeciaganie - 1))}
    ondrop={naDrop}
  >
    {#if bladUdzialow}
      <Pusto ikona="blad" tytul="Nie udało się wczytać udziałów" opis={bladUdzialow}>
        <Przycisk ikona="odswiez" onclick={wczytajUdzialy}>Ponów</Przycisk>
      </Pusto>
    {:else if udzialy && udzialy.length === 0}
      <Pusto ikona="dysk" tytul="Brak udziałów" opis="Nie masz dostępu do żadnego udziału. Poproś administratora o prawa." />
    {:else}
      <div class="naglowek">
        {#if uklad.ponizej('waski')}
          <div class="wybor-udzialu">
            <Wybor etykieta="Udział" ukryjEtykiete opcje={opcjeUdzialow} value={udzial} onchange={(u) => idzDo('/', u)} />
          </div>
        {/if}
        {#if sciezka !== '/'}
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="strzalka_gora" tytul="Folder wyżej (Backspace)"
            onclick={() => idzDo(rodzic(sciezka))} />
        {/if}
        <Okruszki {udzial} {sciezka} idz={(s) => idzDo(s)} />
      </div>

      <PasekNarzedzi
        bind:this={pasek}
        bind:szukaj
        bind:widok
        bind:sortowanie
        bind:wieleNaraz
        {zapis}
        zaznaczono={zazn.wybrane.length}
        {wczytywanie}
        nowyFolder={() => akcje?.nowyFolder()}
        wgrajPliki={() => poleWgrywania?.click()}
        odswiez={() => void wczytaj()}
        pobierzZip={() => akcje?.zip(zazn.wybrane)}
        usunZaznaczone={() => akcje?.usun(zazn.wybrane)}
      />

      <Karta bezMarginesu>
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <div
          class="obszar"
          bind:this={obszar}
          tabindex="0"
          role="grid"
          aria-label="Lista plików"
          aria-multiselectable="true"
          aria-busy={wczytywanie || szukanie}
          onkeydown={klawisz}
          oncontextmenu={menuTla}
        >
          {#if blad}
            {#if blad.kod === 404}
              <Pusto ikona="folder" tytul="Folder nie istnieje" opis={blad.tekst}>
                <Przycisk ikona="dom" onclick={() => idzDo('/')}>Do głównego folderu</Przycisk>
              </Pusto>
            {:else if blad.kod === 403}
              <Pusto ikona="klodka" tytul="Brak dostępu" opis={blad.tekst} />
            {:else}
              <Pusto ikona="blad" tytul="Nie udało się wczytać folderu" opis={blad.tekst}>
                <Przycisk ikona="odswiez" onclick={() => void wczytaj()}>Ponów</Przycisk>
              </Pusto>
            {/if}
          {:else if !wczytano || (szukanie && !wyniki)}
            <div class="szkielet"><Szkielet linie={8} wysokosc={widok === 'kafelki' ? 'kafel' : 'wiersz'} /></div>
          {:else if wyniki && wyniki.length === 0}
            <Pusto ikona="szukaj" tytul="Nic nie znaleziono" opis="Spróbuj innej nazwy albo szukaj w folderze wyżej." />
          {:else if !wyniki && !wpisy.length}
            <Pusto ikona="folder_otwarty" tytul="Pusty folder"
              opis={zapis ? 'Przeciągnij tu pliki albo użyj przycisku Wgraj.' : 'W tym folderze nie ma plików.'} />
          {:else if widok === 'lista'}
            <TabelaPlikow bind:this={lista} wpisy={widoczne} zaznaczone={zazn.zbior} kursor={zazn.kursor}
              sort={sortKlucz} {malejaco} {wieleNaraz} {zapytanie} z={zdarzenia} />
          {:else}
            <KafelkiPlikow bind:this={kafelki} wpisy={widoczne} {udzial} zaznaczone={zazn.zbior} kursor={zazn.kursor}
              {wieleNaraz} {zapytanie} z={zdarzenia} />
          {/if}

          {#if przeciaganie > 0}
            <StrefaUpuszczania {zapis} {sciezka} />
          {/if}
        </div>
      </Karta>

      <div class="stopka opis">
        {#if wyniki}
          Wyniki wyszukiwania: {wyniki.length}
        {:else}
          {wpisy.filter((w) => w.folder).length} folderów, {wpisy.filter((w) => !w.folder).length} plików
        {/if}
        {#if wczytywanie}<span> · Wczytywanie…</span>{/if}
        <span class="skroty">Enter otwiera, Delete usuwa, F2 zmienia nazwę, Ctrl+K polecenia, ? skróty</span>
      </div>

      <ListaWgrywania {kolejka} />
    {/if}
  </section>
</div>

<input bind:this={poleWgrywania} type="file" multiple hidden onchange={wybranoPliki} />

<AkcjePlikow
  bind:this={akcje}
  {udzial}
  {sciezka}
  {zapis}
  {otworz}
  wgraj={() => poleWgrywania?.click()}
  zaznaczWszystko={() => zazn.wszystko()}
  {poZmianie}
  poZamknieciuMenu={() => obszar?.focus()}
/>
<Podglad bind:otwarte={podglad.otwarte} bind:wpis={podglad.wpis} {udzial} pliki={widoczne} />
<MojePobrania />

<style>
  .strona {
    display: grid;
    grid-template-columns: var(--uklad-pasek-w) minmax(0, 1fr);
    gap: var(--odstep-7);
    min-width: 0;
    min-height: 100%;
  }
  .strona.waska {
    grid-template-columns: minmax(0, 1fr);
  }
  .kolumna {
    position: sticky;
    top: 0;
    align-self: start;
  }
  .glowna {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
    min-width: 0;
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-height: var(--wysokosc-przycisk);
  }
  .wybor-udzialu {
    width: var(--kolumna-l);
  }
  .obszar {
    position: relative;
    min-height: calc(var(--wysokosc-wiersz-tabeli) * 8);
    outline: none;
  }
  .obszar:focus-visible {
    box-shadow: inset 0 0 0 var(--odstep-ramka) var(--kolor-niebieski);
    border-radius: var(--promien-karta);
  }
  .szkielet {
    padding: var(--odstep-l);
  }
  .stopka {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-s);
  }
  .skroty {
    margin-left: auto;
  }
  .waska .skroty {
    display: none;
  }
</style>
