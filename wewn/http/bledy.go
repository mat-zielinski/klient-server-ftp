package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"

	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/miniatury"
	"skrytka/wewn/uslugi/transfery"
	"skrytka/wewn/uslugi/udostepnienia"
)

// bladZadania: błąd z gotowym kodem HTTP i komunikatem dla człowieka.
type bladZadania struct {
	kod       int
	komunikat string
}

func (b *bladZadania) Error() string { return b.komunikat }

func nowyBlad(kod int, komunikat string) error { return &bladZadania{kod: kod, komunikat: komunikat} }

// Błędy widoku.
var (
	errBrakSesji       = nowyBlad(nethttp.StatusUnauthorized, "Zaloguj się.")
	errCSRF            = nowyBlad(nethttp.StatusForbidden, "Brak nagłówka ochrony żądania.")
	errZlyJSON         = nowyBlad(nethttp.StatusBadRequest, "Nieprawidłowe dane w żądaniu.")
	errZaDuze          = nowyBlad(nethttp.StatusRequestEntityTooLarge, "Treść żądania jest za duża.")
	errNieobslugiwane  = nowyBlad(nethttp.StatusNotImplemented, "Ta operacja nie jest jeszcze dostępna.")
	errKreatorZrobiony = nowyBlad(nethttp.StatusForbidden, "Kreator został już zakończony.")
	errZlaLiczba       = nowyBlad(nethttp.StatusBadRequest, "Nieprawidłowa liczba w parametrze.")
	errZlaData         = nowyBlad(nethttp.StatusBadRequest, "Nieprawidłowa data (oczekiwany format RFC 3339).")
	errKorzen          = nowyBlad(nethttp.StatusBadRequest, "Tej operacji nie można wykonać na folderze głównym udziału.")
	errBrakZapytania   = nowyBlad(nethttp.StatusBadRequest, "Podaj tekst do wyszukania.")
	errNieznanyAdres   = nowyBlad(nethttp.StatusNotFound, "Nieznany adres API.")
	errMetoda          = nowyBlad(nethttp.StatusMethodNotAllowed, "Niedozwolona metoda.")
)

// mapowanieBledow: błędy rdzenia i usług na kody wg docs/api.md (kolejność ma znaczenie).
var mapowanieBledow = []struct {
	err       error
	kod       int
	komunikat string
}{
	{rdzen.ErrNieZnaleziono, nethttp.StatusNotFound, "Nie znaleziono."},
	{rdzen.ErrBrakPrawa, nethttp.StatusForbidden, "Brak uprawnień."},
	{rdzen.ErrTylkoOdczyt, nethttp.StatusForbidden, "Udział jest tylko do odczytu."},
	{rdzen.ErrPozaUdzialem, nethttp.StatusBadRequest, "Ścieżka wychodzi poza udział."},
	{rdzen.ErrZlaNazwa, nethttp.StatusBadRequest, "Niedozwolona nazwa."},
	{rdzen.ErrZleDane, nethttp.StatusBadRequest, "Nieprawidłowe dane."},
	{rdzen.ErrKontoZablokowane, nethttp.StatusTooManyRequests, "Za dużo nieudanych prób. Spróbuj ponownie później."},
	{rdzen.ErrJuzIstnieje, nethttp.StatusConflict, "Taki element już istnieje."},
	{rdzen.ErrWygaslo, nethttp.StatusGone, "Link wygasł albo wyczerpano limit pobrań."},
	{udostepnienia.ErrZlyLink, nethttp.StatusBadRequest, "Niepoprawne dane linku."},
	{magazyn.ErrZlePrzesuniecie, nethttp.StatusConflict, "Nieprawidłowe przesunięcie kawałka."},
	{transfery.ErrPrzerwany, nethttp.StatusConflict, "Transfer przerwany przez administratora."},
	{konta.ErrSlabeHaslo, nethttp.StatusBadRequest, "Hasło jest za krótkie."},
	{konta.ErrZbytDlugieHaslo, nethttp.StatusBadRequest, "Hasło jest za długie albo ma niedozwolone znaki."},
	{konta.ErrOstatniAdmin, nethttp.StatusConflict, "Nie można odebrać uprawnień ani usunąć ostatniego aktywnego administratora."},
	{miniatury.ErrZlyRozmiar, nethttp.StatusBadRequest, "Nieznany rozmiar miniatury."},
	{miniatury.ErrNieObraz, nethttp.StatusNotFound, "Brak miniatury dla tego pliku."},
	{miniatury.ErrNieObslugiwany, nethttp.StatusNotFound, "Brak miniatury dla tego pliku."},
	{miniatury.ErrZaDuzy, nethttp.StatusNotFound, "Brak miniatury dla tego pliku."},
	{miniatury.ErrUszkodzony, nethttp.StatusNotFound, "Brak miniatury dla tego pliku."},
	{dziennik.ErrZlyZakres, nethttp.StatusBadRequest, "Nieprawidłowy zakres czasu."},
	{dziennik.ErrZlyKrok, nethttp.StatusBadRequest, "Nieprawidłowy krok wykresu."},
	{dziennik.ErrZaDuzoPunktow, nethttp.StatusBadRequest, "Za drobny krok wykresu dla tego zakresu."},
}

// kodBledu zwraca kod HTTP i komunikat; nieznany błąd = 500 (szczegóły tylko w dzienniku).
func kodBledu(err error) (int, string) {
	var bz *bladZadania
	if errors.As(err, &bz) {
		return bz.kod, bz.komunikat
	}
	var za *nethttp.MaxBytesError
	if errors.As(err, &za) {
		return errZaDuze.(*bladZadania).kod, errZaDuze.Error()
	}
	for _, m := range mapowanieBledow {
		if errors.Is(err, m.err) {
			return m.kod, m.komunikat
		}
	}
	return nethttp.StatusInternalServerError, "Błąd serwera. Szczegóły w dzienniku."
}

// blad odpowiada błędem; 500 trafia do dziennika serwera.
func (s *serwer) blad(w nethttp.ResponseWriter, r *nethttp.Request, err error) {
	kod, komunikat := kodBledu(err)
	if kod == nethttp.StatusInternalServerError {
		s.dziennik.Zapisz(dziennik.Wpis{
			Czas: s.teraz(), Rodzaj: dziennik.RodzajSerwer, Kto: loginZ(r), IP: adresIP(r), Protokol: protokol(r),
			Wynik: dziennik.WynikBlad, Opis: r.Method + " " + r.URL.Path + ": " + err.Error(),
		})
	}
	piszBlad(w, kod, komunikat)
}

// bladLogowania: złe dane logowania (konto albo hasło linku) = 401 zamiast 400.
func (s *serwer) bladLogowania(w nethttp.ResponseWriter, r *nethttp.Request, err error) {
	if errors.Is(err, rdzen.ErrZleDane) {
		piszBlad(w, nethttp.StatusUnauthorized, "Błędny login lub hasło.")
		return
	}
	s.blad(w, r, err)
}

func piszBlad(w nethttp.ResponseWriter, kod int, komunikat string) {
	piszJSON(w, kod, map[string]string{"blad": komunikat})
}

// piszJSON wysyła odpowiedź JSON (UTF-8, bez cache).
func piszJSON(w nethttp.ResponseWriter, kod int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(kod)
	_ = json.NewEncoder(w).Encode(v) // celowo: klient mógł się rozłączyć, nagłówki już wysłane
}

// pusto: odpowiedź `{}`.
func pusto(w nethttp.ResponseWriter) { piszJSON(w, nethttp.StatusOK, struct{}{}) }
