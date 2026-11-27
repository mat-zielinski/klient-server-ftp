package main

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"
)

// warstwa opisuje miejsce pakietu w układzie warstw.
type warstwa struct {
	Komponent string // np. "rdzen", "uslugi/konta", "cmd/skrytka"
	Nazwa     string // klucz w Warstwy
	Poziom    int
}

type rodzajPakietu int

const (
	pakietWarstwy   rodzajPakietu = iota // sprawdzany
	pakietPominiety                      // poza warstwami (np. narzedzia_dev)
	pakietNieznany                       // w wewn/, ale nie w Warstwy
)

// warstwaKatalogu przypisuje katalog (względny, z "/") do warstwy.
func warstwaKatalogu(katalog string) (warstwa, rodzajPakietu) {
	czesci := strings.Split(katalog, "/")
	switch {
	case czesci[0] == "web":
		return warstwa{Komponent: "web", Nazwa: "web", Poziom: Warstwy["web"]}, pakietWarstwy
	case czesci[0] == KatalogCmd:
		return warstwa{Komponent: katalog, Nazwa: KatalogCmd, Poziom: Warstwy[KatalogCmd]}, pakietWarstwy
	case czesci[0] != KatalogWewn:
		return warstwa{}, pakietPominiety
	case len(czesci) < 2:
		return warstwa{}, pakietNieznany
	}
	nazwa := czesci[1]
	poziom, ok := Warstwy[nazwa]
	if !ok {
		return warstwa{Nazwa: nazwa}, pakietNieznany
	}
	komponent := nazwa
	if nazwa == "uslugi" && len(czesci) > 2 {
		komponent = "uslugi/" + czesci[2]
	}
	return warstwa{Komponent: komponent, Nazwa: nazwa, Poziom: poziom}, pakietWarstwy
}

// sprawdzArchitekture: importy tylko w dół, usługi bez importów między sobą, rdzeń bez dysku, sieci i zegara.
func sprawdzArchitekture(p *projekt) []Naruszenie {
	var wynik []Naruszenie
	for _, pk := range p.Pakiety {
		w, rodzaj := warstwaKatalogu(pk.Katalog)
		if rodzaj == pakietPominiety {
			continue
		}
		if rodzaj == pakietNieznany {
			wynik = append(wynik, Naruszenie{Regula: "architektura.nieznany_pakiet", Plik: pk.Katalog,
				Opis: fmt.Sprintf("pakiet %s nie należy do żadnej warstwy (dopisz go do Warstwy w narzedzia_dev/straznik/limity.go albo przenieś)", pk.Katalog)})
			continue
		}
		for _, f := range pk.Pliki {
			wynik = append(wynik, p.sprawdzImporty(pk, w, f)...)
			if w.Nazwa == "rdzen" && !f.Test {
				wynik = append(wynik, p.sprawdzRdzen(f)...)
			}
		}
	}
	return append(wynik, sprawdzSystem(p)...)
}

func (p *projekt) sprawdzImporty(pk *pakiet, w warstwa, f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	for _, imp := range f.Drzewo.Imports {
		sciezka, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !strings.HasPrefix(sciezka, p.Modul+"/") {
			continue
		}
		cel := strings.TrimPrefix(sciezka, p.Modul+"/")
		if opis := ocenImport(w, cel); opis != "" {
			wynik = append(wynik, Naruszenie{Regula: regulaImportu(w, cel), Plik: f.Sciezka,
				Linia: p.linia(imp.Pos()), Nazwa: cel, Opis: fmt.Sprintf("%s importuje %s: %s", pk.Katalog, cel, opis)})
		}
	}
	return wynik
}

// ocenImport zwraca opis naruszenia ("" = import dozwolony).
func ocenImport(z warstwa, cel string) string {
	wc, rodzaj := warstwaKatalogu(cel)
	switch {
	case rodzaj == pakietPominiety:
		return "import pakietu spoza warstw (narzędzia deweloperskie nie wchodzą do programu)"
	case rodzaj == pakietNieznany:
		return "import pakietu spoza warstw"
	case wc.Komponent == z.Komponent:
		return ""
	case wc.Nazwa == "uslugi" && z.Nazwa == "uslugi":
		return "pakiety uslugi/* nie importują się nawzajem (wspólne rzeczy do rdzenia albo magazynu, składanie w aplikacji)"
	case wc.Poziom >= z.Poziom:
		return fmt.Sprintf("import tylko w dół (warstwa %s nie może sięgać do warstwy %s)", z.Nazwa, wc.Nazwa)
	}
	return ""
}

func regulaImportu(z warstwa, cel string) string {
	wc, _ := warstwaKatalogu(cel)
	if wc.Nazwa == "uslugi" && z.Nazwa == "uslugi" {
		return "architektura.uslugi"
	}
	return "architektura.warstwa"
}

// sprawdzRdzen: rdzeń bez dysku, sieci, procesów i bez wywołania time.Now (czas jako argument).
func (p *projekt) sprawdzRdzen(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	for _, imp := range f.Drzewo.Imports {
		sciezka, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !zakazanyWRdzeniu(sciezka) {
			continue
		}
		wynik = append(wynik, Naruszenie{Regula: "architektura.rdzen_import", Plik: f.Sciezka, Linia: p.linia(imp.Pos()),
			Nazwa: sciezka, Opis: fmt.Sprintf("rdzeń importuje %q: rdzeń bez dysku, sieci i procesów (to robi magazyn / warstwy wyżej)", sciezka)})
	}
	czas := nazwaImportu(f.Drzewo, "time")
	ast.Inspect(f.Drzewo, func(n ast.Node) bool {
		if e, ok := n.(ast.Expr); ok {
			if nazwa, ok := wywolanie(e, czas); ok && nazwa == "Now" {
				wynik = append(wynik, Naruszenie{Regula: "architektura.rdzen_czas", Plik: f.Sciezka, Linia: p.linia(e.Pos()),
					Opis: "rdzeń woła time.Now(): czas podaje wywołujący (argument)"})
			}
		}
		return true
	})
	return wynik
}

func zakazanyWRdzeniu(sciezka string) bool {
	for _, z := range ZakazaneWRdzeniu {
		if (strings.HasSuffix(z, "/") && strings.HasPrefix(sciezka, z)) || sciezka == z {
			return true
		}
	}
	return false
}
