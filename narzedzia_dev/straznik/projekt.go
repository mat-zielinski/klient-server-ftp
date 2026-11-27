package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// plikGo to jeden sparsowany plik źródłowy.
type plikGo struct {
	Sciezka string // względem korzenia projektu, z ukośnikami "/"
	Drzewo  *ast.File
	Test    bool
	Linie   int
}

// pakiet to katalog z plikami .go (pliki testów razem z resztą).
type pakiet struct {
	Katalog string // względem korzenia, z ukośnikami; "." = korzeń
	Pliki   []*plikGo
}

// projekt to cały moduł Go wczytany do sprawdzenia.
type projekt struct {
	Korzen  string
	Modul   string
	Fset    *token.FileSet
	Pakiety []*pakiet
}

// Naruszenie to jedno zgłoszenie strażnika.
type Naruszenie struct {
	Regula string
	Plik   string
	Linia  int
	Nazwa  string // np. nazwa funkcji; dopasowanie wyjątków
	Opis   string
}

// Tekst zwraca naruszenie w formie jednej linii (plik:linia: [reguła] opis).
func (n Naruszenie) Tekst() string {
	miejsce := n.Plik
	if n.Linia > 0 {
		miejsce += ":" + strconv.Itoa(n.Linia)
	}
	return fmt.Sprintf("%s: [%s] %s", miejsce, n.Regula, n.Opis)
}

var wzorCelowo = regexp.MustCompile(`//\s*celowo:\s*\S`)

// czyPominacKatalog: katalogi, których narzędzia Go i strażnik nie sprawdzają.
func czyPominacKatalog(nazwa string) bool {
	if strings.HasPrefix(nazwa, ".") || strings.HasPrefix(nazwa, "_") {
		return true
	}
	for _, p := range KatalogiPomijane {
		if nazwa == p {
			return true
		}
	}
	return false
}

// czytajModul zwraca nazwę modułu z go.mod.
func czytajModul(korzen string) (string, error) {
	f, err := os.Open(filepath.Join(korzen, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("brak go.mod w %s: %w", korzen, err)
	}
	defer f.Close()
	skaner := bufio.NewScanner(f)
	for skaner.Scan() {
		linia := strings.TrimSpace(skaner.Text())
		if strings.HasPrefix(linia, "module ") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(linia, "module ")), `"`), nil
		}
	}
	return "", errors.New("go.mod bez linii module")
}

// wczytajProjekt parsuje wszystkie pliki .go modułu (bez katalogów pomijanych).
func wczytajProjekt(korzen string) (*projekt, error) {
	modul, err := czytajModul(korzen)
	if err != nil {
		return nil, err
	}
	p := &projekt{Korzen: korzen, Modul: modul, Fset: token.NewFileSet()}
	wgKatalogu := map[string]*pakiet{}
	err = filepath.WalkDir(korzen, func(sciezka string, d fs.DirEntry, blad error) error {
		if blad != nil {
			return blad
		}
		if d.IsDir() {
			if sciezka != korzen && czyPominacKatalog(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		return p.dodajPlik(sciezka, wgKatalogu)
	})
	if err != nil {
		return nil, err
	}
	for _, pk := range wgKatalogu {
		p.Pakiety = append(p.Pakiety, pk)
	}
	sort.Slice(p.Pakiety, func(i, j int) bool { return p.Pakiety[i].Katalog < p.Pakiety[j].Katalog })
	return p, nil
}

func (p *projekt) dodajPlik(sciezka string, wgKatalogu map[string]*pakiet) error {
	tresc, err := os.ReadFile(sciezka)
	if err != nil {
		return err
	}
	drzewo, err := parser.ParseFile(p.Fset, sciezka, tresc, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("błąd składni: %w", err)
	}
	wzgledna, err := filepath.Rel(p.Korzen, sciezka)
	if err != nil {
		return err
	}
	wzgledna = filepath.ToSlash(wzgledna)
	katalog := filepath.ToSlash(filepath.Dir(wzgledna))
	pk := wgKatalogu[katalog]
	if pk == nil {
		pk = &pakiet{Katalog: katalog}
		wgKatalogu[katalog] = pk
	}
	pk.Pliki = append(pk.Pliki, &plikGo{
		Sciezka: wzgledna,
		Drzewo:  drzewo,
		Test:    strings.HasSuffix(wzgledna, "_test.go"),
		Linie:   policzLinie(tresc),
	})
	return nil
}

// policzLinie liczy linie jak edytor (ostatnia linia bez znaku nowej linii też się liczy).
func policzLinie(tresc []byte) int {
	if len(tresc) == 0 {
		return 0
	}
	n := strings.Count(string(tresc), "\n")
	if tresc[len(tresc)-1] != '\n' {
		n++
	}
	return n
}

func (p *projekt) linia(poz token.Pos) int {
	return p.Fset.Position(poz).Line
}

// nazwaImportu zwraca nazwę, pod jaką plik widzi pakiet o danej ścieżce ("" = brak importu).
func nazwaImportu(f *ast.File, sciezka string) string {
	for _, imp := range f.Imports {
		s, err := strconv.Unquote(imp.Path.Value)
		if err != nil || s != sciezka {
			continue
		}
		if imp.Name != nil {
			if imp.Name.Name == "_" || imp.Name.Name == "." {
				return ""
			}
			return imp.Name.Name
		}
		return s[strings.LastIndex(s, "/")+1:]
	}
	return ""
}

// wywolanie sprawdza, czy wyrażenie to wywołanie pakiet.Funkcja; zwraca nazwę funkcji.
func wywolanie(e ast.Expr, nazwaPakietu string) (string, bool) {
	call, ok := e.(*ast.CallExpr)
	if !ok || nazwaPakietu == "" {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != nazwaPakietu {
		return "", false
	}
	return sel.Sel.Name, true
}

// komentarzCelowo: czy w liniach [od, do] pliku jest komentarz „// celowo: <powód>”.
func (p *projekt) komentarzCelowo(f *ast.File, od, do int) bool {
	for _, grupa := range f.Comments {
		for _, k := range grupa.List {
			l := p.linia(k.Pos())
			if l >= od && l <= do && wzorCelowo.MatchString(k.Text) {
				return true
			}
		}
	}
	return false
}

// wDrzewie: czy katalog leży w drzewie podanego prefiksu ("cmd" pasuje do "cmd/x").
func wDrzewie(katalog, prefiks string) bool {
	return katalog == prefiks || strings.HasPrefix(katalog, prefiks+"/")
}
