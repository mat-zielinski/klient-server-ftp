// Wspólna matematyka wykresów SVG: skale, ładne podziałki, gładkie ścieżki, obserwacja rozmiaru.
// Bez bibliotek; wszystko w jednostkach SVG (liczby bez jednostek), wygląd w CSS przez tokeny.

import type { Action } from 'svelte/action'
import { WYKRES } from './stale'

export type Barwa = 'niebieski' | 'zielony' | 'fioletowy' | 'pomaranczowy' | 'rozowy' | 'zolty'

/** Kolejność barw serii (pierścień, rankingi), jak kolejne serie w stacji. */
export const BARWY_SERII: Barwa[] = ['niebieski', 'zielony', 'fioletowy', 'pomaranczowy', 'rozowy', 'zolty']

/** Zmienna CSS barwy serii, np. `var(--kolor-niebieski)`. */
export function kolorBarwy(barwa: Barwa): string {
  return `var(--kolor-${barwa})`
}

/** Punkt serii czasowej: `t` w milisekundach od epoki, `v` wartość. */
export interface PunktCzasu { t: number; v: number }

/** Kategoria wykresu słupkowego: opis osi, tytuł dymka, wartość serii A (słupki) i B (linia). */
export interface PunktSlupka { etykieta: string; opis: string; a: number; b: number }

/** Opis serii: nazwa w legendzie i dymku, barwa, format liczb osi i dymka. */
export interface Seria { nazwa: string; barwa: Barwa; format: (v: number) => string; os?: RodzajOsi }

/** Wycinek pierścienia albo pozycja rankingu. */
export interface Pozycja { nazwa: string; wartosc: number; opis?: string }

/** Skala liniowa z dziedziny [d0, d1] na zakres [r0, r1]. */
export function liniowa(d0: number, d1: number, r0: number, r1: number): (x: number) => number {
  const rozpietosc = d1 - d0 || 1
  return (x: number) => r0 + ((x - d0) / rozpietosc) * (r1 - r0)
}

/** Ładny krok podziałki (1, 2, 2,5, 5 × 10^n) dla zakresu podzielonego na około `ile` części. */
function ladnyKrok(zakres: number, ile: number): number {
  const surowy = zakres / Math.max(1, ile)
  const potega = Math.pow(10, Math.floor(Math.log10(surowy)))
  const wzgledny = surowy / potega
  const mnoznik = WYKRES.ladneMnozniki.find((m) => wzgledny <= m) ?? 10
  return mnoznik * potega
}

/** Rodzaj osi wartości: `bajty` = ładne kroki w jednostkach 1024 (KB, MB…), `liczba` = tylko całkowite. */
export type RodzajOsi = 'zwykla' | 'bajty' | 'liczba'

/** Podziałki osi od 0 do ładnego maksimum ≥ `max`; pusty zakres daje [0, 1]. */
export function podzialki(max: number, rodzaj: RodzajOsi = 'zwykla', ile: number = WYKRES.liczbaPodzialek): number[] {
  if (!(max > 0)) return [0, 1]
  if (rodzaj === 'bajty' && max >= WYKRES.podstawaBajtow) {
    const potega = Math.pow(WYKRES.podstawaBajtow, Math.floor(Math.log(max) / Math.log(WYKRES.podstawaBajtow)))
    return podzialki(max / potega, 'zwykla', ile).map((v) => v * potega)
  }
  let krok = ladnyKrok(max, ile)
  if (rodzaj === 'liczba') krok = Math.max(1, Math.round(krok))
  const wynik: number[] = []
  for (let v = 0; v < max + krok * 0.5 && wynik.length <= ile * 3; v += krok) wynik.push(v)
  if (wynik[wynik.length - 1] < max) wynik.push(wynik[wynik.length - 1] + krok)
  return wynik
}

/** Gładka ścieżka monotoniczna (Fritsch–Carlson): nie przestrzeliwuje ponad dane, jak w UniFi. */
export function sciezkaGladka(xs: number[], ys: number[]): string {
  const n = xs.length
  if (n === 0) return ''
  if (n === 1) return `M${xs[0]},${ys[0]}`
  const nachylenia = styczne(xs, ys)
  let d = `M${xs[0]},${ys[0]}`
  for (let i = 0; i < n - 1; i++) {
    const h = (xs[i + 1] - xs[i]) / 3
    d += ` C${xs[i] + h},${ys[i] + h * nachylenia[i]} ${xs[i + 1] - h},${ys[i + 1] - h * nachylenia[i + 1]} ${xs[i + 1]},${ys[i + 1]}`
  }
  return d
}

function styczne(xs: number[], ys: number[]): number[] {
  const n = xs.length
  const delta: number[] = []
  for (let i = 0; i < n - 1; i++) delta.push((ys[i + 1] - ys[i]) / (xs[i + 1] - xs[i] || 1))
  const m: number[] = [delta[0]]
  for (let i = 1; i < n - 1; i++) m.push(delta[i - 1] * delta[i] <= 0 ? 0 : (delta[i - 1] + delta[i]) / 2)
  m.push(delta[n - 2])
  for (let i = 0; i < n - 1; i++) {
    if (delta[i] === 0) { m[i] = 0; m[i + 1] = 0; continue }
    const a = m[i] / delta[i]
    const b = m[i + 1] / delta[i]
    const s = a * a + b * b
    if (s > 9) {
      const t = 3 / Math.sqrt(s)
      m[i] = t * a * delta[i]
      m[i + 1] = t * b * delta[i]
    }
  }
  return m
}

/** Indeks punktu najbliższego `x` w posortowanej rosnąco tablicy (wyszukiwanie binarne). */
export function najblizszy(xs: number[], x: number): number {
  let lo = 0
  let hi = xs.length - 1
  if (hi < 0) return -1
  while (hi - lo > 1) {
    const sr = (lo + hi) >> 1
    if (xs[sr] < x) lo = sr
    else hi = sr
  }
  return Math.abs(xs[lo] - x) <= Math.abs(xs[hi] - x) ? lo : hi
}

/** Obserwuje rozmiar elementu (ResizeObserver) i podaje szerokość i wysokość treści. */
export const obserwujRozmiar: Action<HTMLElement, (szerokosc: number, wysokosc: number) => void> = (wezel, naZmiane) => {
  let wywolaj = naZmiane
  const obserwator = new ResizeObserver((wpisy) => {
    const prost = wpisy[0]?.contentRect
    if (prost && wywolaj) wywolaj(prost.width, prost.height)
  })
  obserwator.observe(wezel)
  return {
    update(nowa) { wywolaj = nowa },
    destroy() { obserwator.disconnect() },
  }
}

/** Unikalny identyfikator elementu SVG (gradient, przycięcie) w obrębie strony. */
let licznikId = 0
export function noweId(przedrostek: string): string {
  licznikId += 1
  return `${przedrostek}-${licznikId}`
}
