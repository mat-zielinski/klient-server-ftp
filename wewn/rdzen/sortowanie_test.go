package rdzen

import (
	"slices"
	"testing"
	"time"
)

func TestPorownajNaturalnie(t *testing.T) {
	przypadki := []struct {
		a, b string
		chce int
	}{
		{"a2", "a10", -1},
		{"a10", "a2", 1},
		{"plik9.txt", "plik10.txt", -1},
		{"A", "a", 0},
		{"Żółw", "żółw", 0},
		{"abc", "ABD", -1},
		{"a", "ą", -1},
		{"ą", "b", -1},
		{"l", "ł", -1},
		{"ł", "m", -1},
		{"z", "ź", -1},
		{"ź", "ż", -1},
		{"o", "ó", -1},
		{"ó", "p", -1},
		{"e", "é", -1},
		{"é", "ę", -1},
		{"Zebra", "żaba", -1},
		{"1", "a", -1},
		{"_a", "a", -1},
		{"a", "ab", -1},
		{"", "a", -1},
		{"", "", 0},
		{"007", "7", 0},
		{"x08", "x9", -1},
		{"99999999999999999999999", "100000000000000000000000", -1},
		{"a1b2", "a1b10", -1},
		{"2024-12-1", "2024-2-1", 1},
		{"z", "я", -1},
	}
	for _, p := range przypadki {
		if got := porownajNaturalnie(p.a, p.b); got != p.chce {
			t.Errorf("porownajNaturalnie(%q, %q) = %d, chcę %d", p.a, p.b, got, p.chce)
		}
	}
}

func nazwy(w []Wpis) []string {
	n := make([]string, len(w))
	for i, x := range w {
		n[i] = x.Nazwa
	}
	return n
}

func TestSortuj(t *testing.T) {
	t0 := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	wpisy := func() []Wpis {
		return []Wpis{
			{Nazwa: "plik10.txt", Rozmiar: 5, Zmieniono: t0.Add(3 * time.Hour)},
			{Nazwa: "Zdjęcia", Folder: true, Zmieniono: t0.Add(1 * time.Hour)},
			{Nazwa: "plik2.txt", Rozmiar: 500, Zmieniono: t0},
			{Nazwa: "ąbc.txt", Rozmiar: 50, Zmieniono: t0.Add(2 * time.Hour)},
			{Nazwa: "archiwum", Folder: true, Zmieniono: t0.Add(5 * time.Hour)},
			{Nazwa: "Abc.txt", Rozmiar: 50, Zmieniono: t0.Add(4 * time.Hour)},
		}
	}
	przypadki := []struct {
		klucz    string
		malejaco bool
		chce     []string
	}{
		{"nazwa", false, []string{"archiwum", "Zdjęcia", "Abc.txt", "ąbc.txt", "plik2.txt", "plik10.txt"}},
		{"nazwa", true, []string{"Zdjęcia", "archiwum", "plik10.txt", "plik2.txt", "ąbc.txt", "Abc.txt"}},
		{"", false, []string{"archiwum", "Zdjęcia", "Abc.txt", "ąbc.txt", "plik2.txt", "plik10.txt"}},
		{"nieznany", false, []string{"archiwum", "Zdjęcia", "Abc.txt", "ąbc.txt", "plik2.txt", "plik10.txt"}},
		{"rozmiar", false, []string{"archiwum", "Zdjęcia", "plik10.txt", "Abc.txt", "ąbc.txt", "plik2.txt"}},
		{"rozmiar", true, []string{"Zdjęcia", "archiwum", "plik2.txt", "ąbc.txt", "Abc.txt", "plik10.txt"}},
		{"zmieniono", false, []string{"Zdjęcia", "archiwum", "plik2.txt", "ąbc.txt", "plik10.txt", "Abc.txt"}},
		{"zmieniono", true, []string{"archiwum", "Zdjęcia", "Abc.txt", "plik10.txt", "ąbc.txt", "plik2.txt"}},
	}
	for _, p := range przypadki {
		w := wpisy()
		Sortuj(w, p.klucz, p.malejaco)
		if got := nazwy(w); !slices.Equal(got, p.chce) {
			t.Errorf("Sortuj(%q, %v) = %v, chcę %v", p.klucz, p.malejaco, got, p.chce)
		}
	}
}

func TestSortujStalaKolejnosc(t *testing.T) {
	// Nazwy równe dla użytkownika ("a" i "A") mają zawsze tę samą kolejność, niezależnie od wejścia.
	a := []Wpis{{Nazwa: "a"}, {Nazwa: "A"}, {Nazwa: "a", Sciezka: "/x/a"}}
	b := []Wpis{{Nazwa: "a", Sciezka: "/x/a"}, {Nazwa: "A"}, {Nazwa: "a"}}
	Sortuj(a, "nazwa", false)
	Sortuj(b, "nazwa", false)
	if !slices.Equal(a, b) {
		t.Errorf("różna kolejność: %v i %v", a, b)
	}
	Sortuj(nil, "nazwa", true) // nie może się wysypać
}

func TestPasuje(t *testing.T) {
	przypadki := []struct {
		nazwa, zapytanie string
		chce             bool
	}{
		{"Żółw.jpg", "zolw", true},
		{"zolw.jpg", "ŻÓŁW", true},
		{"Łódź 2024", "lodz", true},
		{"ZAŻÓŁĆ GĘŚLĄ JAŹŃ", "zazolc gesla jazn", true},
		{"Raport.PDF", "port.p", true},
		{"Café", "cafe", true},
		{"plik.txt", "", true},
		{"plik.txt", "   ", true},
		{"plik.txt", " plik ", true},
		{"plik.txt", "dokument", false},
		{"zolw", "żółwik", false},
		{"abc", "abcd", false},
		{"", "a", false},
	}
	for _, p := range przypadki {
		if got := Pasuje(p.nazwa, p.zapytanie); got != p.chce {
			t.Errorf("Pasuje(%q, %q) = %v, chcę %v", p.nazwa, p.zapytanie, got, p.chce)
		}
	}
}
