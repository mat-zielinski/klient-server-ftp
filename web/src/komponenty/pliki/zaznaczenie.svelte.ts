// Zaznaczenie wpisów jak w Eksploratorze: klik, Ctrl+klik, Shift+klik (zakres od kotwicy), kursor klawiatury.
import type { Wpis } from '../../api/typy'

export class Zaznaczenie {
  zbior = $state(new Set<string>())
  kursor = $state(-1)
  private kotwica = -1

  constructor(private wpisy: () => Wpis[]) {}

  get wybrane(): Wpis[] {
    return this.wpisy().filter((w) => this.zbior.has(w.sciezka))
  }

  get biezacy(): Wpis | undefined {
    return this.kursor >= 0 ? this.wpisy()[this.kursor] : undefined
  }

  wyczysc(): void {
    this.zbior = new Set()
    this.kursor = -1
    this.kotwica = -1
  }

  odznacz(): void {
    this.zbior = new Set()
  }

  wszystko(): void {
    this.zbior = new Set(this.wpisy().map((w) => w.sciezka))
  }

  jeden(i: number): void {
    const w = this.wpisy()[i]
    if (!w) return
    this.zbior = new Set([w.sciezka])
    this.kursor = i
    this.kotwica = i
  }

  przelacz(i: number): void {
    const w = this.wpisy()[i]
    if (!w) return
    const n = new Set(this.zbior)
    if (n.has(w.sciezka)) n.delete(w.sciezka)
    else n.add(w.sciezka)
    this.zbior = n
    this.kursor = i
    this.kotwica = i
  }

  zakres(i: number): void {
    const od = this.kotwica < 0 ? i : this.kotwica
    const [a, b] = od < i ? [od, i] : [i, od]
    this.zbior = new Set(this.wpisy().slice(a, b + 1).map((w) => w.sciezka))
    this.kursor = i
  }

  /** Klik myszą: Shift = zakres, Ctrl / tryb wielu = przełącz, inaczej jeden. */
  klik(i: number, e: MouseEvent, wielu: boolean): void {
    if (e.shiftKey) this.zakres(i)
    else if (e.ctrlKey || e.metaKey || wielu) this.przelacz(i)
    else this.jeden(i)
  }

  /** Prawy przycisk: wpis spoza zaznaczenia zostaje jedynym zaznaczonym. Zwraca cel menu. */
  doMenu(i: number): Wpis[] {
    const w = this.wpisy()[i]
    if (!w) return []
    if (!this.zbior.has(w.sciezka)) this.jeden(i)
    else this.kursor = i
    return this.zbior.size > 1 ? this.wybrane : [w]
  }

  /** Strzałki: przesuwa kursor (z Shift rozszerza zakres); zwraca nowy indeks albo -1. */
  przesun(krok: number, rozszerz: boolean): number {
    const n = this.wpisy().length
    if (!n) return -1
    const nowy = Math.min(n - 1, Math.max(0, this.kursor + krok))
    if (rozszerz) this.zakres(nowy)
    else this.jeden(nowy)
    return nowy
  }
}
