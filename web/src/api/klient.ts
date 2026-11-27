// Klient API Skrytki (docs/api.md): JSON pod /api/, nagłówek X-Skrytka: 1 (ochrona CSRF), błąd = BladApi.

export type Parametry = Record<string, string | number | boolean | null | undefined>

/** Błąd zwrócony przez serwer: kod HTTP (0 = brak połączenia) i komunikat dla człowieka. */
export class BladApi extends Error {
  readonly kod: number
  readonly komunikat: string

  constructor(kod: number, komunikat: string) {
    super(komunikat)
    this.name = 'BladApi'
    this.kod = kod
    this.komunikat = komunikat
  }
}

const PRZEDROSTEK = '/api'
const NAGLOWEK_CSRF = 'X-Skrytka'

/** Komunikaty, gdy serwer nie podał własnego (`{"blad": …}`). */
const KOMUNIKATY: Record<number, string> = {
  0: 'Brak połączenia z serwerem.',
  400: 'Nieprawidłowe dane.',
  401: 'Sesja wygasła. Zaloguj się ponownie.',
  403: 'Brak uprawnień.',
  404: 'Nie znaleziono.',
  409: 'Taki element już istnieje.',
  410: 'Link wygasł.',
  429: 'Za dużo prób. Spróbuj później.',
  500: 'Błąd serwera.',
}

type Sluchacz401 = () => void
let naBrakSesji: Sluchacz401 | null = null

/** Rejestruje reakcję na 401 (sesja wygasła): używa jej stan/sesja. */
export function ustawNaBrakSesji(f: Sluchacz401 | null): void {
  naBrakSesji = f
}

/** Składa zapytanie `?a=1&b=2` (pomija null i undefined, wartości kodowane). */
export function zapytanie(parametry?: Parametry): string {
  if (!parametry) return ''
  const p = new URLSearchParams()
  for (const [k, w] of Object.entries(parametry)) {
    if (w === null || w === undefined) continue
    p.set(k, typeof w === 'boolean' ? (w ? '1' : '0') : String(w))
  }
  const s = p.toString()
  return s ? '?' + s : ''
}

/** Pełny adres `/api/<adres>?…` (adres z ukośnikiem lub bez, z przedrostkiem /api lub bez). */
export function adresApi(adres: string, parametry?: Parametry): string {
  let a = adres.startsWith('/') ? adres : '/' + adres
  if (!a.startsWith(PRZEDROSTEK + '/')) a = PRZEDROSTEK + a
  return a + zapytanie(parametry)
}

async function komunikatBledu(odp: Response): Promise<string> {
  try {
    const dane = (await odp.json()) as { blad?: string }
    if (dane && typeof dane.blad === 'string' && dane.blad) return dane.blad
  } catch {
    // celowo: odpowiedź bez JSON, zostaje komunikat domyślny
  }
  return KOMUNIKATY[odp.status] ?? `Błąd ${odp.status}.`
}

async function zadanie<T>(metoda: string, adres: string, cialo?: unknown, parametry?: Parametry): Promise<T> {
  const naglowki: Record<string, string> = { Accept: 'application/json', [NAGLOWEK_CSRF]: '1' }
  let tresc: BodyInit | undefined
  if (cialo instanceof Blob || cialo instanceof ArrayBuffer || cialo instanceof FormData) {
    tresc = cialo
  } else if (cialo !== undefined) {
    naglowki['Content-Type'] = 'application/json'
    tresc = JSON.stringify(cialo)
  }
  let odp: Response
  try {
    odp = await fetch(adresApi(adres, parametry), {
      method: metoda,
      headers: naglowki,
      body: tresc,
      credentials: 'same-origin',
    })
  } catch {
    throw new BladApi(0, KOMUNIKATY[0])
  }
  if (!odp.ok) {
    const komunikat = await komunikatBledu(odp)
    if (odp.status === 401 && naBrakSesji) naBrakSesji()
    throw new BladApi(odp.status, komunikat)
  }
  if (odp.status === 204) return undefined as T
  const typ = odp.headers.get('Content-Type') ?? ''
  if (typ.includes('application/json')) return (await odp.json()) as T
  return (await odp.text()) as unknown as T
}

export const api = {
  get: <T>(adres: string, parametry?: Parametry) => zadanie<T>('GET', adres, undefined, parametry),
  post: <T>(adres: string, cialo?: unknown, parametry?: Parametry) => zadanie<T>('POST', adres, cialo, parametry),
  put: <T>(adres: string, cialo?: unknown, parametry?: Parametry) => zadanie<T>('PUT', adres, cialo, parametry),
  usun: <T>(adres: string, parametry?: Parametry) => zadanie<T>('DELETE', adres, undefined, parametry),
}

/** Adres pobrania pliku (`GET /api/pobierz`); `podglad` = wyświetlenie w przeglądarce zamiast zapisu. */
export function adresPobrania(udzial: string, sciezka: string, podglad?: boolean): string {
  return adresApi('/pobierz', { udzial, sciezka, podglad: podglad ? 1 : undefined })
}

/** Komunikat z dowolnego błędu (BladApi, Error albo inny). */
export function trescBledu(b: unknown): string {
  if (b instanceof BladApi) return b.komunikat
  if (b instanceof Error) return b.message
  return 'Nieznany błąd.'
}
