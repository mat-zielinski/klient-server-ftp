package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const kodBlokady = "package rdzen\n\nfunc CzyWolno(a bool) error {\n\tif !a {\n\t\treturn errBrak\n\t}\n\treturn nil\n}\n"

// projektMutacji: korzeń z jednym plikiem blokady (wewn/rdzen/b.go).
func projektMutacji(t *testing.T, tresc string) string {
	t.Helper()
	korzen := t.TempDir()
	katalog := filepath.Join(korzen, "wewn", "rdzen")
	if err := os.MkdirAll(katalog, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(katalog, "b.go"), []byte(tresc), 0o644); err != nil {
		t.Fatal(err)
	}
	return korzen
}

func wpisBlokady() Wpis {
	return Wpis{ID: "bramka", Opis: "bramka odmawia", Plik: "wewn/rdzen/b.go",
		Szukaj: "\tif !a {\n\t\treturn errBrak\n\t}\n", Zamien: "", Pakiet: "./wewn/rdzen"}
}

// udawaneTesty: testy są zielone, gdy plik ma blokadę; `zachowanie` decyduje, co dzieje się po mutacji.
func udawaneTesty(t *testing.T, korzen, zachowanie string) uruchamiaczTestow {
	return func(pakiet, wzorzec string) (wynikTestow, error) {
		tresc, err := os.ReadFile(filepath.Join(korzen, "wewn", "rdzen", "b.go"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(tresc), "errBrak") {
			return wynikTestow{Zielone: true, Kompiluje: true}, nil
		}
		switch zachowanie {
		case "czerwone":
			return wynikTestow{Zielone: false, Kompiluje: true}, nil
		case "zielone":
			return wynikTestow{Zielone: true, Kompiluje: true}, nil
		case "nie_buduje":
			return wynikTestow{Zielone: false, Kompiluje: false}, nil
		}
		return wynikTestow{}, errors.New("go nie wystartowało")
	}
}

func TestPrzebieg(t *testing.T) {
	przypadki := []struct {
		zachowanie string
		kod        int
		zawiera    string
	}{
		{"czerwone", KodOK, "OK       bramka"},
		{"zielone", KodZielone, "ZIELONE  bramka"},
		{"nie_buduje", KodBledu, "nie buduje"},
		{"blad", KodBledu, "go nie wystartowało"},
	}
	for _, p := range przypadki {
		t.Run(p.zachowanie, func(t *testing.T) {
			korzen := projektMutacji(t, kodBlokady)
			var wyjscie bytes.Buffer
			kod := przebieg(korzen, []Wpis{wpisBlokady()}, udawaneTesty(t, korzen, p.zachowanie), &straz{}, &wyjscie)
			if kod != p.kod || !strings.Contains(wyjscie.String(), p.zawiera) {
				t.Errorf("kod %d (oczekiwany %d), wyjście:\n%s", kod, p.kod, wyjscie.String())
			}
			sprawdzPrzywrocony(t, korzen, kodBlokady)
		})
	}
}

func sprawdzPrzywrocony(t *testing.T, korzen, oczekiwany string) {
	t.Helper()
	sciezka := filepath.Join(korzen, "wewn", "rdzen", "b.go")
	tresc, err := os.ReadFile(sciezka)
	if err != nil || string(tresc) != oczekiwany {
		t.Errorf("plik nie przywrócony (błąd %v):\n%s", err, tresc)
	}
	for _, rozszerzenie := range []string{RozszerzenieKopii, RozszerzenieTymczasowe} {
		if _, err := os.Stat(sciezka + rozszerzenie); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("zostało %s", rozszerzenie)
		}
	}
}

func TestPrzywrocenieNawetPrzyPanice(t *testing.T) {
	korzen := projektMutacji(t, kodBlokady)
	uruchom := func(string, string) (wynikTestow, error) { panic("awaria w trakcie testów") }
	func() {
		defer func() {
			if recover() == nil {
				t.Error("oczekiwano paniki")
			}
		}()
		_, _ = sprawdzMutacje(korzen, wpisBlokady(), uruchom, &straz{}) // celowo: panika przerywa wywołanie
	}()
	sprawdzPrzywrocony(t, korzen, kodBlokady)
}

func TestPrzywrocenieZKopiiPoPrzerwaniu(t *testing.T) {
	korzen := projektMutacji(t, "zmutowany po twardym przerwaniu")
	kopia := filepath.Join(korzen, "wewn", "rdzen", "b.go"+RozszerzenieKopii)
	if err := os.WriteFile(kopia, []byte(kodBlokady), 0o644); err != nil {
		t.Fatal(err)
	}
	var wyjscie bytes.Buffer
	kod := przebieg(korzen, []Wpis{wpisBlokady()}, udawaneTesty(t, korzen, "czerwone"), &straz{}, &wyjscie)
	if kod != KodOK || !strings.Contains(wyjscie.String(), "przywrócono") {
		t.Errorf("kod %d, wyjście:\n%s", kod, wyjscie.String())
	}
	sprawdzPrzywrocony(t, korzen, kodBlokady)
}

func TestStrazPrzywracaPoSygnale(t *testing.T) {
	korzen := projektMutacji(t, "zmutowany")
	s := &straz{}
	s.ustaw(filepath.Join(korzen, "wewn", "rdzen", "b.go"), []byte(kodBlokady), []byte("zmutowany"))
	if err := s.przywroc(); err != nil {
		t.Fatal(err)
	}
	sprawdzPrzywrocony(t, korzen, kodBlokady)
	if err := s.przywroc(); err != nil {
		t.Errorf("drugie przywrócenie: %v", err)
	}
}

func TestPrzebiegListaNieaktualna(t *testing.T) {
	przypadki := []struct {
		nazwa string
		wpis  func(w *Wpis)
	}{
		{"brak fragmentu", func(w *Wpis) { w.Szukaj = "nie ma tego" }},
		{"brak pliku", func(w *Wpis) { w.Plik = "wewn/rdzen/inny.go" }},
		{"brak pakietu", func(w *Wpis) { w.Pakiet = "./wewn/brak" }},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			korzen := projektMutacji(t, kodBlokady)
			w := wpisBlokady()
			p.wpis(&w)
			var wyjscie bytes.Buffer
			if kod := przebieg(korzen, []Wpis{w}, udawaneTesty(t, korzen, "czerwone"), &straz{}, &wyjscie); kod != KodBledu {
				t.Errorf("kod %d, wyjście:\n%s", kod, wyjscie.String())
			}
		})
	}
}

func TestPrzebiegTestyCzerwoneBezMutacji(t *testing.T) {
	korzen := projektMutacji(t, kodBlokady)
	zawszeCzerwone := func(string, string) (wynikTestow, error) { return wynikTestow{Kompiluje: true}, nil }
	var wyjscie bytes.Buffer
	if kod := przebieg(korzen, []Wpis{wpisBlokady()}, zawszeCzerwone, &straz{}, &wyjscie); kod != KodBledu ||
		!strings.Contains(wyjscie.String(), "BEZ mutacji") {
		t.Errorf("kod %d, wyjście:\n%s", kod, wyjscie.String())
	}
}

func TestPrzebiegWpisDoUzupelnienia(t *testing.T) {
	var wyjscie bytes.Buffer
	w := Wpis{ID: "x", Opis: "blokada", DoUzupelnienia: "czeka na kod rdzenia"}
	nieWolac := func(string, string) (wynikTestow, error) {
		t.Error("testy uruchomione dla wpisu do uzupełnienia")
		return wynikTestow{}, nil
	}
	if kod := przebieg(t.TempDir(), []Wpis{w}, nieWolac, &straz{}, &wyjscie); kod != KodOK ||
		!strings.Contains(wyjscie.String(), "do uzupełnienia") {
		t.Errorf("kod %d, wyjście:\n%s", kod, wyjscie.String())
	}
}

func TestMutuj(t *testing.T) {
	przypadki := []struct {
		nazwa, tekst, szukaj, zamien, wynik string
		blad                                bool
	}{
		{"zwykła", "a\nb\nc\n", "b\n", "x\n", "a\nx\nc\n", false},
		{"CRLF w pliku", "a\r\nb\r\nc\r\n", "a\nb\n", "z\n", "z\r\nc\r\n", false},
		{"brak", "a\n", "b", "x", "", true},
		{"dwa razy", "b b", "b", "x", "", true},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			wynik, err := mutuj(p.tekst, p.szukaj, p.zamien)
			if (err != nil) != p.blad || wynik != p.wynik {
				t.Errorf("wynik %q, błąd %v", wynik, err)
			}
		})
	}
}

func TestBledyWpisu(t *testing.T) {
	przypadki := []struct {
		nazwa string
		wpis  Wpis
		ile   int
	}{
		{"poprawny", wpisBlokady(), 0},
		{"zamien jak szukaj", Wpis{ID: "a", Opis: "b", Plik: "c", Szukaj: "x", Zamien: "x", Pakiet: "./c"}, 1},
		{"pakiet bez ./", Wpis{ID: "a", Opis: "b", Plik: "c", Szukaj: "x", Pakiet: "c"}, 1},
		{"brak pól (puste szukaj = puste zamien)", Wpis{}, 6},
		{"do uzupełnienia wymaga id i opisu", Wpis{DoUzupelnienia: "później"}, 2},
		{"do uzupełnienia poprawny", Wpis{ID: "a", Opis: "b", DoUzupelnienia: "później"}, 0},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if b := bledyWpisu(p.wpis); len(b) != p.ile {
				t.Errorf("błędy %v, oczekiwano %d", b, p.ile)
			}
		})
	}
}

func TestWczytajListe(t *testing.T) {
	przypadki := []struct {
		nazwa, tresc string
		ile          int
		blad         bool
	}{
		{"poprawna", `{"mutacje":[{"id":"a","opis":"b","do_uzupelnienia":"c"}]}`, 1, false},
		{"pusta", `{"mutacje":[]}`, 0, true},
		{"zły JSON", `{`, 0, true},
		{"powtórzone id", `{"mutacje":[{"id":"a","opis":"b","do_uzupelnienia":"c"},{"id":"a","opis":"b","do_uzupelnienia":"c"}]}`, 0, true},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			sciezka := filepath.Join(t.TempDir(), "lista.json")
			if err := os.WriteFile(sciezka, []byte(p.tresc), 0o644); err != nil {
				t.Fatal(err)
			}
			w, err := wczytajListe(sciezka)
			if (err != nil) != p.blad || len(w) != p.ile {
				t.Errorf("wpisy %v, błąd %v", w, err)
			}
		})
	}
}

func TestListaProjektuJestPoprawna(t *testing.T) {
	if _, err := wczytajListe(filepath.Join("..", "mutacje_blokad.json")); err != nil {
		t.Fatal(err)
	}
}

func TestStrazNieNadpisujeCudzejEdycji(t *testing.T) {
	korzen := projektMutacji(t, kodBlokady)
	edycja := kodBlokady + "// dopisane przez innego agenta w trakcie mutacji\n"
	edytuj := func(string, string) (wynikTestow, error) {
		if err := os.WriteFile(filepath.Join(korzen, "wewn", "rdzen", "b.go"), []byte(edycja), 0o644); err != nil {
			t.Fatal(err)
		}
		return wynikTestow{Kompiluje: true}, nil
	}
	_, err := sprawdzMutacje(korzen, wpisBlokady(), edytuj, &straz{})
	if err == nil || !strings.Contains(err.Error(), "zmieniony w trakcie mutacji") {
		t.Errorf("błąd: %v", err)
	}
	sciezka := filepath.Join(korzen, "wewn", "rdzen", "b.go")
	if tresc, _ := os.ReadFile(sciezka); string(tresc) != edycja {
		t.Errorf("cudza edycja nadpisana:\n%s", tresc)
	}
	if oryginal, _ := os.ReadFile(sciezka + RozszerzenieKonfliktu); string(oryginal) != kodBlokady {
		t.Errorf("brak oryginału w kopii konfliktu")
	}
}
