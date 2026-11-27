// Nawigacja po hashu: #/<strona>[/<segmenty>][?klucz=wartosc].
// Strona Pliki: #/pliki/<udzial>/<sciezka…> → parametry { udzial, sciezka: '/a/b' }.
// Ścieżka /s/<id> (link publiczny) → nawigacja.publiczny = id (bez hasha).

export type Parametry = Record<string, string>

export const STRONA_DOMYSLNA = 'pliki'
const STRONA_PLIKOW = 'pliki'
const PRZEDROSTEK_PUBLICZNY = '/s/'

export interface Polozenie {
  strona: string
  parametry: Parametry
}

/** Czyta hash (`#/pliki/Filmy/a/b`) na stronę i parametry. Czysta funkcja (testowalna). */
export function czytajHash(hash: string): Polozenie {
  const bez = hash.replace(/^#\/?/, '')
  const [sciezka, zap = ''] = bez.split('?', 2)
  const segmenty = sciezka.split('/').filter(Boolean).map(dekoduj)
  const strona = segmenty[0] || STRONA_DOMYSLNA
  const parametry: Parametry = {}
  for (const [k, w] of new URLSearchParams(zap)) parametry[k] = w
  if (strona === STRONA_PLIKOW && segmenty.length > 1) {
    parametry.udzial = segmenty[1]
    parametry.sciezka = '/' + segmenty.slice(2).join('/')
  }
  return { strona, parametry }
}

/** Buduje hash z strony i parametrów (odwrotność czytajHash). */
export function zbudujHash(strona: string, parametry: Parametry = {}): string {
  const reszta = { ...parametry }
  let h = '#/' + encodeURIComponent(strona)
  if (strona === STRONA_PLIKOW && reszta.udzial) {
    h += '/' + encodeURIComponent(reszta.udzial)
    const seg = (reszta.sciezka ?? '').split('/').filter(Boolean).map(encodeURIComponent)
    if (seg.length) h += '/' + seg.join('/')
    delete reszta.udzial
    delete reszta.sciezka
  }
  const zap = new URLSearchParams(reszta).toString()
  return zap ? `${h}?${zap}` : h
}

function dekoduj(s: string): string {
  try {
    return decodeURIComponent(s)
  } catch {
    return s // celowo: zły zapis procentowy zostaje dosłownie
  }
}

function idPubliczne(): string | null {
  if (typeof location === 'undefined') return null
  const p = location.pathname
  if (!p.startsWith(PRZEDROSTEK_PUBLICZNY)) return null
  const id = p.slice(PRZEDROSTEK_PUBLICZNY.length).split('/')[0]
  return id ? dekoduj(id) : null
}

class Nawigacja {
  strona = $state(STRONA_DOMYSLNA)
  parametry = $state<Parametry>({})
  /** Id linku publicznego, gdy adres to /s/<id>; wtedy aplikacja pokazuje stronę Publiczny. */
  readonly publiczny: string | null = idPubliczne()

  constructor() {
    if (typeof window === 'undefined') return
    this.wczytaj()
    window.addEventListener('hashchange', () => this.wczytaj())
  }

  private wczytaj(): void {
    const p = czytajHash(location.hash)
    this.strona = p.strona
    this.parametry = p.parametry
  }

  /** Przejście do strony (zmienia hash; wpis w historii przeglądarki). */
  idz(strona: string, parametry: Parametry = {}): void {
    const h = zbudujHash(strona, parametry)
    if (location.hash === h) {
      this.wczytaj()
      return
    }
    location.hash = h
  }

  /** Jak idz, ale bez nowego wpisu w historii (np. poprawka domyślnej strony). */
  zastap(strona: string, parametry: Parametry = {}): void {
    history.replaceState(history.state, '', zbudujHash(strona, parametry))
    this.wczytaj()
  }
}

export const nawigacja = new Nawigacja()

export function idz(strona: string, parametry?: Parametry): void {
  nawigacja.idz(strona, parametry)
}
