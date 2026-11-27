// Pakiet aplikacja: składa warstwy (konfiguracja, magazyn, usługi, HTTP, FTPS), kreator, usługa
// Windows, zamykanie. Jedyny pakiet, który zna wszystkie pozostałe.
package aplikacja

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"skrytka/web"
	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/siec"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/monitor"
	"skrytka/wewn/uslugi/transfery"
	"skrytka/wewn/uslugi/udostepnienia"
)

// Opcje uruchomienia serwera.
type Opcje struct {
	// Konsola: nie otwieraj przeglądarki; wpisy serwera także na standardowe wyjście błędów.
	Konsola bool
	// OtworzPrzegladarke: nil = przeglądarka systemowa (testy podmieniają).
	OtworzPrzegladarke func(adres string) error
	// Teraz: źródło czasu; nil = time.Now.
	Teraz func() time.Time
	// AdresPubliczny: źródło IP zewnętrznego (tryb publiczny z FTPS); nil = siec.AdresPubliczny.
	AdresPubliczny func() string
	// PoStarcie: wołane po uruchomieniu nasłuchu (testy czytają adresy).
	PoStarcie func(a *Aplikacja)
}

// Aplikacja: złożony serwer z usługami i serwerami sieciowymi.
type Aplikacja struct {
	opcje    Opcje
	plikKonf string

	mu      sync.RWMutex // chroni konf
	konf    konfiguracja.Konfiguracja
	zapisMu sync.Mutex // szereguje ZapiszKonfiguracje

	dziennik  *dziennik.Dziennik
	konta     *konta.Konta
	sesje     *konta.Sesje
	udost     *udostepnienia.Udostepnienia
	transfery *transfery.Rejestr
	tls       *tls.Config
	odcisk    string
	statyczne fs.FS
	konsola   *log.Logger

	magazyn   atomic.Pointer[magazyn.Magazyn] // bieżący magazyn (FTPS bierze go przy restarcie)
	uslugiMag atomic.Pointer[uslugiMagazynu]  // indeks, miniatury, sumy bieżącego magazynu
	monitor   *monitor.Monitor
	// polaczeniaHTTP: otwarte połączenia HTTP i HTTPS (dla monitora).
	polaczeniaHTTP atomic.Int64
	// ctx: czas życia serwera (indeksowanie i monitor kończą się z nim).
	ctx context.Context
	// adresPubliczny (string): IP zewnętrzne dla trybu pasywnego FTPS w trybie publicznym, "" = nieznane.
	adresPubliczny atomic.Value
	obsluga        obslugaPrzelaczana
	serwery        serwery
	ftps           *biegFTPS
}

// zloz otwiera konfigurację, dziennik, konta, sesje, udostępnienia, magazyn, rejestr transferów i TLS.
func zloz(ctx context.Context, o Opcje) (*Aplikacja, error) {
	if o.Teraz == nil {
		o.Teraz = time.Now
	}
	if o.AdresPubliczny == nil {
		o.AdresPubliczny = siec.AdresPubliczny
	}
	if o.OtworzPrzegladarke == nil {
		o.OtworzPrzegladarke = otworzPrzegladarke
	}
	a := &Aplikacja{opcje: o, plikKonf: konfiguracja.PlikKonfiguracji(), ctx: ctx}
	a.adresPubliczny.Store("")
	if o.Konsola {
		a.konsola = log.New(os.Stderr, "", log.LstdFlags)
	}
	d, err := dziennik.Otworz(konfiguracja.KatalogLogow())
	if err != nil {
		return nil, err
	}
	a.dziennik = d
	if err := a.zlozUslugi(); err != nil {
		a.zglosBlad("Składanie aplikacji", err)
		a.zamknijDziennik()
		return nil, err
	}
	return a, nil
}

// zlozUslugi: wszystko poza dziennikiem (dziennik jest już otwarty, żeby zapisać błąd).
func (a *Aplikacja) zlozUslugi() error {
	k, err := konfiguracja.Wczytaj(a.plikKonf)
	if err != nil {
		return fmt.Errorf("Nie można wczytać konfiguracji %s: %w", a.plikKonf, err)
	}
	a.konf = k
	if a.konta, err = konta.Otworz(konfiguracja.PlikKont()); err != nil {
		return fmt.Errorf("Nie można otworzyć kont: %w", err)
	}
	a.sesje = konta.NoweSesje(a.konta)
	if a.udost, err = udostepnienia.Otworz(konfiguracja.PlikUdostepnien()); err != nil {
		return fmt.Errorf("Nie można otworzyć udostępnień: %w", err)
	}
	a.transfery = transfery.Nowy(a.naKoniecTransferu)
	a.monitor = monitor.Nowy(monitor.ZLicznikami(a.liczniki))
	if a.tls, err = siec.KonfiguracjaTLS(k); err != nil {
		return fmt.Errorf("Nie można przygotować TLS: %w", err)
	}
	a.odcisk = siec.OdciskCertyfikatu(a.tls)
	a.statyczne = web.Pliki()
	m, err := magazyn.Nowy(k.Udzialy)
	if err != nil {
		// Brak folderu udziału nie blokuje startu: admin poprawi udziały w Ustawieniach.
		a.zglosBlad("Udziały z konfiguracji", err)
		if m, err = magazyn.Nowy(nil); err != nil {
			return fmt.Errorf("Nie można utworzyć pustego magazynu: %w", err)
		}
	}
	a.ustawMagazyn(m)
	return nil
}

// naKoniecTransferu zapisuje zakończony transfer w dzienniku.
func (a *Aplikacja) naKoniecTransferu(s transfery.Stan, wynik string) {
	a.dziennik.Zapisz(wpisTransferu(s, wynik, a.opcje.Teraz()))
}

// wpisTransferu: wpis dziennika z zakończonego transferu (czysta funkcja, testowana).
func wpisTransferu(s transfery.Stan, wynik string, teraz time.Time) dziennik.Wpis {
	rodzaj := dziennik.RodzajPobranie
	if s.Kierunek == transfery.KierunekWgranie {
		rodzaj = dziennik.RodzajWgranie
	}
	ms := int64(0)
	if !s.Start.IsZero() && teraz.After(s.Start) {
		ms = teraz.Sub(s.Start).Milliseconds()
	}
	return dziennik.Wpis{
		Czas: teraz, Rodzaj: rodzaj, Kto: s.Kto, IP: s.IP, Protokol: s.Protokol,
		Udzial: s.Udzial, Sciezka: s.Sciezka, Bajty: s.Przeslano, CzasMs: ms, Wynik: wynik,
	}
}

// zglos zapisuje wpis serwera (start, stop) w dzienniku i, w trybie konsoli, na ekranie.
func (a *Aplikacja) zglos(opis string) {
	a.dziennik.Zapisz(dziennik.Wpis{Rodzaj: dziennik.RodzajSerwer, Wynik: dziennik.WynikOK, Opis: opis})
	if a.konsola != nil {
		a.konsola.Print(opis)
	}
}

// zglosBlad zapisuje błąd serwera w dzienniku i, w trybie konsoli, na ekranie.
func (a *Aplikacja) zglosBlad(gdzie string, err error) {
	opis := gdzie + ": " + err.Error()
	a.dziennik.Zapisz(dziennik.Wpis{Rodzaj: dziennik.RodzajSerwer, Wynik: dziennik.WynikBlad, Opis: opis})
	if a.konsola != nil {
		a.konsola.Print(opis)
	}
}

// zamknijDziennik kończy zapis dziennika; błąd widoczny tylko w konsoli (dziennika już nie ma).
func (a *Aplikacja) zamknijDziennik() {
	if err := a.dziennik.Zamknij(); err != nil && a.konsola != nil {
		a.konsola.Print("Zamykanie dziennika: " + err.Error())
	}
}

// uruchomSerwer: złożenie, start, czekanie na koniec ctx, łagodne zamknięcie.
func uruchomSerwer(ctx context.Context, o Opcje) error {
	ctxSerwera, anuluj := context.WithCancel(ctx)
	defer anuluj()
	a, err := zloz(ctxSerwera, o)
	if err != nil {
		return err
	}
	defer a.zamknijDziennik()
	if err := a.start(ctxSerwera); err != nil {
		a.zglosBlad("Start serwera", err)
		anuluj()
		a.zatrzymaj()
		return err
	}
	if o.PoStarcie != nil {
		o.PoStarcie(a)
	}
	a.otworzKreatorJesliTrzeba()
	select {
	case <-ctxSerwera.Done():
	case err = <-a.serwery.bledy:
		a.zglosBlad("Serwer przestał działać", err)
	}
	anuluj()
	a.zatrzymaj()
	return err
}

// otworzKreatorJesliTrzeba: pierwsze uruchomienie (brak kont) bez --konsola = przeglądarka na kreatorze.
func (a *Aplikacja) otworzKreatorJesliTrzeba() {
	if a.opcje.Konsola || !a.konta.CzyPusto() {
		return
	}
	adres := a.serwery.adresLokalnyHTTPS()
	if adres == "" {
		return
	}
	if err := a.opcje.OtworzPrzegladarke(adres); err != nil {
		a.zglosBlad("Otwieranie przeglądarki", err)
	}
}

// errBrakAdresuPublicznego: żadna usługa nie podała IP zewnętrznego.
var errBrakAdresuPublicznego = errors.New("Nie udało się ustalić adresu publicznego; tryb pasywny FTPS zza routera może nie działać.")

// errBrakNasluchu: żaden adres HTTPS nie dał się otworzyć.
var errBrakNasluchu = errors.New("Nie udało się otworzyć żadnego adresu HTTPS.")
