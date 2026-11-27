package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// nodeNieWolany: styl web bez pliku sprawdz_styl.mjs nie może wołać node.
func nodeNieWolany(t *testing.T) uruchamiaczNode {
	return func(string, string, ...string) (string, error) {
		t.Error("node wywołany, choć nie powinien")
		return "", nil
	}
}

func TestUruchomKodyWyjscia(t *testing.T) {
	przypadki := []struct {
		nazwa    string
		pliki    map[string]string
		argument []string
		kod      int
		zawiera  string
	}{
		{"czysto", map[string]string{"wewn/rdzen/r.go": "package rdzen\n"}, nil, KodOK, "STRAŻNIK: OK"},
		{"naruszenie", map[string]string{"wewn/rdzen/r.go": "package rdzen\nimport \"os\"\nvar _ = os.Args\n"}, nil,
			KodNaruszenia, "[architektura.rdzen_import]"},
		{"tylko jakość pomija architekturę", map[string]string{"wewn/rdzen/r.go": "package rdzen\nimport \"os\"\nvar _ = os.Args\n"},
			[]string{"-tylko", "jakosc"}, KodOK, "STRAŻNIK: OK"},
		{"nieznana grupa", map[string]string{}, []string{"-tylko", "inna"}, KodBladUruchomienia, "nieznana grupa"},
		{"błąd składni", map[string]string{"wewn/rdzen/r.go": "package rdzen\nfunc {\n"}, nil, KodBladUruchomienia, "błąd uruchomienia"},
		{"zły wyjątek", map[string]string{
			"wewn/rdzen/r.go":            "package rdzen\n",
			"narzedzia_dev/wyjatki.json": `{"wyjatki":[{"regula":"jakosc.zlozonosc","plik":"wewn/rdzen/r.go"}]}`,
		}, nil, KodNaruszenia, "[wyjatki.zly_wpis]"},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			korzen := zbudujProjekt(t, p.pliki)
			var wyjscie bytes.Buffer
			kod := uruchom(append([]string{"-katalog", korzen}, p.argument...), &wyjscie, nodeNieWolany(t))
			if kod != p.kod || !strings.Contains(wyjscie.String(), p.zawiera) {
				t.Errorf("kod %d (oczekiwany %d), wyjście:\n%s\n(oczekiwane: %q)", kod, p.kod, wyjscie.String(), p.zawiera)
			}
		})
	}
}

func TestStylWeb(t *testing.T) {
	bladWyjscia := func() error { return &exec.ExitError{} } // węzeł zakończył się kodem różnym od 0
	przypadki := []struct {
		nazwa      string
		wyjscie    string
		err        error
		oczekiwane []string
		opis       string
	}{
		{"styl OK", "", nil, nil, ""},
		{"styl z błędami", "web/src/x.svelte:3: kolor spoza tokenów", bladWyjscia(), []string{"styl.web@web/sprawdz_styl.mjs"}, "kolor spoza tokenów"},
		{"brak node", "", errors.New("exec: \"node\": nie znaleziono"), []string{"styl.web@web/sprawdz_styl.mjs"}, "nie udało się uruchomić node"},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			korzen := zbudujProjekt(t, map[string]string{PlikStyluWeb: "// strażnik stylu\n"})
			var wolanie []string
			node := func(katalog, polecenie string, argumenty ...string) (string, error) {
				wolanie = append([]string{polecenie}, argumenty...)
				return p.wyjscie, p.err
			}
			n, sprawdzono := sprawdzStylWeb(korzen, node)
			if !sprawdzono || strings.Join(wolanie, " ") != "node "+PlikStyluWeb {
				t.Fatalf("sprawdzono=%v, wołanie %v", sprawdzono, wolanie)
			}
			porownaj(t, n, p.oczekiwane...)
			if p.opis != "" && (len(n) == 0 || !strings.Contains(n[0].Opis, p.opis)) {
				t.Errorf("opis %v nie zawiera %q", n, p.opis)
			}
		})
	}
}

func TestStylWebBezPliku(t *testing.T) {
	n, sprawdzono := sprawdzStylWeb(zbudujProjekt(t, nil), nodeNieWolany(t))
	if sprawdzono || len(n) != 0 {
		t.Errorf("bez %s: sprawdzono=%v, naruszenia %v", PlikStyluWeb, sprawdzono, n)
	}
}

func TestPominieteKatalogi(t *testing.T) {
	p := wczytaj(t, map[string]string{
		"wewn/rdzen/r.go":            "package rdzen\n",
		"web/node_modules/x/x.go":    "package x\nimport \"os\"\n",
		"wewn/rdzen/testdata/zly.go": "to nie jest Go",
		"_go-portable/src/x.go":      "to nie jest Go",
		".git/x.go":                  "to nie jest Go",
	})
	if len(p.Pakiety) != 1 || p.Pakiety[0].Katalog != "wewn/rdzen" {
		t.Errorf("pakiety: %+v", p.Pakiety)
	}
}

func TestPoliczLinie(t *testing.T) {
	przypadki := map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\r\nb\r\n": 2}
	for tresc, oczekiwane := range przypadki {
		if n := policzLinie([]byte(tresc)); n != oczekiwane {
			t.Errorf("policzLinie(%q) = %d, oczekiwane %d", tresc, n, oczekiwane)
		}
	}
}
