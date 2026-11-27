// Klawiatura listy plików: strzałki (z Shift rozszerzają zaznaczenie), Home/End, Enter, Delete, F2, Backspace,
// Esc, Ctrl+A, klawisz menu i Shift+F10.

export interface CelKlawiatury {
  /** Widok kafelków: strzałki w bok działają, góra / dół skaczą o wiersz. */
  kafelki: () => boolean
  liczba: () => number
  /** Ile kafelków w wierszu (1 dla listy). */
  wKolumnie: () => number
  przesun: (krok: number, rozszerz: boolean) => void
  otworz: () => void
  usun: () => void
  zmienNazwe: () => void
  wyzej: () => void
  odznacz: () => void
  zaznaczWszystko: () => void
  menu: () => void
}

/** Ile kafelków mieści się w pierwszym wierszu kontenera (elementy z atrybutem data-indeks). */
export function kafelkowWWierszu(obszar: HTMLElement | null): number {
  if (!obszar) return 1
  const el = [...obszar.querySelectorAll<HTMLElement>('[data-indeks]')]
  if (el.length < 2) return 1
  const gora = el[0].offsetTop
  const n = el.findIndex((e) => e.offsetTop !== gora)
  return n < 0 ? el.length : n
}

export function obslugaKlawiszy(c: CelKlawiatury): (e: KeyboardEvent) => void {
  return (e) => {
    if ((e.target as HTMLElement).closest('input, textarea, select, [contenteditable]')) return
    const s = e.shiftKey
    const mapa: Record<string, () => void> = {
      ArrowDown: () => c.przesun(c.wKolumnie(), s),
      ArrowUp: () => c.przesun(-c.wKolumnie(), s),
      ArrowRight: () => c.kafelki() && c.przesun(1, s),
      ArrowLeft: () => c.kafelki() && c.przesun(-1, s),
      Home: () => c.przesun(-c.liczba(), s),
      End: () => c.przesun(c.liczba(), s),
      Enter: c.otworz,
      Delete: c.usun,
      F2: c.zmienNazwe,
      Backspace: c.wyzej,
      Escape: c.odznacz,
      ContextMenu: c.menu,
    }
    let f = mapa[e.key]
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a') f = c.zaznaczWszystko
    else if (s && e.key === 'F10') f = c.menu
    if (!f) return
    e.preventDefault()
    f()
  }
}
