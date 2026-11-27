package konta

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

func TestUrzadzenia(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	t1, err := s.UtworzZOpisem("jan", "10.0.0.1", "Firefox")
	if err != nil {
		t.Fatal(err)
	}
	z.Przesun(time.Minute)
	t2, _ := s.UtworzZOpisem("jan", "10.0.0.2", "Chrome") // celowo: konto istnieje
	_, _ = s.UtworzZOpisem("admin", "10.0.0.3", "Edge")   // celowo: konto istnieje
	z.Przesun(time.Minute)
	s.Sprawdz(t1) // t1 staje się ostatnio używanym

	lista := s.Urzadzenia("jan")
	if len(lista) != 2 {
		t.Fatalf("urządzeń %d, chcę 2", len(lista))
	}
	if lista[0].ID != IDSesji(t1) || lista[0].Przegladarka != "Firefox" || lista[0].IP != "10.0.0.1" ||
		!lista[0].Ostatnio.Equal(z.Teraz()) || !lista[0].Utworzono.Equal(z.Teraz().Add(-2*time.Minute)) {
		t.Fatalf("zła pierwsza pozycja: %+v", lista[0])
	}
	if lista[1].ID != IDSesji(t2) {
		t.Fatalf("zła kolejność: %+v", lista)
	}
	for _, u := range lista {
		if len(u.ID) != 2*konfiguracja.DlugoscIDSesji || strings.Contains(t1+t2, u.ID) {
			t.Fatalf("ID %q ma złą długość albo jest fragmentem tokenu", u.ID)
		}
	}
	if len(s.Urzadzenia("nikt")) != 0 {
		t.Fatal("urządzenia nieistniejącego konta")
	}
}

func TestZakonczUrzadzenie(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	tj, _ := s.UtworzZOpisem("jan", "10.0.0.1", "Firefox")  // celowo: konto istnieje
	ta, _ := s.UtworzZOpisem("admin", "10.0.0.2", "Chrome") // celowo: konto istnieje
	przypadki := []struct {
		nazwa, login, id string
		chce             error
	}{
		{"cudze urządzenie", "jan", IDSesji(ta), rdzen.ErrNieZnaleziono},
		{"nieznane id", "jan", "0000000000000000", rdzen.ErrNieZnaleziono},
		{"puste id", "jan", "", rdzen.ErrNieZnaleziono},
		{"własne", "jan", IDSesji(tj), nil},
		{"drugi raz", "jan", IDSesji(tj), rdzen.ErrNieZnaleziono},
	}
	for _, p := range przypadki {
		if err := s.ZakonczUrzadzenie(p.login, p.id); !errors.Is(err, p.chce) {
			t.Fatalf("%s: %v, chcę %v", p.nazwa, err, p.chce)
		}
	}
	if _, ok := s.Sprawdz(tj); ok {
		t.Fatal("zakończone urządzenie nadal zalogowane")
	}
	if _, ok := s.Sprawdz(ta); !ok {
		t.Fatal("próba zakończenia cudzego urządzenia zakończyła sesję admina")
	}
}

func TestLimitSesjiNaKonto(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	tokeny := make([]string, konfiguracja.MaksSesjiNaKonto)
	for i := range tokeny {
		tokeny[i], _ = s.Utworz("jan") // celowo: konto istnieje
		z.Przesun(time.Second)
	}
	s.Sprawdz(tokeny[0]) // najstarsza utworzona, ale świeżo użyta
	z.Przesun(time.Second)
	ta, _ := s.Utworz("admin") // celowo: konto istnieje
	nowy, _ := s.Utworz("jan") // celowo: konto istnieje
	if n := len(s.Urzadzenia("jan")); n != konfiguracja.MaksSesjiNaKonto {
		t.Fatalf("sesji %d, chcę %d", n, konfiguracja.MaksSesjiNaKonto)
	}
	if _, ok := s.Sprawdz(tokeny[1]); ok {
		t.Fatal("najdawniej używana sesja nie wypadła")
	}
	for _, tok := range []string{tokeny[0], tokeny[2], nowy, ta} {
		if _, ok := s.Sprawdz(tok); !ok {
			t.Fatal("wypadła niewłaściwa sesja")
		}
	}
}

func TestObetnijOpis(t *testing.T) {
	dlugi := strings.Repeat("ż", konfiguracja.MaksDlugoscOpisuUrzadzenia+5)
	przypadki := []struct {
		wej  string
		znak int
	}{
		{"", 0},
		{"Firefox", 7},
		{dlugi, konfiguracja.MaksDlugoscOpisuUrzadzenia},
		{"a\xffb", 3},
	}
	for _, p := range przypadki {
		got := obetnij(p.wej)
		if utf8.RuneCountInString(got) != p.znak || !utf8.ValidString(got) {
			t.Errorf("obetnij(%q) = %q", p.wej, got)
		}
	}
}
