// Stan strony Monitor: odczyt GET /api/monitor, potem próbki na żywo z SSE. Historia 10 min (co 1 s) przycinana
// do okna; historia 24 h (co 1 min) z serwera, dopisywana na żywo co minutę. Brak adresu na serwerze (404) =
// `brakEndpointu`, strona pokazuje wtedy stan pusty zamiast się wysypać.

import { BladApi, trescBledu } from '../../api/klient'
import { pobierzMonitor, sluchajMonitora, type InfoSystemu, type Probka } from '../../api/monitor'
import type { PunktCzasu } from '../wykresy/skala'
import { MONITOR } from './stale'

/** Czas próbki w ms (zły zapis = 0, taka próbka trafia na początek i wypada przy przycinaniu). */
export function czasProbki(p: Probka): number {
  const t = Date.parse(p.czas)
  return Number.isFinite(t) ? t : 0
}

/** Seria punktów wykresu z próbek. */
export function seria(probki: Probka[], wartosc: (p: Probka) => number): PunktCzasu[] {
  return probki.map((p) => ({ t: czasProbki(p), v: Math.max(0, wartosc(p) || 0) }))
}

/** Seria dwóch wartości (np. wysyłanie i odbiór) do WykresDwuSerii. */
export interface PunktDwuSerii { t: number; a: number; b: number }

export function seriaPodwojna(probki: Probka[], a: (p: Probka) => number, b: (p: Probka) => number): PunktDwuSerii[] {
  return probki.map((p) => ({ t: czasProbki(p), a: Math.max(0, a(p) || 0), b: Math.max(0, b(p) || 0) }))
}

/** Ułamek 0..1 (dzielnik zerowy = 0). */
export function udzial(czesc: number, calosc: number): number {
  return calosc > 0 ? Math.min(1, Math.max(0, czesc / calosc)) : 0
}

function przytnij(probki: Probka[], oknoMs: number, koniec: number): Probka[] {
  const granica = koniec - oknoMs - MONITOR.zapasMs
  let i = 0
  while (i < probki.length && czasProbki(probki[i]) < granica) i++
  return i > 0 ? probki.slice(i) : probki
}

export class StanMonitora {
  system = $state.raw<InfoSystemu | null>(null)
  teraz = $state.raw<Probka | null>(null)
  historia = $state.raw<Probka[]>([])
  historiaDoby = $state.raw<Probka[]>([])
  ladowanie = $state(true)
  blad = $state('')
  brakEndpointu = $state(false)
  polaczono = $state(false)
  /** Chwila odczytu `system.czas_pracy_s` i chwila ostatniej próbki (do liczenia czasu pracy na żywo). */
  wczytanoO = $state(Date.now())
  zegar = $state(Date.now())

  /** Czas pracy serwera w sekundach liczony na żywo albo null, gdy nieznany. */
  get czasPracy(): number | null {
    if (!this.system) return null
    return this.system.czas_pracy_s + Math.max(0, (this.zegar - this.wczytanoO) / 1000)
  }

  async wczytaj(): Promise<void> {
    try {
      const odp = await pobierzMonitor()
      this.system = odp.system
      this.wczytanoO = Date.now()
      this.zegar = this.wczytanoO
      const koniec = odp.teraz ? czasProbki(odp.teraz) : Date.now()
      this.historia = przytnij(odp.historia_1s ?? [], MONITOR.oknoNaZywoMs, koniec)
      this.historiaDoby = przytnij(odp.historia_1m ?? [], MONITOR.oknoDobyMs, koniec)
      this.teraz = odp.teraz ?? null
      this.blad = ''
      this.brakEndpointu = false
    } catch (b) {
      this.brakEndpointu = b instanceof BladApi && (b.kod === 404 || b.kod === 405 || b.kod === 501)
      this.blad = trescBledu(b)
    } finally {
      this.ladowanie = false
    }
  }

  dodaj(p: Probka): void {
    const t = czasProbki(p)
    const ostatnia = this.historia[this.historia.length - 1]
    if (ostatnia && czasProbki(ostatnia) >= t) return // powtórka albo stara ramka po wznowieniu
    this.teraz = p
    this.zegar = Date.now()
    this.historia = przytnij([...this.historia, p], MONITOR.oknoNaZywoMs, t)
    const minuta = this.historiaDoby[this.historiaDoby.length - 1]
    if (!minuta || t - czasProbki(minuta) >= MONITOR.krokMinutowyMs) {
      this.historiaDoby = przytnij([...this.historiaDoby, p], MONITOR.oknoDobyMs, t)
    }
    if (this.blad && !this.brakEndpointu) this.blad = ''
  }

  /** Uruchamia odczyt i strumień; zwraca funkcję zatrzymującą. */
  start(): () => void {
    void this.wczytaj()
    return sluchajMonitora(
      (p) => this.dodaj(p),
      (polaczono) => {
        this.polaczono = polaczono
        // strumień ruszył, a odczyt startowy się nie udał (np. serwer wstał później): spróbuj ponownie
        if (polaczono && !this.system) void this.wczytaj()
      },
    )
  }
}
