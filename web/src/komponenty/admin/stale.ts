// Stałe stron administracyjnych (Logowanie, Kreator, Użytkownicy, Ustawienia, Udostępnienia, Publiczny).
// Wszystkie liczby i napisy powtarzane w kilku miejscach są tutaj (zasada „wszystko globalnie”).

import type { Prawo } from '../../api/typy'

/** Minimalna długość hasła konta (kreator, nowe konto, zmiana hasła). Serwer sprawdza to samo. */
export const MIN_DLUGOSC_HASLA = 10

/** Kod HTTP blokady konta po złych hasłach (ErrKontoZablokowane). */
export const KOD_BLOKADY = 429
/** Kod HTTP złych danych logowania albo brakującego hasła linku. */
export const KOD_BRAK_DOSTEPU = 401
/** Kod HTTP linku, który wygasł albo wyczerpał limit pobrań. */
export const KOD_WYGASLO = 410
/** Kod HTTP linku, którego nie ma. */
export const KOD_NIE_ZNALEZIONO = 404
/** Kod HTTP złego hasła linku (gdy serwer odróżnia je od braku hasła). */
export const KOD_ZAKAZ = 403

/** Domyślne wartości kreatora pierwszego uruchomienia. */
export const KREATOR = {
  login: 'admin',
  nazwaUdzialu: 'Udostepnione',
  sciezkaUdzialu: 'D:\\Udostepnione',
  tylkoOdczyt: true,
  tryb: 'vpn' as const,
}

/** Opcje prawa do udziału (kolejność w wyborze). */
export const OPCJE_PRAWA: { wartosc: Prawo; etykieta: string }[] = [
  { wartosc: 'brak', etykieta: 'Brak' },
  { wartosc: 'odczyt', etykieta: 'Odczyt' },
  { wartosc: 'zapis', etykieta: 'Zapis' },
]

/** Pigułka prawa: zapis = zielona, odczyt = niebieska, brak = szara. */
export const STAN_PRAWA: Record<Prawo, 'ok' | 'info' | 'wyl'> = { zapis: 'ok', odczyt: 'info', brak: 'wyl' }

/** Zakres portu TCP. */
export const PORT_MIN = 1
export const PORT_MAX = 65535

/** Nagłówek z hasłem linku publicznego (docs/api.md, Udostępnienia). */
export const NAGLOWEK_HASLA = 'X-Haslo'
/** Przedrostek adresu strony linku publicznego. */
export const PRZEDROSTEK_LINKU = '/s/'
/** Adres API linku publicznego. */
export const API_PUBLICZNY = '/api/publiczny/'

/** Opisy trybów sieci (Kreator, Ustawienia → Sieć). */
export const TRYBY_SIECI = {
  vpn: {
    nazwa: 'VPN (WireGuard)',
    opis:
      'Klient łączy się tunelem WireGuard z routerem i widzi ten komputer pod adresem LAN. Serwer słucha tylko ' +
      'w sieci lokalnej i w tunelu, z internetu jest niewidoczny.',
    znaczek: 'Zalecany',
  },
  publiczny: {
    nazwa: 'Publiczny (port na routerze)',
    opis:
      'Port przekierowany na routerze na ten komputer, dostęp z dowolnego miejsca. Serwer wymusza HTTPS i FTPS ' +
      'oraz blokuje konto po złych hasłach.',
    ostrzezenie:
      'Serwer będzie widoczny z internetu. Używaj długich haseł i certyfikatu Let\u2019s Encrypt albo własnego.',
  },
} as const

/** Po ilu ms zwolnić tymczasowy adres blob pobrania z hasłem (przeglądarka musi zdążyć zacząć zapis). */
export const ZWOLNIENIE_POBRANIA_MS = 60_000
