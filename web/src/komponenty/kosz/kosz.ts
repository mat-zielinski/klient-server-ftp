// Kosz aplikacji (docs/api.md, „Kosz”): typy odpowiedzi, wywołania API i sortowanie listy.
import { api } from '../../api/klient'
import type { Rodzaj, UdzialInfo } from '../../api/typy'

/** Wpis GET /api/kosz (typ lokalny; docelowo w api/typy.ts). */
export interface WpisKosza {
  id: string
  nazwa: string
  /** Oryginalne miejsce w udziale („/a/b.txt”). */
  sciezka: string
  usunieto: string
  rozmiar: number
  folder: boolean
  rodzaj: Rodzaj
}

/** Liczba dni, po których serwer sam opróżnia kosz (KoszOproznijPoDniach w konfiguracji serwera). */
export const DNI_AUTOMATYCZNEGO_OPROZNIANIA = 30

/** Udziały, w których jest kosz: prawo zapisu i nie tylko do odczytu. */
export function udzialyZKoszem(udzialy: UdzialInfo[]): UdzialInfo[] {
  return udzialy.filter((u) => u.prawo === 'zapis' && !u.tylko_odczyt)
}

export const listaKosza = (udzial: string) => api.get<WpisKosza[]>('/kosz', { udzial })

export const przywroc = (udzial: string, id: string, nadpisz: boolean) =>
  api.post<Record<string, never>>('/kosz/przywroc', { id, nadpisz }, { udzial })

export const usunTrwale = (udzial: string, id: string) => api.usun<Record<string, never>>('/kosz', { udzial, id })

export const oproznij = (udzial: string) => api.usun<{ usuniete: number }>('/kosz/wszystko', { udzial })

/** Folder nadrzędny oryginalnego miejsca („/a/b.txt” → „/a”). */
export function folderNadrzedny(sciezka: string): string {
  const i = sciezka.lastIndexOf('/')
  return i <= 0 ? '/' : sciezka.slice(0, i)
}

/** Sortowanie: nazwa, miejsce, usunieto (domyślnie), rozmiar; foldery bez pierwszeństwa (kosz to historia). */
export function sortujKosz(lista: WpisKosza[], klucz: string, malejaco: boolean): WpisKosza[] {
  const kier = malejaco ? -1 : 1
  const w = (e: WpisKosza): string | number => {
    switch (klucz) {
      case 'nazwa':
        return e.nazwa.toLowerCase()
      case 'miejsce':
        return e.sciezka.toLowerCase()
      case 'rozmiar':
        return e.rozmiar
      default:
        return Date.parse(e.usunieto) || 0
    }
  }
  return [...lista].sort((a, b) => {
    const x = w(a)
    const y = w(b)
    return (typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y), 'pl')) * kier
  })
}
