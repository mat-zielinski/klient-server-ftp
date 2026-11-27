// Pozycje menu kontekstowego pliku / folderu, w tym podmenu „Pobierz przez…” z GET /api/protokoly.
import type { OpcjaProtokolu, Protokoly, Wpis } from '../../api/typy'
import type { NazwaIkony } from '../../ikony/ikony'
import type { PozycjaMenu } from '../wspolne/MenuKontekstowe.svelte'
import { maPodglad } from './rodzaje'

export interface AkcjeMenu {
  otworz: (w: Wpis) => void
  protokol: (w: Wpis, o: OpcjaProtokolu, p: Protokoly) => void
  kopiujSciezke: (w: Wpis) => void
  zmienNazwe: (w: Wpis) => void
  usun: (w: Wpis[]) => void
  wlasciwosci: (w: Wpis) => void
  zip: (w: Wpis[]) => void
}

/** Stan wczytywania opcji protokołów dla menu. */
export type StanProtokolow = { stan: 'wczytywanie' } | { stan: 'blad'; komunikat: string } | { stan: 'ok'; dane: Protokoly }

const IKONY_PROTOKOLOW: Record<OpcjaProtokolu['id'], NazwaIkony> = {
  https: 'globus',
  ftps: 'ftp',
  link: 'kopiuj',
  zip: 'archiwum',
  publiczny: 'udostepnij',
}

function podmenuProtokolow(w: Wpis, p: StanProtokolow, akcje: AkcjeMenu): PozycjaMenu[] {
  if (p.stan === 'wczytywanie') return [{ etykieta: 'Wczytywanie…', ikona: 'klepsydra', wylaczona: true }]
  if (p.stan === 'blad') return [{ etykieta: 'Niedostępne', ikona: 'blad', opis: p.komunikat, wylaczona: true }]
  return p.dane.opcje.map((o) => ({
    etykieta: o.nazwa,
    ikona: IKONY_PROTOKOLOW[o.id] ?? 'pobierz',
    opis: o.dostepne ? o.opis : o.powod || 'Niedostępne.',
    wylaczona: !o.dostepne,
    onclick: () => akcje.protokol(w, o, p.dane),
  }))
}

/** Menu dla jednego wpisu. */
export function menuWpisu(w: Wpis, zapis: boolean, p: StanProtokolow, akcje: AkcjeMenu): PozycjaMenu[] {
  const bezZapisu = zapis ? undefined : 'Brak prawa zapisu w tym udziale.'
  const pozycje: PozycjaMenu[] = [
    w.folder
      ? { etykieta: 'Otwórz', ikona: 'folder_otwarty', opis: 'Enter', onclick: () => akcje.otworz(w) }
      : {
          etykieta: 'Podgląd',
          ikona: 'oko',
          opis: maPodglad(w) ? 'Enter' : 'Brak podglądu dla tego rodzaju pliku.',
          wylaczona: !maPodglad(w),
          onclick: () => akcje.otworz(w),
        },
    { etykieta: 'Pobierz przez…', ikona: 'pobierz', podmenu: podmenuProtokolow(w, p, akcje) },
    { etykieta: 'Kopiuj ścieżkę', ikona: 'kopiuj', separator: true, onclick: () => akcje.kopiujSciezke(w) },
    { etykieta: 'Zmień nazwę', ikona: 'olowek', opis: bezZapisu ?? 'F2', wylaczona: !zapis, onclick: () => akcje.zmienNazwe(w) },
    {
      etykieta: 'Usuń', ikona: 'kosz', opis: bezZapisu ?? 'Delete, do kosza', wylaczona: !zapis, niebezpieczna: true,
      onclick: () => akcje.usun([w]),
    },
    { etykieta: 'Właściwości', ikona: 'info', separator: true, onclick: () => akcje.wlasciwosci(w) },
  ]
  return pozycje
}

/** Menu dla kilku zaznaczonych wpisów. */
export function menuWielu(wpisy: Wpis[], zapis: boolean, akcje: AkcjeMenu): PozycjaMenu[] {
  return [
    { etykieta: `Zaznaczono: ${wpisy.length}`, ikona: 'lista', wylaczona: true },
    { etykieta: 'Pobierz jako ZIP', ikona: 'archiwum', separator: true, onclick: () => akcje.zip(wpisy) },
    {
      etykieta: 'Usuń zaznaczone', ikona: 'kosz', opis: zapis ? 'Delete, do kosza' : 'Brak prawa zapisu w tym udziale.',
      wylaczona: !zapis, niebezpieczna: true, onclick: () => akcje.usun(wpisy),
    },
  ]
}

/** Menu pustego miejsca listy (prawy przycisk poza wpisami). */
export function menuTla(
  zapis: boolean,
  a: { nowyFolder: () => void; wgraj: () => void; odswiez: () => void; zaznaczWszystko: () => void },
): PozycjaMenu[] {
  const bez = zapis ? undefined : 'Brak prawa zapisu w tym udziale.'
  return [
    { etykieta: 'Nowy folder', ikona: 'folder_plus', opis: bez, wylaczona: !zapis, onclick: a.nowyFolder },
    { etykieta: 'Wgraj pliki', ikona: 'wgraj', opis: bez, wylaczona: !zapis, onclick: a.wgraj },
    { etykieta: 'Odśwież', ikona: 'odswiez', separator: true, onclick: a.odswiez },
    { etykieta: 'Zaznacz wszystko', ikona: 'ok_kolo', opis: 'Ctrl+A', onclick: a.zaznaczWszystko },
  ]
}
