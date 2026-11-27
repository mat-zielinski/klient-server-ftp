package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// zbudujProjekt zapisuje przykładowe źródła w t.TempDir() (moduł "skrytka") i zwraca korzeń.
func zbudujProjekt(t *testing.T, pliki map[string]string) string {
	t.Helper()
	korzen := t.TempDir()
	wszystkie := map[string]string{"go.mod": "module skrytka\n\ngo 1.27\n"}
	for k, v := range pliki {
		wszystkie[k] = v
	}
	for sciezka, tresc := range wszystkie {
		pelna := filepath.Join(korzen, filepath.FromSlash(sciezka))
		if err := os.MkdirAll(filepath.Dir(pelna), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pelna, []byte(tresc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return korzen
}

func wczytaj(t *testing.T, pliki map[string]string) *projekt {
	t.Helper()
	p, err := wczytajProjekt(zbudujProjekt(t, pliki))
	if err != nil {
		t.Fatalf("wczytajProjekt: %v", err)
	}
	return p
}

// reguly zwraca posortowane „reguła@plik” naruszeń (porównanie bez numerów linii i opisów).
func reguly(n []Naruszenie) []string {
	wynik := make([]string, 0, len(n))
	for _, x := range n {
		wynik = append(wynik, x.Regula+"@"+x.Plik)
	}
	sort.Strings(wynik)
	return wynik
}

func porownaj(t *testing.T, jest []Naruszenie, oczekiwane ...string) {
	t.Helper()
	sort.Strings(oczekiwane)
	if strings.Join(reguly(jest), "\n") != strings.Join(oczekiwane, "\n") {
		var opisy []string
		for _, n := range jest {
			opisy = append(opisy, n.Tekst())
		}
		t.Errorf("naruszenia:\n  %s\noczekiwane:\n  %s", strings.Join(opisy, "\n  "), strings.Join(oczekiwane, "\n  "))
	}
}

// funkcjaZLiniami zwraca źródło funkcji o podanej liczbie linii (razem z linią nagłówka i klamrą).
func funkcjaZLiniami(nazwa string, linie int) string {
	var b strings.Builder
	b.WriteString("func " + nazwa + "() int {\n\tx := 0\n")
	for i := 0; i < linie-4; i++ {
		b.WriteString("\tx++\n")
	}
	b.WriteString("\treturn x\n}\n")
	return b.String()
}

// funkcje zwraca deklaracje funkcji pliku w kolejności źródła.
func funkcje(f *plikGo) []*ast.FuncDecl {
	var wynik []*ast.FuncDecl
	for _, d := range f.Drzewo.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			wynik = append(wynik, fn)
		}
	}
	return wynik
}

func znajdzFunkcje(t *testing.T, p *projekt, nazwa string) *ast.FuncDecl {
	t.Helper()
	for _, pk := range p.Pakiety {
		for _, f := range pk.Pliki {
			for _, fn := range funkcje(f) {
				if fn.Name.Name == nazwa {
					return fn
				}
			}
		}
	}
	t.Fatalf("brak funkcji %s", nazwa)
	return nil
}
