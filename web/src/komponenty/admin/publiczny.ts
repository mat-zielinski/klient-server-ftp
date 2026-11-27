// Dostęp do linku publicznego bez logowania (docs/api.md, linki publiczne). Informacje przez własne fetch, bo hasło
// linku idzie w nagłówku X-Haslo (nie w adresie, żeby nie trafiło do historii i dzienników). Po POPRAWNYM haśle
// serwer ustawia ciasteczko linku (30 min), więc pobieranie i ZIP to ZWYKŁE odnośniki <a href> (bez blob:,
// bez trzymania pliku w pamięci przeglądarki). Ciasteczko wygasło = 401 i ponowny formularz hasła.

import { BladApi } from '../../api/klient'
import { API_PUBLICZNY, NAGLOWEK_HASLA } from './stale'
import type { InfoPubliczne } from './typy'

function naglowki(haslo: string): Record<string, string> {
  const n: Record<string, string> = { Accept: 'application/json' }
  if (haslo) n[NAGLOWEK_HASLA] = haslo
  return n
}

async function blad(odp: Response): Promise<BladApi> {
  let komunikat = ''
  try {
    const d = (await odp.json()) as { blad?: string }
    komunikat = d?.blad ?? ''
  } catch {
    // celowo: odpowiedź bez JSON, komunikat ustala strona po kodzie
  }
  return new BladApi(odp.status, komunikat)
}

function zapytanieSciezki(sciezka?: string): string {
  return sciezka ? '?' + new URLSearchParams({ sciezka }).toString() : ''
}

/**
 * Informacje o linku (plik albo folder z listą; `sciezka` = podfolder względem linku). Puste hasło = poleganie na
 * ciasteczku linku. Rzuca BladApi (401 hasło, 403 złe hasło, 410 wygasł, 404 brak, 400 poza linkiem).
 */
export async function infoLinku(id: string, haslo: string, sciezka?: string): Promise<InfoPubliczne> {
  let odp: Response
  try {
    odp = await fetch(API_PUBLICZNY + encodeURIComponent(id) + zapytanieSciezki(sciezka), {
      headers: naglowki(haslo),
      credentials: 'same-origin',
    })
  } catch {
    throw new BladApi(0, 'Brak połączenia z serwerem.')
  }
  if (!odp.ok) throw await blad(odp)
  return (await odp.json()) as InfoPubliczne
}

/** Adres pobrania pliku z linku (dla folderu: `sciezka` wewnątrz udostępnionego folderu). */
export function adresPobraniaLinku(id: string, sciezka?: string): string {
  return `${API_PUBLICZNY}${encodeURIComponent(id)}/pobierz${zapytanieSciezki(sciezka)}`
}

/** Adres ZIP całego folderu linku albo jego podfolderu. */
export function adresZipLinku(id: string, sciezka?: string): string {
  return `${API_PUBLICZNY}${encodeURIComponent(id)}/zip${zapytanieSciezki(sciezka)}`
}

/** Okruszki ścieżki podfolderu: [{nazwa, sciezka}] od korzenia linku (korzeń ma ścieżkę ''). */
export function okruszki(nazwaKorzenia: string, sciezka: string): { nazwa: string; sciezka: string }[] {
  const wynik = [{ nazwa: nazwaKorzenia, sciezka: '' }]
  let biezaca = ''
  for (const czesc of sciezka.split('/').filter(Boolean)) {
    biezaca += '/' + czesc
    wynik.push({ nazwa: czesc, sciezka: biezaca })
  }
  return wynik
}
