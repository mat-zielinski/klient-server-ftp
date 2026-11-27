// Pakiet http: widok HTTP. Tylko tłumaczy żądania na wywołania usług (docs/api.md), bez logiki.
package http

import (
	"io"
	"io/fs"
	nethttp "net/http"
	"os"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/siec"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/indeks"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/miniatury"
	"skrytka/wewn/uslugi/monitor"
	"skrytka/wewn/uslugi/sumy"
	"skrytka/wewn/uslugi/transfery"
	"skrytka/wewn/uslugi/udostepnienia"
)

// Zaleznosci składane w wewn/aplikacja.
type Zaleznosci struct {
	// Konfiguracja zwraca bieżącą konfigurację; ZapiszKonfiguracje zapisuje (zmiana portów = restart).
	Konfiguracja       func() konfiguracja.Konfiguracja
	ZapiszKonfiguracje func(konfiguracja.Konfiguracja) error
	Magazyn            *magazyn.Magazyn
	Konta              *konta.Konta
	Sesje              *konta.Sesje
	Dziennik           *dziennik.Dziennik
	Transfery          *transfery.Rejestr
	Udostepnienia      *udostepnienia.Udostepnienia
	// Statyczne: zbudowany interfejs (web/dist), index.html dla nieznanych ścieżek (SPA).
	Statyczne fs.FS
	// OdciskCertyfikatu do menu protokołów; pusty przy HTTP.
	OdciskCertyfikatu string
	// Usługi fali 2; nil = funkcja wyłączona (adres zwraca 503).
	Monitor   *monitor.Monitor
	Indeks    *indeks.Indeks
	Miniatury *miniatury.Miniatury
	Sumy      *sumy.Sumy
}

// Nowy buduje obsługę wszystkich adresów (API + /s/<id> + pliki interfejsu).
func Nowy(z Zaleznosci) nethttp.Handler {
	u := uslugi{
		konfiguracja:  z.Konfiguracja,
		zapiszKonfig:  z.ZapiszKonfiguracje,
		magazyn:       z.Magazyn,
		konta:         z.Konta,
		sesje:         z.Sesje,
		dziennik:      z.Dziennik,
		transfery:     rejestrUslugi{r: z.Transfery},
		udostepnienia: z.Udostepnienia,
		statyczne:     z.Statyczne,
		odcisk:        z.OdciskCertyfikatu,
		siec:          siecSystemowa(),
		teraz:         time.Now,
		tykacz:        tykaczSystemowy,
		zajetosc:      monitor.ZajetoscDysku,
	}
	u.dodatki = dodatkiZ(z)
	return nowySerwer(u).obsluga()
}

// Interfejsy usług: pozwalają testom podstawić atrapy (sygnatury jak w kontraktach pakietów).
type (
	uslugaMagazynu interface {
		Udzialy() []konfiguracja.Udzial
		Udzial(nazwa string) (konfiguracja.Udzial, error)
		Lista(udzial, sciezka string) ([]rdzen.Wpis, error)
		Info(udzial, sciezka string) (rdzen.Wpis, error)
		Otworz(udzial, sciezka string) (*os.File, rdzen.Wpis, error)
		DopiszKawalek(udzial, sciezka string, przesuniecie int64, r io.Reader) (int64, error)
		RozmiarKawalkow(udzial, sciezka string) (int64, error)
		Zatwierdz(udzial, sciezka string, nadpisz bool) error
		NowyFolder(udzial, sciezka string) error
		ZmienNazwe(udzial, z, na string) error
		DoKosza(udzial, sciezka string) error
		Szukaj(udzial, sciezka, zapytanie string, limit int) ([]rdzen.Wpis, error)
		ZipDo(w io.Writer, udzial string, sciezki []string) (int64, error)
		RozmiarFolderu(udzial, sciezka string) (int64, error)
	}
	uslugaKont interface {
		CzyPusto() bool
		ZalogujZKodem(login, haslo, kod, ip string) (rdzen.Uzytkownik, error)
		Start2FA(login string) (string, string, error)
		Wlacz2FA(login, kod string) error
		Wylacz2FA(login, haslo string) error
		Ma2FA(login string) bool
		Uzytkownik(login string) (rdzen.Uzytkownik, error)
		Lista() []rdzen.Uzytkownik
		Dodaj(u rdzen.Uzytkownik, haslo string) error
		Zmien(u rdzen.Uzytkownik) error
		UstawHaslo(login, haslo string) error
		Usun(login string) error
	}
	uslugaSesji interface {
		UtworzZOpisem(login, ip, przegladarka string) (string, error)
		Urzadzenia(login string) []konta.Urzadzenie
		ZakonczUrzadzenie(login, id string) error
		Sprawdz(token string) (rdzen.Uzytkownik, bool)
		Zakoncz(token string)
		ZakonczWszystkie(login string)
	}
	uslugaDziennika interface {
		Zapisz(w dziennik.Wpis)
		Czytaj(f dziennik.Filtr) ([]dziennik.Wpis, error)
		Statystyki(od, do time.Time, krok time.Duration) (dziennik.Statystyki, error)
		EksportCSV(w io.Writer, f dziennik.Filtr) error
	}
	uslugaUdostepnien interface {
		Utworz(l udostepnienia.Link, haslo string) (udostepnienia.Link, error)
		Lista(autor string) []udostepnienia.Link
		Pobierz(id string) (udostepnienia.Link, error)
		Usun(id string) error
		Sprawdz(id, haslo string) (udostepnienia.Link, error)
		ZaliczPobranie(id string) error
	}
	// transfer: uchwyt jednego przesyłu (transfery.Transfer).
	transfer interface {
		PiszDo(dst io.Writer) io.Writer
		Zakoncz(wynik string)
	}
	uslugaTransferow interface {
		Rozpocznij(s transfery.Stan) transfer
		Aktywne(kto string) []transfery.Stan
		Sluchaj(kto string) (<-chan transfery.Zdarzenie, func())
		PredkoscLaczna() int64
		// Przerwij: errNieobslugiwane, gdy rejestr nie umie przerywać.
		Przerwij(id string) error
	}
)

// uslugaSieci: funkcje pakietu siec (w testach atrapy).
type uslugaSieci struct {
	adresyLokalne  func() []siec.Adres
	adresPubliczny func() string
	sprawdzPort    func(port int) (bool, string)
	regulaZapory   func(nazwa string, porty []int) error
}

func siecSystemowa() uslugaSieci {
	return uslugaSieci{
		adresyLokalne:  siec.AdresyLokalne,
		adresPubliczny: siec.AdresPubliczny,
		sprawdzPort:    siec.SprawdzPort,
		regulaZapory:   siec.RegulaZapory,
	}
}

// tykaczSystemowy: kanał co okres i funkcja zatrzymania (w testach atrapa bez czekania).
func tykaczSystemowy(okres time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(okres)
	return t.C, t.Stop
}

// uslugi: wszystko, czego potrzebuje widok.
type uslugi struct {
	konfiguracja  func() konfiguracja.Konfiguracja
	zapiszKonfig  func(konfiguracja.Konfiguracja) error
	magazyn       uslugaMagazynu
	konta         uslugaKont
	sesje         uslugaSesji
	dziennik      uslugaDziennika
	transfery     uslugaTransferow
	udostepnienia uslugaUdostepnien
	statyczne     fs.FS
	odcisk        string
	siec          uslugaSieci
	teraz         func() time.Time
	tykacz        func(okres time.Duration) (<-chan time.Time, func())
	zajetosc      func(sciezka string) (monitor.Zajetosc, error)
	dodatki
}

// serwer: stan widoku (bez logiki biznesowej).
type serwer struct {
	uslugi
	start        time.Time
	kreatorMutex sync.Mutex
	statyczne    *pamiecStatycznych
	publiczny    pamiecAdresuPublicznego
}

func nowySerwer(u uslugi) *serwer {
	return &serwer{uslugi: u, start: u.teraz(), statyczne: nowaPamiecStatycznych(u.statyczne)}
}

// obsluga: trasy + pośrednicy (od zewnątrz: odzyskiwanie, nagłówki, sesja, CSRF).
func (s *serwer) obsluga() nethttp.Handler {
	m := nethttp.NewServeMux()
	s.trasySesji(m)
	s.trasyPlikow(m)
	s.trasyTransferow(m)
	s.trasyDziennika(m)
	s.trasyUdostepnien(m)
	s.trasyAdmina(m)
	s.trasyFali2(m)
	m.HandleFunc("/", s.plikStatyczny)
	return s.odzyskiwanie(naglowkiBezpieczenstwa(s.sesjaWKontekscie(ochronaCSRF(m))))
}

// rejestrUslugi dopasowuje *transfery.Rejestr do uslugaTransferow.
type rejestrUslugi struct{ r *transfery.Rejestr }

func (a rejestrUslugi) Rozpocznij(s transfery.Stan) transfer { return a.r.Rozpocznij(s) }
func (a rejestrUslugi) Aktywne(kto string) []transfery.Stan  { return a.r.Aktywne(kto) }
func (a rejestrUslugi) PredkoscLaczna() int64                { return a.r.PredkoscLaczna() }

func (a rejestrUslugi) Sluchaj(kto string) (<-chan transfery.Zdarzenie, func()) {
	return a.r.Sluchaj(kto)
}

// Przerwij używa metody rejestru, jeśli ją ma (dodaje agent „transfery”).
func (a rejestrUslugi) Przerwij(id string) error {
	switch p := any(a.r).(type) {
	case interface{ Przerwij(string) error }:
		return p.Przerwij(id)
	case interface{ Przerwij(string) bool }:
		if !p.Przerwij(id) {
			return rdzen.ErrNieZnaleziono
		}
		return nil
	}
	return errNieobslugiwane
}
