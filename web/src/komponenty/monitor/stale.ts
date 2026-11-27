// Stałe strony Monitor (okna czasu, limity historii, progi wskaźników, zakresy mapy ciepła) w jednym miejscu.

export const MS_SEKUNDA = 1000
export const MS_MINUTA = 60 * MS_SEKUNDA
export const MS_GODZINA = 60 * MS_MINUTA
export const MS_DZIEN = 24 * MS_GODZINA

export const MONITOR = {
  /** Okno wykresów na żywo (historia_1s). */
  oknoNaZywoMs: 10 * MS_MINUTA,
  /** Okno historii minutowej (historia_1m). */
  oknoDobyMs: MS_DZIEN,
  /** Zapas ponad okno, zanim stare próbki zostaną usunięte (linia wchodzi zza lewej krawędzi). */
  zapasMs: 5 * MS_SEKUNDA,
  /** Odstęp próbek dopisywanych na żywo do historii minutowej. */
  krokMinutowyMs: MS_MINUTA,
  /** Co ile odświeżać zajętość udziałów (GET /api/pulpit) i statystyki indeksu. */
  odswiezanieDyskuMs: 60 * MS_SEKUNDA,
  odswiezanieIndeksuMs: 5 * MS_MINUTA,
  /** Wskaźniki: od pierwszego progu żółty, od drugiego czerwony. */
  progOstrzezenia: 0.7,
  progAlarmu: 0.9,
  /** Ile pozycji w rankingu udziałów. */
  pozycjiRankingu: 8,
} as const

/** Okres wykresów: na żywo (10 min co 1 s) albo doba (24 h co 1 min). */
export type Okres = 'na_zywo' | 'doba'

export const OKRESY: { id: Okres; etykieta: string }[] = [
  { id: 'na_zywo', etykieta: '10 min' },
  { id: 'doba', etykieta: '24 h' },
]

/** Zakres mapy ciepła w dniach wstecz (wliczając dziś). */
export const ZAKRESY_MAPY: { id: string; etykieta: string; dni: number }[] = [
  { id: '7', etykieta: '7 dni', dni: 7 },
  { id: '30', etykieta: '30 dni', dni: 30 },
  { id: '90', etykieta: '90 dni', dni: 90 },
]

/** Polskie nazwy rodzajów plików z indeksu (nieznany rodzaj = nazwa z serwera). */
export const NAZWY_RODZAJOW: Record<string, string> = {
  obraz: 'Obrazy',
  wideo: 'Wideo',
  dzwiek: 'Dźwięk',
  pdf: 'PDF',
  tekst: 'Tekst',
  archiwum: 'Archiwa',
  inny: 'Inne',
  folder: 'Foldery',
}

/** Polskie nazwy rodzajów interfejsu i stan pigułki. */
export const RODZAJE_INTERFEJSU: Record<string, { nazwa: string; stan: 'ok' | 'info' | 'ostrz' | 'wyl' }> = {
  lan: { nazwa: 'LAN', stan: 'info' },
  wireguard: { nazwa: 'WireGuard', stan: 'ok' },
  publiczny: { nazwa: 'Publiczny', stan: 'ostrz' },
  petla: { nazwa: 'Pętla', stan: 'wyl' },
}
