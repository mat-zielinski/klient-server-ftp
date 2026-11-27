// Zakres dat Dziennika: dziś, 7 dni, 30 dni albo własny (od–do z pól daty), krok wykresu godzina / dzień.

import { DZIENNIK, MS_DZIEN } from './stale'

export type RodzajZakresu = 'dzis' | 'tydzien' | 'miesiac' | 'wlasny'
export type Krok = 'godzina' | 'dzien'

export interface Zakres {
  rodzaj: RodzajZakresu
  /** Dla zakresu własnego: daty `RRRR-MM-DD` (włącznie). */
  od: string
  do: string
  krok: Krok
}

export const ZAKRESY: { id: RodzajZakresu; etykieta: string }[] = [
  { id: 'dzis', etykieta: 'Dziś' },
  { id: 'tydzien', etykieta: '7 dni' },
  { id: 'miesiac', etykieta: '30 dni' },
  { id: 'wlasny', etykieta: 'Własny' },
]

export const KROKI: { id: Krok; etykieta: string }[] = [
  { id: 'godzina', etykieta: 'Godzina' },
  { id: 'dzien', etykieta: 'Dzień' },
]

function poczatekDnia(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

/** Data `RRRR-MM-DD` w czasie lokalnym (do pól `<input type="date">`). */
export function dataPola(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const dz = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${dz}`
}

function zPola(tekst: string): Date | null {
  const czesci = /^(\d{4})-(\d{2})-(\d{2})$/.exec(tekst)
  if (!czesci) return null
  return new Date(Number(czesci[1]), Number(czesci[2]) - 1, Number(czesci[3]))
}

/** Granice zakresu [od, do) jako daty; zakres własny z błędnymi polami wraca do dzisiejszego dnia. */
export function granice(z: Zakres, teraz: Date): { od: Date; do: Date } {
  const dzis = poczatekDnia(teraz)
  const wstecz = (dni: number) => new Date(dzis.getTime() - (dni - 1) * MS_DZIEN)
  switch (z.rodzaj) {
    case 'tydzien':
      return { od: wstecz(DZIENNIK.dniTydzien), do: teraz }
    case 'miesiac':
      return { od: wstecz(DZIENNIK.dniMiesiac), do: teraz }
    case 'wlasny': {
      const od = zPola(z.od) ?? dzis
      const doDnia = zPola(z.do) ?? dzis
      const [a, b] = od <= doDnia ? [od, doDnia] : [doDnia, od]
      return { od: a, do: new Date(b.getFullYear(), b.getMonth(), b.getDate() + 1) }
    }
    default:
      return { od: dzis, do: teraz }
  }
}

/** Parametry zapytania `od`, `do` (RFC 3339) dla API dziennika. */
export function parametryZakresu(z: Zakres, teraz: Date): { od: string; do: string } {
  const g = granice(z, teraz)
  return { od: g.od.toISOString(), do: g.do.toISOString() }
}

/** Krok domyślny dla zakresu: dziś godzinowo, dłuższe dziennie. */
export function krokDomyslny(rodzaj: RodzajZakresu): Krok {
  return rodzaj === 'dzis' ? 'godzina' : 'dzien'
}

/** Krótki opis punktu na osi: „14:00” albo „27.11”. */
export function etykietaPunktu(iso: string, krok: Krok): string {
  const d = new Date(iso)
  if (krok === 'godzina') return d.toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString('pl-PL', { day: '2-digit', month: '2-digit' })
}

/** Pełny opis przedziału w dymku: „27.11.2026, 14:00–15:00” albo „piątek, 27.11.2026”. */
export function opisPunktu(iso: string, krok: Krok): string {
  const d = new Date(iso)
  const dzien = d.toLocaleDateString('pl-PL', { day: '2-digit', month: '2-digit', year: 'numeric' })
  if (krok === 'dzien') return `${d.toLocaleDateString('pl-PL', { weekday: 'long' })}, ${dzien}`
  const godz = (x: Date) => x.toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit' })
  return `${dzien}, ${godz(d)}–${godz(new Date(d.getTime() + MS_DZIEN / 24))}`
}
