package dziennik

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

// zegar: wstrzykiwany czas (bez time.Sleep).
type zegar struct {
	mu sync.Mutex
	t  time.Time
}

func (z *zegar) teraz() time.Time {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.t
}

func (z *zegar) ustaw(t time.Time) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.t = t
}

func data(r int, m time.Month, d, g, min int) time.Time {
	return time.Date(r, m, d, g, min, 0, 0, time.Local)
}

func nowy(t *testing.T, z *zegar, o Opcje) (*Dziennik, string) {
	t.Helper()
	kat := filepath.Join(t.TempDir(), "logi")
	o.Teraz = z.teraz
	d, err := OtworzZOpcjami(kat, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Zamknij() }) // celowo: test mógł już zamknąć
	return d, kat
}

func TestZapisIOdczyt(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 10, 0)}
	d, kat := nowy(t, z, Opcje{})
	d.Zapisz(Wpis{Rodzaj: RodzajLogowanie, Kto: "jan", IP: "10.0.0.2"})
	dlugi := strings.Repeat("ż", konfiguracja.DziennikOpisMax+20)
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 11, 0), Rodzaj: RodzajPobranie, Kto: "jan", Opis: dlugi})

	wpisy, err := d.Czytaj(Filtr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(wpisy) != 2 {
		t.Fatalf("liczba wpisów %d, oczekiwano 2", len(wpisy))
	}
	if wpisy[0].Rodzaj != RodzajPobranie || wpisy[1].Rodzaj != RodzajLogowanie {
		t.Errorf("zła kolejność (najnowsze pierwsze): %v, %v", wpisy[0].Rodzaj, wpisy[1].Rodzaj)
	}
	if !wpisy[1].Czas.Equal(z.teraz()) {
		t.Errorf("pusty czas nie został uzupełniony zegarem: %v", wpisy[1].Czas)
	}
	if n := len([]rune(wpisy[0].Opis)); n != konfiguracja.DziennikOpisMax {
		t.Errorf("opis nie przycięty: %d znaków", n)
	}
	if err := d.Zamknij(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(kat, "dziennik_2026-11.jsonl")); err != nil {
		t.Errorf("brak pliku miesiąca: %v", err)
	}
	// Ponowne otwarcie czyta to samo z dysku.
	d2, err := OtworzZOpcjami(kat, Opcje{Teraz: z.teraz})
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Zamknij()
	if w, _ := d2.Czytaj(Filtr{}); len(w) != 2 {
		t.Errorf("po ponownym otwarciu %d wpisów", len(w))
	}
}

func TestZamknijDwaRazyIZapisPoZamknieciu(t *testing.T) {
	z := &zegar{t: data(2026, 11, 1, 0, 0)}
	d, _ := nowy(t, z, Opcje{})
	if err := d.Zamknij(); err != nil {
		t.Fatal(err)
	}
	d.Zapisz(Wpis{Rodzaj: RodzajSerwer}) // nie może wywołać paniki
	if err := d.Zamknij(); err != nil {
		t.Errorf("drugie Zamknij: %v", err)
	}
}

func TestPrzelomMiesiaca(t *testing.T) {
	z := &zegar{t: data(2026, 11, 1, 0, 5)}
	d, kat := nowy(t, z, Opcje{})
	d.Zapisz(Wpis{Czas: data(2026, 10, 31, 23, 59), Rodzaj: RodzajPobranie, Sciezka: "/pazdziernik"})
	d.Zapisz(Wpis{Czas: data(2026, 11, 1, 0, 1), Rodzaj: RodzajPobranie, Sciezka: "/listopad"})
	d.Zapisz(Wpis{Czas: data(2026, 9, 15, 12, 0), Rodzaj: RodzajPobranie, Sciezka: "/wrzesien"})

	wszystkie, err := d.Czytaj(Filtr{})
	if err != nil {
		t.Fatal(err)
	}
	var sciezki []string
	for _, w := range wszystkie {
		sciezki = append(sciezki, w.Sciezka)
	}
	if got := strings.Join(sciezki, ","); got != "/listopad,/pazdziernik,/wrzesien" {
		t.Errorf("kolejność przez miesiące: %s", got)
	}
	for _, m := range []string{"2026-09", "2026-10", "2026-11"} {
		if _, err := os.Stat(filepath.Join(kat, "dziennik_"+m+".jsonl")); err != nil {
			t.Errorf("brak pliku %s: %v", m, err)
		}
	}
	paz, _ := d.Czytaj(Filtr{Od: data(2026, 10, 1, 0, 0), Do: data(2026, 11, 1, 0, 0)})
	if len(paz) != 1 || paz[0].Sciezka != "/pazdziernik" {
		t.Errorf("zakres październik: %+v", paz)
	}
}

func TestFiltr(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 8, 0), Rodzaj: RodzajPobranie, Kto: "jan", Sciezka: "/Filmy/Lato.mkv"})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 9, 0), Rodzaj: RodzajPobranie, Kto: "ola", Sciezka: "/a.txt"})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 10, 0), Rodzaj: RodzajLogowanie, Kto: "jan", Opis: "Udane LATO"})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 11, 0), Rodzaj: RodzajZmiana, Kto: "ola"})

	przypadki := []struct {
		nazwa string
		f     Filtr
		ile   int
	}{
		{"bez filtru", Filtr{}, 4},
		{"rodzaj", Filtr{Rodzaj: RodzajPobranie}, 2},
		{"kto", Filtr{Kto: "jan"}, 2},
		{"szukaj w ścieżce i opisie bez wielkości liter", Filtr{Szukaj: "lato"}, 2},
		{"limit", Filtr{Limit: 3}, 3},
		{"od włącznie", Filtr{Od: data(2026, 11, 27, 10, 0)}, 2},
		{"do wyłącznie", Filtr{Do: data(2026, 11, 27, 10, 0)}, 2},
		{"łącznie", Filtr{Kto: "ola", Rodzaj: RodzajPobranie, Szukaj: "a.txt"}, 1},
		{"nic", Filtr{Kto: "nikt"}, 0},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w, err := d.Czytaj(p.f)
			if err != nil {
				t.Fatal(err)
			}
			if len(w) != p.ile {
				t.Errorf("%d wpisów, oczekiwano %d", len(w), p.ile)
			}
		})
	}
}

func TestUszkodzonaLiniaPomijana(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, kat := nowy(t, z, Opcje{})
	plik := filepath.Join(kat, "dziennik_2026-10.jsonl")
	tresc := `{"czas":"2026-10-02T10:00:00Z","rodzaj":"pobranie"}` + "\n{zepsute\n" +
		`{"czas":"2026-10-03T10:00:00Z","rodzaj":"pobranie"}`
	if err := os.WriteFile(plik, []byte(tresc), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := d.Czytaj(Filtr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 2 {
		t.Errorf("%d wpisów, oczekiwano 2 (uszkodzona linia pominięta)", len(w))
	}
}

func TestPamiecZamknietegoMiesiaca(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	d.Zapisz(Wpis{Czas: data(2026, 10, 5, 10, 0), Rodzaj: RodzajPobranie})
	d.Zapisz(Wpis{Czas: data(2026, 11, 5, 10, 0), Rodzaj: RodzajPobranie})
	if w, _ := d.Czytaj(Filtr{}); len(w) != 2 {
		t.Fatalf("%d wpisów", len(w))
	}
	d.pamiec.mu.Lock()
	_, pazW := d.pamiec.pozycje["2026-10"]
	_, lisW := d.pamiec.pozycje["2026-11"]
	d.pamiec.mu.Unlock()
	if !pazW || lisW {
		t.Errorf("pamięć: październik %v (oczekiwano true), listopad %v (oczekiwano false)", pazW, lisW)
	}
	// Dopisanie do zamkniętego miesiąca (np. pobranie z czasem sprzed północy) unieważnia pamięć.
	d.Zapisz(Wpis{Czas: data(2026, 10, 31, 23, 0), Rodzaj: RodzajPobranie})
	if w, _ := d.Czytaj(Filtr{}); len(w) != 3 {
		t.Errorf("po dopisaniu do zamkniętego miesiąca %d wpisów, oczekiwano 3", len(w))
	}
}

func TestPamiecWyrzucaNajstarsze(t *testing.T) {
	p := nowaPamiec(2)
	info := fakeInfo{}
	for _, k := range []string{"2026-01", "2026-02", "2026-03"} {
		p.wstaw(k, info, nil)
	}
	if _, ok := p.pozycje["2026-01"]; ok || len(p.pozycje) != 2 {
		t.Errorf("pamięć nie wyrzuciła najstarszej pozycji: %v", p.kolejnosc)
	}
}

type fakeInfo struct{ os.FileInfo }

func (fakeInfo) Size() int64        { return 1 }
func (fakeInfo) ModTime() time.Time { return time.Time{} }

func TestRownolegleZapisz(t *testing.T) {
	const gorutyny, naGorutyne = 20, 100
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{Bufor: gorutyny * naGorutyne})
	var wg sync.WaitGroup
	for g := range gorutyny {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range naGorutyne {
				d.Zapisz(Wpis{Rodzaj: RodzajPobranie, Kto: fmt.Sprintf("k%d", g), Bajty: int64(i)})
			}
		}()
	}
	wg.Wait()
	w, err := d.Czytaj(Filtr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != gorutyny*naGorutyne {
		t.Errorf("%d wpisów, oczekiwano %d", len(w), gorutyny*naGorutyne)
	}
}

func TestPrzepelnienieBufora(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	wszedl, pusc := make(chan struct{}), make(chan struct{})
	var raz sync.Once
	hak := func() { raz.Do(func() { close(wszedl); <-pusc }) }
	d, _ := nowy(t, z, Opcje{Bufor: 1, przedWpisem: hak})

	d.Zapisz(Wpis{Rodzaj: RodzajPobranie, Opis: "1"}) // gorutyna bierze go i staje w haku
	<-wszedl
	d.Zapisz(Wpis{Rodzaj: RodzajPobranie, Opis: "2"}) // zajmuje jedyne miejsce w kanale
	d.Zapisz(Wpis{Rodzaj: RodzajPobranie, Opis: "3"}) // przepada
	d.Zapisz(Wpis{Rodzaj: RodzajPobranie, Opis: "4"}) // przepada
	close(pusc)

	w, err := d.Czytaj(Filtr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 3 {
		t.Fatalf("%d wpisów, oczekiwano 3 (dwa zapisane + wpis o przepełnieniu)", len(w))
	}
	o, _ := d.Czytaj(Filtr{Rodzaj: RodzajSerwer})
	if len(o) != 1 || !strings.Contains(o[0].Opis, "utracono 2") {
		t.Errorf("brak wpisu o przepełnieniu: %+v", o)
	}
}

func TestPrzytnij(t *testing.T) {
	przypadki := []struct {
		wej string
		max int
		wyj string
	}{
		{"", 3, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"żółć", 2, "żó"},
		{"ąę", 5, "ąę"},
	}
	for _, p := range przypadki {
		if got := przytnij(p.wej, p.max); got != p.wyj {
			t.Errorf("przytnij(%q, %d) = %q, oczekiwano %q", p.wej, p.max, got, p.wyj)
		}
	}
}

func TestKluczZNazwy(t *testing.T) {
	przypadki := []struct {
		nazwa string
		klucz string
		ok    bool
	}{
		{"dziennik_2026-11.jsonl", "2026-11", true},
		{"dziennik_2026-13.jsonl", "", false},
		{"dziennik_2026-1.jsonl", "", false},
		{"bledy.log", "", false},
		{"dziennik_2026-11.jsonl.tmp", "", false},
	}
	for _, p := range przypadki {
		k, ok := kluczZNazwy(p.nazwa)
		if k != p.klucz || ok != p.ok {
			t.Errorf("kluczZNazwy(%q) = %q, %v", p.nazwa, k, ok)
		}
	}
}
