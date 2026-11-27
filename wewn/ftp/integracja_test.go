package ftp

import (
	"context"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/transfery"
)

// serwerTestowy: prawdziwe magazyn, konta, transfery i bramka; dziennik zbierany w pamięci.
type serwerTestowy struct {
	adres   string
	pula    *x509.CertPool
	katalog map[string]string // udział -> folder na dysku
	konta   *konta.Konta
	d       *dziennikTest
	stop    context.CancelFunc
	wynik   chan error
	mu      sync.Mutex
	koniec  []transfery.Stan
	wyniki  []string
}

func (s *serwerTestowy) transfery() ([]transfery.Stan, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]transfery.Stan(nil), s.koniec...), append([]string(nil), s.wyniki...)
}

func nowySerwerTestowy(t *testing.T) *serwerTestowy {
	t.Helper()
	baza := t.TempDir()
	s := &serwerTestowy{katalog: map[string]string{}, d: &dziennikTest{}, wynik: make(chan error, 1)}
	var udzialy []konfiguracja.Udzial
	for _, n := range []string{"publiczny", "tajny", "archiwum"} {
		p := filepath.Join(baza, n)
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
		s.katalog[n] = p
		udzialy = append(udzialy, konfiguracja.Udzial{Nazwa: n, Sciezka: p, TylkoOdczyt: n == "archiwum"})
	}
	if err := os.WriteFile(filepath.Join(s.katalog["publiczny"], "a.txt"), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := magazyn.Nowy(udzialy)
	if err != nil {
		t.Skipf("czeka na magazyn: %v", err)
	}
	s.konta, err = konta.Otworz(filepath.Join(baza, "konta.json"))
	if err != nil {
		t.Skipf("czeka na konta: %v", err)
	}
	dodaj := func(login string, ftps bool, prawa map[string]rdzen.Prawo) {
		if err := s.konta.Dodaj(rdzen.Uzytkownik{Login: login, FTPS: ftps, Prawa: prawa}, hasloTestowe); err != nil {
			t.Skipf("czeka na konta: %v", err)
		}
	}
	dodaj("ola", true, map[string]rdzen.Prawo{"publiczny": rdzen.PrawoZapis, "archiwum": rdzen.PrawoOdczyt})
	dodaj("web", false, map[string]rdzen.Prawo{"publiczny": rdzen.PrawoOdczyt})
	rej := transfery.Nowy(func(st transfery.Stan, w string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.koniec = append(s.koniec, st)
		s.wyniki = append(s.wyniki, w)
	})
	cfgTLS, pula := certyfikatTestowy(t)
	s.pula = pula
	u := uslugi{
		tls: cfgTLS, magazyn: m, konta: s.konta, dziennik: s.d,
		rozpocznij: rozpoczynanie(rej), czyWolno: rdzen.CzyWolno,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.adres = ln.Addr().String()
	ctx, stop := context.WithCancel(context.Background())
	s.stop = stop
	go func() { s.wynik <- uruchomNa(ctx, u, ln) }()
	t.Cleanup(func() {
		stop()
		<-s.wynik
	})
	return s
}

func TestFTPSBezTLSOdrzucone(t *testing.T) {
	s := nowySerwerTestowy(t)
	k := polacz(t, s.adres, s.pula)
	if kod, _ := k.polecenie("USER ola"); kod != 421 {
		t.Fatalf("USER bez TLS: %d", kod)
	}
	k2 := polacz(t, s.adres, s.pula)
	if kod, _ := k2.polecenie("PASS %s", hasloTestowe); kod != 421 {
		t.Fatalf("PASS bez TLS: %d", kod)
	}
	if len(s.d.kopia()) != 0 {
		t.Fatal("logowanie bez TLS dotarło do kont")
	}
}

func TestFTPSDaneBezTLSOdrzucone(t *testing.T) {
	s := nowySerwerTestowy(t)
	k := polacz(t, s.adres, s.pula)
	k.zaloguj("ola", hasloTestowe)
	k.wymagaj(200, "PROT C")
	d := k.dane(false)
	defer d.Close()
	if kod, _ := k.polecenie("RETR /publiczny/a.txt"); kod == 150 {
		t.Fatal("transfer danych bez TLS przyjęty")
	}
}

func TestFTPSLogowanieOdrzucone(t *testing.T) {
	testy := []struct{ login, haslo string }{
		{"ola", "zle-haslo-xyz"},
		{"web", hasloTestowe}, // poprawne hasło, ale konto bez FTPS
		{"nikt", hasloTestowe},
	}
	s := nowySerwerTestowy(t)
	for _, tt := range testy {
		k := polacz(t, s.adres, s.pula)
		k.tls()
		k.wymagaj(331, "USER %s", tt.login)
		k.wymagaj(530, "PASS %s", tt.haslo)
	}
	w := s.d.kopia()
	if len(w) != len(testy) {
		t.Fatalf("dziennik: %+v", w)
	}
	for _, e := range w {
		if e.Rodzaj != dziennik.RodzajLogowanieBlad || e.IP != "127.0.0.1" || strings.Contains(e.Opis, "haslo") {
			t.Errorf("wpis: %+v", e)
		}
	}
}

func TestFTPSPrzebieg(t *testing.T) {
	s := nowySerwerTestowy(t)
	k := polacz(t, s.adres, s.pula)
	k.zaloguj("ola", hasloTestowe)

	if kod, n := k.lista("/"); kod != 226 || strings.Join(n, ",") != "publiczny,archiwum" {
		t.Fatalf("korzeń: %d %v", kod, n)
	}
	if kod, _ := k.polecenie("CWD /tajny"); kod != 550 {
		t.Fatalf("CWD do ukrytego udziału: %d", kod)
	}
	if kod, b := k.pobierz("/publiczny/a.txt", 0); kod != 226 || string(b) != "0123456789" {
		t.Fatalf("RETR: %d %q", kod, b)
	}
	if kod, b := k.pobierz("/publiczny/a.txt", 6); kod != 226 || string(b) != "6789" {
		t.Fatalf("REST+RETR: %d %q", kod, b)
	}
	if kod := k.wyslij("/publiczny/nowy.bin", []byte("wgrane")); kod != 226 {
		t.Fatalf("STOR: %d", kod)
	}
	if b, err := os.ReadFile(filepath.Join(s.katalog["publiczny"], "nowy.bin")); err != nil || string(b) != "wgrane" {
		t.Fatalf("plik po STOR: %q %v", b, err)
	}
	if kod := k.wyslij("/archiwum/x.bin", []byte("x")); kod == 226 || kod == 150 {
		t.Fatalf("STOR w archiwum: %d", kod)
	}
	if kod := k.wyslij("/x.bin", []byte("x")); kod == 226 || kod == 150 {
		t.Fatalf("STOR w korzeniu: %d", kod)
	}
	k.wymagaj(257, "MKD /publiczny/folder")
	k.wymagaj(350, "RNFR /publiczny/nowy.bin")
	k.wymagaj(250, "RNTO /publiczny/folder/nowy.bin")
	k.wymagaj(250, "DELE /publiczny/folder/nowy.bin")
	if _, err := os.Stat(filepath.Join(s.katalog["publiczny"], "folder", "nowy.bin")); !os.IsNotExist(err) {
		t.Fatalf("plik nie zniknął: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.katalog["publiczny"], konfiguracja.KatalogKosza)); err != nil {
		t.Fatalf("brak kosza: %v", err)
	}
	k.wymagaj(250, "RMD /publiczny/folder")

	stany, wyniki := s.transfery()
	if strings.Join(wyniki, ",") != "ok,ok,ok" {
		t.Fatalf("wyniki transferów: %v", wyniki)
	}
	for _, st := range stany {
		if st.Kto != "ola" || st.Protokol != "ftps" || st.IP != "127.0.0.1" {
			t.Errorf("stan: %+v", st)
		}
	}
	if stany[2].Kierunek != transfery.KierunekWgranie || stany[2].Przeslano != 6 {
		t.Errorf("wgranie: %+v", stany[2])
	}
}

func TestFTPSZamkniecieKontekstem(t *testing.T) {
	s := nowySerwerTestowy(t)
	k := polacz(t, s.adres, s.pula)
	k.zaloguj("ola", hasloTestowe)
	s.stop()
	if err := <-s.wynik; err != nil {
		t.Fatalf("Uruchom po anulowaniu: %v", err)
	}
	s.wynik <- nil // dla t.Cleanup
	if kod, _ := k.polecenie("NOOP"); kod != 0 {
		t.Fatalf("połączenie żyje po zamknięciu serwera: %d", kod)
	}
	if _, err := net.Dial("tcp", s.adres); err == nil {
		t.Fatal("serwer dalej przyjmuje połączenia")
	}
}
