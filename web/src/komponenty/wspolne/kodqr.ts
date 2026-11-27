// Koder QR bez bibliotek (ISO/IEC 18004): tryb bajtowy (UTF-8), poziom korekcji M, wersje 1–10.
// Wynik: macierz modułów (true = ciemny). Rysuje ją KodQR.svelte. Sprawdzenie: opis w docs/obszary/web-powloka.md.

/** Struktura bloków dla poziomu M: [kodów korekcji na blok, [liczba bloków, kodów danych w bloku][]]. */
const BLOKI_M: [number, [number, number][]][] = [
  [10, [[1, 16]]],
  [16, [[1, 28]]],
  [26, [[1, 44]]],
  [18, [[2, 32]]],
  [24, [[2, 43]]],
  [16, [[4, 27]]],
  [18, [[4, 31]]],
  [22, [[2, 38], [2, 39]]],
  [22, [[3, 36], [2, 37]]],
  [26, [[4, 43], [1, 44]]],
]

/** Środki wzorów wyrównania dla wersji 1–10. */
const WYROWNANIE: number[][] = [
  [],
  [6, 18],
  [6, 22],
  [6, 26],
  [6, 30],
  [6, 34],
  [6, 22, 38],
  [6, 24, 42],
  [6, 26, 46],
  [6, 28, 50],
]

const WERSJA_MAKS = 10
const POZIOM_M = 0b00
const MASKA_FORMATU = 0x5412
const WIELOMIAN_FORMATU = 0x537
const WIELOMIAN_WERSJI = 0x1f25
const WERSJA_Z_INFO = 7
const TRYB_BAJTOWY = 0b0100
const WYPELNIENIE = [0xec, 0x11]

// ------------------------------------------------------------------ GF(256) i Reed-Solomon
const EXP = new Uint8Array(512)
const LOG = new Uint8Array(256)
{
  let x = 1
  for (let i = 0; i < 255; i++) {
    EXP[i] = x
    LOG[x] = i
    x <<= 1
    if (x & 0x100) x ^= 0x11d
  }
  for (let i = 255; i < 512; i++) EXP[i] = EXP[i - 255]
}

function mnoz(a: number, b: number): number {
  return a && b ? EXP[LOG[a] + LOG[b]] : 0
}

function generator(stopien: number): number[] {
  let g = [1]
  for (let i = 0; i < stopien; i++) {
    const nowy = new Array<number>(g.length + 1).fill(0)
    for (let j = 0; j < g.length; j++) {
      nowy[j] ^= g[j]
      nowy[j + 1] ^= mnoz(g[j], EXP[i])
    }
    g = nowy
  }
  return g
}

/** Kody korekcji Reed-Solomona dla bloku danych. */
export function korekcja(dane: number[], ile: number): number[] {
  const g = generator(ile)
  const reszta = [...dane, ...new Array<number>(ile).fill(0)]
  for (let i = 0; i < dane.length; i++) {
    const wsp = reszta[i]
    if (!wsp) continue
    for (let j = 0; j < g.length; j++) reszta[i + j] ^= mnoz(g[j], wsp)
  }
  return reszta.slice(dane.length)
}

// ------------------------------------------------------------------ BCH: format i wersja
function bch(wartosc: number, wielomian: number, bity: number): number {
  const stopien = Math.floor(Math.log2(wielomian))
  let r = wartosc << stopien
  for (let i = bity + stopien - 1; i >= stopien; i--) if ((r >> i) & 1) r ^= wielomian << (i - stopien)
  return (wartosc << stopien) | r
}

/** 15 bitów informacji o formacie (poziom M, maska 0–7), już z maską 0x5412. */
export function bityFormatu(maska: number): number {
  return bch((POZIOM_M << 3) | maska, WIELOMIAN_FORMATU, 5) ^ MASKA_FORMATU
}

/** 18 bitów informacji o wersji (od wersji 7). */
export function bityWersji(wersja: number): number {
  return bch(wersja, WIELOMIAN_WERSJI, 6)
}

// ------------------------------------------------------------------ dane
function pojemnosc(wersja: number): number {
  return BLOKI_M[wersja - 1][1].reduce((s, [n, k]) => s + n * k, 0)
}

function bitowLicznika(wersja: number): number {
  return wersja < 10 ? 8 : 16
}

/** Najmniejsza wersja mieszcząca `bajtow` bajtów albo null. */
export function dobierzWersje(bajtow: number): number | null {
  for (let v = 1; v <= WERSJA_MAKS; v++) {
    if (4 + bitowLicznika(v) + bajtow * 8 <= pojemnosc(v) * 8) return v
  }
  return null
}

/** Kody danych (z trybem, licznikiem, terminatorem i wypełnieniem). */
export function kodyDanych(bajty: Uint8Array, wersja: number): number[] {
  const bity: number[] = []
  const dopisz = (w: number, n: number) => {
    for (let i = n - 1; i >= 0; i--) bity.push((w >> i) & 1)
  }
  const pojemnoscBitow = pojemnosc(wersja) * 8
  dopisz(TRYB_BAJTOWY, 4)
  dopisz(bajty.length, bitowLicznika(wersja))
  for (const b of bajty) dopisz(b, 8)
  dopisz(0, Math.min(4, pojemnoscBitow - bity.length))
  while (bity.length % 8) bity.push(0)
  const kody: number[] = []
  for (let i = 0; i < bity.length; i += 8) kody.push(bity.slice(i, i + 8).reduce((a, b) => (a << 1) | b, 0))
  for (let i = 0; kody.length < pojemnosc(wersja); i++) kody.push(WYPELNIENIE[i % 2])
  return kody
}

/** Przeplot bloków danych i korekcji (kolejność zapisu w symbolu). */
export function przeplot(kody: number[], wersja: number): number[] {
  const [ileKorekcji, grupy] = BLOKI_M[wersja - 1]
  const dane: number[][] = []
  let poz = 0
  for (const [n, k] of grupy) {
    for (let i = 0; i < n; i++) {
      dane.push(kody.slice(poz, poz + k))
      poz += k
    }
  }
  const kor = dane.map((b) => korekcja(b, ileKorekcji))
  const wynik: number[] = []
  const maks = Math.max(...dane.map((b) => b.length))
  for (let i = 0; i < maks; i++) for (const b of dane) if (i < b.length) wynik.push(b[i])
  for (let i = 0; i < ileKorekcji; i++) for (const b of kor) wynik.push(b[i])
  return wynik
}

// ------------------------------------------------------------------ macierz
type Macierz = boolean[][]

interface Symbol {
  rozmiar: number
  moduly: Macierz
  funkcyjne: Macierz
}

function nowySymbol(wersja: number): Symbol {
  const rozmiar = 17 + 4 * wersja
  const pusta = () => Array.from({ length: rozmiar }, () => new Array<boolean>(rozmiar).fill(false))
  return { rozmiar, moduly: pusta(), funkcyjne: pusta() }
}

function ustaw(s: Symbol, x: number, y: number, ciemny: boolean): void {
  s.moduly[y][x] = ciemny
  s.funkcyjne[y][x] = true
}

function wzorPozycji(s: Symbol, cx: number, cy: number): void {
  for (let dy = -4; dy <= 4; dy++) {
    for (let dx = -4; dx <= 4; dx++) {
      const x = cx + dx
      const y = cy + dy
      if (x < 0 || y < 0 || x >= s.rozmiar || y >= s.rozmiar) continue
      const d = Math.max(Math.abs(dx), Math.abs(dy))
      ustaw(s, x, y, d !== 2 && d !== 4)
    }
  }
}

function wzorWyrownania(s: Symbol, cx: number, cy: number): void {
  for (let dy = -2; dy <= 2; dy++)
    for (let dx = -2; dx <= 2; dx++) ustaw(s, cx + dx, cy + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1)
}

function wzoryStale(s: Symbol, wersja: number): void {
  const n = s.rozmiar
  for (let i = 0; i < n; i++) {
    ustaw(s, 6, i, i % 2 === 0)
    ustaw(s, i, 6, i % 2 === 0)
  }
  wzorPozycji(s, 3, 3)
  wzorPozycji(s, n - 4, 3)
  wzorPozycji(s, 3, n - 4)
  const poz = WYROWNANIE[wersja - 1]
  for (const y of poz) {
    for (const x of poz) {
      const naPozycji = (x === 6 && y === 6) || (x === 6 && y === poz.at(-1)) || (y === 6 && x === poz.at(-1))
      if (!naPozycji) wzorWyrownania(s, x, y)
    }
  }
  wpiszFormat(s, 0) // rezerwacja miejsc (nadpisane po wyborze maski)
  if (wersja >= WERSJA_Z_INFO) {
    const b = bityWersji(wersja)
    for (let i = 0; i < 18; i++) {
      const bit = ((b >> i) & 1) === 1
      const a = Math.floor(i / 3)
      const c = n - 11 + (i % 3)
      ustaw(s, a, c, bit)
      ustaw(s, c, a, bit)
    }
  }
}

function wpiszFormat(s: Symbol, maska: number): void {
  const b = bityFormatu(maska)
  const n = s.rozmiar
  const bit = (i: number) => ((b >> i) & 1) === 1
  for (let i = 0; i <= 5; i++) ustaw(s, 8, i, bit(i))
  ustaw(s, 8, 7, bit(6))
  ustaw(s, 8, 8, bit(7))
  ustaw(s, 7, 8, bit(8))
  for (let i = 9; i < 15; i++) ustaw(s, 14 - i, 8, bit(i))
  for (let i = 0; i < 8; i++) ustaw(s, n - 1 - i, 8, bit(i))
  for (let i = 8; i < 15; i++) ustaw(s, 8, n - 15 + i, bit(i))
  ustaw(s, 8, n - 8, true) // ciemny moduł
}

function wpiszDane(s: Symbol, kody: number[]): void {
  const n = s.rozmiar
  let i = 0
  const wszystkieBity = kody.length * 8
  for (let prawa = n - 1; prawa >= 1; prawa -= 2) {
    if (prawa === 6) prawa = 5
    for (let k = 0; k < n; k++) {
      for (let j = 0; j < 2; j++) {
        const x = prawa - j
        const wGore = ((prawa + 1) & 2) === 0
        const y = wGore ? n - 1 - k : k
        if (s.funkcyjne[y][x]) continue
        if (i < wszystkieBity) s.moduly[y][x] = ((kody[i >> 3] >> (7 - (i & 7))) & 1) === 1
        i++
      }
    }
  }
}

const MASKI: ((x: number, y: number) => boolean)[] = [
  (x, y) => (x + y) % 2 === 0,
  (_x, y) => y % 2 === 0,
  (x) => x % 3 === 0,
  (x, y) => (x + y) % 3 === 0,
  (x, y) => (Math.floor(x / 3) + Math.floor(y / 2)) % 2 === 0,
  (x, y) => ((x * y) % 2) + ((x * y) % 3) === 0,
  (x, y) => (((x * y) % 2) + ((x * y) % 3)) % 2 === 0,
  (x, y) => (((x + y) % 2) + ((x * y) % 3)) % 2 === 0,
]

function zamaskuj(s: Symbol, maska: number): void {
  const f = MASKI[maska]
  for (let y = 0; y < s.rozmiar; y++)
    for (let x = 0; x < s.rozmiar; x++) if (!s.funkcyjne[y][x] && f(x, y)) s.moduly[y][x] = !s.moduly[y][x]
}

/** Kara za wzór (reguły N1–N4 normy); im mniej, tym lepsza maska. */
export function kara(m: Macierz): number {
  const n = m.length
  let wynik = 0
  const linia = (pobierz: (i: number) => boolean) => {
    let w = 0
    let ciag = 1
    for (let i = 1; i <= n; i++) {
      if (i < n && pobierz(i) === pobierz(i - 1)) ciag++
      else {
        if (ciag >= 5) w += 3 + ciag - 5
        ciag = 1
      }
    }
    for (let i = 0; i + 11 <= n; i++) {
      const wz = Array.from({ length: 11 }, (_, k) => pobierz(i + k))
      const a = [true, false, true, true, true, false, true, false, false, false, false]
      if (a.every((v, k) => wz[k] === v) || a.every((v, k) => wz[10 - k] === v)) w += 40
    }
    return w
  }
  for (let y = 0; y < n; y++) wynik += linia((i) => m[y][i])
  for (let x = 0; x < n; x++) wynik += linia((i) => m[i][x])
  for (let y = 0; y < n - 1; y++)
    for (let x = 0; x < n - 1; x++) {
      const c = m[y][x]
      if (c === m[y][x + 1] && c === m[y + 1][x] && c === m[y + 1][x + 1]) wynik += 3
    }
  const ciemne = m.reduce((s, w) => s + w.filter(Boolean).length, 0)
  wynik += Math.floor(Math.abs((ciemne * 20) / (n * n) - 10)) * 10
  return wynik
}

/** Symbol QR dla tekstu (UTF-8). `maska` wymusza maskę (testy), inaczej najmniejsza kara. Null = za długi tekst. */
export function zakoduj(tekst: string, maska?: number): { wersja: number; maska: number; moduly: Macierz } | null {
  const bajty = new TextEncoder().encode(tekst)
  const wersja = dobierzWersje(bajty.length)
  if (wersja === null) return null
  const kody = przeplot(kodyDanych(bajty, wersja), wersja)
  let najlepszy: { maska: number; moduly: Macierz; kara: number } | null = null
  for (let m = 0; m < MASKI.length; m++) {
    if (maska !== undefined && m !== maska) continue
    const s = nowySymbol(wersja)
    wzoryStale(s, wersja)
    wpiszDane(s, kody)
    zamaskuj(s, m)
    wpiszFormat(s, m)
    const k = kara(s.moduly)
    if (!najlepszy || k < najlepszy.kara) najlepszy = { maska: m, moduly: s.moduly, kara: k }
  }
  return najlepszy && { wersja, maska: najlepszy.maska, moduly: najlepszy.moduly }
}

/** Odczyt kodów danych z symbolu (do testu w obie strony): zdejmuje maskę i czyta w kolejności zapisu. */
export function odczytajKody(moduly: Macierz, wersja: number, maska: number): number[] {
  const s = nowySymbol(wersja)
  wzoryStale(s, wersja)
  const n = s.rozmiar
  const f = MASKI[maska]
  const bity: number[] = []
  for (let prawa = n - 1; prawa >= 1; prawa -= 2) {
    if (prawa === 6) prawa = 5
    for (let k = 0; k < n; k++) {
      for (let j = 0; j < 2; j++) {
        const x = prawa - j
        const y = ((prawa + 1) & 2) === 0 ? n - 1 - k : k
        if (s.funkcyjne[y][x]) continue
        bity.push(moduly[y][x] !== f(x, y) ? 1 : 0)
      }
    }
  }
  const kody: number[] = []
  for (let i = 0; i + 8 <= bity.length; i += 8) kody.push(bity.slice(i, i + 8).reduce((a, b) => (a << 1) | b, 0))
  return kody
}
