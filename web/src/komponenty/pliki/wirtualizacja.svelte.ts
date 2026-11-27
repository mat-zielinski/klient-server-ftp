// Wirtualizacja listy i kafelków: renderujemy tylko widoczne wiersze + bufor, resztę zastępują odstępy.
// Wysokość wiersza mierzymy na żywo z wyrenderowanych elementów [data-indeks] (pierwszy pomiar: token
// --wysokosc-wiersz-tabeli), więc działa dla tabeli i siatki kafelków bez liczb wpisanych w kod.
import { BUFOR_WIRTUALIZACJI, POCZATKOWE_WIERSZE } from './stale'

/** Długość w jednostkach ekranu jako CSS bez wpisywania „px” (1 jednostka = token --odstep-ramka). */
export function dlugosc(n: number): string {
  return `calc(var(--odstep-ramka) * ${Math.max(0, Math.round(n))})`
}

function rodzicPrzewijania(el: HTMLElement): HTMLElement | null {
  for (let p = el.parentElement; p; p = p.parentElement) {
    const o = getComputedStyle(p).overflowY
    if ((o === 'auto' || o === 'scroll') && p.scrollHeight > p.clientHeight) return p
  }
  return null
}

function tokenWiersza(el: HTMLElement): number {
  return parseFloat(getComputedStyle(el).getPropertyValue('--wysokosc-wiersz-tabeli')) || 1
}

export class Wirtualizacja {
  start = $state(0)
  koniec = $state(POCZATKOWE_WIERSZE)
  /** Wysokość jednego wiersza (jednostki ekranu). */
  wiersz = $state(0)
  private el: HTMLElement | null = null
  /** Odległość pierwszego wiersza od góry kontenera (nagłówek tabeli, margines siatki). */
  private przesuniecie = 0
  private ramka = 0

  constructor(
    private liczba: () => number,
    private kolumny: () => number = () => 1,
  ) {}

  get gora(): number {
    return (this.start / this.kol) * this.wiersz
  }

  get dol(): number {
    const wierszy = Math.ceil(this.liczba() / this.kol)
    return Math.max(0, (wierszy - Math.ceil(this.koniec / this.kol)) * this.wiersz)
  }

  private get kol(): number {
    return Math.max(1, this.kolumny())
  }

  /** Podłącza kontener; zwraca sprzątanie. */
  podlacz(el: HTMLElement): () => void {
    this.el = el
    this.wiersz = tokenWiersza(el)
    const naZmiane = () => this.zaplanuj()
    addEventListener('scroll', naZmiane, true)
    addEventListener('resize', naZmiane)
    const obserwator = new ResizeObserver(naZmiane)
    obserwator.observe(el)
    this.aktualizuj()
    return () => {
      removeEventListener('scroll', naZmiane, true)
      removeEventListener('resize', naZmiane)
      obserwator.disconnect()
      cancelAnimationFrame(this.ramka)
      this.el = null
    }
  }

  zaplanuj(): void {
    cancelAnimationFrame(this.ramka)
    this.ramka = requestAnimationFrame(() => this.aktualizuj())
  }

  private zmierz(): void {
    if (!this.el) return
    const el = this.el.querySelectorAll<HTMLElement>('[data-indeks]')
    if (!el.length) return
    const pierwszy = el[0].getBoundingClientRect()
    const kol = this.kol
    const h = el.length > kol ? el[kol].getBoundingClientRect().top - pierwszy.top : el[0].offsetHeight
    if (h > 0) this.wiersz = h
    const indeks = Number(el[0].dataset.indeks) || 0
    this.przesuniecie = pierwszy.top - this.el.getBoundingClientRect().top - Math.floor(indeks / kol) * this.wiersz
  }

  /** Widoczny pas ekranu: część wspólna okna i przewijanego rodzica. */
  private widok(): { gora: number; dol: number; rodzic: HTMLElement | null } {
    const rodzic = this.el ? rodzicPrzewijania(this.el) : null
    const r = rodzic?.getBoundingClientRect()
    return { gora: Math.max(0, r?.top ?? 0), dol: Math.min(innerHeight, r?.bottom ?? innerHeight), rodzic }
  }

  aktualizuj(): void {
    if (!this.el) return
    this.zmierz()
    const w = this.widok()
    const poczatek = this.el.getBoundingClientRect().top + this.przesuniecie
    const kol = this.kol
    const pierwszy = Math.floor((w.gora - poczatek) / this.wiersz)
    const ostatni = Math.ceil((w.dol - poczatek) / this.wiersz)
    const start = Math.max(0, pierwszy - BUFOR_WIRTUALIZACJI) * kol
    const koniec = Math.min(this.liczba(), Math.max(0, ostatni + BUFOR_WIRTUALIZACJI) * kol)
    if (start !== this.start) this.start = Math.min(start, Math.max(0, koniec - kol))
    if (koniec !== this.koniec) this.koniec = Math.max(koniec, this.start)
  }

  /** Przewija tak, żeby wpis o danym indeksie był widoczny (także gdy nie jest wyrenderowany). */
  przewinDo(indeks: number): void {
    if (!this.el) return
    const w = this.widok()
    const gora = this.el.getBoundingClientRect().top + this.przesuniecie + Math.floor(indeks / this.kol) * this.wiersz
    const cel = w.rodzic ?? document.scrollingElement
    if (!cel) return
    if (gora < w.gora) cel.scrollTop -= w.gora - gora
    else if (gora + this.wiersz > w.dol) cel.scrollTop += gora + this.wiersz - w.dol
    this.zaplanuj()
  }
}
