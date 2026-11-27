// Pakiet dziennik: dziennik serwera logi/dziennik_RRRR-MM.jsonl (jeden JSON na linię) + statystyki
// do wykresów. Nigdy hasło ani token. KONTRAKT (właściciel: agent „dziennik”).
//
// Implementacja: zapis.go (gorutyna zapisująca), odczyt.go (Czytaj, pamięć podręczna miesięcy),
// statystyki.go, csv.go.
package dziennik

import "time"

// Rodzaje wpisów.
const (
	RodzajLogowanie     = "logowanie"
	RodzajLogowanieBlad = "logowanie_blad"
	RodzajPobranie      = "pobranie" // zakończone (Wynik ok / przerwane / blad)
	RodzajWgranie       = "wgranie"
	RodzajZmiana        = "zmiana" // nazwa, folder, kosz
	RodzajUdostepnienie = "udostepnienie"
	RodzajAdministracja = "administracja"
	RodzajSerwer        = "serwer" // start, stop, błąd
)

// Wyniki.
const (
	WynikOK        = "ok"
	WynikPrzerwane = "przerwane"
	WynikBlad      = "blad"
)

// Wpis dziennika.
type Wpis struct {
	Czas     time.Time `json:"czas"`
	Rodzaj   string    `json:"rodzaj"`
	Kto      string    `json:"kto"` // login albo "link:<id>"
	IP       string    `json:"ip"`
	Protokol string    `json:"protokol"` // "https" | "http" | "ftps" | ""
	Udzial   string    `json:"udzial"`
	Sciezka  string    `json:"sciezka"`
	Bajty    int64     `json:"bajty"`
	CzasMs   int64     `json:"czas_ms"`
	Wynik    string    `json:"wynik"`
	Opis     string    `json:"opis"`
}

// Filtr do odczytu (puste pola = bez ograniczenia). Zakres czasu: Od włącznie, Do wyłącznie.
type Filtr struct {
	Od, Do time.Time
	Rodzaj string
	Kto    string
	Szukaj string // w ścieżce i opisie, bez rozróżniania wielkości liter
	Limit  int
}

// PunktWykresu: jeden przedział czasu.
type PunktWykresu struct {
	Czas       time.Time `json:"czas"`
	Bajty      int64     `json:"bajty"`
	Pobrania   int       `json:"pobrania"`
	SredniaBps int64     `json:"srednia_bps"`
}

// Statystyki do strony „Dziennik pobrań”.
type Statystyki struct {
	Wykres                []PunktWykresu    `json:"wykres"` // bajty i liczba pobrań w przedziałach
	NajczestszePliki      []PozycjaRankingu `json:"najczestsze_pliki"`
	NajaktywniejsiKlienci []PozycjaRankingu `json:"najaktywniejsi_klienci"`
	Protokoly             map[string]int64  `json:"protokoly"` // bajty wg protokołu
	SumaBajtow            int64             `json:"suma_bajtow"`
	SumaPobran            int               `json:"suma_pobran"`
}

// PozycjaRankingu dla list „top”.
type PozycjaRankingu struct {
	Nazwa  string `json:"nazwa"`
	Bajty  int64  `json:"bajty"`
	Liczba int    `json:"liczba"`
}
