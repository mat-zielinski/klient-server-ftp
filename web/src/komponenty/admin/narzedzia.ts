// Czyste funkcje stron administracyjnych: sprawdzanie haseł i portów, wykrywanie zmian wymagających restartu,
// źródło certyfikatu, adres linku publicznego, kopiowanie do schowka.

import type { Konfiguracja, Link } from '../../api/typy'
import { MIN_DLUGOSC_HASLA, PORT_MAX, PORT_MIN, PRZEDROSTEK_LINKU } from './stale'
import type { ZrodloCertyfikatu } from './typy'

/** Błąd hasła (pusty napis = hasło dobre). `powtorzone` pomijane, gdy undefined. */
export function bladHasla(haslo: string, powtorzone?: string): string {
  if (haslo.length < MIN_DLUGOSC_HASLA) return `Hasło musi mieć co najmniej ${MIN_DLUGOSC_HASLA} znaków.`
  if (powtorzone !== undefined && haslo !== powtorzone) return 'Hasła nie są takie same.'
  return ''
}

/** Błąd portu (pusty napis = port dobry). */
export function bladPortu(port: number): string {
  if (!Number.isInteger(port) || port < PORT_MIN || port > PORT_MAX) return `Port od ${PORT_MIN} do ${PORT_MAX}.`
  return ''
}

/** Pola konfiguracji, których zmiana działa dopiero po restarcie serwera. */
const POLA_RESTARTU: (keyof Konfiguracja)[] = [
  'tryb_sieci',
  'adres_nasluchu',
  'port_http',
  'port_https',
  'port_ftps',
  'ftps_wlaczone',
  'certyfikat',
  'klucz',
  'domena_lets_encrypt',
]

/** Czy pole zmienione względem zapisanej konfiguracji wymaga restartu. */
export function poleWymagaRestartu(zapisana: Konfiguracja, biezaca: Konfiguracja, pole: keyof Konfiguracja): boolean {
  if (pole === 'porty_pasywne_ftp') {
    return zapisana.porty_pasywne_ftp[0] !== biezaca.porty_pasywne_ftp[0] ||
      zapisana.porty_pasywne_ftp[1] !== biezaca.porty_pasywne_ftp[1]
  }
  return POLA_RESTARTU.includes(pole) && zapisana[pole] !== biezaca[pole]
}

/** Czy jakakolwiek zmiana wymaga restartu. */
export function wymagaRestartu(zapisana: Konfiguracja, biezaca: Konfiguracja): boolean {
  return [...POLA_RESTARTU, 'porty_pasywne_ftp' as const].some((p) => poleWymagaRestartu(zapisana, biezaca, p))
}

/** Źródło certyfikatu z pól konfiguracji: domena = Let's Encrypt, ścieżka pliku = z pliku, nic = własny. */
export function zrodloCertyfikatu(k: Konfiguracja): ZrodloCertyfikatu {
  if (k.domena_lets_encrypt.trim()) return 'letsencrypt'
  if (k.certyfikat.trim()) return 'plik'
  return 'wlasny'
}

/** Pełny adres linku publicznego (serwer zwykle podaje `adres`, inaczej składamy z bieżącego hosta). */
export function adresLinku(link: Link): string {
  if (link.adres) return link.adres
  return `${location.origin}${PRZEDROSTEK_LINKU}${encodeURIComponent(link.id)}`
}

/** Czy data jest ustawiona (pusta albo zerowa data Go „0001-01-01” = brak). */
export function maDate(iso: string | undefined): iso is string {
  if (!iso) return false
  const t = Date.parse(iso)
  return Number.isFinite(t) && t > 0
}

/** Czy link wygasł (data minęła). Brak daty = bez wygasania. */
export function wygasl(link: Pick<Link, 'wygasa'>, teraz: number = Date.now()): boolean {
  return maDate(link.wygasa) && Date.parse(link.wygasa) <= teraz
}

/** Czy link wyczerpał limit pobrań (0 = bez limitu). */
export function wyczerpany(link: Pick<Link, 'limit_pobran' | 'pobrano'>): boolean {
  return link.limit_pobran > 0 && link.pobrano >= link.limit_pobran
}

/** Kopiuje tekst do schowka; zwraca false, gdy przeglądarka nie pozwoliła. */
export async function kopiuj(tekst: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(tekst)
      return true
    }
  } catch {
    // celowo: przechodzimy do zapasowej metody poniżej
  }
  const pole = document.createElement('textarea')
  pole.value = tekst
  pole.setAttribute('readonly', '')
  pole.className = 'tylko-czytnik'
  document.body.appendChild(pole)
  pole.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false // celowo: stara przeglądarka bez execCommand
  }
  pole.remove()
  return ok
}

/** Głęboka kopia konfiguracji (porównanie z zapisaną wersją). */
export function kopiaKonfiguracji(k: Konfiguracja): Konfiguracja {
  return {
    ...k,
    porty_pasywne_ftp: [k.porty_pasywne_ftp[0], k.porty_pasywne_ftp[1]],
    udzialy: k.udzialy.map((u) => ({ ...u })),
  }
}
