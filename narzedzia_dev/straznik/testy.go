package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// sprawdzTesty: pakiety wewn/** z logiką mają testy; w testach bez time.Sleep i bez t.Skip bez powodu.
func sprawdzTesty(p *projekt) []Naruszenie {
	var wynik []Naruszenie
	for _, pk := range p.Pakiety {
		if wDrzewie(pk.Katalog, KatalogWewn) && maLogike(pk) && !maTesty(pk) {
			wynik = append(wynik, Naruszenie{Regula: "testy.brak_testow", Plik: pk.Katalog,
				Opis: fmt.Sprintf("pakiet %s ma logikę, a nie ma plików _test.go", pk.Katalog)})
		}
		for _, f := range pk.Pliki {
			if f.Test {
				wynik = append(wynik, p.sprawdzPlikTestow(f)...)
			}
		}
	}
	return wynik
}

// maLogike: któryś plik (poza testami i kontrakt.go) ma funkcję z ciałem.
func maLogike(pk *pakiet) bool {
	for _, f := range pk.Pliki {
		if f.Test || strings.HasSuffix(f.Sciezka, "/"+PlikKontraktu) {
			continue
		}
		for _, d := range f.Drzewo.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
				return true
			}
		}
	}
	return false
}

func maTesty(pk *pakiet) bool {
	for _, f := range pk.Pliki {
		if f.Test {
			return true
		}
	}
	return false
}

func (p *projekt) sprawdzPlikTestow(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	czas := nazwaImportu(f.Drzewo, "time")
	ast.Inspect(f.Drzewo, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if nazwa, ok := wywolanie(call, czas); ok && nazwa == "Sleep" {
			wynik = append(wynik, Naruszenie{Regula: "testy.sleep", Plik: f.Sciezka, Linia: p.linia(call.Pos()),
				Opis: "time.Sleep w teście: zegar wstrzykiwany albo czekanie na warunek (kanał, WaitGroup)"})
		}
		if opis := opisSkipa(call); opis != "" {
			wynik = append(wynik, Naruszenie{Regula: "testy.skip_bez_powodu", Plik: f.Sciezka, Linia: p.linia(call.Pos()), Opis: opis})
		}
		return true
	})
	return wynik
}

// opisSkipa: t.Skip / t.Skipf muszą mieć niepusty powód; t.SkipNow() nie ma go wcale.
func opisSkipa(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if _, ok := sel.X.(*ast.Ident); !ok {
		return ""
	}
	switch sel.Sel.Name {
	case "SkipNow":
		return "SkipNow() pomija test bez powodu: użyj t.Skip(\"<powód>\")"
	case "Skip", "Skipf":
		if len(call.Args) == 0 {
			return sel.Sel.Name + "() bez powodu: podaj powód pominięcia"
		}
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			tekst, err := strconv.Unquote(lit.Value)
			if err == nil && strings.TrimSpace(tekst) == "" {
				return sel.Sel.Name + "(\"\") z pustym powodem"
			}
		}
	}
	return ""
}
