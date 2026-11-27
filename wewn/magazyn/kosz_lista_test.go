package magazyn

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// znacznikTestu: folder kosza dla stalyCzas.
var znacznikTestu = stalyCzas.Format(konfiguracja.MagazynFormatZnacznikaKosza)

// idZ koduje identyfikator tak jak serwer (do testów złych identyfikatorów).
func idZ(surowy string) string { return base64.RawURLEncoding.EncodeToString([]byte(surowy)) }

// wpisPo zwraca wpis kosza o danej oryginalnej ścieżce.
func wpisPo(t *testing.T, m *Magazyn, sciezka string) WpisKosza {
	t.Helper()
	l, err := m.ListaKosza("dane")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range l {
		if w.Sciezka == sciezka {
			return w
		}
	}
	t.Fatalf("brak w koszu: %s (%+v)", sciezka, l)
	return WpisKosza{}
}

func doKosza(t *testing.T, m *Magazyn, sciezka string) {
	t.Helper()
	if err := m.DoKosza("dane", sciezka); err != nil {
		t.Fatalf("DoKosza %s: %v", sciezka, err)
	}
}

func TestListaKosza(t *testing.T) {
	m, korzen := nowyTestowy(t)
	if l, err := m.ListaKosza("dane"); err != nil || len(l) != 0 {
		t.Fatalf("pusty kosz: %v %v", l, err)
	}
	if _, err := m.ListaKosza("nieznany"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("nieznany udział: %v", err)
	}
	plik(t, filepath.Join(korzen, "f", "a.txt"), "12345")
	plik(t, filepath.Join(korzen, "g", "x", "1.txt"), "12")
	plik(t, filepath.Join(korzen, "g", "2.txt"), "123")
	doKosza(t, m, "/f/a.txt")
	m.zegar = func() time.Time { return stalyCzas.Add(time.Hour) }
	doKosza(t, m, "/g")
	l, err := m.ListaKosza("dane")
	if err != nil {
		t.Fatal(err)
	}
	oczekiwane := []WpisKosza{
		{Nazwa: "g", Sciezka: "/g", Usunieto: stalyCzas.Add(time.Hour), Rozmiar: 5, Folder: true},
		{Nazwa: "a.txt", Sciezka: "/f/a.txt", Usunieto: stalyCzas, Rozmiar: 5},
	}
	if len(l) != len(oczekiwane) {
		t.Fatalf("lista: %+v", l)
	}
	for i, o := range oczekiwane {
		w := l[i]
		if w.Nazwa != o.Nazwa || w.Sciezka != o.Sciezka || !w.Usunieto.Equal(o.Usunieto) ||
			w.Rozmiar != o.Rozmiar || w.Folder != o.Folder || w.ID == "" {
			t.Errorf("wpis %d: %+v, oczekiwano %+v", i, w, o)
		}
		if strings.ContainsAny(w.ID, `/\:.`) {
			t.Errorf("ID z niebezpiecznymi znakami: %q", w.ID)
		}
	}
}

func TestListaKoszaStaryUklad(t *testing.T) {
	m, korzen := nowyTestowy(t)
	folder := filepath.Join(korzen, konfiguracja.KatalogKosza, znacznikTestu)
	plik(t, filepath.Join(folder, "f", "a.txt"), "1")
	plik(t, filepath.Join(folder, "x.txt"), "1")
	plik(t, filepath.Join(folder, "d", "1.txt"), "1")
	plik(t, filepath.Join(folder, "d", "2.txt"), "1")
	plik(t, filepath.Join(korzen, konfiguracja.KatalogKosza, "obcy", "z.txt"), "1")
	l, err := m.ListaKosza("dane")
	if err != nil {
		t.Fatal(err)
	}
	jest := map[string]bool{}
	for _, w := range l {
		jest[w.Sciezka] = true
	}
	for _, s := range []string{"/f/a.txt", "/x.txt", "/d"} {
		if !jest[s] {
			t.Errorf("brak wpisu %s w %+v", s, l)
		}
	}
	if len(l) != 3 {
		t.Errorf("za dużo wpisów (folder bez znacznika?): %+v", l)
	}
	if err := m.Przywroc("dane", wpisPo(t, m, "/f/a.txt").ID, false); err != nil || czytaj(t, filepath.Join(korzen, "f", "a.txt")) != "1" {
		t.Errorf("przywrócenie ze starego układu: %v", err)
	}
}

func TestPrzywroc(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "a", "b", "c.txt"), "treść")
	doKosza(t, m, "/a/b/c.txt")
	doKosza(t, m, "/a")
	if err := m.Przywroc("dane", wpisPo(t, m, "/a/b/c.txt").ID, false); err != nil {
		t.Fatal(err)
	}
	if czytaj(t, filepath.Join(korzen, "a", "b", "c.txt")) != "treść" {
		t.Error("plik nie wrócił na miejsce (foldery nadrzędne)")
	}
	if _, err := os.Stat(filepath.Join(korzen, konfiguracja.KatalogKosza, znacznikTestu)); !os.IsNotExist(err) {
		t.Errorf("pusty folder znacznika został: %v", err)
	}
	if _, err := os.Stat(filepath.Join(korzen, konfiguracja.KatalogKosza, znacznikTestu+konfiguracja.MagazynPrzyrostekOpisuKosza)); !os.IsNotExist(err) {
		t.Errorf("opis znacznika został: %v", err)
	}
	if l, _ := m.ListaKosza("dane"); len(l) != 1 || l[0].Sciezka != "/a" {
		t.Errorf("w koszu powinien zostać tylko /a: %+v", l)
	}
}

func TestPrzywrocKolizja(t *testing.T) {
	testy := []struct {
		nazwa    string
		nadpisz  bool
		blad     error
		tresc    string
		wKoszu   int
		folderem bool
	}{
		{"zajęte bez nadpisania", false, rdzen.ErrJuzIstnieje, "nowy", 1, false},
		{"zajęte z nadpisaniem", true, nil, "stary", 1, false},
		{"zajęte folderem z nadpisaniem", true, nil, "stary", 1, true},
	}
	for _, tc := range testy {
		t.Run(tc.nazwa, func(t *testing.T) {
			m, korzen := nowyTestowy(t)
			cel := filepath.Join(korzen, "p.txt")
			plik(t, cel, "stary")
			doKosza(t, m, "/p.txt")
			if tc.folderem {
				plik(t, filepath.Join(cel, "w.txt"), "nowy")
			} else {
				plik(t, cel, "nowy")
			}
			err := m.Przywroc("dane", wpisPo(t, m, "/p.txt").ID, tc.nadpisz)
			if !errors.Is(err, tc.blad) {
				t.Fatalf("błąd %v, oczekiwano %v", err, tc.blad)
			}
			if got := czytaj(t, cel); got != tc.tresc {
				t.Errorf("treść %q, oczekiwano %q", got, tc.tresc)
			}
			if l, _ := m.ListaKosza("dane"); len(l) != tc.wKoszu {
				t.Errorf("w koszu %d, oczekiwano %d (nadpisany element ma trafić do kosza)", len(l), tc.wKoszu)
			}
		})
	}
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "k", "p.txt"), "1")
	doKosza(t, m, "/k/p.txt")
	if err := os.Remove(filepath.Join(korzen, "k")); err != nil {
		t.Fatal(err)
	}
	plik(t, filepath.Join(korzen, "k"), "plik zamiast folderu")
	if err := m.Przywroc("dane", wpisPo(t, m, "/k/p.txt").ID, false); !errors.Is(err, rdzen.ErrJuzIstnieje) {
		t.Errorf("plik zamiast folderu nadrzędnego: %v", err)
	}
}

func TestUsunTrwale(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "d", "a.txt"), "1")
	plik(t, filepath.Join(korzen, "b.txt"), "2")
	doKosza(t, m, "/d")
	doKosza(t, m, "/b.txt")
	id := wpisPo(t, m, "/d").ID
	if err := m.UsunTrwale("dane", id); err != nil {
		t.Fatal(err)
	}
	if l, _ := m.ListaKosza("dane"); len(l) != 1 || l[0].Sciezka != "/b.txt" {
		t.Errorf("po usunięciu: %+v", l)
	}
	if err := m.UsunTrwale("dane", id); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("drugie usunięcie: %v", err)
	}
	if err := m.Przywroc("dane", id, false); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("przywrócenie usuniętego trwale: %v", err)
	}
}

// TestUsunTrwaleJunctionWKoszu: usunięcie folderu z junction do sekretu nie rusza sekretu.
func TestUsunTrwaleJunctionWKoszu(t *testing.T) {
	m, korzen := nowyTestowy(t)
	sekret := filepath.Join(filepath.Dir(korzen), "sekret")
	plik(t, filepath.Join(sekret, "s.txt"), "tajne")
	if err := os.MkdirAll(filepath.Join(korzen, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	junction(t, filepath.Join(korzen, "d", "j"), sekret)
	doKosza(t, m, "/d")
	if _, err := m.Oproznij("dane", 0); err != nil {
		t.Fatal(err)
	}
	if czytaj(t, filepath.Join(sekret, "s.txt")) != "tajne" {
		t.Fatal("opróżnianie kosza usunęło plik spoza udziału przez junction")
	}
}

func TestKoszUcieczkaPrzezID(t *testing.T) {
	m, korzen := nowyTestowy(t)
	baza := filepath.Dir(korzen)
	plik(t, filepath.Join(baza, "sekret.txt"), "tajne")
	plik(t, filepath.Join(korzen, "x.txt"), "1")
	doKosza(t, m, "/x.txt")
	zn := znacznikTestu
	testy := []struct {
		nazwa, id string
		blad      error
	}{
		{"puste", "", ErrZlyIDKosza},
		{"nie base64", "!!!", ErrZlyIDKosza},
		{"za długie", strings.Repeat("A", konfiguracja.MagazynMaksDlugoscIDKosza+1), ErrZlyIDKosza},
		{"bez ścieżki", idZ(zn + "/"), ErrZlyIDKosza},
		{"bez znacznika", idZ("x.txt"), ErrZlyIDKosza},
		{"zły znacznik", idZ("obcy/x.txt"), ErrZlyIDKosza},
		{"znacznik _1", idZ(zn + "_1/x.txt"), ErrZlyIDKosza},
		{"znacznik _02", idZ(zn + "_02/x.txt"), ErrZlyIDKosza},
		{"kropki w górę", idZ(zn + "/../../sekret.txt"), nil},
		{"kropki na początku", idZ("../sekret.txt"), ErrZlyIDKosza},
		{"ukośnik wsteczny", idZ(zn + `/..\..\sekret.txt`), nil},
		{"dysk", idZ(zn + "/C:/sekret.txt"), nil},
		{"podwójny ukośnik", idZ(zn + "//x.txt"), nil},
		{"ukryta nazwa", idZ(zn + "/" + konfiguracja.KatalogKosza + "/x.txt"), rdzen.ErrNieZnaleziono},
		{"inna ścieżka niż w opisie", idZ(zn + "/y.txt"), rdzen.ErrNieZnaleziono},
	}
	for _, tc := range testy {
		t.Run(tc.nazwa, func(t *testing.T) {
			for op, f := range map[string]func() error{
				"Przywroc":   func() error { return m.Przywroc("dane", tc.id, true) },
				"UsunTrwale": func() error { return m.UsunTrwale("dane", tc.id) },
			} {
				err := f()
				if err == nil || (tc.blad != nil && !errors.Is(err, tc.blad)) {
					t.Errorf("%s: %v, oczekiwano %v", op, err, tc.blad)
				}
			}
		})
	}
	if czytaj(t, filepath.Join(baza, "sekret.txt")) != "tajne" {
		t.Error("sekret ruszony")
	}
	if len(mustLista(t, m)) != 1 {
		t.Error("prawdziwy wpis zniknął")
	}
}

func mustLista(t *testing.T, m *Magazyn) []WpisKosza {
	t.Helper()
	l, err := m.ListaKosza("dane")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// TestKoszUcieczkaPrzezJunction: dowiązanie w folderze znacznika i kosz będący dowiązaniem nie wyprowadzają poza kosz.
func TestKoszUcieczkaPrzezJunction(t *testing.T) {
	m, korzen := nowyTestowy(t)
	sekret := filepath.Join(filepath.Dir(korzen), "sekret")
	plik(t, filepath.Join(sekret, "s.txt"), "tajne")
	folder := filepath.Join(korzen, konfiguracja.KatalogKosza, znacznikTestu)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	junction(t, filepath.Join(folder, "j"), sekret)
	id := idZ(znacznikTestu + "/j/s.txt")
	if err := m.Przywroc("dane", id, false); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("Przywroc przez junction: %v", err)
	}
	if err := m.UsunTrwale("dane", id); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("UsunTrwale przez junction: %v", err)
	}
	if czytaj(t, filepath.Join(sekret, "s.txt")) != "tajne" {
		t.Fatal("plik spoza udziału ruszony")
	}
	if _, err := os.Stat(filepath.Join(korzen, "j", "s.txt")); err == nil {
		t.Fatal("plik spoza udziału przywrócony do udziału")
	}
	// Cały kosz jako junction do obcego folderu.
	m2, korzen2 := nowyTestowy(t)
	obcy := filepath.Join(filepath.Dir(korzen2), "obcy")
	plik(t, filepath.Join(obcy, znacznikTestu, "s.txt"), "tajne")
	junction(t, filepath.Join(korzen2, konfiguracja.KatalogKosza), obcy)
	if _, err := m2.ListaKosza("dane"); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("ListaKosza na junction: %v", err)
	}
	if err := m2.Przywroc("dane", idZ(znacznikTestu+"/s.txt"), false); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("Przywroc z kosza-junction: %v", err)
	}
	if n, err := m2.Oproznij("dane", 0); !errors.Is(err, rdzen.ErrPozaUdzialem) || n != 0 {
		t.Errorf("Oproznij kosza-junction: %d %v", n, err)
	}
	if czytaj(t, filepath.Join(obcy, znacznikTestu, "s.txt")) != "tajne" {
		t.Fatal("obcy folder ruszony")
	}
}

func TestOproznijWgWieku(t *testing.T) {
	dzien := 24 * time.Hour
	testy := []struct {
		nazwa      string
		starszeNiz time.Duration
		usuniete   int
		zostaje    []string
	}{
		{"starsze niż 30 dni", 30 * dzien, 2, []string{"/c.txt"}},
		{"starsze niż 40 dni", 40 * dzien, 1, []string{"/b.txt", "/c.txt"}},
		{"wszystkie", 0, 3, nil},
	}
	for _, tc := range testy {
		t.Run(tc.nazwa, func(t *testing.T) {
			m, korzen := nowyTestowy(t)
			for i, n := range []string{"a.txt", "b.txt", "c.txt"} {
				plik(t, filepath.Join(korzen, n), "1")
				czas := stalyCzas.Add(time.Duration(i) * 10 * dzien) // a: 0, b: +10 dni, c: +20 dni
				m.zegar = func() time.Time { return czas }
				doKosza(t, m, "/"+n)
			}
			m.zegar = func() time.Time { return stalyCzas.Add(45 * dzien) } // wiek: a 45, b 35, c 25 dni
			n, err := m.Oproznij("dane", tc.starszeNiz)
			if err != nil || n != tc.usuniete {
				t.Fatalf("Oproznij: %d %v, oczekiwano %d", n, err, tc.usuniete)
			}
			l := mustLista(t, m)
			if len(l) != len(tc.zostaje) {
				t.Fatalf("zostało %+v, oczekiwano %v", l, tc.zostaje)
			}
			for i := range tc.zostaje {
				if l[len(l)-1-i].Sciezka != tc.zostaje[i] {
					t.Errorf("zostało %+v, oczekiwano %v", l, tc.zostaje)
				}
			}
		})
	}
	m, _ := nowyTestowy(t)
	if n, err := m.Oproznij("dane", 0); err != nil || n != 0 {
		t.Errorf("opróżnianie bez kosza: %d %v", n, err)
	}
	if _, err := m.Oproznij("nieznany", 0); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("nieznany udział: %v", err)
	}
}

func TestKoszTylkoOdczyt(t *testing.T) {
	m, _ := nowyTestowy(t)
	id := idZ(znacznikTestu + "/a.txt")
	operacje := map[string]func() error{
		"Przywroc":   func() error { return m.Przywroc("archiwum", id, false) },
		"UsunTrwale": func() error { return m.UsunTrwale("archiwum", id) },
		"Oproznij": func() error {
			_, err := m.Oproznij("archiwum", 0)
			return err
		},
	}
	for op, f := range operacje {
		if err := f(); !errors.Is(err, rdzen.ErrTylkoOdczyt) {
			t.Errorf("%s w udziale tylko do odczytu: %v", op, err)
		}
	}
}

func TestCzasZnacznika(t *testing.T) {
	testy := map[string]bool{
		znacznikTestu: true, znacznikTestu + "_2": true, znacznikTestu + "_15": true,
		znacznikTestu + "_1": false, znacznikTestu + "_02": false, znacznikTestu + "_x": false,
		znacznikTestu + "x": false, "2026-13-01_00-00-00": false, "": false, "obcy": false,
	}
	for nazwa, ok := range testy {
		if _, got := czasZnacznika(nazwa, time.UTC); got != ok {
			t.Errorf("%q: %v, oczekiwano %v", nazwa, got, ok)
		}
	}
}
