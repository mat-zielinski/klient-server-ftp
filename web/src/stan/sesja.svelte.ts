// Sesja: stan aplikacji z GET /api/stan (wersja, nazwa, kreator, zalogowany użytkownik, motyw).

import { api, trescBledu, ustawNaBrakSesji } from '../api/klient'
import type { StanAplikacji } from '../api/typy'
import { motyw } from './motyw.svelte'

class Sesja {
  stan = $state<StanAplikacji | null>(null)
  /** Pierwsze wczytanie stanu zakończone (z sukcesem albo błędem). */
  wczytano = $state(false)
  /** Komunikat, gdy /api/stan nie odpowiada. */
  blad = $state('')

  get zalogowany(): boolean {
    return !!this.stan?.zalogowany && !!this.stan.uzytkownik
  }

  get admin(): boolean {
    return !!this.stan?.uzytkownik?.admin
  }

  async odswiez(): Promise<void> {
    try {
      this.stan = await api.get<StanAplikacji>('/stan')
      this.blad = ''
      motyw.zSerwera(this.stan.motyw)
    } catch (b) {
      this.blad = trescBledu(b)
    } finally {
      this.wczytano = true
    }
  }

  async wyloguj(): Promise<void> {
    try {
      await api.post('/wylogowanie')
    } catch {
      // celowo: nawet przy błędzie serwera kończymy sesję po stronie interfejsu
    }
    if (this.stan) this.stan = { ...this.stan, zalogowany: false, uzytkownik: null }
    await this.odswiez()
  }
}

export const sesja = new Sesja()

// 401 z dowolnego żądania: sesja wygasła, odśwież stan (aplikacja pokaże Logowanie)
ustawNaBrakSesji(() => {
  if (sesja.stan?.zalogowany) void sesja.odswiez()
})
