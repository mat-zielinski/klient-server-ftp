package main

import "testing"

func TestZastosujWyjatki(t *testing.T) {
	dluga := Naruszenie{Regula: "jakosc.dlugosc_funkcji", Plik: "wewn/a/a.go", Nazwa: "F"}
	inna := Naruszenie{Regula: "jakosc.dlugosc_funkcji", Plik: "wewn/a/a.go", Nazwa: "G"}
	wszystkie := map[string]bool{"architektura": true, "jakosc": true, "testy": true, "styl": true}
	powod := "parser stanów, podział pogorszy czytelność"
	przypadki := []struct {
		nazwa      string
		naruszenia []Naruszenie
		wyjatki    []Wyjatek
		grupy      map[string]bool
		oczekiwane []string
	}{
		{"bez wyjątków", []Naruszenie{dluga}, nil, wszystkie, []string{"jakosc.dlugosc_funkcji@wewn/a/a.go"}},
		{"wyjątek z nazwą obejmuje tylko tę funkcję", []Naruszenie{dluga, inna},
			[]Wyjatek{{Regula: dluga.Regula, Plik: dluga.Plik, Nazwa: "F", Uzasadnienie: powod}}, wszystkie,
			[]string{"jakosc.dlugosc_funkcji@wewn/a/a.go"}},
		{"wyjątek bez nazwy obejmuje cały plik", []Naruszenie{dluga, inna},
			[]Wyjatek{{Regula: dluga.Regula, Plik: dluga.Plik, Uzasadnienie: powod}}, wszystkie, nil},
		{"wyjątek bez uzasadnienia nie działa", []Naruszenie{dluga},
			[]Wyjatek{{Regula: dluga.Regula, Plik: dluga.Plik}}, wszystkie,
			[]string{"jakosc.dlugosc_funkcji@wewn/a/a.go", "wyjatki.zly_wpis@" + PlikWyjatkow}},
		{"nieużywany wyjątek", nil,
			[]Wyjatek{{Regula: dluga.Regula, Plik: dluga.Plik, Uzasadnienie: powod}}, wszystkie,
			[]string{"wyjatki.nieuzywany@" + PlikWyjatkow}},
		{"nieużywany, ale grupa nie sprawdzana", nil,
			[]Wyjatek{{Regula: dluga.Regula, Plik: dluga.Plik, Uzasadnienie: powod}}, map[string]bool{"testy": true}, nil},
		{"inna reguła nie pasuje", []Naruszenie{dluga},
			[]Wyjatek{{Regula: "jakosc.zlozonosc", Plik: dluga.Plik, Uzasadnienie: powod}}, wszystkie,
			[]string{"jakosc.dlugosc_funkcji@wewn/a/a.go", "wyjatki.nieuzywany@" + PlikWyjatkow}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			porownaj(t, zastosujWyjatki(p.naruszenia, p.wyjatki, p.grupy), p.oczekiwane...)
		})
	}
}

func TestWczytajWyjatki(t *testing.T) {
	przypadki := []struct {
		nazwa string
		pliki map[string]string
		ile   int
		blad  bool
	}{
		{"brak pliku", nil, 0, false},
		{"pusta lista", map[string]string{PlikWyjatkow: `{"_opis":"x","wyjatki":[]}`}, 0, false},
		{"jeden wpis", map[string]string{PlikWyjatkow: `{"wyjatki":[{"regula":"a.b","plik":"c","uzasadnienie":"powód"}]}`}, 1, false},
		{"zły JSON", map[string]string{PlikWyjatkow: `{`}, 0, true},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w, err := wczytajWyjatki(zbudujProjekt(t, p.pliki))
			if (err != nil) != p.blad || len(w) != p.ile {
				t.Errorf("wyjątki %v, błąd %v", w, err)
			}
		})
	}
}
