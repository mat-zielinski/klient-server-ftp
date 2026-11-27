package magazyn

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

func TestNowy(t *testing.T) {
	baza := t.TempDir()
	plik(t, filepath.Join(baza, "plik.txt"), "x")
	testy := []struct {
		nazwa   string
		udzialy []konfiguracja.Udzial
		blad    bool
	}{
		{"pusty", nil, false},
		{"poprawny", []konfiguracja.Udzial{{Nazwa: "a", Sciezka: baza}}, false},
		{"brak folderu", []konfiguracja.Udzial{{Nazwa: "a", Sciezka: filepath.Join(baza, "nie")}}, true},
		{"plik zamiast folderu", []konfiguracja.Udzial{{Nazwa: "a", Sciezka: filepath.Join(baza, "plik.txt")}}, true},
		{"powtórzona nazwa", []konfiguracja.Udzial{{Nazwa: "a", Sciezka: baza}, {Nazwa: "a", Sciezka: baza}}, true},
		{"pusta nazwa", []konfiguracja.Udzial{{Nazwa: "", Sciezka: baza}}, true},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if _, err := Nowy(tt.udzialy); (err != nil) != tt.blad {
				t.Errorf("błąd = %v, oczekiwano błędu: %v", err, tt.blad)
			}
		})
	}
}

func TestUdzialy(t *testing.T) {
	m, _ := nowyTestowy(t)
	u := m.Udzialy()
	if len(u) != 2 || u[0].Nazwa != "dane" || u[1].Nazwa != "archiwum" {
		t.Errorf("kolejność udziałów: %v", u)
	}
	u[0].Nazwa = "zmienione"
	if m.Udzialy()[0].Nazwa != "dane" {
		t.Error("Udzialy zwraca wewnętrzny wycinek")
	}
	if _, err := m.Udzial("brak"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Udzial(brak) = %v", err)
	}
	if x, err := m.Udzial("archiwum"); err != nil || !x.TylkoOdczyt {
		t.Errorf("Udzial(archiwum) = %v, %v", x, err)
	}
	if _, err := m.Lista("brak", "/"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Lista w nieznanym udziale = %v", err)
	}
}

func TestListaInfoOtworz(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "zdjecia", "a.jpg"), "12345")
	plik(t, filepath.Join(korzen, "b.txt"), "b")
	lista, err := m.Lista("dane", "/")
	if err != nil || len(lista) != 2 {
		t.Fatalf("Lista = %v, %v", lista, err)
	}
	l, _ := m.Lista("dane", "/zdjecia")
	if len(l) != 1 || l[0].Sciezka != "/zdjecia/a.jpg" || l[0].Rozmiar != 5 || l[0].Rodzaj != rdzen.RodzajObraz {
		t.Errorf("wpis: %+v", l)
	}
	testy := []struct {
		sciezka string
		blad    error
	}{
		{"/b.txt", nil},
		{"zdjecia/a.jpg", nil},
		{"/nie.txt", rdzen.ErrNieZnaleziono},
		{"/../x", rdzen.ErrPozaUdzialem},
		{"/zdjecia", rdzen.ErrNieZnaleziono}, // folder w Otworz
	}
	for _, tt := range testy {
		f, w, err := m.Otworz("dane", tt.sciezka)
		if !errors.Is(err, tt.blad) {
			t.Errorf("Otworz(%s) = %v, oczekiwano %v", tt.sciezka, err, tt.blad)
		}
		if f != nil {
			_ = f.Close() // celowo: test
			if w.Folder || w.Nazwa == "" {
				t.Errorf("Otworz(%s) wpis %+v", tt.sciezka, w)
			}
		}
	}
	if w, err := m.Info("dane", "/"); err != nil || w.Nazwa != "dane" || !w.Folder || w.Sciezka != "/" {
		t.Errorf("Info(/) = %+v, %v", w, err)
	}
	if _, err := m.Lista("dane", "/b.txt"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Lista(plik) = %v", err)
	}
}

func TestZapisz(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "jest.txt"), "stary")
	if err := os.Mkdir(filepath.Join(korzen, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	testy := []struct {
		nazwa, sciezka string
		nadpisz        bool
		blad           error
		tresc          string
	}{
		{"nowy", "/nowy.txt", false, nil, "dane"},
		{"istnieje bez nadpisania", "/jest.txt", false, rdzen.ErrJuzIstnieje, "stary"},
		{"nadpisanie", "/jest.txt", true, nil, "dane"},
		{"folder pod nazwą", "/folder", true, rdzen.ErrJuzIstnieje, ""},
		{"brak rodzica", "/brak/x.txt", false, rdzen.ErrNieZnaleziono, ""},
		{"korzeń", "/", true, rdzen.ErrZlaNazwa, ""},
		{"zła nazwa", "/a<b.txt", false, rdzen.ErrZlaNazwa, ""},
		{"ukryta nazwa", "/x" + konfiguracja.MagazynPrzyrostekCzesci, false, rdzen.ErrNieZnaleziono, ""},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			n, err := m.Zapisz("dane", tt.sciezka, strings.NewReader("dane"), tt.nadpisz)
			if !errors.Is(err, tt.blad) {
				t.Fatalf("błąd %v, oczekiwano %v", err, tt.blad)
			}
			if err == nil && n != 4 {
				t.Errorf("zapisano %d bajtów", n)
			}
			if tt.tresc != "" {
				if got := czytaj(t, filepath.Join(korzen, filepath.FromSlash(tt.sciezka))); got != tt.tresc {
					t.Errorf("treść %q, oczekiwano %q", got, tt.tresc)
				}
			}
		})
	}
	// przerwany strumień: brak celu i brak pliku tymczasowego
	_, err := m.Zapisz("dane", "/przerwany.txt", io.MultiReader(strings.NewReader("abc"), zepsuty{}), false)
	if err == nil || czytaj(t, filepath.Join(korzen, "przerwany.txt")) != "" {
		t.Errorf("przerwany zapis: %v", err)
	}
	sprawdzBrakTymczasowych(t, korzen)
}

type zepsuty struct{}

func (zepsuty) Read([]byte) (int, error) { return 0, errors.New("zerwane połączenie") }

func sprawdzBrakTymczasowych(t *testing.T, folder string) {
	t.Helper()
	wpisy, _ := os.ReadDir(folder)
	for _, w := range wpisy {
		if strings.HasPrefix(w.Name(), konfiguracja.MagazynPrefiksTymczasowy) {
			t.Errorf("został plik tymczasowy %s", w.Name())
		}
	}
}

func TestZapiszWspolbieznie(t *testing.T) {
	m, korzen := nowyTestowy(t)
	const n = 16
	var wg sync.WaitGroup
	var mu sync.Mutex
	udane := 0
	for i := range n {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := m.Zapisz("dane", "/wspolny.txt", strings.NewReader(strconv.Itoa(i)), false); err == nil {
				mu.Lock()
				udane++
				mu.Unlock()
			} else if !errors.Is(err, rdzen.ErrJuzIstnieje) {
				t.Errorf("nieoczekiwany błąd: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := m.Zapisz("dane", "/p"+strconv.Itoa(i)+".txt", strings.NewReader("x"), false); err != nil {
				t.Errorf("zapis równoległy: %v", err)
			}
		}()
	}
	wg.Wait()
	if udane != 1 {
		t.Errorf("bez nadpisania udało się %d zapisów tego samego pliku, oczekiwano 1", udane)
	}
	sprawdzBrakTymczasowych(t, korzen)
}

func TestKawalki(t *testing.T) {
	m, korzen := nowyTestowy(t)
	czesc := filepath.Join(korzen, "duzy.bin"+konfiguracja.MagazynPrzyrostekCzesci)
	kroki := []struct {
		nazwa        string
		przesuniecie int64
		dane         string
		blad         error
		poKroku      string
	}{
		{"pierwszy", 0, "abc", nil, "abc"},
		{"drugi", 3, "def", nil, "abcdef"},
		{"luka", 10, "x", ErrZlePrzesuniecie, "abcdef"},
		{"ujemne", -1, "x", ErrZlePrzesuniecie, "abcdef"},
		{"ponowienie po zerwaniu", 3, "DEF", nil, "abcDEF"},
		{"wznowienie", 6, "ghi", nil, "abcDEFghi"},
	}
	for _, k := range kroki {
		n, err := m.DopiszKawalek("dane", "/duzy.bin", k.przesuniecie, strings.NewReader(k.dane))
		if !errors.Is(err, k.blad) {
			t.Fatalf("%s: błąd %v, oczekiwano %v", k.nazwa, err, k.blad)
		}
		if err == nil && n != int64(len(k.dane)) {
			t.Errorf("%s: zapisano %d", k.nazwa, n)
		}
		if got := czytaj(t, czesc); got != k.poKroku {
			t.Errorf("%s: część %q, oczekiwano %q", k.nazwa, got, k.poKroku)
		}
	}
	if r, err := m.RozmiarKawalkow("dane", "/duzy.bin"); err != nil || r != 9 {
		t.Errorf("RozmiarKawalkow = %d, %v", r, err)
	}
	if l, _ := m.Lista("dane", "/"); len(l) != 0 {
		t.Errorf("część widoczna w liście: %v", l)
	}
	plik(t, filepath.Join(korzen, "duzy.bin"), "stary")
	if err := m.Zatwierdz("dane", "/duzy.bin", false); !errors.Is(err, rdzen.ErrJuzIstnieje) {
		t.Errorf("Zatwierdz bez nadpisania na istniejący = %v", err)
	}
	if err := m.Zatwierdz("dane", "/duzy.bin", true); err != nil {
		t.Fatal(err)
	}
	if czytaj(t, filepath.Join(korzen, "duzy.bin")) != "abcDEFghi" || czytaj(t, czesc) != "" {
		t.Error("po zatwierdzeniu zła treść albo została część")
	}
	if err := m.Zatwierdz("dane", "/inny.bin", false); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Zatwierdz bez części = %v", err)
	}
	if r, err := m.RozmiarKawalkow("dane", "/inny.bin"); err != nil || r != 0 {
		t.Errorf("RozmiarKawalkow bez części = %d, %v", r, err)
	}
}

func TestNowyFolderIZmianaNazwy(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "a.txt"), "a")
	plik(t, filepath.Join(korzen, "b.txt"), "b")
	plik(t, filepath.Join(korzen, "f", "w.txt"), "w")
	testy := []struct {
		nazwa string
		op    func() error
		blad  error
	}{
		{"nowy folder", func() error { return m.NowyFolder("dane", "/nowy") }, nil},
		{"folder istnieje", func() error { return m.NowyFolder("dane", "/f") }, rdzen.ErrJuzIstnieje},
		{"folder bez rodzica", func() error { return m.NowyFolder("dane", "/x/y") }, rdzen.ErrNieZnaleziono},
		{"folder kosza", func() error { return m.NowyFolder("dane", "/"+konfiguracja.KatalogKosza) }, rdzen.ErrNieZnaleziono},
		{"zmiana nazwy", func() error { return m.ZmienNazwe("dane", "/a.txt", "/c.txt") }, nil},
		{"cel zajęty", func() error { return m.ZmienNazwe("dane", "/c.txt", "/b.txt") }, rdzen.ErrJuzIstnieje},
		{"tylko wielkość liter", func() error { return m.ZmienNazwe("dane", "/c.txt", "/C.txt") }, nil},
		{"przeniesienie", func() error { return m.ZmienNazwe("dane", "/b.txt", "/f/b.txt") }, nil},
		{"folder do wnętrza", func() error { return m.ZmienNazwe("dane", "/f", "/f/pod") }, rdzen.ErrZlaNazwa},
		{"korzeń", func() error { return m.ZmienNazwe("dane", "/", "/x") }, rdzen.ErrZlaNazwa},
		{"brak źródła", func() error { return m.ZmienNazwe("dane", "/nie", "/x") }, rdzen.ErrNieZnaleziono},
		{"poza udział", func() error { return m.ZmienNazwe("dane", "/f", "/../f") }, rdzen.ErrPozaUdzialem},
		{"do kosza przez nazwę", func() error { return m.ZmienNazwe("dane", "/f", "/"+konfiguracja.KatalogKosza) }, rdzen.ErrNieZnaleziono},
	}
	for _, tt := range testy {
		if err := tt.op(); !errors.Is(err, tt.blad) {
			t.Errorf("%s: %v, oczekiwano %v", tt.nazwa, err, tt.blad)
		}
	}
	if czytaj(t, filepath.Join(korzen, "f", "b.txt")) != "b" {
		t.Error("przeniesienie nie zadziałało")
	}
	if w, err := m.Info("dane", "/C.txt"); err != nil || w.Nazwa != "C.txt" {
		t.Errorf("zmiana wielkości liter: %+v %v", w, err)
	}
}

func TestDoKosza(t *testing.T) {
	m, korzen := nowyTestowy(t)
	znacznik := stalyCzas.Format(konfiguracja.MagazynFormatZnacznikaKosza)
	kosz := filepath.Join(korzen, konfiguracja.KatalogKosza)
	plik(t, filepath.Join(korzen, "f", "a.txt"), "1")
	if err := m.DoKosza("dane", "/f/a.txt"); err != nil {
		t.Fatal(err)
	}
	plik(t, filepath.Join(korzen, "f", "a.txt"), "2")
	if err := m.DoKosza("dane", "/f/a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := m.DoKosza("dane", "/f"); err != nil {
		t.Fatal(err)
	}
	oczekiwane := map[string]string{
		filepath.Join(kosz, znacznik, "f", "a.txt"):      "1",
		filepath.Join(kosz, znacznik+"_2", "f", "a.txt"): "2",
		filepath.Join(kosz, znacznik+"_3", "f"):          "",
	}
	for s, tresc := range oczekiwane {
		if _, err := os.Stat(s); err != nil {
			t.Errorf("brak w koszu: %s", s)
		} else if tresc != "" && czytaj(t, s) != tresc {
			t.Errorf("%s: zła treść", s)
		}
	}
	if l, _ := m.Lista("dane", "/"); len(l) != 0 {
		t.Errorf("kosz widoczny albo plik nie zniknął: %v", l)
	}
	wKoszu := "/" + konfiguracja.KatalogKosza + "/" + znacznik + "/f/a.txt"
	for op, err := range operacjeNaSciezce(m, "dane", wKoszu) {
		if !errors.Is(err, rdzen.ErrNieZnaleziono) {
			t.Errorf("%s w koszu z zewnątrz: %v, oczekiwano ErrNieZnaleziono", op, err)
		}
	}
	if _, err := m.Lista("dane", "/"+strings.ToUpper(konfiguracja.KatalogKosza)); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Lista kosza wielkimi literami: %v", err)
	}
	testy := []struct {
		sciezka string
		blad    error
	}{
		{"/", rdzen.ErrZlaNazwa},
		{"/nie.txt", rdzen.ErrNieZnaleziono},
		{"/" + konfiguracja.KatalogKosza, rdzen.ErrNieZnaleziono},
		{"/../x", rdzen.ErrPozaUdzialem},
	}
	for _, tt := range testy {
		if err := m.DoKosza("dane", tt.sciezka); !errors.Is(err, tt.blad) {
			t.Errorf("DoKosza(%s) = %v, oczekiwano %v", tt.sciezka, err, tt.blad)
		}
	}
}

func TestSzukaj(t *testing.T) {
	m, korzen := nowyTestowy(t)
	for i := range 10 {
		plik(t, filepath.Join(korzen, "a", "b", "raport"+strconv.Itoa(i)+".txt"), "x")
	}
	plik(t, filepath.Join(korzen, "inne.txt"), "x")
	testy := []struct {
		nazwa, sciezka, zapytanie string
		limit, ile                int
	}{
		{"wszystkie", "/", "raport", 0, 10},
		{"limit", "/", "raport", 3, 3},
		{"folder", "/", "b", 0, 1},
		{"od podfolderu", "/a/b", "inne", 0, 0},
	}
	for _, tt := range testy {
		w, err := m.Szukaj("dane", tt.sciezka, tt.zapytanie, tt.limit)
		if err != nil || len(w) != tt.ile {
			t.Errorf("%s: %d wyników (%v), oczekiwano %d", tt.nazwa, len(w), err, tt.ile)
		}
	}
	w, _ := m.Szukaj("dane", "/", "raport3", 0)
	if len(w) != 1 || w[0].Sciezka != "/a/b/raport3.txt" {
		t.Errorf("ścieżka wyniku: %+v", w)
	}
	if _, err := m.Szukaj("dane", "/inne.txt", "x", 0); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("Szukaj w pliku = %v", err)
	}
}

func TestZip(t *testing.T) {
	m, korzen := nowyTestowy(t)
	duzy := strings.Repeat("Skrytka tekst do kompresji. ", 100000)
	pliki := map[string]string{
		"f/a.txt":        duzy,
		"f/zdjecie.JPG":  "obraz",
		"f/pod/film.mp4": "wideo",
		"f/Żółw ąę.txt":  "polskie",
		"b.txt":          "b",
	}
	var suma int64
	for s, tresc := range pliki {
		plik(t, filepath.Join(korzen, filepath.FromSlash(s)), tresc)
		suma += int64(len(tresc))
	}
	if err := os.Mkdir(filepath.Join(korzen, "f", "pusty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r, err := m.RozmiarFolderu("dane", "/f"); err != nil || r != suma-1 {
		t.Errorf("RozmiarFolderu = %d, %v; oczekiwano %d", r, err, suma-1)
	}
	if r, err := m.RozmiarFolderu("dane", "/b.txt"); err != nil || r != 1 {
		t.Errorf("RozmiarFolderu(plik) = %d, %v", r, err)
	}
	buf := &bytes.Buffer{}
	n, err := m.ZipDo(buf, "dane", []string{"/f", "/b.txt", "/b.txt"})
	if err != nil || n != suma+1 {
		t.Fatalf("ZipDo = %d, %v; oczekiwano %d", n, err, suma+1)
	}
	if buf.Len() > len(duzy)/10 {
		t.Errorf("tekst nie został skompresowany: %d bajtów", buf.Len())
	}
	wyn := rozpakuj(t, buf.Bytes())
	for s, tresc := range pliki {
		if got, ok := wyn[s]; !ok || got.tresc != tresc {
			t.Errorf("ZIP: %s brak albo zła treść", s)
		}
	}
	if wyn["b (2).txt"].tresc != "b" {
		t.Error("powtórzona nazwa nie dostała przyrostka (2)")
	}
	metody := map[string]uint16{"f/zdjecie.JPG": zip.Store, "f/pod/film.mp4": zip.Store, "f/a.txt": zip.Deflate}
	for s, met := range metody {
		if wyn[s].metoda != met {
			t.Errorf("%s: metoda %d, oczekiwano %d", s, wyn[s].metoda, met)
		}
	}
	if _, ok := wyn["f/pusty/"]; !ok {
		t.Error("brak pustego folderu w ZIP")
	}
	if _, err := m.ZipDo(io.Discard, "dane", []string{"/nie"}); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("ZipDo(brak) = %v", err)
	}
}

type pozycjaZip struct {
	tresc  string
	metoda uint16
}

// rozpakuj czyta całe archiwum do mapy nazwa → treść.
func rozpakuj(t *testing.T, dane []byte) map[string]pozycjaZip {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(dane), int64(len(dane)))
	if err != nil {
		t.Fatalf("niepoprawny ZIP: %v", err)
	}
	wyn := map[string]pozycjaZip{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close() // celowo: test
		if err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		wyn[f.Name] = pozycjaZip{string(b), f.Method}
	}
	return wyn
}
