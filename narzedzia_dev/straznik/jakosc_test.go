package main

import (
	"strings"
	"testing"
)

func TestJakoscRozmiarPliku(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		linie      int
		oczekiwane []string
	}{
		{"na granicy", PlikMaxLinii, nil},
		{"za duży", PlikMaxLinii + 1, []string{"jakosc.rozmiar_pliku@wewn/rdzen/r.go"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			tresc := "package rdzen\n" + strings.Repeat("// linia\n", p.linie-1)
			porownaj(t, sprawdzJakosc(wczytaj(t, map[string]string{"wewn/rdzen/r.go": tresc})), p.oczekiwane...)
		})
	}
}

func TestJakoscDlugoscFunkcji(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		plik       string
		linie      int
		oczekiwane []string
	}{
		{"na granicy", "wewn/rdzen/r.go", FunkcjaMaxLinii, nil},
		{"za długa", "wewn/rdzen/r.go", FunkcjaMaxLinii + 1, []string{"jakosc.dlugosc_funkcji@wewn/rdzen/r.go"}},
		{"w teście wolno", "wewn/rdzen/r_test.go", FunkcjaMaxLinii + 20, nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			tresc := "package rdzen\n" + funkcjaZLiniami("F", p.linie)
			porownaj(t, sprawdzJakosc(wczytaj(t, map[string]string{p.plik: tresc})), p.oczekiwane...)
		})
	}
}

func TestZlozonosc(t *testing.T) {
	przypadki := []struct {
		nazwa string
		cialo string
		wynik int
	}{
		{"prosta", "return", 1},
		{"if i for", "if a { }\nfor i := 0; i < 1; i++ { }", 3},
		{"range i &&", "for range []int{} { if a && b || c { } }", 5},
		{"switch bez default", "switch { case a: \ncase b:\ndefault: }", 3},
		{"select", "var c chan int\nselect { case <-c:\ndefault: }", 2},
		{"funkcja anonimowa liczona", "f := func() { if a { } }\nf()", 2},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			pr := wczytaj(t, map[string]string{"wewn/rdzen/r.go": "package rdzen\nvar a, b, c bool\nfunc F() {\n" + p.cialo + "\n}\n"})
			fn := znajdzFunkcje(t, pr, "F")
			if z := zlozonosc(fn); z != p.wynik {
				t.Errorf("złożoność = %d, oczekiwane %d", z, p.wynik)
			}
		})
	}
}

func TestJakoscZlozonoscIArgumenty(t *testing.T) {
	warunki := strings.Repeat("if a { }\n", ZlozonoscMax)
	przypadki := []struct {
		nazwa      string
		tresc      string
		oczekiwane []string
	}{
		{"złożoność na granicy", "func F() {\n" + strings.Repeat("if a { }\n", ZlozonoscMax-1) + "}\n", nil},
		{"złożoność za duża", "func F() {\n" + warunki + "}\n", []string{"jakosc.zlozonosc@wewn/rdzen/r.go"}},
		{"8 argumentów", "func F(a, b, c, d, e, f, g int, h string) {}\n", nil},
		{"9 argumentów", "func F(a, b, c, d, e, f, g, h int, i string) {}\n", []string{"jakosc.argumenty@wewn/rdzen/r.go"}},
		{"metoda: odbiorca się nie liczy", "type T struct{}\nfunc (t *T) F(a, b, c, d, e, f, g, h int) {}\n", nil},
		{"argumenty bez nazw", "func F(int, int, int, int, int, int, int, int, int) {}\n", []string{"jakosc.argumenty@wewn/rdzen/r.go"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			tresc := "package rdzen\nvar a bool\n" + p.tresc
			porownaj(t, sprawdzJakosc(wczytaj(t, map[string]string{"wewn/rdzen/r.go": tresc})), p.oczekiwane...)
		})
	}
}

func TestJakoscWypisywanie(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		plik       string
		tresc      string
		oczekiwane []string
	}{
		{"fmt.Println", "wewn/http/h.go", "import \"fmt\"\nfunc F() { fmt.Println(1) }", []string{"jakosc.wypisywanie@wewn/http/h.go"}},
		{"fmt.Printf z aliasem", "wewn/http/h.go", "import f \"fmt\"\nfunc F() { f.Printf(\"x\") }", []string{"jakosc.wypisywanie@wewn/http/h.go"}},
		{"println", "wewn/http/h.go", "func F() { println(1) }", []string{"jakosc.wypisywanie@wewn/http/h.go"}},
		{"log.Printf", "wewn/http/h.go", "import \"log\"\nfunc F() { log.Printf(\"x\") }", []string{"jakosc.wypisywanie@wewn/http/h.go"}},
		{"log.Fatal", "wewn/http/h.go", "import \"log\"\nfunc F() { log.Fatal(1) }", []string{"jakosc.wypisywanie@wewn/http/h.go"}},
		{"w teście też nie", "wewn/http/h_test.go", "import \"fmt\"\nfunc F() { fmt.Print(1) }", []string{"jakosc.wypisywanie@wewn/http/h_test.go"}},
		{"fmt.Sprintf wolno", "wewn/http/h.go", "import \"fmt\"\nvar _ = fmt.Sprintf(\"x\")", nil},
		{"fmt.Fprintf wolno", "wewn/http/h.go", "import (\n\"fmt\"\n\"io\"\n)\nfunc F(w io.Writer) { fmt.Fprintf(w, \"x\") }", nil},
		{"w cmd wolno", "cmd/skrytka/main.go", "import \"fmt\"\nfunc F() { fmt.Println(1) }", nil},
		{"zmienna o nazwie fmt bez importu", "wewn/http/h.go", "type t struct{}\nfunc (t) Println() {}\nfunc F() { var fmt t; fmt.Println() }", nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			naglowek := "package http\n"
			if strings.HasPrefix(p.plik, "cmd/") {
				naglowek = "package main\n"
			}
			porownaj(t, sprawdzJakosc(wczytaj(t, map[string]string{p.plik: naglowek + p.tresc + "\n"})), p.oczekiwane...)
		})
	}
}

func TestJakoscCicheBledy(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		tresc      string
		oczekiwane []string
	}{
		{"_ = f()", "func F() { _ = g() }", []string{"jakosc.zignorowany_blad@wewn/http/h.go"}},
		{"_, _ = f()", "func F() { _, _ = g2() }", []string{"jakosc.zignorowany_blad@wewn/http/h.go"}},
		{"_ = f() z powodem w linii", "func F() { _ = g() // celowo: zamykanie po odczycie\n}", nil},
		{"_ = f() z powodem nad linią", "func F() {\n// celowo: zamykanie po odczycie\n_ = g()\n}", nil},
		{"celowo bez powodu", "func F() { _ = g() // celowo:\n}", []string{"jakosc.zignorowany_blad@wewn/http/h.go"}},
		{"_ = zmienna wolno", "func F() { x := 1; _ = x }", nil},
		{"v, _ := f() wolno", "func F() { v, _ := g2(); _ = v }", nil},
		{"pusty if err", "func F() { if err := g(); err != nil {\n} }", []string{"jakosc.pusty_blok_bledu@wewn/http/h.go"}},
		{"pusty if nil odwrotnie", "func F() { err := g(); if nil != err {} }", []string{"jakosc.pusty_blok_bledu@wewn/http/h.go"}},
		{"pusty if z powodem", "func F() { if err := g(); err != nil {\n// celowo: brak pliku to stan poprawny\n} }", nil},
		{"if z treścią", "func F() error { if err := g(); err != nil { return err }; return nil }", nil},
		{"pusty if bez nil", "func F() { if a {} }", nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			tresc := "package http\nvar a bool\nfunc g() error { return nil }\nfunc g2() (int, error) { return 0, nil }\n" + p.tresc + "\n"
			porownaj(t, sprawdzJakosc(wczytaj(t, map[string]string{"wewn/http/h.go": tresc})), p.oczekiwane...)
		})
	}
}

func TestNazwaFunkcji(t *testing.T) {
	pr := wczytaj(t, map[string]string{"wewn/rdzen/r.go": "package rdzen\ntype T struct{}\ntype G[X any] struct{}\nfunc F() {}\nfunc (T) M() {}\nfunc (*T) N() {}\nfunc (G[X]) O() {}\n"})
	oczekiwane := []string{"F", "T.M", "T.N", "G.O"}
	var jest []string
	for _, f := range pr.Pakiety[0].Pliki {
		for _, fn := range funkcje(f) {
			jest = append(jest, nazwaFunkcji(fn))
		}
	}
	if strings.Join(jest, ",") != strings.Join(oczekiwane, ",") {
		t.Errorf("nazwy = %v, oczekiwane %v", jest, oczekiwane)
	}
}
