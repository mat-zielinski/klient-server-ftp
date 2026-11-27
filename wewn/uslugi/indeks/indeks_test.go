package indeks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

var czasTestu = time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)

// drzewo zakłada pliki (ścieżka względna -> treść) w katalogu.
func drzewo(t *testing.T, kat string, pliki map[string]string) {
	t.Helper()
	for p, tresc := range pliki {
		pelna := filepath.Join(kat, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(pelna), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pelna, []byte(tresc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// nowyIndeks: magazyn z udziałem "dom" w t.TempDir i indeks bez przerw.
func nowyIndeks(t *testing.T, pliki map[string]string) (*Indeks, string) {
	t.Helper()
	kat := t.TempDir()
	drzewo(t, kat, pliki)
	m, err := magazyn.Nowy([]konfiguracja.Udzial{{Nazwa: "dom", Sciezka: kat}})
	if err != nil {
		t.Fatal(err)
	}
	x := Nowy(m, ZPrzerwa(0), ZOpoznieniem(0), ZRownolegloscia(3), ZZegarem(func() time.Time { return czasTestu }))
	return x, kat
}

var plikiTestowe = map[string]string{
	"Zdjęcia/Wakacje 2024/Żółw na plaży.JPG": "jpg",
	"Zdjęcia/Wakacje 2024/zachód.png":        "png!",
	"Dokumenty/raport roczny.pdf":            "pdf12",
	"Dokumenty/Raport kwartalny.docx":        "docx",
	"notatki.txt":                            "tekst",
	"desktop.ini":                            "ukryty",
	".skrytka_kosz/2026/raport stary.txt":    "kosz",
}

func sciezki(w []rdzen.Wpis) []string {
	var s []string
	for _, x := range w {
		s = append(s, x.Sciezka)
	}
	sort.Strings(s)
	return s
}

func TestSzukaj(t *testing.T) {
	x, _ := nowyIndeks(t, plikiTestowe)
	if x.Gotowy("dom") || x.Szukaj("dom", "/", "raport", 0) != nil {
		t.Fatal("indeks gotowy przed przejściem")
	}
	if err := x.Przeindeksuj(context.Background(), "dom"); err != nil {
		t.Fatal(err)
	}
	if !x.Gotowy("dom") || x.Gotowy("obcy") {
		t.Fatal("Gotowy")
	}
	przypadki := []struct {
		start, q string
		limit    int
		chce     string
	}{
		{"/", "raport", 0, "[/Dokumenty/Raport kwartalny.docx /Dokumenty/raport roczny.pdf]"},
		{"/", "raport roczny", 0, "[/Dokumenty/raport roczny.pdf]"},
		{"/", "  roczny   RAPORT ", 0, "[/Dokumenty/raport roczny.pdf]"},
		{"/", "raport zolw", 0, "[]"},
		{"/", "zolw", 0, "[/Zdjęcia/Wakacje 2024/Żółw na plaży.JPG]"},
		{"/", "ŻÓŁW plaz", 0, "[/Zdjęcia/Wakacje 2024/Żółw na plaży.JPG]"},
		{"/", "zdjecia", 0, "[/Zdjęcia]"},
		{"/", "wakacje", 0, "[/Zdjęcia/Wakacje 2024]"},
		{"/Dokumenty", "raport", 0, "[/Dokumenty/Raport kwartalny.docx /Dokumenty/raport roczny.pdf]"},
		{"Zdjęcia", "raport", 0, "[]"},
		{"/Zdjęcia", ".", 0, "[/Zdjęcia/Wakacje 2024/zachód.png /Zdjęcia/Wakacje 2024/Żółw na plaży.JPG]"},
		{"/Dok", "raport", 0, "[]"}, // przedrostek nazwy to nie podfolder
		{"/", "desktop", 0, "[]"},
		{"/", "stary", 0, "[]"}, // kosz ukryty
		{"/", "", 2, "[/Dokumenty /Zdjęcia]"},
	}
	for _, p := range przypadki {
		w := x.Szukaj("dom", p.start, p.q, p.limit)
		if got := fmt.Sprint(sciezki(w)); got != p.chce && !(p.chce == "[]" && len(w) == 0) {
			t.Errorf("Szukaj(%q, %q) = %s, chcę %s", p.start, p.q, got, p.chce)
		}
	}
	for _, zla := range []string{"../", "/a/../../b", "C:/Windows"} {
		if w := x.Szukaj("dom", zla, "", 0); w != nil {
			t.Errorf("start %q: %v", zla, w)
		}
	}
	if w := x.Szukaj("obcy", "/", "", 0); w != nil {
		t.Errorf("obcy udział: %v", w)
	}
}

func TestWpisZIndeksu(t *testing.T) {
	x, kat := nowyIndeks(t, plikiTestowe)
	if err := x.Przeindeksuj(context.Background(), "dom"); err != nil {
		t.Fatal(err)
	}
	w := x.Szukaj("dom", "/", "roczny", 0)
	fi, err := os.Stat(filepath.Join(kat, "Dokumenty", "raport roczny.pdf"))
	if err != nil || len(w) != 1 {
		t.Fatal(err, w)
	}
	chce := rdzen.Wpis{Nazwa: "raport roczny.pdf", Sciezka: "/Dokumenty/raport roczny.pdf", Rozmiar: 5,
		Zmieniono: fi.ModTime(), Rodzaj: rdzen.RodzajPDF}
	if !w[0].Zmieniono.Equal(chce.Zmieniono) {
		t.Errorf("czas %v != %v", w[0].Zmieniono, chce.Zmieniono)
	}
	w[0].Zmieniono = chce.Zmieniono
	if w[0] != chce {
		t.Errorf("%+v", w[0])
	}
}

func TestLimitSzukania(t *testing.T) {
	pliki := map[string]string{}
	for i := range konfiguracja.MagazynMaksLimitSzukania + 10 {
		pliki[fmt.Sprintf("f/p%05d.txt", i)] = ""
	}
	x, _ := nowyIndeks(t, pliki)
	if err := x.Przeindeksuj(context.Background(), "dom"); err != nil {
		t.Fatal(err)
	}
	przypadki := []struct{ limit, chce int }{
		{0, konfiguracja.MagazynDomyslnyLimitSzukania},
		{-1, konfiguracja.MagazynDomyslnyLimitSzukania},
		{7, 7},
		{1 << 30, konfiguracja.MagazynMaksLimitSzukania},
	}
	for _, p := range przypadki {
		if n := len(x.Szukaj("dom", "/", "p", p.limit)); n != p.chce {
			t.Errorf("limit %d: %d wyników, chcę %d", p.limit, n, p.chce)
		}
	}
}

func TestOdswiezenie(t *testing.T) {
	x, kat := nowyIndeks(t, plikiTestowe)
	ctx := context.Background()
	if err := x.Przeindeksuj(ctx, "dom"); err != nil {
		t.Fatal(err)
	}
	drzewo(t, kat, map[string]string{"Dokumenty/raport nowy.txt": "x"})
	if err := os.Remove(filepath.Join(kat, "Dokumenty", "raport roczny.pdf")); err != nil {
		t.Fatal(err)
	}
	if n := len(x.Szukaj("dom", "/", "raport", 0)); n != 2 {
		t.Fatalf("przed odświeżeniem %d", n)
	}
	if err := x.Przeindeksuj(ctx, "dom"); err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprint(sciezki(x.Szukaj("dom", "/", "raport", 0)))
	if got != "[/Dokumenty/Raport kwartalny.docx /Dokumenty/raport nowy.txt]" {
		t.Errorf("po odświeżeniu %s", got)
	}
	if err := x.Przeindeksuj(ctx, "obcy"); err == nil {
		t.Error("nieznany udział bez błędu")
	}
}

func TestPrzerwanePrzejscieZostawiaStaryZbior(t *testing.T) {
	x, kat := nowyIndeks(t, plikiTestowe)
	if err := x.Przeindeksuj(context.Background(), "dom"); err != nil {
		t.Fatal(err)
	}
	drzewo(t, kat, map[string]string{"nowy raport.txt": ""})
	ctx, anuluj := context.WithCancel(context.Background())
	anuluj()
	if err := x.Przeindeksuj(ctx, "dom"); err == nil {
		t.Fatal("przerwane przejście bez błędu")
	}
	if n := len(x.Szukaj("dom", "/", "raport", 0)); n != 2 {
		t.Errorf("stary zbiór utracony: %d", n)
	}
}

func TestUruchomIOdswiez(t *testing.T) {
	x, kat := nowyIndeks(t, plikiTestowe)
	przejscia := make(chan string, 8)
	x.poPrzejsciu = func(u string) { przejscia <- u }
	ctx, anuluj := context.WithCancel(context.Background())
	defer anuluj()
	x.Uruchom(ctx)
	czekajNa := func() {
		t.Helper()
		select {
		case <-przejscia:
		case <-time.After(10 * time.Second):
			t.Fatal("brak przejścia")
		}
	}
	czekajNa()
	if !x.Gotowy("dom") {
		t.Fatal("niegotowy po pierwszym przejściu")
	}
	drzewo(t, kat, map[string]string{"świeży.txt": ""})
	x.Odswiez("dom")
	x.Odswiez("obcy") // nieznany: nic
	czekajNa()
	if n := len(x.Szukaj("dom", "/", "swiezy", 0)); n != 1 {
		t.Errorf("po Odswiez %d", n)
	}
}

func TestStatystyki(t *testing.T) {
	x, _ := nowyIndeks(t, plikiTestowe)
	s := x.Statystyki()
	if len(s) != 1 || s[0].Nazwa != "dom" || s[0].Pliki != 0 || !s[0].Zindeksowano.IsZero() || s[0].WgRodzaju == nil {
		t.Fatalf("przed przejściem %+v", s)
	}
	if err := x.Przeindeksuj(context.Background(), "dom"); err != nil {
		t.Fatal(err)
	}
	s = x.Statystyki()
	chce := StatystykaUdzialu{Nazwa: "dom", Pliki: 5, Foldery: 3, Bajty: 3 + 4 + 5 + 4 + 5, Zindeksowano: czasTestu}
	got := s[0]
	wg := got.WgRodzaju
	got.WgRodzaju = nil
	if fmt.Sprint(got) != fmt.Sprint(chce) {
		t.Errorf("%+v, chcę %+v", got, chce)
	}
	chceRodzaje := map[string]LicznikRodzaju{
		rdzen.RodzajObraz: {2, 7}, rdzen.RodzajPDF: {1, 5}, rdzen.RodzajTekst: {1, 5}, rdzen.RodzajInny: {1, 4},
	}
	if fmt.Sprint(wg) != fmt.Sprint(chceRodzaje) {
		t.Errorf("wg rodzaju %v, chcę %v", wg, chceRodzaje)
	}
	wg[rdzen.RodzajObraz] = LicznikRodzaju{}
	if x.Statystyki()[0].WgRodzaju[rdzen.RodzajObraz].Pliki != 2 {
		t.Error("wynik Statystyki współdzieli mapę z indeksem")
	}
}

// Zwijanie indeksu musi dawać to samo co rdzen.Pasuje (tabela ogonków skopiowana).
func TestZgodnoscZRdzeniem(t *testing.T) {
	nazwy := []string{"Żółw.JPG", "ĄĆĘŁŃÓŚŹŻ", "zażółć gęślą jaźń", "Ärger Öl Über", "Ñandú", "Ελληνικά", "plain.txt", "ŠKODA"}
	zapytania := []string{"zolw", "acelnoszz", "zazolc", "gesla", "jazn", "arger", "ol", "uber", "nandu", "ελλ", "PLAIN", "skoda", "x", ""}
	for _, n := range nazwy {
		for _, q := range zapytania {
			if a, b := pasujeWszystkie(zwin(n), slowaZapytania(q)), rdzen.Pasuje(n, q); a != b {
				t.Errorf("%q / %q: indeks %v, rdzen %v", n, q, a, b)
			}
		}
	}
}

func TestSlowaZapytania(t *testing.T) {
	przypadki := []struct{ q, chce string }{
		{"", "[]"},
		{"  ", "[]"},
		{"a bbb cc", "[bbb cc a]"},
		{"Żółw  PLAŻA", "[plaza zolw]"},
	}
	for _, p := range przypadki {
		if got := fmt.Sprint(slowaZapytania(p.q)); got != p.chce {
			t.Errorf("%q: %s, chcę %s", p.q, got, p.chce)
		}
	}
}
