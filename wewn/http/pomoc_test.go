package http

import (
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
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

const hasloTestowe = "BardzoTajne123"

// atrapaDziennika zapamiętuje wpisy i ostatni filtr (bez dysku).
type atrapaDziennika struct {
	mu     sync.Mutex
	wpisy  []dziennik.Wpis
	filtry []dziennik.Filtr
}

func (d *atrapaDziennika) Zapisz(w dziennik.Wpis) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wpisy = append(d.wpisy, w)
}

func (d *atrapaDziennika) Czytaj(f dziennik.Filtr) ([]dziennik.Wpis, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.filtry = append(d.filtry, f)
	wynik := []dziennik.Wpis{}
	for _, w := range d.wpisy {
		if (f.Rodzaj == "" || w.Rodzaj == f.Rodzaj) && (f.Kto == "" || w.Kto == f.Kto) {
			wynik = append(wynik, w)
		}
	}
	return wynik, nil
}

func (d *atrapaDziennika) Statystyki(od, do time.Time, krok time.Duration) (dziennik.Statystyki, error) {
	return dziennik.Statystyki{}, nil
}

func (d *atrapaDziennika) EksportCSV(w io.Writer, f dziennik.Filtr) error {
	_, err := d.Czytaj(f)
	return err
}

func (d *atrapaDziennika) kopia() []dziennik.Wpis {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]dziennik.Wpis(nil), d.wpisy...)
}

func (d *atrapaDziennika) ostatniFiltr() dziennik.Filtr {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.filtry[len(d.filtry)-1]
}

// atrapaTykacza: kanały taktów wg okresu, sterowane z testu.
type atrapaTykacza struct {
	mu     sync.Mutex
	kanaly map[time.Duration]chan time.Time
}

func (a *atrapaTykacza) tykacz(okres time.Duration) (<-chan time.Time, func()) {
	return a.kanal(okres), func() {}
}

func (a *atrapaTykacza) kanal(okres time.Duration) chan time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.kanaly[okres] == nil {
		a.kanaly[okres] = make(chan time.Time, 1)
	}
	return a.kanaly[okres]
}

// srodowisko: prawdziwe usługi w t.TempDir(), atrapy dziennika, sieci i zegara.
type srodowisko struct {
	t        *testing.T
	s        *serwer
	h        nethttp.Handler
	katalog  string
	konta    *konta.Konta
	sesje    *konta.Sesje
	dziennik *atrapaDziennika
	rejestr  *transfery.Rejestr
	linki    *udostepnienia.Udostepnienia
	tykacz   *atrapaTykacza
	konfig   konfiguracja.Konfiguracja
	zapisane []konfiguracja.Konfiguracja
	koniec   *zakonczone
}

// Udziały: Filmy (jan: odczyt), Tajne (jan: brak), Wspolne (jan: zapis), Archiwum tylko do odczytu.
func noweSrodowisko(t *testing.T, zKontami bool) *srodowisko {
	t.Helper()
	kat := t.TempDir()
	udzialy := []konfiguracja.Udzial{}
	for _, n := range []string{"Filmy", "Tajne", "Wspolne", "Archiwum"} {
		d := filepath.Join(kat, n)
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
		udzialy = append(udzialy, konfiguracja.Udzial{Nazwa: n, Sciezka: d, TylkoOdczyt: n == "Archiwum"})
	}
	m, err := magazyn.Nowy(udzialy)
	if err != nil {
		t.Fatal(err)
	}
	k, err := konta.Otworz(filepath.Join(kat, "konta.json"))
	if err != nil {
		t.Fatal(err)
	}
	l, err := udostepnienia.Otworz(filepath.Join(kat, "linki.json"))
	if err != nil {
		t.Fatal(err)
	}
	koniec := &zakonczone{}
	rej := transfery.NowyZOpcjami(koniec.dodaj, transfery.ZTykaczem(nil))
	t.Cleanup(rej.Zamknij)
	sr := &srodowisko{t: t, katalog: kat, konta: k, sesje: konta.NoweSesje(k), dziennik: &atrapaDziennika{}, koniec: koniec,
		rejestr: rej, linki: l, tykacz: &atrapaTykacza{kanaly: map[time.Duration]chan time.Time{}}}
	sr.konfig = konfiguracja.Domyslna()
	sr.konfig.Udzialy = udzialy
	if zKontami {
		sr.dodajKonto(rdzen.Uzytkownik{Login: "admin", Admin: true, FTPS: true})
		sr.dodajKonto(rdzen.Uzytkownik{Login: "jan", Prawa: map[string]rdzen.Prawo{
			"Filmy": rdzen.PrawoOdczyt, "Tajne": rdzen.PrawoBrak, "Wspolne": rdzen.PrawoZapis,
			"Archiwum": rdzen.PrawoZapis,
		}})
	}
	u := uslugi{
		konfiguracja: func() konfiguracja.Konfiguracja { return sr.konfig },
		zapiszKonfig: func(k konfiguracja.Konfiguracja) error {
			sr.konfig = k
			sr.zapisane = append(sr.zapisane, k)
			return nil
		},
		magazyn: m, konta: k, sesje: sr.sesje, dziennik: sr.dziennik, transfery: rejestrUslugi{r: rej},
		udostepnienia: l, statyczne: statyczneTestowe(), odcisk: "AB:CD",
		siec: uslugaSieci{
			adresyLokalne:  func() []siec.Adres { return []siec.Adres{{Interfejs: "lan", IP: "192.168.0.10", Rodzaj: "lan"}} },
			adresPubliczny: func() string { return "203.0.113.5" },
			sprawdzPort:    func(int) (bool, string) { return true, "" },
			regulaZapory:   func(string, []int) error { return nil },
		},
		teraz:    time.Now,
		tykacz:   sr.tykacz.tykacz,
		zajetosc: monitor.ZajetoscDysku,
		dodatki: dodatki{
			indeks: indeks.Nowy(m), miniatury: miniatury.Nowe(m, filepath.Join(kat, "miniatury")), sumy: sumy.Nowe(m),
		},
	}
	sr.s = nowySerwer(u)
	sr.h = sr.s.obsluga()
	return sr
}

func statyczneTestowe() fstest.MapFS {
	return fstest.MapFS{
		"index.html":               {Data: []byte("<!doctype html><title>Skrytka</title>")},
		"assets/app-AbCdEf12.js":   {Data: []byte(strings.Repeat("console.log('skrytka');", 50))},
		"assets/styl-Zz99Yy88.css": {Data: []byte("body{margin:0}")},
		"favicon.svg":              {Data: []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")},
		"sw.js":                    {Data: []byte("self.addEventListener(1)")},
		"manifest.webmanifest":     {Data: []byte("{}")},
	}
}

func (sr *srodowisko) dodajKonto(u rdzen.Uzytkownik) {
	sr.t.Helper()
	if err := sr.konta.Dodaj(u, hasloTestowe); err != nil {
		sr.t.Fatal(err)
	}
}

// token sesji dla loginu ("" = bez sesji).
func (sr *srodowisko) token(login string) string {
	sr.t.Helper()
	if login == "" {
		return ""
	}
	t, err := sr.sesje.Utworz(login)
	if err != nil {
		sr.t.Fatal(err)
	}
	return t
}

// plik tworzy plik w udziale.
func (sr *srodowisko) plik(udzial, wzgledna, tresc string) {
	sr.t.Helper()
	p := filepath.Join(sr.katalog, udzial, filepath.FromSlash(wzgledna))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		sr.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(tresc), 0o600); err != nil {
		sr.t.Fatal(err)
	}
}

func (sr *srodowisko) istnieje(udzial, wzgledna string) bool {
	_, err := os.Stat(filepath.Join(sr.katalog, udzial, filepath.FromSlash(wzgledna)))
	return err == nil
}

// zadanie: żądanie z sesją loginu i nagłówkiem CSRF (csrf=false = bez nagłówka).
type zadanie struct {
	metoda, adres, cialo, login string
	bezCSRF                     bool
	// ciasteczko: gotowy token sesji (zamiast nowej sesji dla login).
	ciasteczko string
	naglowki   map[string]string
}

func (sr *srodowisko) wykonaj(z zadanie) *httptest.ResponseRecorder {
	sr.t.Helper()
	var cialo io.Reader
	if z.cialo != "" {
		cialo = strings.NewReader(z.cialo)
	}
	r := httptest.NewRequest(z.metoda, z.adres, cialo)
	r.RemoteAddr = "192.168.0.50:51000"
	if !z.bezCSRF {
		r.Header.Set(konfiguracja.NaglowekCSRF, konfiguracja.WartoscCSRF)
	}
	t := z.ciasteczko
	if t == "" {
		t = sr.token(z.login)
	}
	if t != "" {
		r.AddCookie(&nethttp.Cookie{Name: konfiguracja.NazwaCiasteczkaSesji, Value: t})
	}
	for k, v := range z.naglowki {
		r.Header.Set(k, v)
		if k == "Host" {
			r.Host = v
		}
	}
	w := httptest.NewRecorder()
	sr.h.ServeHTTP(w, r)
	return w
}

// zakonczone: callback naKoniec rejestru (jedno źródło wpisów pobrań w aplikacji).
type zakonczone struct {
	mu     sync.Mutex
	stany  []transfery.Stan
	wyniki []string
}

func (z *zakonczone) dodaj(s transfery.Stan, wynik string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.stany = append(z.stany, s)
	z.wyniki = append(z.wyniki, wynik)
}

func (z *zakonczone) ile() int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return len(z.stany)
}

func (z *zakonczone) kopia() ([]transfery.Stan, []string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	return append([]transfery.Stan(nil), z.stany...), append([]string(nil), z.wyniki...)
}
