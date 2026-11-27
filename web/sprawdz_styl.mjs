// Strażnik stylu interfejsu: kolory i rozmiary tylko w src/styl/tokeny.css.
// Błąd, gdy w innym pliku src/ lub public/ (.svelte, .ts, .css, .js, .svg, .webmanifest) jest kolor #hex, rgb(…), hsl(…) albo liczba z px / rem
// (poza zerem). Komentarze też są sprawdzane (żeby nie przemycać wartości). Uruchomienie: npm run straznik.
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const KORZEN = fileURLToPath(new URL('.', import.meta.url))
const KATALOGI = ['src', 'public']
// wyjątki: tokeny (jedyne źródło) i manifest PWA (format JSON, przeglądarka nie zna tam zmiennych CSS)
const WYJATKI = new Set([join('src', 'styl', 'tokeny.css'), join('public', 'manifest.webmanifest')])
const ROZSZERZENIA = /\.(svelte|ts|css|js|mjs|html|svg|webmanifest)$/

const REGULY = [
  { nazwa: 'kolor #hex', wzor: /#[0-9a-fA-F]{3,8}(?![0-9a-zA-Z_-])/g },
  { nazwa: 'kolor rgb()/rgba()', wzor: /\brgba?\s*\(/gi },
  { nazwa: 'kolor hsl()/hsla()', wzor: /\bhsla?\s*\(/gi },
  { nazwa: 'rozmiar px/rem', wzor: /(?<![\w.-])(\d*\.?\d+)(px|rem)\b/g, zero: true },
]

function pliki(katalog) {
  const wynik = []
  for (const nazwa of readdirSync(katalog)) {
    const sciezka = join(katalog, nazwa)
    if (statSync(sciezka).isDirectory()) wynik.push(...pliki(sciezka))
    else if (ROZSZERZENIA.test(nazwa)) wynik.push(sciezka)
  }
  return wynik
}

const bledy = []
const wszystkie = KATALOGI.filter((k) => existsSync(join(KORZEN, k))).flatMap((k) => pliki(join(KORZEN, k)))
for (const plik of wszystkie) {
  const wzgledna = relative(KORZEN, plik)
  if (WYJATKI.has(wzgledna)) continue
  const linie = readFileSync(plik, 'utf8').split(/\r?\n/)
  linie.forEach((linia, i) => {
    for (const r of REGULY) {
      for (const m of linia.matchAll(r.wzor)) {
        if (r.zero && parseFloat(m[1]) === 0) continue
        bledy.push(`${wzgledna}:${i + 1}: ${r.nazwa}: „${m[0]}” (użyj zmiennej z styl/tokeny.css)`)
      }
    }
  })
}

if (bledy.length) {
  console.error(`Strażnik stylu: ${bledy.length} naruszeń.`)
  for (const b of bledy) console.error('  ' + b)
  process.exit(1)
}
console.log('Strażnik stylu: OK (kolory i rozmiary tylko w styl/tokeny.css).')
