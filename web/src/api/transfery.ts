// Transfery na żywo przez SSE (GET /api/transfery/strumien): zdarzenia start | postep | koniec oraz suma co 1 s.
// EventSource sam wznawia połączenie; gdy przeglądarka go zamknie (np. 401 lub błąd serwera), łączymy ponownie
// z rosnącą przerwą.

import type { SumaTransferow, ZdarzenieTransferu } from './typy'

const ADRES = '/api/transfery/strumien'
const TYPY = ['start', 'postep', 'koniec'] as const
const PRZERWA_START_MS = 1000
const PRZERWA_MAKS_MS = 30000

export function sluchajTransferow(
  naZdarzenie: (z: ZdarzenieTransferu) => void,
  naSume: (s: SumaTransferow) => void,
): () => void {
  let zrodlo: EventSource | null = null
  let zamkniete = false
  let przerwa = PRZERWA_START_MS
  let zegar: ReturnType<typeof setTimeout> | null = null

  const czytaj = <T>(e: MessageEvent): T | null => {
    try {
      return JSON.parse(e.data as string) as T
    } catch {
      return null // celowo: uszkodzona ramka SSE jest pomijana
    }
  }

  const polacz = () => {
    if (zamkniete) return
    const es = new EventSource(ADRES, { withCredentials: true })
    zrodlo = es
    for (const typ of TYPY) {
      es.addEventListener(typ, (e) => {
        const z = czytaj<ZdarzenieTransferu>(e as MessageEvent)
        if (!z) return
        // serwer wysyła w data obiekt zdarzenia; gdyby wysłał sam stan, uzupełniamy typ z nazwy zdarzenia
        naZdarzenie(z.stan ? { ...z, typ } : { typ, stan: z as unknown as ZdarzenieTransferu['stan'] })
      })
    }
    es.addEventListener('suma', (e) => {
      const s = czytaj<SumaTransferow>(e as MessageEvent)
      if (s) naSume(s)
    })
    es.onopen = () => {
      przerwa = PRZERWA_START_MS
    }
    es.onerror = () => {
      if (es.readyState !== EventSource.CLOSED || zamkniete) return
      zegar = setTimeout(polacz, przerwa)
      przerwa = Math.min(przerwa * 2, PRZERWA_MAKS_MS)
    }
  }

  polacz()
  return () => {
    zamkniete = true
    if (zegar) clearTimeout(zegar)
    zrodlo?.close()
  }
}
