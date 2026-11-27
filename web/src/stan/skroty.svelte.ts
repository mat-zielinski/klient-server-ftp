// Rejestr poleceń i skróty klawiszowe.
// Każda strona może dopisać swoje polecenia (pojawiają się w palecie Ctrl+K, a skrót działa w całej aplikacji):
//   const wyrejestruj = zarejestrujPolecenie({ id: 'pliki.nowy-folder', nazwa: 'Nowy folder', ikona: 'folder_plus',
//                                              skrot: 'Shift+N', akcja: () => …, widoczne: () => … })
//   $effect(() => zarejestrujPolecenie({...}))   // efekt zwraca wyrejestrowanie; albo onMount / onDestroy
// Rejestr jest bezpieczny w $effect: to zwykła Map, a zmianę ogłasza licznik wersji zapisywany w untrack(), więc
// rejestracja ani nie czyta, ani nie śledzi reaktywnego stanu efektu, który ją woła (bez pętli efektów).
// Skróty wbudowane: Ctrl+K (lub Cmd+K) paleta, „/” szukanie (polecenie o id SZUKAJ, a bez niego paleta),
// „?” pomoc skrótów, Esc zamyka paletę i pomoc.
// Format skrótu: „Ctrl+K”, „Shift+N”, „Alt+1”, „Delete”, „F2”, „N” (Ctrl = Ctrl albo Cmd na Macu).

import { untrack } from 'svelte'
import type { NazwaIkony } from '../ikony/ikony'

export interface Polecenie {
  id: string
  nazwa: string
  ikona?: NazwaIkony
  skrot?: string
  akcja: () => void
  /** Polecenie aktywne tylko, gdy zwraca true (np. tylko na stronie Pliki, tylko admin). */
  widoczne?: () => boolean
  /** Grupa w palecie i w pomocy (np. „Strony”, „Pliki”); domyślnie „Polecenia”. */
  grupa?: string
  /** Dodatkowe słowa do wyszukiwania (nie są wyświetlane). */
  slowa?: string
}

/** Id polecenia, które uruchamia „/” (rejestruje je strona z wyszukiwarką, np. Pliki). */
export const SZUKAJ = 'szukaj'
export const SKROT_PALETY = 'Ctrl+K'
export const SKROT_SZUKANIA = '/'
export const SKROT_POMOCY = '?'
export const GRUPA_DOMYSLNA = 'Polecenia'

class Skroty {
  /** Rejestr (zwykła Map, nie $state): kolejność = kolejność rejestracji. */
  private readonly mapa = new Map<string, Polecenie>()
  /** Licznik zmian rejestru: czytany przez `polecenia` (śledzenie), zapisywany tylko w untrack(). */
  private wersja = $state(0)
  paleta = $state(false)
  pomoc = $state(false)
  private zainstalowane = false

  /** Wszystkie zarejestrowane polecenia (reaktywne przez licznik wersji). */
  get polecenia(): Polecenie[] {
    void this.wersja
    return [...this.mapa.values()]
  }

  /** Dodaje polecenie (to samo id zastępuje poprzednie); nie czyta stanu reaktywnego. */
  dodaj(p: Polecenie): () => void {
    this.mapa.delete(p.id)
    this.mapa.set(p.id, p)
    this.zmieniono()
    return () => {
      if (this.mapa.get(p.id) !== p) return // już zastąpione nowszym poleceniem o tym id
      this.mapa.delete(p.id)
      this.zmieniono()
    }
  }

  private zmieniono(): void {
    untrack(() => {
      this.wersja += 1
    })
  }

  /** Polecenia aktywne w tej chwili (widoczne() === true lub brak warunku). */
  get aktywne(): Polecenie[] {
    return this.polecenia.filter((p) => !p.widoczne || p.widoczne())
  }

  otworzPalete(): void {
    this.pomoc = false
    this.paleta = true
  }

  otworzPomoc(): void {
    this.paleta = false
    this.pomoc = true
  }

  /** Jeden słuchacz klawiatury dla całej aplikacji (App.svelte). */
  instaluj(): () => void {
    if (this.zainstalowane || typeof window === 'undefined') return () => {}
    this.zainstalowane = true
    const f = (e: KeyboardEvent) => this.klawisz(e)
    window.addEventListener('keydown', f)
    return () => {
      window.removeEventListener('keydown', f)
      this.zainstalowane = false
    }
  }

  private klawisz(e: KeyboardEvent): void {
    if (e.defaultPrevented || e.isComposing) return
    if (pasujeSkrot(SKROT_PALETY, e)) {
      e.preventDefault()
      if (this.paleta) this.paleta = false
      else this.otworzPalete()
      return
    }
    if (this.paleta || this.pomoc || oknoModalneOtwarte()) return
    const wPolu = poleTekstowe(e.target)
    if (!wPolu && pasujeSkrot(SKROT_POMOCY, e)) {
      e.preventDefault()
      this.otworzPomoc()
      return
    }
    if (!wPolu && pasujeSkrot(SKROT_SZUKANIA, e)) {
      e.preventDefault()
      const szukaj = this.aktywne.find((p) => p.id === SZUKAJ)
      if (szukaj) szukaj.akcja()
      else this.otworzPalete()
      return
    }
    for (const p of this.aktywne) {
      if (!p.skrot || !pasujeSkrot(p.skrot, e)) continue
      if (wPolu && !maModyfikator(p.skrot)) continue
      e.preventDefault()
      p.akcja()
      return
    }
  }
}

export const skroty = new Skroty()

/** Dodaje polecenie (to samo id zastępuje poprzednie); zwraca funkcję wyrejestrowania. */
export function zarejestrujPolecenie(p: Polecenie): () => void {
  return skroty.dodaj(p)
}

// ------------------------------------------------------------------ skróty: dopasowanie i opis
interface Rozbior {
  ctrl: boolean
  shift: boolean
  alt: boolean
  klawisz: string
}

function rozbierz(skrot: string): Rozbior {
  const czesci = skrot.split('+').map((c) => c.trim())
  // „Ctrl++” → ostatnia część pusta, klawisz to „+”
  const klawisz = czesci.at(-1) === '' ? '+' : (czesci.at(-1) ?? '')
  const mod = czesci.slice(0, -1).map((c) => c.toLowerCase())
  return {
    ctrl: mod.includes('ctrl') || mod.includes('cmd'),
    shift: mod.includes('shift'),
    alt: mod.includes('alt'),
    klawisz: klawisz.toLowerCase(),
  }
}

/** Czy zdarzenie klawiatury pasuje do skrótu. Shift nie jest sprawdzany dla znaków, które go wymagają („?”). */
export function pasujeSkrot(skrot: string, e: KeyboardEvent): boolean {
  const r = rozbierz(skrot)
  const ctrl = e.ctrlKey || e.metaKey
  if (r.ctrl !== ctrl || r.alt !== e.altKey) return false
  const klawisz = e.key.toLowerCase()
  const znak = r.klawisz.length === 1 && !/[a-z0-9]/.test(r.klawisz)
  if (!znak && r.shift !== e.shiftKey) return false
  if (klawisz === r.klawisz) return true
  // Alt+1 / Shift+1 na układach, gdzie key to znak: porównanie po kodzie klawisza
  return /^[a-z0-9]$/.test(r.klawisz) && e.code.toLowerCase() === (/\d/.test(r.klawisz) ? 'digit' : 'key') + r.klawisz
}

function maModyfikator(skrot: string): boolean {
  const r = rozbierz(skrot)
  return r.ctrl || r.alt || r.klawisz.length > 1
}

/** Klawisze skrótu do wyświetlenia w <kbd> („Ctrl”, „K”); na Macu Ctrl jako „Cmd”. */
export function klawiszeSkrotu(skrot: string): string[] {
  const mac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
  return skrot
    .split(/\+(?!$)/)
    .map((c) => c.trim())
    .map((c) => (c.toLowerCase() === 'ctrl' && mac ? 'Cmd' : c.length === 1 ? c.toUpperCase() : c))
}

function poleTekstowe(cel: EventTarget | null): boolean {
  if (!(cel instanceof HTMLElement)) return false
  if (cel.isContentEditable) return true
  if (cel instanceof HTMLTextAreaElement || cel instanceof HTMLSelectElement) return true
  if (!(cel instanceof HTMLInputElement)) return false
  return !['checkbox', 'radio', 'button', 'submit', 'range', 'color', 'file'].includes(cel.type)
}

function oknoModalneOtwarte(): boolean {
  return typeof document !== 'undefined' && !!document.querySelector('dialog[open]')
}
