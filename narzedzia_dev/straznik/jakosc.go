package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

// sprawdzJakosc: rozmiar pliku, długość / złożoność / argumenty funkcji, wypisywanie poza cmd, ciche błędy.
func sprawdzJakosc(p *projekt) []Naruszenie {
	var wynik []Naruszenie
	for _, pk := range p.Pakiety {
		for _, f := range pk.Pliki {
			if f.Linie > PlikMaxLinii {
				wynik = append(wynik, Naruszenie{Regula: "jakosc.rozmiar_pliku", Plik: f.Sciezka,
					Opis: fmt.Sprintf("plik ma %d linii (limit %d): podziel według tematu", f.Linie, PlikMaxLinii)})
			}
			if !f.Test {
				wynik = append(wynik, p.sprawdzFunkcje(f)...)
			}
			if !wDrzewie(pk.Katalog, KatalogCmd) {
				wynik = append(wynik, p.sprawdzWypisywanie(f)...)
			}
			wynik = append(wynik, p.sprawdzCicheBledy(f)...)
		}
	}
	return wynik
}

// nazwaFunkcji: "Funkcja" albo "Typ.Metoda".
func nazwaFunkcji(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	typ := fn.Recv.List[0].Type
	if gw, ok := typ.(*ast.StarExpr); ok {
		typ = gw.X
	}
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name + "." + fn.Name.Name
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name + "." + fn.Name.Name
		}
	}
	return fn.Name.Name
}

func (p *projekt) sprawdzFunkcje(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	for _, d := range f.Drzewo.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		nazwa := nazwaFunkcji(fn)
		nowe := func(regula, opis string) {
			wynik = append(wynik, Naruszenie{Regula: regula, Plik: f.Sciezka, Linia: p.linia(fn.Pos()), Nazwa: nazwa, Opis: opis})
		}
		if linie := p.linia(fn.End()) - p.linia(fn.Pos()) + 1; linie > FunkcjaMaxLinii {
			nowe("jakosc.dlugosc_funkcji", fmt.Sprintf("funkcja %s ma %d linii (limit %d)", nazwa, linie, FunkcjaMaxLinii))
		}
		if z := zlozonosc(fn); z > ZlozonoscMax {
			nowe("jakosc.zlozonosc", fmt.Sprintf("funkcja %s ma złożoność %d (limit %d)", nazwa, z, ZlozonoscMax))
		}
		if a := liczbaArgumentow(fn.Type.Params); a > ArgumentyMax {
			nowe("jakosc.argumenty", fmt.Sprintf("funkcja %s ma %d argumentów (limit %d): zbierz je w strukturę", nazwa, a, ArgumentyMax))
		}
	}
	return wynik
}

// zlozonosc liczy złożoność cyklomatyczną jak gocyclo (funkcje anonimowe wliczone do otaczającej).
func zlozonosc(fn *ast.FuncDecl) int {
	z := 1
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch w := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			z++
		case *ast.CaseClause:
			if w.List != nil {
				z++
			}
		case *ast.CommClause:
			if w.Comm != nil {
				z++
			}
		case *ast.BinaryExpr:
			if w.Op == token.LAND || w.Op == token.LOR {
				z++
			}
		}
		return true
	})
	return z
}

func liczbaArgumentow(lista *ast.FieldList) int {
	if lista == nil {
		return 0
	}
	n := 0
	for _, pole := range lista.List {
		n += max(1, len(pole.Names))
	}
	return n
}

// zakazaneWypisywanie: funkcje pakietów, które piszą prosto na konsolę (komunikaty idą przez dziennik).
var zakazaneWypisywanie = map[string][]string{
	"fmt": {"Print", "Printf", "Println"},
	"log": {"Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln", "Panic", "Panicf", "Panicln"},
}

func (p *projekt) sprawdzWypisywanie(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	nazwy := map[string]string{}
	for pakietStd := range zakazaneWypisywanie {
		if n := nazwaImportu(f.Drzewo, pakietStd); n != "" {
			nazwy[n] = pakietStd
		}
	}
	ast.Inspect(f.Drzewo, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if opis := opisWypisywania(call, nazwy); opis != "" {
			wynik = append(wynik, Naruszenie{Regula: "jakosc.wypisywanie", Plik: f.Sciezka, Linia: p.linia(call.Pos()),
				Opis: opis + " poza cmd/: komunikaty przez dziennik"})
		}
		return true
	})
	return wynik
}

func opisWypisywania(call *ast.CallExpr, nazwy map[string]string) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if fun.Name == "print" || fun.Name == "println" {
			return fun.Name + "()"
		}
	case *ast.SelectorExpr:
		x, ok := fun.X.(*ast.Ident)
		if !ok {
			return ""
		}
		pakietStd, ok := nazwy[x.Name]
		if !ok {
			return ""
		}
		for _, z := range zakazaneWypisywanie[pakietStd] {
			if fun.Sel.Name == z {
				return pakietStd + "." + z + "()"
			}
		}
	}
	return ""
}

// sprawdzCicheBledy: `_ = f()` i pusty `if x != nil {}` bez komentarza „// celowo: <powód>”.
func (p *projekt) sprawdzCicheBledy(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	ast.Inspect(f.Drzewo, func(n ast.Node) bool {
		switch w := n.(type) {
		case *ast.AssignStmt:
			if czyOdrzucenieWyniku(w) {
				l := p.linia(w.Pos())
				if !p.komentarzCelowo(f.Drzewo, l-1, l) {
					wynik = append(wynik, Naruszenie{Regula: "jakosc.zignorowany_blad", Plik: f.Sciezka, Linia: l,
						Opis: "wynik wywołania odrzucony przez `_ =` bez komentarza „// celowo: <powód>”"})
				}
			}
		case *ast.IfStmt:
			if czyPustyIfNil(w) && !p.komentarzCelowo(f.Drzewo, p.linia(w.Pos()), p.linia(w.Body.End())) {
				wynik = append(wynik, Naruszenie{Regula: "jakosc.pusty_blok_bledu", Plik: f.Sciezka, Linia: p.linia(w.Pos()),
					Opis: "pusty blok `if … != nil {}` (błąd połknięty) bez komentarza „// celowo: <powód>”"})
			}
		}
		return true
	})
	return wynik
}

func czyOdrzucenieWyniku(a *ast.AssignStmt) bool {
	if a.Tok != token.ASSIGN || len(a.Rhs) != 1 {
		return false
	}
	if _, ok := a.Rhs[0].(*ast.CallExpr); !ok {
		return false
	}
	for _, l := range a.Lhs {
		if id, ok := l.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	return true
}

func czyPustyIfNil(w *ast.IfStmt) bool {
	if len(w.Body.List) != 0 {
		return false
	}
	b, ok := w.Cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	return czyNil(b.X) || czyNil(b.Y)
}

func czyNil(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}
