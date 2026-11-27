// Pakiet rdzen: czysta logika bez dysku, sieci i czasu systemowego.
// KONTRAKT: typy i sygnatury w tym pliku są wspólne dla wszystkich warstw; zmiana = uzgodnienie z sesją główną.
package rdzen

import (
	"errors"
	"time"
)

// Wpis to plik albo folder w udziale.
type Wpis struct {
	Nazwa string `json:"nazwa"`
	// Sciezka względna w udziale, zawsze z "/" i od "/" (np. "/zdjecia/a.jpg").
	Sciezka   string    `json:"sciezka"`
	Folder    bool      `json:"folder"`
	Rozmiar   int64     `json:"rozmiar"`
	Zmieniono time.Time `json:"zmieniono"`
	// Rodzaj: RodzajObraz, RodzajWideo … (RodzajPliku).
	Rodzaj string `json:"rodzaj"`
}

// Rodzaje plików do ikon i podglądu.
const (
	RodzajFolder   = "folder"
	RodzajObraz    = "obraz"
	RodzajWideo    = "wideo"
	RodzajDzwiek   = "dzwiek"
	RodzajPDF      = "pdf"
	RodzajTekst    = "tekst"
	RodzajArchiwum = "archiwum"
	RodzajInny     = "inny"
)

// Prawo użytkownika do udziału.
type Prawo string

const (
	PrawoBrak   Prawo = "brak"
	PrawoOdczyt Prawo = "odczyt"
	PrawoZapis  Prawo = "zapis"
)

// Uzytkownik konta (hasło tylko jako hash argon2id).
type Uzytkownik struct {
	Login     string           `json:"login"`
	HashHasla string           `json:"hash_hasla"`
	Admin     bool             `json:"admin"`
	Prawa     map[string]Prawo `json:"prawa"` // klucz: nazwa udziału
	Wylaczony bool             `json:"wylaczony"`
	FTPS      bool             `json:"ftps"` // czy może logować się przez FTPS
}

// Operacja sprawdzana przez bramkę.
type Operacja string

const (
	OpOdczyt        Operacja = "odczyt"        // lista, podgląd, wyszukiwanie
	OpPobranie      Operacja = "pobranie"      // pobranie pliku / ZIP
	OpWgranie       Operacja = "wgranie"       // nowy plik, nowy folder
	OpZmiana        Operacja = "zmiana"        // zmiana nazwy, przeniesienie
	OpUsuniecie     Operacja = "usuniecie"     // do kosza
	OpUdostepnienie Operacja = "udostepnienie" // tworzenie linku
	OpAdministracja Operacja = "administracja" // konta, ustawienia, dziennik
)

// Błędy wspólne (warstwa HTTP mapuje je na kody odpowiedzi).
var (
	ErrPozaUdzialem     = errors.New("ścieżka poza udziałem")
	ErrNieZnaleziono    = errors.New("nie znaleziono")
	ErrBrakPrawa        = errors.New("brak uprawnień")
	ErrJuzIstnieje      = errors.New("już istnieje")
	ErrZlaNazwa         = errors.New("niedozwolona nazwa")
	ErrZleDane          = errors.New("błędne dane logowania")
	ErrKontoZablokowane = errors.New("konto chwilowo zablokowane")
	ErrWygaslo          = errors.New("link wygasł")
	ErrTylkoOdczyt      = errors.New("udział tylko do odczytu")
)
