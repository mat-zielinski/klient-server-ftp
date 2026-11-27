package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

func TestZakresPasywny(t *testing.T) {
	testy := []struct {
		wej [2]int
		ok  bool
	}{
		{[2]int{50000, 50100}, true},
		{[2]int{50000, 50000}, true},
		{[2]int{0, 0}, false},
		{[2]int{50100, 50000}, false},
		{[2]int{-1, 10}, false},
		{[2]int{60000, 70000}, false},
	}
	for _, tt := range testy {
		r := zakresPasywny(tt.wej)
		if (r != nil) != tt.ok {
			t.Errorf("%v: %v", tt.wej, r)
		}
		if r != nil && (r.Start != tt.wej[0] || r.End != tt.wej[1]) {
			t.Errorf("%v: %+v", tt.wej, r)
		}
	}
}

func TestUstawieniaWymuszajaTLS(t *testing.T) {
	s := noweSrodowisko(t)
	s.u.kfg.PortyPasywne = [2]int{50000, 50010}
	ust, err := nowySterownik(s.u, nil).GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if ust.TLSRequired != ftpserver.MandatoryEncryption || !ust.DisableActiveMode ||
		ust.PasvConnectionsCheck != ftpserver.IPMatchRequired || ust.IdleTimeout != konfiguracja.FTPBezczynnoscS {
		t.Fatalf("ustawienia: %+v", ust)
	}
	if r, ok := ust.PassiveTransferPortRange.(ftpserver.PortRange); !ok || r.Start != 50000 || r.End != 50010 {
		t.Fatalf("porty: %+v", ust.PassiveTransferPortRange)
	}
}

func TestAdresPublicznyWPASV(t *testing.T) {
	testy := []struct{ wej, cel string }{
		{"", ""},
		{"203.0.113.7", "203.0.113.7"},
	}
	for _, tt := range testy {
		s := noweSrodowisko(t)
		s.u.adresPubliczny = tt.wej
		ust, err := nowySterownik(s.u, nil).GetSettings()
		if err != nil || ust.PublicHost != tt.cel {
			t.Errorf("%q: PublicHost %q, %v", tt.wej, ust.PublicHost, err)
		}
	}
}

func TestTLSMinimum12(t *testing.T) {
	s := noweSrodowisko(t)
	if _, err := nowySterownik(s.u, nil).GetTLSConfig(); !errors.Is(err, errBrakTLS) {
		t.Fatalf("brak TLS: %v", err)
	}
	s.u.tls = &tls.Config{MinVersion: tls.VersionTLS10}
	c, err := nowySterownik(s.u, nil).GetTLSConfig()
	if err != nil || c.MinVersion != tls.VersionTLS12 {
		t.Fatalf("%v %v", c, err)
	}
	if s.u.tls.MinVersion != tls.VersionTLS10 {
		t.Fatal("zmieniono oryginał zamiast kopii")
	}
}

func TestUruchomBezTLS(t *testing.T) {
	s := noweSrodowisko(t)
	if err := uruchomNa(context.Background(), s.u, nil); !errors.Is(err, errBrakTLS) {
		t.Fatalf("%v", err)
	}
	if err := Uruchom(context.Background(), Zaleznosci{}); !errors.Is(err, errBrakZaleznosci) {
		t.Fatalf("%v", err)
	}
}

func TestLimitPolaczenNaIP(t *testing.T) {
	s := noweSrodowisko(t)
	st := nowySterownik(s.u, nil)
	var id uint32
	for i := 0; i < konfiguracja.FTPMaksPolaczenNaIP; i++ {
		id++
		if _, err := st.ClientConnected(kontekst(id, "10.0.0.1")); err != nil {
			t.Fatalf("połączenie %d odrzucone: %v", i, err)
		}
	}
	id++
	nadmiar := kontekst(id, "10.0.0.1")
	if _, err := st.ClientConnected(nadmiar); !errors.Is(err, errZaDuzoPolaczen) {
		t.Fatalf("ponad limit: %v", err)
	}
	st.ClientDisconnected(nadmiar)
	id++
	if _, err := st.ClientConnected(kontekst(id, "10.0.0.2")); err != nil {
		t.Fatalf("inny adres: %v", err)
	}
	st.ClientDisconnected(kontekst(1, "10.0.0.1"))
	st.ClientDisconnected(kontekst(1, "10.0.0.1")) // drugie rozłączenie nie może zdjąć licznika drugi raz
	id++
	if _, err := st.ClientConnected(kontekst(id, "10.0.0.1")); err != nil {
		t.Fatalf("po rozłączeniu: %v", err)
	}
	id++
	if _, err := st.ClientConnected(kontekst(id, "10.0.0.1")); err == nil {
		t.Fatal("licznik zdjęty dwa razy")
	}
}

func TestRozlaczWszystkich(t *testing.T) {
	s := noweSrodowisko(t)
	st := nowySterownik(s.u, nil)
	a, b := kontekst(1, "10.0.0.1"), kontekst(2, "10.0.0.2")
	_, _ = st.ClientConnected(a) // celowo: limit nieistotny w tym teście
	_, _ = st.ClientConnected(b) // celowo: j.w.
	st.rozlaczWszystkich()
	if !a.zamkniety || !b.zamkniety {
		t.Fatal("nie zamknięto klientów")
	}
}

func TestLogowanie(t *testing.T) {
	testy := []struct {
		nazwa, login, haslo string
		blokuj              bool
		ok                  bool
		cel                 error
		opis                string
	}{
		{"dobre", "ola", hasloTestowe, false, true, nil, ""},
		{"złe hasło", "ola", "zle-haslo-xyz", false, false, rdzen.ErrZleDane, rdzen.ErrZleDane.Error()},
		{"brak konta", "nikt", "zle-haslo-xyz", false, false, rdzen.ErrZleDane, rdzen.ErrZleDane.Error()},
		{"konto bez FTPS", "web", hasloTestowe, false, false, rdzen.ErrZleDane, errBrakFTPS.Error()},
		{"blokada", "ola", hasloTestowe, true, false, rdzen.ErrKontoZablokowane, rdzen.ErrKontoZablokowane.Error()},
	}
	for _, tt := range testy {
		s := noweSrodowisko(t)
		s.k.blokuj = tt.blokuj
		cc := kontekst(1, "192.168.0.7")
		drv, err := nowySterownik(s.u, nil).AuthUser(cc, tt.login, tt.haslo)
		if tt.ok != (err == nil) || tt.ok != (drv != nil) {
			t.Fatalf("%s: %v %v", tt.nazwa, drv, err)
		}
		if !tt.ok && !errors.Is(err, tt.cel) {
			t.Errorf("%s: %v", tt.nazwa, err)
		}
		w := s.d.kopia()
		if len(w) != 1 {
			t.Fatalf("%s: dziennik %+v", tt.nazwa, w)
		}
		if strings.Contains(w[0].Opis, tt.haslo) || strings.Contains(w[0].Kto, tt.haslo) {
			t.Errorf("%s: hasło w dzienniku", tt.nazwa)
		}
		oczekiwany := dziennik.RodzajLogowanie
		if !tt.ok {
			oczekiwany = dziennik.RodzajLogowanieBlad
		}
		if w[0].Rodzaj != oczekiwany || w[0].IP != "192.168.0.7" || w[0].Protokol != "ftps" || w[0].Kto != tt.login {
			t.Errorf("%s: wpis %+v", tt.nazwa, w[0])
		}
		if !tt.ok && w[0].Opis != tt.opis {
			t.Errorf("%s: opis %q", tt.nazwa, w[0].Opis)
		}
		if tt.ok && cc.sciezka != "/" {
			t.Errorf("%s: ścieżka startowa %q", tt.nazwa, cc.sciezka)
		}
	}
}
