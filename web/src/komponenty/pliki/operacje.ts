// Operacje na plikach przez API (docs/api.md, sekcje Pliki i Fala 2). Błędy wychodzą jako BladApi z klienta.
import { api, adresApi, BladApi } from '../../api/klient'
import type { Protokoly, UdzialInfo, Wpis } from '../../api/typy'
import { pobierzAdres, polacz, rodzic } from './narzedzia'
import { LIMIT_WYNIKOW_SZUKANIA } from './stale'

export type KluczSortu = 'nazwa' | 'rozmiar' | 'zmieniono'

export const udzialy = () => api.get<UdzialInfo[]>('/udzialy')

export async function lista(udzial: string, sciezka: string, sort: KluczSortu, malejaco: boolean): Promise<Wpis[]> {
  const odp = await api.get<{ sciezka: string; wpisy: Wpis[] | null }>('/lista', { udzial, sciezka, sort, malejaco })
  return uporzadkuj(odp.wpisy ?? [], sort, malejaco)
}

/** Czy wpis leży w folderze (lub jego podfolderach). */
export function wFolderze(w: { sciezka: string }, folder: string): boolean {
  return folder === '/' || w.sciezka.startsWith(folder.replace(/\/+$/, '') + '/')
}

/** Szukanie: najpierw indeks (szybki, cały udział, zawężamy do folderu), przy 404 / 503 zwykłe /api/szukaj. */
export async function szukaj(udzial: string, sciezka: string, q: string): Promise<Wpis[]> {
  try {
    const r = await api.get<Wpis[] | null>('/indeks/szukaj', { udzial, q, limit: LIMIT_WYNIKOW_SZUKANIA })
    return (r ?? []).filter((w) => wFolderze(w, sciezka))
  } catch (e) {
    if (!(e instanceof BladApi) || (e.kod !== 404 && e.kod !== 503)) throw e
  }
  return (await api.get<Wpis[] | null>('/szukaj', { udzial, sciezka, q })) ?? []
}

export const protokoly = (udzial: string, sciezka: string) => api.get<Protokoly>('/protokoly', { udzial, sciezka })

export const adresMiniatury = (udzial: string, sciezka: string, rozmiar: 's' | 'm') =>
  adresApi('/miniatura', { udzial, sciezka, rozmiar })

export async function suma(udzial: string, sciezka: string): Promise<string> {
  return (await api.get<{ sha256: string }>('/suma', { udzial, sciezka })).sha256
}

export const nowyFolder = (udzial: string, folder: string, nazwa: string) =>
  api.post('/folder', undefined, { udzial, sciezka: polacz(folder, nazwa) })

/** Zmiana nazwy w tym samym folderze: z i na to pełne ścieżki w udziale (ustalenie sesji głównej). */
export const zmienNazwe = (udzial: string, w: Wpis, nazwa: string) =>
  api.post('/zmien', { z: w.sciezka, na: polacz(rodzic(w.sciezka), nazwa) }, { udzial })

/** Usuwa wpisy do kosza po kolei; zwraca listę nieudanych z komunikatem. */
export async function usun(udzial: string, wpisy: Wpis[]): Promise<{ wpis: Wpis; blad: string }[]> {
  const bledy: { wpis: Wpis; blad: string }[] = []
  for (const w of wpisy) {
    try {
      await api.usun('/plik', { udzial, sciezka: w.sciezka })
    } catch (e) {
      bledy.push({ wpis: w, blad: e instanceof Error ? e.message : String(e) })
    }
  }
  return bledy
}

/** Adres ZIP: GET /api/zip?udzial&sciezka=..&sciezka=.. (strumień prosto do pliku, bez trzymania w pamięci). */
export function adresZip(udzial: string, sciezki: string[]): string {
  const p = new URLSearchParams({ udzial })
  for (const s of sciezki) p.append('sciezka', s)
  return adresApi('/zip') + '?' + p.toString()
}

export function pobierzZip(udzial: string, wpisy: Wpis[]): void {
  if (wpisy.length) pobierzAdres(adresZip(udzial, wpisy.map((w) => w.sciezka)))
}

const porownajNazwy = new Intl.Collator('pl', { numeric: true, sensitivity: 'base' }).compare

/** Foldery zawsze pierwsze, potem wybrany klucz (porządek naturalny nazw, jak w Eksploratorze). */
export function uporzadkuj(wpisy: Wpis[], sort: KluczSortu, malejaco: boolean): Wpis[] {
  const kier = malejaco ? -1 : 1
  return [...wpisy].sort((a, b) => {
    if (a.folder !== b.folder) return a.folder ? -1 : 1
    let r = 0
    if (sort === 'rozmiar') r = a.rozmiar - b.rozmiar
    else if (sort === 'zmieniono') r = Date.parse(a.zmieniono) - Date.parse(b.zmieniono)
    if (r === 0) r = porownajNazwy(a.nazwa, b.nazwa)
    return r * kier
  })
}
