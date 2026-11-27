// Typy stron administracyjnych, których nie ma w kontrakcie src/api/typy.ts (odpowiedzi opisane w docs/api.md).

import type { Adres, Konfiguracja, Uzytkownik, Wpis } from '../../api/typy'

export type TrybSieci = Konfiguracja['tryb_sieci']

/** Odpowiedź GET /api/siec. */
export interface InfoSieci {
  adresy: Adres[]
  publiczny: string
  tryb: string
  porty: { http: number; https: number; ftps: number }
  porty_otwarte: Record<string, boolean>
  odcisk_certyfikatu: string
}

/** Ciało POST /api/kreator. */
export interface DaneKreatora {
  login: string
  haslo: string
  udzial: { nazwa: string; sciezka: string; tylko_odczyt: boolean }
  tryb_sieci: TrybSieci
}

/** Ciało POST /api/konta (konto z hasłem); PUT bez hasła. */
export type KontoZHaslem = Uzytkownik & { haslo?: string }

/** Odpowiedź PUT /api/ustawienia (pola opcjonalne: serwer może zwrócić `{}`). */
export interface WynikZapisu {
  restart?: boolean
  komunikat?: string
}

/** Skąd serwer bierze certyfikat HTTPS (wyliczane z pól Konfiguracja). */
export type ZrodloCertyfikatu = 'wlasny' | 'plik' | 'letsencrypt'

/** Odpowiedź GET /api/publiczny/<id>: link do pliku albo folderu. */
export interface InfoPubliczne {
  nazwa: string
  folder: boolean
  rozmiar: number
  wygasa?: string
  limit_pobran?: number
  pobrano?: number
  /** Tylko dla folderu: zawartość (ścieżki względem udostępnionego folderu). */
  wpisy?: Wpis[]
}
