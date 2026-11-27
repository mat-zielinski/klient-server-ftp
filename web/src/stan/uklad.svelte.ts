// Układ responsywny: szerokość okna i progi z tokenów (--prog-telefon, --prog-waski, --prog-sredni).
// Media queries w CSS nie przyjmują var(), więc progi czytamy tu raz z tokeny.css (jedno źródło prawdy).

export type Prog = 'telefon' | 'waski' | 'sredni'

const DOMYSLNE: Record<Prog, number> = { telefon: 640, waski: 1024, sredni: 1280 }

function czytajProgi(): Record<Prog, number> {
  if (typeof document === 'undefined') return DOMYSLNE
  const styl = getComputedStyle(document.documentElement)
  const wynik = { ...DOMYSLNE }
  for (const k of Object.keys(DOMYSLNE) as Prog[]) {
    const n = parseFloat(styl.getPropertyValue(`--prog-${k}`))
    if (Number.isFinite(n) && n > 0) wynik[k] = n
  }
  return wynik
}

class Uklad {
  szerokosc = $state(typeof window === 'undefined' ? DOMYSLNE.sredni : window.innerWidth)
  progi = $state.raw<Record<Prog, number>>(DOMYSLNE)

  constructor() {
    if (typeof window === 'undefined') return
    window.addEventListener('resize', () => {
      this.szerokosc = window.innerWidth
    })
  }

  /** Wywołać po wczytaniu stylów (main.ts). */
  odczytajProgi(): void {
    this.progi = czytajProgi()
    this.szerokosc = window.innerWidth
  }

  /** Okno węższe niż próg. */
  ponizej(prog: Prog): boolean {
    return this.szerokosc < this.progi[prog]
  }

  get telefon(): boolean {
    return this.ponizej('telefon')
  }

  get waski(): boolean {
    return this.ponizej('waski')
  }
}

export const uklad = new Uklad()
