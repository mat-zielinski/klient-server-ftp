// Monitor serwera (docs/api.md, Fala 2): typy odpowiedzi, odczyt GET /api/monitor, strumień SSE
// /api/monitor/strumien (zdarzenie `probka` co 1 s), mapa ciepła pobrań i statystyki indeksu.
// Typy są tutaj (nie w typy.ts), bo należą do obszaru web-monitor.

import { api } from './klient'

/** Ruch jednego interfejsu sieciowego w próbce. */
export interface SiecInterfejsu {
  interfejs: string
  /** Np. `lan`, `wireguard`, `publiczny`, `petla` (serwer może dodać inne). */
  rodzaj: string
  wysylanie_bps: number
  odbior_bps: number
}

/** Próbka monitora (co 1 s w historii 10 min, co 1 min w historii 24 h). */
export interface Probka {
  czas: string
  cpu_proc: number
  ram_uzyte: number
  ram_razem: number
  dysk_odczyt_bps: number
  dysk_zapis_bps: number
  siec: SiecInterfejsu[]
  wysylanie_bps: number
  odbior_bps: number
  polaczenia: number
  transfery: number
  gorutyny: number
  pamiec_procesu: number
}

/** Opis maszyny, na której działa serwer. */
export interface InfoSystemu {
  os: string
  arch: string
  cpu_rdzenie: number
  ram_razem: number
  host: string
  czas_pracy_s: number
}

/** Odpowiedź GET /api/monitor. */
export interface StanMonitoraApi {
  teraz: Probka
  historia_1s: Probka[]
  historia_1m: Probka[]
  system: InfoSystemu
}

/** Liczba plików i bajtów jednego rodzaju w udziale. */
export interface LicznikRodzaju { pliki: number; bajty: number }

/** Statystyki indeksu jednego udziału. */
export interface StatystykiUdzialu {
  nazwa: string
  pliki: number
  foldery: number
  bajty: number
  wg_rodzaju: Record<string, LicznikRodzaju>
  /** Czas ostatniego pełnego indeksowania (RFC 3339), pusty = jeszcze nie zindeksowano. */
  zindeksowano: string
}

/** Odpowiedź GET /api/indeks/statystyki. */
export interface StatystykiIndeksu { udzialy: StatystykiUdzialu[] }

/** Odpowiedź GET /api/dziennik/mapa: 7 wierszy (pon..niedz) po 24 godziny, bajty pobrań. */
export interface MapaPobran { komorki: number[][] }

const ADRES_STRUMIENIA = '/api/monitor/strumien'
const PRZERWA_START_MS = 1000
const PRZERWA_MAKS_MS = 30000

export const pobierzMonitor = () => api.get<StanMonitoraApi>('/monitor')
export const pobierzStatystykiIndeksu = () => api.get<StatystykiIndeksu>('/indeks/statystyki')
export const pobierzMape = (od: string, doCzasu: string) => api.get<MapaPobran>('/dziennik/mapa', { od, do: doCzasu })

/**
 * Słucha próbek na żywo. `naStan(true)` po otwarciu połączenia, `naStan(false)` po jego utracie.
 * Zamknięte przez przeglądarkę połączenie (np. 404, 401) wznawia z rosnącą przerwą. Zwraca funkcję zamykającą.
 */
export function sluchajMonitora(naProbke: (p: Probka) => void, naStan: (polaczono: boolean) => void): () => void {
  let zrodlo: EventSource | null = null
  let zamkniete = false
  let przerwa = PRZERWA_START_MS
  let zegar: ReturnType<typeof setTimeout> | null = null

  const polacz = () => {
    if (zamkniete) return
    const es = new EventSource(ADRES_STRUMIENIA, { withCredentials: true })
    zrodlo = es
    es.addEventListener('probka', (e) => {
      try {
        naProbke(JSON.parse((e as MessageEvent).data as string) as Probka)
      } catch {
        // celowo: uszkodzona ramka SSE jest pomijana
      }
    })
    es.onopen = () => {
      przerwa = PRZERWA_START_MS
      naStan(true)
    }
    es.onerror = () => {
      naStan(false)
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
