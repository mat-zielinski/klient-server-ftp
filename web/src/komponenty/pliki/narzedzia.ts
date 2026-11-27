// Drobne narzędzia strony Pliki: ścieżki, schowek, wyzwalanie pobrania.

/** Łączy ścieżkę folderu z nazwą wpisu („/a” + „b” = „/a/b”). */
export function polacz(folder: string, nazwa: string): string {
  const baza = folder.endsWith('/') ? folder : folder + '/'
  return baza + nazwa
}

/** Folder nadrzędny („/a/b” → „/a”, „/a” → „/”). */
export function rodzic(sciezka: string): string {
  const i = sciezka.replace(/\/+$/, '').lastIndexOf('/')
  return i <= 0 ? '/' : sciezka.slice(0, i)
}

/** Ostatni człon ścieżki. */
export function nazwaZeSciezki(sciezka: string): string {
  const czesci = sciezka.split('/').filter(Boolean)
  return czesci.length ? czesci[czesci.length - 1] : '/'
}

/** Człony ścieżki bez pustych („/a/b” → [„a”, „b”]). */
export function czesciSciezki(sciezka: string): string[] {
  return sciezka.split('/').filter(Boolean)
}

/** Normalizuje ścieżkę do postaci „/a/b” (zawsze z ukośnikiem na początku). */
export function normalizuj(sciezka: string): string {
  return '/' + czesciSciezki(sciezka).join('/')
}

/** Kopiuje tekst do schowka; działa też bez HTTPS (zapasowo przez zaznaczenie). */
export async function kopiuj(tekst: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(tekst)
      return true
    }
  } catch {
    // celowo: przechodzimy do sposobu zapasowego
  }
  const pole = document.createElement('textarea')
  pole.value = tekst
  pole.setAttribute('readonly', '')
  pole.style.position = 'fixed'
  pole.style.opacity = '0'
  document.body.appendChild(pole)
  pole.select()
  let wynik = false
  try {
    wynik = document.execCommand('copy')
  } catch {
    wynik = false
  }
  document.body.removeChild(pole)
  return wynik
}

/** Rozpoczyna pobranie adresu (przeglądarka sama obsłuży Content-Disposition). */
export function pobierzAdres(adres: string, nazwa = ''): void {
  const a = document.createElement('a')
  a.href = adres
  if (nazwa) a.download = nazwa
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}


/** Buduje zapytanie URL z parametrów (pomija puste). */
export function zapytanie(parametry: Record<string, string | number | boolean | undefined>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(parametry)) {
    if (v === undefined || v === '') continue
    p.set(k, typeof v === 'boolean' ? (v ? '1' : '0') : String(v))
  }
  return p.toString()
}

/** Odczytuje komunikat błędu z odpowiedzi `{"blad": "…"}`. */
export function komunikatBledu(tekst: string, kod: number): string {
  try {
    const j = JSON.parse(tekst) as { blad?: string }
    if (j.blad) return j.blad
  } catch {
    // celowo: odpowiedź bez JSON
  }
  return kod === 0 ? 'Brak połączenia z serwerem.' : `Błąd serwera (${kod}).`
}

/** Dzieli tekst na fragmenty pasujące do słów zapytania (bez wielkości liter) i resztę. */
export function fragmentyPodswietlenia(tekst: string, zapytanie: string): { tekst: string; trafienie: boolean }[] {
  const slowa = zapytanie.trim().toLowerCase().split(/\s+/).filter(Boolean)
  if (!slowa.length) return [{ tekst, trafienie: false }]
  const male = tekst.toLowerCase()
  const znak = new Array<boolean>(tekst.length).fill(false)
  for (const s of slowa) {
    for (let i = male.indexOf(s); i >= 0; i = male.indexOf(s, i + 1)) znak.fill(true, i, i + s.length)
  }
  const wynik: { tekst: string; trafienie: boolean }[] = []
  for (let i = 0; i < tekst.length; i++) {
    const ostatni = wynik[wynik.length - 1]
    if (ostatni && ostatni.trafienie === znak[i]) ostatni.tekst += tekst[i]
    else wynik.push({ tekst: tekst[i], trafienie: znak[i] })
  }
  return wynik
}
