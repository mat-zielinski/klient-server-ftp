// Dostęp do linku publicznego bez logowania (GET /api/publiczny/<id>, …/pobierz). Własne fetch zamiast
// api.get, bo hasło linku idzie w nagłówku X-Haslo (nie w adresie, żeby nie trafiło do historii i dzienników).

import { BladApi } from '../../api/klient'
import { API_PUBLICZNY, NAGLOWEK_HASLA, ZWOLNIENIE_POBRANIA_MS } from './stale'
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

/** Informacje o linku (plik albo folder z listą). Rzuca BladApi (401 hasło, 410 wygasł, 404 brak). */
export async function infoLinku(id: string, haslo: string): Promise<InfoPubliczne> {
  let odp: Response
  try {
    odp = await fetch(API_PUBLICZNY + encodeURIComponent(id), { headers: naglowki(haslo), credentials: 'same-origin' })
  } catch {
    throw new BladApi(0, 'Brak połączenia z serwerem.')
  }
  if (!odp.ok) throw await blad(odp)
  return (await odp.json()) as InfoPubliczne
}

/** Adres pobrania pliku z linku (dla folderu: `sciezka` wewnątrz udostępnionego folderu). */
export function adresPobraniaLinku(id: string, sciezka?: string): string {
  const q = sciezka ? '?' + new URLSearchParams({ sciezka }).toString() : ''
  return `${API_PUBLICZNY}${encodeURIComponent(id)}/pobierz${q}`
}

/** Pobranie pliku z hasłem: fetch z nagłówkiem, potem zapis przez tymczasowy adres blob. */
export async function pobierzZHaslem(id: string, haslo: string, nazwa: string, sciezka?: string): Promise<void> {
  let odp: Response
  try {
    odp = await fetch(adresPobraniaLinku(id, sciezka), { headers: { [NAGLOWEK_HASLA]: haslo }, credentials: 'same-origin' })
  } catch {
    throw new BladApi(0, 'Brak połączenia z serwerem.')
  }
  if (!odp.ok) throw await blad(odp)
  const blob = await odp.blob()
  const adres = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = adres
  a.download = nazwa
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(adres), ZWOLNIENIE_POBRANIA_MS)
}
