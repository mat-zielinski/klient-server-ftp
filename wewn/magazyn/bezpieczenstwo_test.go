package magazyn

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// operacjeNaSciezce wywołuje każdą operację magazynu na ścieżce i zwraca błędy po nazwie operacji.
func operacjeNaSciezce(m *Magazyn, udzial, s string) map[string]error {
	wyniki := map[string]error{}
	_, wyniki["Lista"] = m.Lista(udzial, s)
	_, wyniki["Info"] = m.Info(udzial, s)
	f, _, err := m.Otworz(udzial, s)
	if f != nil {
		_ = f.Close() // celowo: test
	}
	wyniki["Otworz"] = err
	_, wyniki["Zapisz"] = m.Zapisz(udzial, s+"/nowy.txt", strings.NewReader("x"), true)
	_, wyniki["DopiszKawalek"] = m.DopiszKawalek(udzial, s+"/k.bin", 0, strings.NewReader("x"))
	wyniki["NowyFolder"] = m.NowyFolder(udzial, s+"/nowy")
	_, wyniki["Szukaj"] = m.Szukaj(udzial, s, "sekret", 0)
	_, wyniki["ZipDo"] = m.ZipDo(io.Discard, udzial, []string{s})
	_, wyniki["RozmiarFolderu"] = m.RozmiarFolderu(udzial, s)
	return wyniki
}

// sprawdzUcieczke: każda operacja przez dowiązanie wychodzące poza udział kończy się ErrPozaUdzialem,
// a na zewnątrz nic nie powstało.
func sprawdzUcieczke(t *testing.T, m *Magazyn, zewn string, link string) {
	t.Helper()
	for op, err := range operacjeNaSciezce(m, "dane", link) {
		if !errors.Is(err, rdzen.ErrPozaUdzialem) {
			t.Errorf("%s przez %s: błąd %v, oczekiwano ErrPozaUdzialem", op, link, err)
		}
	}
	for _, s := range []string{link + "/sekret.txt"} {
		if f, _, err := m.Otworz("dane", s); err == nil {
			_ = f.Close() // celowo: test
			t.Errorf("Otworz(%s) odczytał plik spoza udziału", s)
		}
	}
	for _, n := range []string{"nowy.txt", "k.bin", "nowy", "k.bin" + konfiguracja.MagazynPrzyrostekCzesci} {
		if _, err := os.Lstat(filepath.Join(zewn, n)); err == nil {
			t.Errorf("poza udziałem powstał %s", n)
		}
	}
}

// przygotujZewnetrzny: folder obok udziału z plikiem sekret.txt.
func przygotujZewnetrzny(t *testing.T, korzen string) string {
	zewn := filepath.Join(filepath.Dir(korzen), "zewnatrz")
	plik(t, filepath.Join(zewn, "sekret.txt"), "tajne")
	return zewn
}

func TestUcieczkaPrzezJunction(t *testing.T) {
	m, korzen := nowyTestowy(t)
	zewn := przygotujZewnetrzny(t, korzen)
	plik(t, filepath.Join(korzen, "wewn", "a.txt"), "a")
	junction(t, filepath.Join(korzen, "j"), zewn)
	// junction zagnieżdżony w podfolderze i junction prowadzący do wnętrza udziału
	junction(t, filepath.Join(korzen, "wewn", "j2"), zewn)
	junction(t, filepath.Join(korzen, "dobry"), filepath.Join(korzen, "wewn"))

	sprawdzUcieczke(t, m, zewn, "/j")
	sprawdzUcieczke(t, m, zewn, "/wewn/j2")

	lista, err := m.Lista("dane", "/")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range lista {
		if w.Nazwa == "j" {
			t.Error("Lista pokazuje junction prowadzący poza udział")
		}
	}
	if !maNazwe(lista, "dobry") {
		t.Error("Lista ukrywa junction prowadzący do wnętrza udziału")
	}
	f, _, err := m.Otworz("dane", "/dobry/a.txt")
	if err != nil {
		t.Fatalf("junction wewnątrz udziału powinien działać: %v", err)
	}
	_ = f.Close() // celowo: test
	zip := &bytes.Buffer{}
	if _, err := m.ZipDo(zip, "dane", []string{"/"}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(zip.Bytes(), []byte("sekret")) {
		t.Error("ZIP korzenia zawiera plik spoza udziału")
	}
	if wyn, _ := m.Szukaj("dane", "/", "sekret", 0); len(wyn) > 0 {
		t.Error("Szukaj znalazł plik spoza udziału")
	}
}

func TestUcieczkaPrzezSymlink(t *testing.T) {
	m, korzen := nowyTestowy(t)
	zewn := przygotujZewnetrzny(t, korzen)
	symlink(t, zewn, filepath.Join(korzen, "abs"))
	symlink(t, filepath.Join("..", "zewnatrz"), filepath.Join(korzen, "wzgl"))
	symlink(t, filepath.Join("..", "zewnatrz", "sekret.txt"), filepath.Join(korzen, "plik.txt"))
	// pętla dowiązań
	symlink(t, "petla2", filepath.Join(korzen, "petla1"))
	symlink(t, "petla1", filepath.Join(korzen, "petla2"))

	sprawdzUcieczke(t, m, zewn, "/abs")
	sprawdzUcieczke(t, m, zewn, "/wzgl")
	if _, err := m.Info("dane", "/plik.txt"); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("Info(dowiązanie do pliku poza udziałem) = %v", err)
	}
	if _, err := m.Info("dane", "/petla1"); !errors.Is(err, rdzen.ErrPozaUdzialem) {
		t.Errorf("pętla dowiązań: %v", err)
	}
	lista, err := m.Lista("dane", "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 0 {
		t.Errorf("Lista pokazuje dowiązania poza udział: %v", lista)
	}
	// DoKosza na dowiązaniu przenosi samo dowiązanie, cel zostaje nietknięty
	symlink(t, filepath.Join(korzen, "cel"), filepath.Join(korzen, "dowiazanie"))
	plik(t, filepath.Join(korzen, "cel", "x.txt"), "x")
	if err := m.DoKosza("dane", "/dowiazanie"); err != nil {
		t.Fatal(err)
	}
	if czytaj(t, filepath.Join(korzen, "cel", "x.txt")) != "x" {
		t.Error("DoKosza na dowiązaniu ruszył jego cel")
	}
}

func TestRozwiazWUdziale(t *testing.T) {
	korzen := t.TempDir()
	plik(t, filepath.Join(korzen, "a", "b.txt"), "b")
	testy := []struct {
		nazwa, wej, wyj string
		blad            error
	}{
		{"korzeń", korzen, korzen, nil},
		{"istniejący", filepath.Join(korzen, "a", "b.txt"), filepath.Join(korzen, "a", "b.txt"), nil},
		{"nieistniejąca końcówka", filepath.Join(korzen, "a", "x", "y"), filepath.Join(korzen, "a", "x", "y"), nil},
		{"poza", filepath.Dir(korzen), "", rdzen.ErrPozaUdzialem},
		{"sąsiad z prefiksem", korzen + "x", "", rdzen.ErrPozaUdzialem},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			wyj, err := rozwiazWUdziale(korzen, tt.wej)
			if !errors.Is(err, tt.blad) || (err == nil && wyj != tt.wyj) {
				t.Errorf("= %q, %v; oczekiwano %q, %v", wyj, err, tt.wyj, tt.blad)
			}
		})
	}
}

func TestBezwzglednyCel(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("ścieżki Windows")
	}
	testy := []struct{ biezacy, cel, wyj string }{
		{`C:\u\a`, `D:\x`, `D:\x`},
		{`C:\u\a`, `..\b`, `C:\u\b`},
		{`C:\u\a`, `\Windows`, `C:\Windows`},
		{`C:\u\a`, `b\c`, `C:\u\a\b\c`},
	}
	for _, tt := range testy {
		if w := bezwzglednyCel(tt.biezacy, tt.cel); !strings.EqualFold(w, tt.wyj) {
			t.Errorf("bezwzglednyCel(%q, %q) = %q, oczekiwano %q", tt.biezacy, tt.cel, w, tt.wyj)
		}
	}
}

func TestTylkoOdczyt(t *testing.T) {
	m, _ := nowyTestowy(t)
	ro := m.udzialy["archiwum"].korzen
	plik(t, filepath.Join(ro, "a.txt"), "a")
	zapisy := map[string]func() error{
		"Zapisz": func() error {
			_, err := m.Zapisz("archiwum", "/b.txt", strings.NewReader("b"), true)
			return err
		},
		"DopiszKawalek": func() error {
			_, err := m.DopiszKawalek("archiwum", "/b.txt", 0, strings.NewReader("b"))
			return err
		},
		"Zatwierdz":  func() error { return m.Zatwierdz("archiwum", "/b.txt", true) },
		"NowyFolder": func() error { return m.NowyFolder("archiwum", "/f") },
		"ZmienNazwe": func() error { return m.ZmienNazwe("archiwum", "/a.txt", "/c.txt") },
		"DoKosza":    func() error { return m.DoKosza("archiwum", "/a.txt") },
	}
	for op, f := range zapisy {
		if err := f(); !errors.Is(err, rdzen.ErrTylkoOdczyt) {
			t.Errorf("%s w udziale tylko do odczytu: %v", op, err)
		}
	}
	if czytaj(t, filepath.Join(ro, "a.txt")) != "a" {
		t.Error("plik w udziale tylko do odczytu zmieniony")
	}
	if wpisy, err := m.Lista("archiwum", "/"); err != nil || len(wpisy) != 1 {
		t.Errorf("odczyt w udziale tylko do odczytu: %v %v", wpisy, err)
	}
}

func TestUkrywanie(t *testing.T) {
	m, korzen := nowyTestowy(t)
	ukryte := []string{
		"desktop.ini", "Thumbs.db", "$RECYCLE.BIN/x", "System Volume Information/y",
		konfiguracja.KatalogKosza + "/2026/stary.txt", "film.mp4" + konfiguracja.MagazynPrzyrostekCzesci,
		konfiguracja.MagazynPrefiksTymczasowy + "123", "pod/DESKTOP.INI", "lost+found/z", ".Trash-1000/files/s",
	}
	for _, u := range ukryte {
		plik(t, filepath.Join(korzen, filepath.FromSlash(u)), "sekret ukryty")
	}
	plik(t, filepath.Join(korzen, "widoczny.txt"), "w")
	lista, err := m.Lista("dane", "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 2 || !maNazwe(lista, "widoczny.txt") || !maNazwe(lista, "pod") {
		t.Errorf("Lista pokazuje ukryte: %v", lista)
	}
	if l, _ := m.Lista("dane", "/pod"); len(l) != 0 {
		t.Errorf("Lista podfolderu pokazuje ukryte: %v", l)
	}
	for _, u := range ukryte {
		if _, err := m.Info("dane", "/"+u); !errors.Is(err, rdzen.ErrNieZnaleziono) {
			t.Errorf("Info(%s) = %v, oczekiwano ErrNieZnaleziono", u, err)
		}
	}
	if w, _ := m.Szukaj("dane", "/", "sekret", 0); len(w) != 0 {
		t.Errorf("Szukaj znalazł ukryte: %v", w)
	}
	if w, _ := m.Szukaj("dane", "/", "ini", 0); len(w) != 0 {
		t.Errorf("Szukaj znalazł desktop.ini: %v", w)
	}
	buf := &bytes.Buffer{}
	if _, err := m.ZipDo(buf, "dane", []string{"/"}); err != nil {
		t.Fatal(err)
	}
	if n := len(rozpakuj(t, buf.Bytes())); n != 3 { // dane/, dane/pod/, dane/widoczny.txt
		t.Errorf("ZIP ma %d pozycji, oczekiwano 3 (bez ukrytych)", n)
	}
	if r, _ := m.RozmiarFolderu("dane", "/"); r != 1 {
		t.Errorf("RozmiarFolderu liczy ukryte: %d", r)
	}
}

func TestCzyUkryta(t *testing.T) {
	testy := map[string]bool{
		"a.txt": false, ".skrytka_kosz": true, ".SKRYTKA_KOSZ": true, "x.skrytka_czesc": true,
		".skrytka_tmp_9": true, "Desktop.ini": true, "thumbs.db": true, "$Recycle.Bin": true,
		"System Volume Information": true, "desktop.ini.txt": false, ".gitignore": false,
		"lost+found": true, ".Trash-1000": true, ".trash-0": true, "Trash": false, "lost+found.txt": false,
	}
	for n, oczek := range testy {
		if czyUkryta(n) != oczek {
			t.Errorf("czyUkryta(%q) != %v", n, oczek)
		}
	}
}

func maNazwe(w []rdzen.Wpis, nazwa string) bool {
	for _, x := range w {
		if x.Nazwa == nazwa {
			return true
		}
	}
	return false
}
