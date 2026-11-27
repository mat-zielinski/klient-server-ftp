// Typy i drobne odwzorowania Pulpitu i Dziennika, których nie ma w kontrakcie src/api/typy.ts.

import type { Adres } from '../../api/typy'

/** Odpowiedź GET /api/siec (docs/api.md, „Administracja”). */
export interface InfoSieci {
  adresy: Adres[]
  publiczny: string
  tryb: string
  porty: { http: number; https: number; ftps: number }
  porty_otwarte: Record<string, boolean>
  odcisk_certyfikatu: string
}

export type StanPigulki = 'ok' | 'info' | 'blad' | 'ostrz' | 'wyl'

/** Pigułka protokołu: szyfrowane spokojnie, HTTP bez szyfrowania jako ostrzeżenie. */
export function stanProtokolu(protokol: string): StanPigulki {
  switch (protokol.toLowerCase()) {
    case 'https':
      return 'info'
    case 'ftps':
      return 'ok'
    case 'http':
      return 'ostrz'
    default:
      return 'wyl'
  }
}

/** Pigułka wyniku wpisu dziennika. */
export function stanWyniku(wynik: string): StanPigulki {
  switch (wynik) {
    case 'ok':
      return 'ok'
    case 'przerwane':
      return 'ostrz'
    case 'blad':
      return 'blad'
    default:
      return 'wyl'
  }
}

const NAZWY_WYNIKU: Record<string, string> = { ok: 'OK', przerwane: 'Przerwane', blad: 'Błąd' }

export function nazwaWyniku(wynik: string): string {
  return NAZWY_WYNIKU[wynik] ?? (wynik || 'Brak')
}

/** Nazwa pliku z pełnej ścieżki (ostatni człon). */
export function nazwaPliku(sciezka: string): string {
  const czesci = sciezka.split('/').filter(Boolean)
  return czesci[czesci.length - 1] ?? sciezka
}

/** Kopiuje tekst do schowka; zwraca, czy się udało. */
export async function kopiuj(tekst: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(tekst)
    return true
  } catch {
    return false // celowo: brak uprawnień do schowka (np. HTTP bez TLS) zgłasza wywołujący
  }
}

/** Odmiana liczebnika: 1 pobranie, 2 pobrania, 5 pobrań. */
export function odmiana(n: number, jeden: string, kilka: string, wiele: string): string {
  if (n === 1) return jeden
  const j = n % 10
  const d = n % 100
  return j >= 2 && j <= 4 && (d < 12 || d > 14) ? kilka : wiele
}

/** Rodzaje wpisów dziennika (pakiet dziennik) z nazwami do filtra i tabeli. */
export const RODZAJE_WPISU: { wartosc: string; etykieta: string }[] = [
  { wartosc: 'pobranie', etykieta: 'Pobranie' },
  { wartosc: 'wgranie', etykieta: 'Wgranie' },
  { wartosc: 'zmiana', etykieta: 'Zmiana' },
  { wartosc: 'udostepnienie', etykieta: 'Udostępnienie' },
  { wartosc: 'logowanie', etykieta: 'Logowanie' },
  { wartosc: 'logowanie_blad', etykieta: 'Nieudane logowanie' },
  { wartosc: 'administracja', etykieta: 'Administracja' },
  { wartosc: 'serwer', etykieta: 'Serwer' },
]

export function nazwaRodzaju(rodzaj: string): string {
  return RODZAJE_WPISU.find((r) => r.wartosc === rodzaj)?.etykieta ?? rodzaj
}

/** Nazwa protokołu w statystykach: pusty klucz = „nieznany”. */
export function nazwaProtokolu(protokol: string): string {
  return protokol ? protokol.toUpperCase() : 'Nieznany'
}
