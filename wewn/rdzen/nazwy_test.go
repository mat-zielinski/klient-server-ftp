package rdzen

import (
	"errors"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
)

func TestSprawdzNazwe(t *testing.T) {
	przypadki := []struct {
		nazwa string
		dobra bool
	}{
		{"plik.txt", true},
		{"Żółw ąę.jpg", true},
		{".gitignore", true},
		{"a..b", true},
		{" spacja na poczatku", true},
		{"console", true},
		{"nul_", true},
		{"com10.txt", true},
		{"lpt", true},
		{"CONx.txt", true},
		{strings.Repeat("a", konfiguracja.MaksDlugoscNazwy), true},
		{"", false},
		{".", false},
		{"..", false},
		{"...", false},
		{".. ", false},
		{"plik.", false},
		{"plik ", false},
		{"plik. ", false},
		{"a/b", false},
		{`a\b`, false},
		{"/", false},
		{"C:", false},
		{"plik:strumien", false},
		{"plik::$DATA", false},
		{"a*", false},
		{"a?", false},
		{`a"`, false},
		{"a<", false},
		{"a>", false},
		{"a|", false},
		{"a\x00", false},
		{"a\x1f", false},
		{"a\x7f", false},
		{"a\u009b", false},
		{"a‮b", false},
		{"a⁦b", false},
		{"\xc3\x28", false},
		{"CON", false},
		{"con", false},
		{"Con.txt", false},
		{"PRN", false},
		{"AUX.log", false},
		{"NUL", false},
		{"nul.tar.gz", false},
		{"NUL .txt", false},
		{"COM1", false},
		{"com9.txt", false},
		{"COM0", false},
		{"LPT1", false},
		{"lpt9.doc", false},
		{"LPT²", false},
		{"CONOUT$", false},
		{"CLOCK$", false},
		{konfiguracja.KatalogKosza, false},
		{strings.ToUpper(konfiguracja.KatalogKosza), false},
		{strings.Repeat("a", konfiguracja.MaksDlugoscNazwy+1), false},
	}
	for _, p := range przypadki {
		err := SprawdzNazwe(p.nazwa)
		if p.dobra && err != nil {
			t.Errorf("SprawdzNazwe(%q) = %v, chcę nil", p.nazwa, err)
		}
		if !p.dobra && !errors.Is(err, ErrZlaNazwa) {
			t.Errorf("SprawdzNazwe(%q) = %v, chcę ErrZlaNazwa", p.nazwa, err)
		}
	}
}

func TestRodzajPliku(t *testing.T) {
	przypadki := []struct {
		nazwa  string
		folder bool
		chce   string
	}{
		{"zdjecia.jpg", true, RodzajFolder},
		{"folder", true, RodzajFolder},
		{"a.JPG", false, RodzajObraz},
		{"a.webp", false, RodzajObraz},
		{"film.MkV", false, RodzajWideo},
		{"utwor.mp3", false, RodzajDzwiek},
		{"umowa.pdf", false, RodzajPDF},
		{"notatki.txt", false, RodzajTekst},
		{"skrypt.ts", false, RodzajTekst},
		{"kopia.tar.gz", false, RodzajArchiwum},
		{"a.zip", false, RodzajArchiwum},
		{"bez_rozszerzenia", false, RodzajInny},
		{".gitignore", false, RodzajInny},
		{"program.exe", false, RodzajInny},
		{"kropka.", false, RodzajInny},
		{"", false, RodzajInny},
	}
	for _, p := range przypadki {
		if got := RodzajPliku(p.nazwa, p.folder); got != p.chce {
			t.Errorf("RodzajPliku(%q, %v) = %q, chcę %q", p.nazwa, p.folder, got, p.chce)
		}
	}
}

func TestRozszerzeniaBezDuplikatow(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("zbudujRodzaje: %v", r)
		}
	}()
	if len(zbudujRodzaje()) == 0 {
		t.Fatal("pusta mapa rodzajów")
	}
}
