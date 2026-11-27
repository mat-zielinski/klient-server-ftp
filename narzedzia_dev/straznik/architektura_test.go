package main

import "testing"

func TestWarstwaKatalogu(t *testing.T) {
	przypadki := []struct {
		katalog   string
		komponent string
		poziom    int
		rodzaj    rodzajPakietu
	}{
		{"wewn/konfiguracja", "konfiguracja", 0, pakietWarstwy},
		{"wewn/rdzen", "rdzen", 1, pakietWarstwy},
		{"wewn/magazyn", "magazyn", 2, pakietWarstwy},
		{"wewn/uslugi/konta", "uslugi/konta", 3, pakietWarstwy},
		{"wewn/uslugi/konta/wew", "uslugi/konta", 3, pakietWarstwy},
		{"wewn/http", "http", 5, pakietWarstwy},
		{"wewn/http/api", "http", 5, pakietWarstwy},
		{"wewn/ftp", "ftp", 5, pakietWarstwy},
		{"wewn/siec", "siec", 4, pakietWarstwy},
		{"wewn/aplikacja", "aplikacja", 6, pakietWarstwy},
		{"cmd/skrytka", "cmd/skrytka", 7, pakietWarstwy},
		{"web", "web", 0, pakietWarstwy},
		{"wewn/inne", "", 0, pakietNieznany},
		{"wewn", "", 0, pakietNieznany},
		{"narzedzia_dev/straznik", "", 0, pakietPominiety},
	}
	for _, p := range przypadki {
		w, r := warstwaKatalogu(p.katalog)
		if r != p.rodzaj || w.Komponent != p.komponent || w.Poziom != p.poziom {
			t.Errorf("%s: jest (%q, %d, %d), oczekiwane (%q, %d, %d)", p.katalog, w.Komponent, w.Poziom, r, p.komponent, p.poziom, p.rodzaj)
		}
	}
}

func TestArchitekturaImporty(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		pliki      map[string]string
		oczekiwane []string
	}{
		{"w dół wolno", map[string]string{
			"wewn/konfiguracja/k.go": "package konfiguracja\n",
			"wewn/rdzen/r.go":        "package rdzen\nimport _ \"skrytka/wewn/konfiguracja\"\n",
			"wewn/magazyn/m.go":      "package magazyn\nimport (\n_ \"skrytka/wewn/rdzen\"\n_ \"skrytka/wewn/konfiguracja\"\n)\n",
			"wewn/uslugi/konta/k.go": "package konta\nimport _ \"skrytka/wewn/magazyn\"\n",
			"wewn/http/h.go":         "package http\nimport _ \"skrytka/wewn/uslugi/konta\"\n",
			"wewn/aplikacja/a.go":    "package aplikacja\nimport _ \"skrytka/wewn/http\"\n",
			"cmd/skrytka/main.go":    "package main\nimport _ \"skrytka/wewn/aplikacja\"\nfunc main() {}\n",
		}, nil},
		{"w górę nie wolno", map[string]string{
			"wewn/rdzen/r.go":   "package rdzen\nimport _ \"skrytka/wewn/magazyn\"\n",
			"wewn/magazyn/m.go": "package magazyn\n",
		}, []string{"architektura.warstwa@wewn/rdzen/r.go"}},
		{"ta sama warstwa nie wolno", map[string]string{
			"wewn/http/h.go": "package http\nimport _ \"skrytka/wewn/ftp\"\n",
			"wewn/ftp/f.go":  "package ftp\n",
		}, []string{"architektura.warstwa@wewn/http/h.go"}},
		{"http i ftp nad siecią wolno", map[string]string{
			"wewn/http/h.go":           "package http\nimport _ \"skrytka/wewn/siec\"\n",
			"wewn/ftp/f.go":            "package ftp\nimport _ \"skrytka/wewn/siec\"\n",
			"wewn/siec/s.go":           "package siec\nimport _ \"skrytka/wewn/uslugi/monitor\"\n",
			"wewn/uslugi/monitor/m.go": "package monitor\n",
		}, nil},
		{"sieć nie sięga do http", map[string]string{
			"wewn/siec/s.go": "package siec\nimport _ \"skrytka/wewn/http\"\n",
			"wewn/http/h.go": "package http\n",
		}, []string{"architektura.warstwa@wewn/siec/s.go"}},
		{"usługa nie sięga do sieci", map[string]string{
			"wewn/uslugi/monitor/m.go": "package monitor\nimport _ \"skrytka/wewn/siec\"\n",
			"wewn/siec/s.go":           "package siec\n",
		}, []string{"architektura.warstwa@wewn/uslugi/monitor/m.go"}},
		{"indeks może importować magazyn", map[string]string{
			"wewn/uslugi/indeks/i.go": "package indeks\nimport _ \"skrytka/wewn/magazyn\"\n",
			"wewn/magazyn/m.go":       "package magazyn\n",
		}, nil},
		{"usługi między sobą nie", map[string]string{
			"wewn/uslugi/konta/k.go":    "package konta\nimport _ \"skrytka/wewn/uslugi/dziennik\"\n",
			"wewn/uslugi/dziennik/d.go": "package dziennik\n",
		}, []string{"architektura.uslugi@wewn/uslugi/konta/k.go"}},
		{"podpakiet tego samego komponentu wolno", map[string]string{
			"wewn/http/api/a.go": "package api\nimport _ \"skrytka/wewn/http\"\n",
			"wewn/http/h.go":     "package http\n",
		}, nil},
		{"nieznany pakiet w wewn", map[string]string{
			"wewn/inne/i.go": "package inne\n",
		}, []string{"architektura.nieznany_pakiet@wewn/inne"}},
		{"import narzędzi deweloperskich", map[string]string{
			"wewn/http/h.go":       "package http\nimport _ \"skrytka/narzedzia_dev/x\"\n",
			"narzedzia_dev/x/x.go": "package x\n",
		}, []string{"architektura.warstwa@wewn/http/h.go"}},
		{"cmd z cmd", map[string]string{
			"cmd/a/main.go": "package main\nimport _ \"skrytka/cmd/b\"\n",
			"cmd/b/b.go":    "package b\n",
		}, []string{"architektura.warstwa@cmd/a/main.go"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			porownaj(t, sprawdzArchitekture(wczytaj(t, p.pliki)), p.oczekiwane...)
		})
	}
}

func TestArchitekturaRdzen(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		tresc      string
		oczekiwane []string
	}{
		{"czysta logika", "package rdzen\nimport (\n\"path\"\n\"strings\"\n\"time\"\n)\nfunc F(t time.Time) string { return path.Clean(strings.ToLower(t.String())) }\n", nil},
		{"os", "package rdzen\nimport \"os\"\nvar _ = os.Args\n", []string{"architektura.rdzen_import@wewn/rdzen/r.go"}},
		{"os/exec", "package rdzen\nimport \"os/exec\"\nvar _ = exec.Command\n", []string{"architektura.rdzen_import@wewn/rdzen/r.go"}},
		{"net/http", "package rdzen\nimport \"net/http\"\nvar _ = http.Get\n", []string{"architektura.rdzen_import@wewn/rdzen/r.go"}},
		{"net", "package rdzen\nimport \"net\"\nvar _ = net.Dial\n", []string{"architektura.rdzen_import@wewn/rdzen/r.go"}},
		{"time.Now", "package rdzen\nimport \"time\"\nfunc F() time.Time { return time.Now() }\n", []string{"architektura.rdzen_czas@wewn/rdzen/r.go"}},
		{"time.Now z aliasem", "package rdzen\nimport cz \"time\"\nfunc F() cz.Time { return cz.Now().UTC() }\n", []string{"architektura.rdzen_czas@wewn/rdzen/r.go"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			porownaj(t, sprawdzArchitekture(wczytaj(t, map[string]string{"wewn/rdzen/r.go": p.tresc})), p.oczekiwane...)
		})
	}
}

func TestArchitekturaRdzenTestyMogaUzywacDysku(t *testing.T) {
	p := wczytaj(t, map[string]string{
		"wewn/rdzen/r.go":      "package rdzen\n",
		"wewn/rdzen/r_test.go": "package rdzen\nimport (\n\"os\"\n\"time\"\n)\nvar _ = os.Args\nvar _ = time.Now()\n",
	})
	porownaj(t, sprawdzArchitekture(p))
}

func TestArchitekturaMagazynMozeOs(t *testing.T) {
	p := wczytaj(t, map[string]string{"wewn/magazyn/m.go": "package magazyn\nimport (\n\"os\"\n\"time\"\n)\nvar _ = os.Args\nvar _ = time.Now()\n"})
	porownaj(t, sprawdzArchitekture(p))
}
