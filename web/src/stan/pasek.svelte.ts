// Stan paska bocznego: zwinięty do ikon (zapamiętane w przeglądarce) i szuflada na telefonie.

const KLUCZ = 'skrytka_pasek_zwiniety'

function wczytaj(): boolean {
  try {
    return localStorage.getItem(KLUCZ) === '1'
  } catch {
    return false // celowo: przeglądarka bez localStorage
  }
}

class Pasek {
  /** Użytkownik zwinął pasek do samych ikon. */
  zwiniety = $state(wczytaj())
  /** Szuflada otwarta (tylko telefon). */
  szuflada = $state(false)

  przelaczZwiniecie(): void {
    this.zwiniety = !this.zwiniety
    try {
      localStorage.setItem(KLUCZ, this.zwiniety ? '1' : '0')
    } catch {
      // celowo: brak zapisu nie blokuje zwinięcia
    }
  }

  otworzSzuflade(): void {
    this.szuflada = true
  }

  zamknijSzuflade(): void {
    this.szuflada = false
  }
}

export const pasek = new Pasek()
