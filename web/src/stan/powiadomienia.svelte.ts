// Powiadomienia (toasty): powiadom(tekst, rodzaj) dokłada wpis, który znika sam po czasie z tokenu --czas-toast.
// Wyświetla je komponenty/wspolne/Powiadomienia.svelte (osadzony w App.svelte).

export type RodzajPowiadomienia = 'ok' | 'info' | 'blad' | 'ostrz'

export interface Powiadomienie {
  id: number
  tekst: string
  rodzaj: RodzajPowiadomienia
}

const CZAS_DOMYSLNY_MS = 4500
const CZAS_BLEDU_MNOZNIK = 2
const MAKS_NARAZ = 5

class Powiadomienia {
  lista = $state<Powiadomienie[]>([])
  private nastepny = 1

  dodaj(tekst: string, rodzaj: RodzajPowiadomienia): number {
    const id = this.nastepny++
    this.lista = [...this.lista, { id, tekst, rodzaj }].slice(-MAKS_NARAZ)
    const czas = czasZTokenu() * (rodzaj === 'blad' ? CZAS_BLEDU_MNOZNIK : 1)
    setTimeout(() => this.zamknij(id), czas)
    return id
  }

  zamknij(id: number): void {
    this.lista = this.lista.filter((p) => p.id !== id)
  }
}

function czasZTokenu(): number {
  if (typeof document === 'undefined') return CZAS_DOMYSLNY_MS
  const w = getComputedStyle(document.documentElement).getPropertyValue('--czas-toast').trim()
  const n = parseFloat(w)
  if (!Number.isFinite(n)) return CZAS_DOMYSLNY_MS
  return w.endsWith('ms') ? n : n * 1000
}

export const powiadomienia = new Powiadomienia()

export function powiadom(tekst: string, rodzaj: RodzajPowiadomienia = 'info'): number {
  return powiadomienia.dodaj(tekst, rodzaj)
}
