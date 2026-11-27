// Zdarzenia widoków listy i kafelków przekazywane do strony Pliki (jeden kształt dla obu widoków).
import type { Wpis } from '../../api/typy'

export interface ZdarzeniaWpisow {
  /** Klik: zaznacza (Ctrl dokłada, Shift zakres). */
  klik: (w: Wpis, indeks: number, e: MouseEvent) => void
  /** Pole wyboru w trybie wielu: przełącza jeden wpis. */
  przelacz: (w: Wpis, indeks: number) => void
  /** Dwuklik / Enter: folder = wejście, plik = podgląd. */
  otworz: (w: Wpis) => void
  /** Prawy przycisk albo przycisk „więcej”. */
  menu: (w: Wpis, indeks: number, e: MouseEvent) => void
  sortuj: (klucz: string, malejaco: boolean) => void
}
