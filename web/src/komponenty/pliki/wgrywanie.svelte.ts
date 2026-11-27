// Kolejka wgrywania: kawałki po 8 MB przez PUT /api/wgraj, wznawianie z GET /api/wgraj, kilka plików naraz.
import { komunikatBledu, polacz, zapytanie } from './narzedzia'
import type { PlikDoWgrania } from './upuszczanie'
import {
  KAWALEK_WGRYWANIA,
  PONOWIENIA_KAWALKA,
  PRZERWA_PONOWIENIA_MS,
  ROWNOLEGLE_WGRYWANIA,
  ZNIKANIE_ZAKONCZONYCH_MS,
} from './stale'

export type StanWgrywania = 'czeka' | 'trwa' | 'gotowe' | 'blad' | 'istnieje' | 'anulowane'

export interface ZadanieWgrywania {
  id: number
  plik: File
  udzial: string
  sciezka: string
  przeslano: number
  stan: StanWgrywania
  komunikat: string
  nadpisz: boolean
  predkosc: number
}

class BladWgrywania extends Error {
  constructor(public kod: number, komunikat: string) {
    super(komunikat)
  }
}

let licznik = 0

/** Wysyła jeden kawałek przez XHR (postęp wewnątrz kawałka) i zwraca łączną liczbę przesłanych bajtów. */
function wyslijKawalek(
  z: ZadanieWgrywania,
  od: number,
  naPostep: (bajty: number) => void,
  przerwij: (x: XMLHttpRequest) => void,
): Promise<number> {
  const doKonca = Math.min(od + KAWALEK_WGRYWANIA, z.plik.size)
  const koniec = doKonca >= z.plik.size
  const q = zapytanie({ udzial: z.udzial, sciezka: z.sciezka, przesuniecie: od, koniec, nadpisz: z.nadpisz })
  return new Promise((ok, zle) => {
    const x = new XMLHttpRequest()
    przerwij(x)
    x.open('PUT', '/api/wgraj?' + q)
    x.setRequestHeader('X-Skrytka', '1')
    x.setRequestHeader('Content-Type', 'application/octet-stream')
    // pełny rozmiar pliku: serwer pokazuje wgranie na żywo z postępem
    x.setRequestHeader('X-Rozmiar-Calkowity', String(z.plik.size))
    x.upload.onprogress = (e) => naPostep(od + e.loaded)
    x.onload = () => {
      if (x.status >= 200 && x.status < 300) {
        try {
          ok((JSON.parse(x.responseText) as { przeslano: number }).przeslano)
        } catch {
          ok(doKonca)
        }
      } else zle(new BladWgrywania(x.status, komunikatBledu(x.responseText, x.status)))
    }
    x.onerror = () => zle(new BladWgrywania(0, komunikatBledu('', 0)))
    x.onabort = () => zle(new BladWgrywania(-1, 'Anulowano.'))
    x.send(z.plik.slice(od, doKonca))
  })
}

async function juzPrzeslano(z: ZadanieWgrywania): Promise<number> {
  if (z.nadpisz) return 0
  const r = await fetch('/api/wgraj?' + zapytanie({ udzial: z.udzial, sciezka: z.sciezka }), {
    headers: { 'X-Skrytka': '1' },
  })
  if (!r.ok) return 0
  try {
    const n = ((await r.json()) as { przeslano: number }).przeslano
    return n > 0 && n < z.plik.size ? n : 0
  } catch {
    return 0
  }
}

const czekaj = (ms: number) => new Promise((r) => setTimeout(r, ms))

export class KolejkaWgrywania {
  zadania = $state<ZadanieWgrywania[]>([])
  /** Wołane po każdym udanym wgraniu (np. odświeżenie listy). */
  naGotowe: (z: ZadanieWgrywania) => void = () => {}
  private xhr = new Map<number, XMLHttpRequest>()

  get aktywne(): number {
    return this.zadania.filter((z) => z.stan === 'trwa' || z.stan === 'czeka').length
  }

  dodaj(pliki: PlikDoWgrania[], udzial: string, folder: string): void {
    for (const { plik, wzgledna } of pliki) {
      this.zadania.push({
        id: ++licznik, plik, udzial, sciezka: polacz(folder, wzgledna), przeslano: 0,
        stan: 'czeka', komunikat: '', nadpisz: false, predkosc: 0,
      })
    }
    this.uruchom()
  }

  nadpisz(id: number): void {
    const z = this.zadania.find((x) => x.id === id)
    if (!z) return
    z.nadpisz = true
    z.przeslano = 0
    z.stan = 'czeka'
    z.komunikat = ''
    this.uruchom()
  }

  ponow(id: number): void {
    const z = this.zadania.find((x) => x.id === id)
    if (!z) return
    z.stan = 'czeka'
    z.komunikat = ''
    this.uruchom()
  }

  anuluj(id: number): void {
    const z = this.zadania.find((x) => x.id === id)
    if (!z) return
    this.xhr.get(id)?.abort()
    z.stan = 'anulowane'
    this.usunPozniej(id)
  }

  usun(id: number): void {
    this.zadania = this.zadania.filter((z) => z.id !== id)
  }

  wyczyscZakonczone(): void {
    this.zadania = this.zadania.filter((z) => z.stan === 'trwa' || z.stan === 'czeka')
  }

  private usunPozniej(id: number): void {
    setTimeout(() => this.usun(id), ZNIKANIE_ZAKONCZONYCH_MS)
  }

  private uruchom(): void {
    let trwa = this.zadania.filter((z) => z.stan === 'trwa').length
    for (const z of this.zadania) {
      if (trwa >= ROWNOLEGLE_WGRYWANIA) break
      if (z.stan !== 'czeka') continue
      z.stan = 'trwa'
      trwa++
      void this.wgraj(z).finally(() => this.uruchom())
    }
  }

  private async wgraj(zadanie: ZadanieWgrywania): Promise<void> {
    // pracujemy na obiekcie z $state, żeby zmiany były widoczne
    const z = this.zadania.find((x) => x.id === zadanie.id) ?? zadanie
    try {
      z.przeslano = await juzPrzeslano(z)
      let start = performance.now()
      let bajtyStart = z.przeslano
      do {
        let proby = 0
        for (;;) {
          try {
            z.przeslano = await wyslijKawalek(z, z.przeslano, (b) => {
              z.przeslano = b
              const dt = (performance.now() - start) / 1000
              if (dt > 1) {
                z.predkosc = (b - bajtyStart) / dt
                start = performance.now()
                bajtyStart = b
              }
            }, (x) => this.xhr.set(z.id, x))
            break
          } catch (e) {
            const kod = e instanceof BladWgrywania ? e.kod : 0
            if (kod !== 0 || ++proby > PONOWIENIA_KAWALKA) throw e
            await czekaj(PRZERWA_PONOWIENIA_MS)
            z.przeslano = await juzPrzeslano(z)
          }
        }
      } while (z.przeslano < z.plik.size)
      z.stan = 'gotowe'
      this.naGotowe(z)
      this.usunPozniej(z.id)
    } catch (e) {
      if (z.stan === 'anulowane') return
      const kod = e instanceof BladWgrywania ? e.kod : 0
      z.stan = kod === 409 ? 'istnieje' : 'blad'
      z.komunikat = kod === 409 ? 'Plik już istnieje.' : (e as Error).message
    } finally {
      this.xhr.delete(z.id)
    }
  }
}
