<script lang="ts">
  // Akcje na plikach: menu kontekstowe (z „Pobierz przez…” z GET /api/protokoly) i okna akcji: FTPS, link
  // publiczny, nowy folder / zmiana nazwy, właściwości, usuwanie. Strona woła wyeksportowane funkcje.
  import { adresPobrania, trescBledu } from '../../api/klient'
  import type { OpcjaProtokolu, Protokoly, Wpis } from '../../api/typy'
  import MenuKontekstowe, { type PozycjaMenu } from '../wspolne/MenuKontekstowe.svelte'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import OknoFtps from './OknoFtps.svelte'
  import OknoLinkuPublicznego from './OknoLinkuPublicznego.svelte'
  import OknoNazwy from './OknoNazwy.svelte'
  import OknoWlasciwosci from './OknoWlasciwosci.svelte'
  import OknoUsuwania from './OknoUsuwania.svelte'
  import { menuTla, menuWielu, menuWpisu, type AkcjeMenu, type StanProtokolow } from './menu'
  import { kopiuj, pobierzAdres } from './narzedzia'
  import * as op from './operacje'

  let {
    udzial,
    sciezka,
    zapis,
    otworz,
    wgraj,
    zaznaczWszystko,
    poZmianie,
    poZamknieciuMenu,
  }: {
    udzial: string
    sciezka: string
    zapis: boolean
    otworz: (w: Wpis) => void
    wgraj: () => void
    zaznaczWszystko: () => void
    /** Po zmianie zawartości folderu (odświeżenie listy, czyszczenie zaznaczenia). */
    poZmianie: () => void
    poZamknieciuMenu: () => void
  } = $props()

  let menu = $state<{ x: number; y: number; cel: Wpis[] | null } | null>(null)
  let stanProt = $state<StanProtokolow>({ stan: 'wczytywanie' })
  let ftps = $state<{ otwarte: boolean; opcja: OpcjaProtokolu | null; dane: Protokoly | null }>({
    otwarte: false, opcja: null, dane: null,
  })
  let link = $state({ otwarte: false, sciezka: '' })
  let nazwa = $state<{ otwarte: boolean; wpis: Wpis | null }>({ otwarte: false, wpis: null })
  let wlasciwosci = $state<{ otwarte: boolean; wpis: Wpis | null }>({ otwarte: false, wpis: null })
  let usuwanie = $state<{ otwarte: boolean; wpisy: Wpis[] }>({ otwarte: false, wpisy: [] })

  let numerMenu = 0
  /** Otwiera menu w punkcie; cel null = menu pustego miejsca. */
  export async function otworzMenu(x: number, y: number, cel: Wpis[] | null) {
    menu = { x, y, cel }
    if (!cel || cel.length !== 1) return
    const numer = ++numerMenu
    stanProt = { stan: 'wczytywanie' }
    try {
      const dane = await op.protokoly(udzial, cel[0].sciezka)
      if (numer === numerMenu) stanProt = { stan: 'ok', dane }
    } catch (e) {
      if (numer === numerMenu) stanProt = { stan: 'blad', komunikat: trescBledu(e) }
    }
  }

  export function nowyFolder() {
    if (zapis) nazwa = { otwarte: true, wpis: null }
  }

  export function zmienNazwe(w: Wpis) {
    if (zapis) nazwa = { otwarte: true, wpis: w }
  }

  export function usun(wpisy: Wpis[]) {
    if (!wpisy.length) return
    if (!zapis) {
      powiadom('Brak prawa zapisu w tym udziale.', 'ostrz')
      return
    }
    usuwanie = { otwarte: true, wpisy }
  }

  export function zip(wpisy: Wpis[]) {
    op.pobierzZip(udzial, wpisy)
  }

  async function kopiujZKomunikatem(tekst: string, ok: string) {
    const udane = await kopiuj(tekst)
    powiadom(udane ? ok : 'Nie udało się skopiować.', udane ? 'ok' : 'blad')
  }

  function wybierzProtokol(w: Wpis, o: OpcjaProtokolu, p: Protokoly) {
    const bezposredni = o.adres || adresPobrania(udzial, w.sciezka)
    if (o.id === 'https') pobierzAdres(bezposredni)
    else if (o.id === 'zip') pobierzAdres(o.adres || op.adresZip(udzial, [w.sciezka]))
    else if (o.id === 'link')
      void kopiujZKomunikatem(new URL(bezposredni, location.origin).href, 'Skopiowano link bezpośredni. Działa po zalogowaniu.')
    else if (o.id === 'ftps') ftps = { otwarte: true, opcja: o, dane: p }
    else if (o.id === 'publiczny') link = { otwarte: true, sciezka: w.sciezka }
  }

  const akcje: AkcjeMenu = {
    otworz: (w) => otworz(w),
    protokol: wybierzProtokol,
    kopiujSciezke: (w) => void kopiujZKomunikatem(`${udzial}${w.sciezka}`, 'Skopiowano ścieżkę.'),
    zmienNazwe,
    usun,
    wlasciwosci: (w) => (wlasciwosci = { otwarte: true, wpis: w }),
    zip,
  }

  const pozycje = $derived.by((): PozycjaMenu[] => {
    if (!menu) return []
    if (!menu.cel) return menuTla(zapis, { nowyFolder, wgraj, odswiez: poZmianie, zaznaczWszystko })
    if (menu.cel.length > 1) return menuWielu(menu.cel, zapis, akcje)
    return menuWpisu(menu.cel[0], zapis, stanProt, akcje)
  })

  async function zatwierdzNazwe(nowa: string): Promise<boolean> {
    const w = nazwa.wpis
    try {
      if (w) {
        if (nowa === w.nazwa) return true
        await op.zmienNazwe(udzial, w, nowa)
        powiadom('Zmieniono nazwę.', 'ok')
      } else {
        await op.nowyFolder(udzial, sciezka, nowa)
        powiadom('Utworzono folder.', 'ok')
      }
      poZmianie()
      return true
    } catch (e) {
      powiadom(trescBledu(e), 'blad')
      return false
    }
  }
</script>

{#if menu}
  <MenuKontekstowe x={menu.x} y={menu.y} {pozycje} zamknij={() => {
    menu = null
    poZamknieciuMenu()
  }} />
{/if}

<OknoFtps bind:otwarte={ftps.otwarte} opcja={ftps.opcja} adresy={ftps.dane?.adresy ?? []}
  odcisk={ftps.dane?.odcisk_certyfikatu ?? ''} />
<OknoLinkuPublicznego bind:otwarte={link.otwarte} {udzial} sciezka={link.sciezka} />
<OknoNazwy bind:otwarte={nazwa.otwarte} tytul={nazwa.wpis ? 'Zmień nazwę' : 'Nowy folder'}
  poczatkowa={nazwa.wpis?.nazwa ?? ''} przycisk={nazwa.wpis ? 'Zmień' : 'Utwórz'} zatwierdz={zatwierdzNazwe} />
<OknoWlasciwosci bind:otwarte={wlasciwosci.otwarte} wpis={wlasciwosci.wpis} {udzial} />
<OknoUsuwania bind:otwarte={usuwanie.otwarte} {udzial} wpisy={usuwanie.wpisy} poUsunieciu={poZmianie} />
