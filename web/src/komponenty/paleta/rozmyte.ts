// Rozmyte dopasowanie dla palety poleceń: litery zapytania muszą wystąpić w tekście w tej kolejności
// (bez wielkości liter i polskich znaków: „zolc” pasuje do „Żółć”). Wynik wyższy = lepsze dopasowanie:
// premia za kolejne litery, początek słowa i początek tekstu; kara za długość tekstu.

const PREMIA_CIAG = 5
const PREMIA_SLOWO = 8
const PREMIA_POCZATEK = 10
const KARA_DLUGOSC = 0.1

/** Małe litery bez znaków diakrytycznych (ł → l osobno, bo NFD jej nie rozkłada). Długość bez zmian. */
export function uprosc(t: string): string {
  return t
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/ł/g, 'l')
}

export interface Dopasowanie {
  wynik: number
  /** Indeksy dopasowanych liter w tekście (do podświetlenia). */
  pozycje: number[]
}

/** Dopasowanie zapytania do tekstu albo null. Puste zapytanie pasuje do wszystkiego z wynikiem 0. */
export function dopasuj(zapytanie: string, tekst: string): Dopasowanie | null {
  const z = uprosc(zapytanie.trim()).replace(/\s+/g, ' ')
  if (!z) return { wynik: 0, pozycje: [] }
  const t = uprosc(tekst)
  // całe zapytanie jako ciąg: najlepszy przypadek
  const wprost = t.indexOf(z)
  if (wprost >= 0) {
    const pozycje = Array.from({ length: z.length }, (_, i) => wprost + i)
    const slowo = wprost === 0 || /[\s\-_/:.]/.test(t[wprost - 1])
    return { wynik: 100 + (slowo ? PREMIA_SLOWO : 0) + (wprost === 0 ? PREMIA_POCZATEK : 0) - t.length * KARA_DLUGOSC, pozycje }
  }
  const pozycje: number[] = []
  let wynik = 0
  let j = 0
  for (let i = 0; i < t.length && j < z.length; i++) {
    if (z[j] === ' ') {
      j++
      i--
      continue
    }
    if (t[i] !== z[j]) continue
    if (pozycje.length && pozycje[pozycje.length - 1] === i - 1) wynik += PREMIA_CIAG
    if (i === 0) wynik += PREMIA_POCZATEK
    else if (/[\s\-_/:.]/.test(t[i - 1])) wynik += PREMIA_SLOWO
    wynik += 1
    pozycje.push(i)
    j++
  }
  if (j < z.length) return null
  return { wynik: wynik - t.length * KARA_DLUGOSC, pozycje }
}

/** Filtruje i sortuje elementy wg dopasowania (najlepsze pierwsze, przy remisie kolejność wejścia). */
export function filtruj<T>(elementy: T[], zapytanie: string, tekst: (e: T) => string): { e: T; d: Dopasowanie }[] {
  const wynik: { e: T; d: Dopasowanie; i: number }[] = []
  elementy.forEach((e, i) => {
    const d = dopasuj(zapytanie, tekst(e))
    if (d) wynik.push({ e, d, i })
  })
  wynik.sort((a, b) => b.d.wynik - a.d.wynik || a.i - b.i)
  return wynik.map(({ e, d }) => ({ e, d }))
}
