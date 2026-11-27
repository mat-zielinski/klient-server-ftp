// Formatowanie po polsku: rozmiary (KB/MB/GB, podstawa 1024), prędkości (MB/s), czasy, daty.

const JEDNOSTKI = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
const KROK = 1024
const STREFA = 'pl-PL'

function liczba(n: number, miejsca: number): string {
  return n.toLocaleString(STREFA, { minimumFractionDigits: 0, maximumFractionDigits: miejsca })
}

/** Rozmiar w bajtach: „0 B”, „512 B”, „1,5 KB”, „12,3 MB”, „1,07 GB”. */
export function rozmiar(b: number): string {
  if (!Number.isFinite(b) || b < 0) return '—'
  let i = 0
  let w = b
  while (w >= KROK && i < JEDNOSTKI.length - 1) {
    w /= KROK
    i++
  }
  const miejsca = i === 0 ? 0 : w < 10 ? 2 : w < 100 ? 1 : 0
  return `${liczba(w, miejsca)} ${JEDNOSTKI[i]}`
}

/** Prędkość w bajtach na sekundę: „850 KB/s”, „11,2 MB/s”. */
export function predkosc(bps: number): string {
  if (!Number.isFinite(bps) || bps <= 0) return '0 B/s'
  return rozmiar(bps) + '/s'
}

/** Czas trwania w sekundach: „45 s”, „3 min 20 s”, „2 godz. 5 min”, „3 dni 4 godz.”. */
export function czasTrwania(s: number): string {
  if (!Number.isFinite(s) || s < 0) return '—'
  const c = Math.round(s)
  if (c < 60) return `${c} s`
  const min = Math.floor(c / 60)
  if (min < 60) {
    const sek = c % 60
    return sek ? `${min} min ${sek} s` : `${min} min`
  }
  const godz = Math.floor(min / 60)
  if (godz < 24) {
    const m = min % 60
    return m ? `${godz} godz. ${m} min` : `${godz} godz.`
  }
  const dni = Math.floor(godz / 24)
  const g = godz % 24
  const dniTekst = `${dni} ${dni === 1 ? 'dzień' : 'dni'}`
  return g ? `${dniTekst} ${g} godz.` : dniTekst
}

function wczytajDate(iso: string): Date | null {
  if (!iso) return null
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) || d.getFullYear() < 1971 ? null : d
}

/** Data i godzina: „27.11.2026, 14:05”. Pusta lub zerowa data = „—”. */
export function data(iso: string): string {
  const d = wczytajDate(iso)
  if (!d) return '—'
  return d.toLocaleString(STREFA, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Data względna: „przed chwilą”, „2 min temu”, „3 godz. temu”, „wczoraj”, „5 dni temu”, „za 2 dni”;
 *  starsze niż 30 dni = zwykła data. `teraz` do testów. */
export function dataWzgledna(iso: string, teraz: number = Date.now()): string {
  const d = wczytajDate(iso)
  if (!d) return '—'
  const roznica = Math.round((teraz - d.getTime()) / 1000)
  const przyszlosc = roznica < 0
  const s = Math.abs(roznica)
  const opis = (t: string) => (przyszlosc ? `za ${t}` : `${t} temu`)
  if (s < 45) return przyszlosc ? 'za chwilę' : 'przed chwilą'
  if (s < 3600) return opis(`${Math.max(1, Math.round(s / 60))} min`)
  if (s < 86400) return opis(`${Math.round(s / 3600)} godz.`)
  const dni = Math.round(s / 86400)
  if (dni === 1) return przyszlosc ? 'jutro' : 'wczoraj'
  if (dni <= 30) return opis(`${dni} dni`)
  return data(iso)
}

/** Procent 0..1 → „42%”. */
export function procent(u: number): string {
  if (!Number.isFinite(u)) return '—'
  return `${Math.round(Math.min(1, Math.max(0, u)) * 100)}%`
}

/** Liczba z odstępami tysięcy: „12 345”. */
export function liczbaCala(n: number): string {
  return Number.isFinite(n) ? Math.round(n).toLocaleString(STREFA) : '—'
}
