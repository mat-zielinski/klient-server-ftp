// Motyw: wybór użytkownika 'auto' (domyślny, wg prefers-color-scheme systemu), 'ciemny' albo 'jasny'.
// Na <html> zawsze trafia motyw WYNIKOWY (data-motyw="ciemny" | "jasny"), więc tokeny.css zna tylko dwa motywy.
// Wybór zapamiętany w przeglądarce (localStorage); bez wyboru obowiązuje motyw z konfiguracji serwera, a bez niego auto.
// Meta theme-color (pasek przeglądarki na telefonie, PWA) dostaje kolor paska bocznego z tokenów.

export type Motyw = 'ciemny' | 'jasny'
export type WyborMotywu = 'auto' | Motyw

const KLUCZ = 'skrytka_motyw'
const ZAPYTANIE_JASNY = '(prefers-color-scheme: light)'
const KOLEJNOSC: WyborMotywu[] = ['auto', 'ciemny', 'jasny']

export const NAZWY_MOTYWOW: Record<WyborMotywu, string> = {
  auto: 'Motyw automatyczny',
  ciemny: 'Motyw ciemny',
  jasny: 'Motyw jasny',
}

function zapamietany(): WyborMotywu | null {
  try {
    const w = localStorage.getItem(KLUCZ)
    return w === 'auto' || w === 'jasny' || w === 'ciemny' ? w : null
  } catch {
    return null // celowo: przeglądarka bez localStorage
  }
}

function zapytanie(): MediaQueryList | null {
  return typeof matchMedia === 'undefined' ? null : matchMedia(ZAPYTANIE_JASNY)
}

class Motywy {
  wybor = $state<WyborMotywu>(zapamietany() ?? 'auto')
  private systemJasny = $state(zapytanie()?.matches ?? false)

  /** Motyw wynikowy (to, co widać). */
  get biezacy(): Motyw {
    if (this.wybor === 'auto') return this.systemJasny ? 'jasny' : 'ciemny'
    return this.wybor
  }

  constructor() {
    zapytanie()?.addEventListener('change', (e) => {
      this.systemJasny = e.matches
      this.zastosuj()
    })
    this.zastosuj()
  }

  private zastosuj(): void {
    if (typeof document === 'undefined') return
    document.documentElement.dataset.motyw = this.biezacy
    const kolor = getComputedStyle(document.documentElement).getPropertyValue('--kolor-pasek').trim()
    let meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
    if (!meta) {
      meta = document.createElement('meta')
      meta.name = 'theme-color'
      document.head.append(meta)
    }
    if (kolor) meta.content = kolor
  }

  ustaw(w: WyborMotywu, zapamietaj = true): void {
    this.wybor = w
    this.zastosuj()
    if (!zapamietaj) return
    try {
      localStorage.setItem(KLUCZ, w)
    } catch {
      // celowo: brak zapisu nie blokuje zmiany motywu
    }
  }

  /** Następna opcja w kolejności: automatyczny → ciemny → jasny → automatyczny. */
  przelacz(): void {
    this.ustaw(KOLEJNOSC[(KOLEJNOSC.indexOf(this.wybor) + 1) % KOLEJNOSC.length])
  }

  /** Motyw z serwera obowiązuje tylko, gdy użytkownik nie wybrał własnego. */
  zSerwera(m: string | undefined): void {
    if (zapamietany()) return
    if (m === 'jasny' || m === 'ciemny' || m === 'auto') this.ustaw(m, false)
  }
}

export const motyw = new Motywy()
