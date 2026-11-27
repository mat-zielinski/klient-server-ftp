// Monitor transferów na żywo dla Pulpitu: lista aktywnych (GET /api/transfery + zdarzenia SSE),
// seria łącznej prędkości z ostatnich 5 minut (zdarzenia „suma” co 1 s), przerywanie (DELETE /api/transfery?id).

import { api, trescBledu } from '../../api/klient'
import { sluchajTransferow } from '../../api/transfery'
import type { StanTransferu, SumaTransferow, ZdarzenieTransferu } from '../../api/typy'
import { NA_ZYWO } from '../wykresy/stale'
import type { PunktCzasu } from '../wykresy/skala'

export class MonitorTransferow {
  transfery = $state.raw<StanTransferu[]>([])
  seria = $state.raw<PunktCzasu[]>([])
  suma = $state<SumaTransferow>({ predkosc_bps: 0, aktywne: 0 })
  /** Rośnie przy każdym zakończonym transferze (Pulpit odświeża wtedy liczniki „dziś”). */
  zakonczone = $state(0)
  /** Czy przyszła już pierwsza suma (do tego czasu kafle pokazują stan z /api/pulpit). */
  polaczono = $state(false)
  /** Błąd wczytania listy transferów (pusty = brak błędu). */
  blad = $state('')

  #zatrzymaj: (() => void) | null = null

  /** Wczytuje bieżące transfery i zaczyna słuchać strumienia. Zwraca funkcję sprzątającą. */
  start(): () => void {
    void this.wczytaj()
    this.#zatrzymaj = sluchajTransferow(
      (z) => this.naZdarzenie(z),
      (s) => this.naSume(s, Date.now()),
    )
    return () => this.stop()
  }

  stop(): void {
    this.#zatrzymaj?.()
    this.#zatrzymaj = null
  }

  async wczytaj(): Promise<void> {
    try {
      const lista = await api.get<StanTransferu[]>('/transfery')
      this.transfery = Array.isArray(lista) ? lista : []
      this.blad = ''
    } catch (b) {
      this.blad = trescBledu(b)
    }
  }

  naZdarzenie(z: ZdarzenieTransferu): void {
    const id = z.stan.id
    if (z.typ === 'koniec') {
      this.transfery = this.transfery.filter((t) => t.id !== id)
      this.zakonczone += 1
      return
    }
    const i = this.transfery.findIndex((t) => t.id === id)
    if (i < 0) this.transfery = [...this.transfery, z.stan]
    else this.transfery = this.transfery.map((t, j) => (j === i ? z.stan : t))
  }

  naSume(s: SumaTransferow, teraz: number): void {
    this.suma = s
    this.polaczono = true
    const granica = teraz - NA_ZYWO.oknoMs - NA_ZYWO.opoznienieMs - NA_ZYWO.krokMs
    const zostaje = this.seria.filter((p) => p.t >= granica)
    this.seria = [...zostaje, { t: teraz, v: s.predkosc_bps }]
  }

  async przerwij(id: string): Promise<void> {
    await api.usun('/transfery', { id })
    this.transfery = this.transfery.filter((t) => t.id !== id)
  }
}
