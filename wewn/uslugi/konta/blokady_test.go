package konta

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

func TestZaloguj(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	if err := k.Zmien(rdzen.Uzytkownik{Login: "jan", Wylaczony: true}); err != nil {
		t.Fatal(err)
	}
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "ola"}, "haslo-oli-12"); err != nil {
		t.Fatal(err)
	}
	przypadki := []struct {
		nazwa, login, haslo string
		chce                error
	}{
		{"dobre hasło", "admin", hasloAdmina, nil},
		{"złe hasło", "admin", hasloAdmina + "x", rdzen.ErrZleDane},
		{"wielkość liter loginu", "Admin", hasloAdmina, rdzen.ErrZleDane},
		{"nieistniejący login", "nikt", hasloAdmina, rdzen.ErrZleDane},
		{"wyłączone konto, dobre hasło", "jan", hasloJana, rdzen.ErrZleDane},
		{"cudze hasło", "ola", hasloAdmina, rdzen.ErrZleDane},
		{"bardzo długie hasło", "ola", strings.Repeat("a", 10000), rdzen.ErrZleDane},
	}
	for i, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			u, err := k.Zaloguj(p.login, p.haslo, fmt.Sprintf("10.0.0.%d", i))
			if !errors.Is(err, p.chce) {
				t.Fatalf("błąd %v, chcę %v", err, p.chce)
			}
			if err == nil && (u.Login != p.login || u.HashHasla != "") {
				t.Fatalf("zły użytkownik albo widoczny hash: %+v", u)
			}
		})
	}
}

func TestZalogujNieistniejacyLoginTezLiczyHash(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	wywolania := 0
	k.porownaj = func(hash, haslo string) bool {
		wywolania++
		return sprawdzHaslo(hash, haslo)
	}
	if _, err := k.Zaloguj("nikt", "jakies-haslo-1", "10.0.0.1"); !errors.Is(err, rdzen.ErrZleDane) {
		t.Fatalf("błąd %v", err)
	}
	if wywolania != 1 || !strings.HasPrefix(k.atrapa, prefiksArgon) {
		t.Fatalf("dla nieistniejącego loginu nie policzono hasha (wywołania=%d)", wywolania)
	}
}

// zleProby wykonuje n złych logowań.
func zleProby(t *testing.T, k *Konta, n int, login func(i int) string, ip func(i int) string) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := k.Zaloguj(login(i), "zle-haslo-xyz", ip(i)); !errors.Is(err, rdzen.ErrZleDane) {
			t.Fatalf("próba %d: %v, chcę ErrZleDane", i, err)
		}
	}
}

func staly(s string) func(int) string { return func(int) string { return s } }

func rozne(prefiks string) func(int) string {
	return func(i int) string { return fmt.Sprintf("%s%d", prefiks, i) }
}

func TestBlokadaLoginu(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	zleProby(t, k, konfiguracja.MaksBlednychLogowan, staly("jan"), rozne("10.1.0."))
	// Dobre hasło z nowego IP: login zablokowany.
	if _, err := k.Zaloguj("jan", hasloJana, "10.9.9.9"); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("po %d złych próbach: %v, chcę blokadę", konfiguracja.MaksBlednychLogowan, err)
	}
	// Inny login z tych samych IP działa.
	if _, err := k.Zaloguj("admin", hasloAdmina, "10.9.9.8"); err != nil {
		t.Fatalf("blokada objęła inny login: %v", err)
	}
	z.Przesun(konfiguracja.CzasBlokady - time.Second)
	if _, err := k.Zaloguj("jan", hasloJana, "10.9.9.9"); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("blokada zdjęta za wcześnie: %v", err)
	}
	z.Przesun(time.Second)
	if _, err := k.Zaloguj("jan", hasloJana, "10.9.9.9"); err != nil {
		t.Fatalf("po czasie blokady: %v", err)
	}
}

func TestBlokadaIP(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	zleProby(t, k, konfiguracja.MaksBlednychLogowan, rozne("nikt"), staly("10.2.0.1"))
	if _, err := k.Zaloguj("admin", hasloAdmina, "10.2.0.1"); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("IP po złych próbach: %v, chcę blokadę", err)
	}
	if _, err := k.Zaloguj("admin", hasloAdmina, "10.2.0.2"); err != nil {
		t.Fatalf("blokada objęła inne IP: %v", err)
	}
	z.Przesun(konfiguracja.CzasBlokady)
	if _, err := k.Zaloguj("admin", hasloAdmina, "10.2.0.1"); err != nil {
		t.Fatalf("po czasie blokady: %v", err)
	}
}

func TestBlokadaOknoCzasu(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	zleProby(t, k, konfiguracja.MaksBlednychLogowan-1, staly("jan"), staly("10.3.0.1"))
	z.Przesun(konfiguracja.OknoBlednychLogowan + time.Second)
	zleProby(t, k, konfiguracja.MaksBlednychLogowan-1, staly("jan"), staly("10.3.0.1"))
	if _, err := k.Zaloguj("jan", hasloJana, "10.3.0.1"); err != nil {
		t.Fatalf("stare próby spoza okna liczone: %v", err)
	}
}

func TestDobreLogowanieCzysciLicznikLoginu(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	zleProby(t, k, konfiguracja.MaksBlednychLogowan-1, staly("jan"), rozne("10.4.0."))
	if _, err := k.Zaloguj("jan", hasloJana, "10.4.1.1"); err != nil {
		t.Fatal(err)
	}
	zleProby(t, k, konfiguracja.MaksBlednychLogowan-1, staly("jan"), rozne("10.4.2."))
	if _, err := k.Zaloguj("jan", hasloJana, "10.4.3.1"); err != nil {
		t.Fatalf("licznik loginu nie wyczyszczony po dobrym logowaniu: %v", err)
	}
}

func TestBlokadySprzatanie(t *testing.T) {
	b := noweBlokady()
	teraz := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < konfiguracja.ProgSprzataniaBlokad; i++ {
		b.zarejestrujBlad(teraz, kluczIP(fmt.Sprint(i)))
	}
	b.zarejestrujBlad(teraz.Add(konfiguracja.OknoBlednychLogowan+time.Second), kluczIP("nowy"))
	if len(b.wpisy) != 1 {
		t.Fatalf("po sprzątaniu %d wpisów, chcę 1", len(b.wpisy))
	}
}
