// KONTRAKT typów API (lustro docs/api.md i struktur Go). Zmiana = uzgodnienie z sesją główną.

export type Prawo = 'brak' | 'odczyt' | 'zapis'
export type Rodzaj = 'folder' | 'obraz' | 'wideo' | 'dzwiek' | 'pdf' | 'tekst' | 'archiwum' | 'inny'

export interface Wpis { nazwa: string; sciezka: string; folder: boolean; rozmiar: number; zmieniono: string; rodzaj: Rodzaj }
export interface Uzytkownik { login: string; admin: boolean; prawa: Record<string, Prawo>; wylaczony: boolean; ftps: boolean }
export interface StanAplikacji { wersja: string; nazwa: string; kreator: boolean; zalogowany: boolean; uzytkownik: Uzytkownik | null; motyw: 'ciemny' | 'jasny' }
export interface UdzialInfo { nazwa: string; tylko_odczyt: boolean; prawo: Prawo }

export interface OpcjaProtokolu {
  id: 'https' | 'ftps' | 'link' | 'zip' | 'publiczny'
  nazwa: string; opis: string; adres?: string; port?: number; dostepne: boolean; powod?: string
}
export interface Adres { interfejs: string; ip: string; rodzaj: 'lan' | 'wireguard' | 'publiczny' | 'petla' }
export interface Protokoly { opcje: OpcjaProtokolu[]; adresy: Adres[]; odcisk_certyfikatu: string }

export interface StanTransferu {
  id: string; kto: string; ip: string; protokol: 'https' | 'http' | 'ftps'; kierunek: 'pobranie' | 'wgranie'
  udzial: string; sciezka: string; rozmiar: number; przeslano: number; predkosc_bps: number
  start: string; pozostalo_s: number; historia: number[]
}
export interface ZdarzenieTransferu { typ: 'start' | 'postep' | 'koniec'; stan: StanTransferu; wynik?: 'ok' | 'przerwane' | 'blad' }
export interface SumaTransferow { predkosc_bps: number; aktywne: number }

export interface WpisDziennika {
  czas: string; rodzaj: string; kto: string; ip: string; protokol: string; udzial: string
  sciezka: string; bajty: number; czas_ms: number; wynik: string; opis: string
}
export interface PunktWykresu { czas: string; bajty: number; pobrania: number; srednia_bps: number }
export interface PozycjaRankingu { nazwa: string; bajty: number; liczba: number }
export interface Statystyki {
  wykres: PunktWykresu[]; najczestsze_pliki: PozycjaRankingu[]; najaktywniejsi_klienci: PozycjaRankingu[]
  protokoly: Record<string, number>; suma_bajtow: number; suma_pobran: number
}

export interface Link {
  id: string; udzial: string; sciezka: string; autor: string; utworzono: string; wygasa: string
  ma_haslo: boolean; limit_pobran: number; pobrano: number; adres?: string
}
export interface Udzial { nazwa: string; sciezka: string; tylko_odczyt: boolean }
export interface Konfiguracja {
  tryb_sieci: 'vpn' | 'publiczny'; adres_nasluchu: string; port_http: number; port_https: number; port_ftps: number
  porty_pasywne_ftp: [number, number]; ftps_wlaczone: boolean; udzialy: Udzial[]; certyfikat: string; klucz: string
  domena_lets_encrypt: string; motyw: string
}
export interface Pulpit {
  aktywne: number; predkosc_bps: number; dzis: { bajty: number; pobrania: number }
  udzialy: { nazwa: string; wolne: number; razem: number }[]; czas_pracy_s: number
}
