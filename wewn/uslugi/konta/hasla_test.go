package konta

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestZahaszujFormatPHCDomyslneParametry(t *testing.T) {
	h, err := zahaszuj("dowolne-haslo-1", domyslneParametry())
	if err != nil {
		t.Fatal(err)
	}
	wzor := regexp.MustCompile(`^\$argon2id\$v=19\$m=65536,t=2,p=2\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)
	if !wzor.MatchString(h) {
		t.Fatalf("zły format hasha: %q", h)
	}
	if !sprawdzHaslo(h, "dowolne-haslo-1") || sprawdzHaslo(h, "dowolne-haslo-2") {
		t.Fatal("weryfikacja z parametrami domyślnymi nie działa")
	}
}

func TestSprawdzHaslo(t *testing.T) {
	h, err := zahaszuj("poprawne-haslo", tanieParametry)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := zahaszuj("poprawne-haslo", tanieParametry)
	if err != nil {
		t.Fatal(err)
	}
	if h == h2 {
		t.Fatal("ta sama sól dla dwóch haszowań")
	}
	czesci := strings.Split(h, "$")
	przypadki := []struct {
		nazwa, hash, haslo string
		chce               bool
	}{
		{"dobre", h, "poprawne-haslo", true},
		{"złe", h, "poprawne-haslO", false},
		{"puste", h, "", false},
		{"pusty hash", "", "poprawne-haslo", false},
		{"inny algorytm", strings.Replace(h, "argon2id", "argon2i", 1), "poprawne-haslo", false},
		{"zła wersja", strings.Replace(h, "v=19", "v=16", 1), "poprawne-haslo", false},
		{"pamięć ponad limit", strings.Replace(h, "m=64,", "m=99999999,", 1), "poprawne-haslo", false},
		{"zero wątków", strings.Replace(h, "p=1", "p=0", 1), "poprawne-haslo", false},
		{"brak klucza", strings.Join(czesci[:5], "$"), "poprawne-haslo", false},
		{"zepsuta sól", strings.Replace(h, czesci[4], "!!!", 1), "poprawne-haslo", false},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if got := sprawdzHaslo(p.hash, p.haslo); got != p.chce {
				t.Fatalf("sprawdzHaslo = %v, chcę %v", got, p.chce)
			}
		})
	}
}

func TestSprawdzDlugoscHasla(t *testing.T) {
	przypadki := []struct {
		haslo string
		chce  error
	}{
		{"krotkie", ErrSlabeHaslo},
		{"123456789", ErrSlabeHaslo},
		{"1234567890", nil},
		{"zażółćgęśl", nil}, // 10 znaków, więcej bajtów
		{"zażółćgęś", ErrSlabeHaslo},
		{strings.Repeat("a", 1025), ErrZbytDlugieHaslo},
		{"aaaaaaaaaa\xff", ErrZbytDlugieHaslo},
	}
	for _, p := range przypadki {
		if err := sprawdzDlugoscHasla(p.haslo); !errors.Is(err, p.chce) {
			t.Errorf("%q: %v, chcę %v", p.haslo, err, p.chce)
		}
	}
}
