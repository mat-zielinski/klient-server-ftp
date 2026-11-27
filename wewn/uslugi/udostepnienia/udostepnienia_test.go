package udostepnienia

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// zegar: wstrzykiwany czas, przesuwany ręcznie (bez time.Sleep).
type zegar struct {
	mu sync.Mutex
	t  time.Time
}

func (z *zegar) teraz() time.Time { z.mu.Lock(); defer z.mu.Unlock(); return z.t }
func (z *zegar) przesun(d time.Duration) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.t = z.t.Add(d)
}

var poczatek = time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)

func nowe(t *testing.T) (*Udostepnienia, *zegar, string) {
	t.Helper()
	plik := filepath.Join(t.TempDir(), konfiguracja.PlikUdostepnienNazwa)
	u, err := Otworz(plik)
	if err != nil {
		t.Fatalf("Otworz: %v", err)
	}
	z := &zegar{t: poczatek}
	u.UstawZegar(z.teraz)
	return u, z, plik
}

func utworz(t *testing.T, u *Udostepnienia, l Link, haslo string) Link {
	t.Helper()
	if l.Udzial == "" {
		l.Udzial = "dom"
	}
	if l.Autor == "" {
		l.Autor = "ala"
	}
	wynik, err := u.Utworz(l, haslo)
	if err != nil {
		t.Fatalf("Utworz: %v", err)
	}
	return wynik
}

func TestUtworzWalidacja(t *testing.T) {
	u, _, _ := nowe(t)
	przypadki := []struct {
		nazwa string
		link  Link
		haslo string
		blad  error
	}{
		{"brak udziału", Link{Sciezka: "/a"}, "", ErrZlyLink},
		{"ujemny limit", Link{Udzial: "dom", Sciezka: "/a", LimitPobran: -1}, "", ErrZlyLink},
		{"data w przeszłości", Link{Udzial: "dom", Sciezka: "/a", Wygasa: poczatek.Add(-time.Hour)}, "", ErrZlyLink},
		{"data równa teraz", Link{Udzial: "dom", Sciezka: "/a", Wygasa: poczatek}, "", ErrZlyLink},
		{"za długie hasło", Link{Udzial: "dom", Sciezka: "/a"}, strings.Repeat("x", konfiguracja.MaksDlugoscHaslaLinku+1), ErrZlyLink},
		{"wyjście z udziału", Link{Udzial: "dom", Sciezka: "/../tajne"}, "", rdzen.ErrPozaUdzialem},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if _, err := u.Utworz(p.link, p.haslo); !errors.Is(err, p.blad) {
				t.Fatalf("błąd %v, oczekiwano %v", err, p.blad)
			}
		})
	}
	if n := len(u.Lista("")); n != 0 {
		t.Fatalf("odrzucone linki trafiły na listę: %d", n)
	}
}

func TestUtworzPolaIWygasniecie(t *testing.T) {
	u, _, _ := nowe(t)
	przypadki := []struct {
		nazwa  string
		wygasa time.Time
		chce   time.Time
	}{
		{"domyślna ważność", time.Time{}, poczatek.Add(konfiguracja.DomyslnaWaznoscLinku)},
		{"własna data", poczatek.Add(48 * time.Hour), poczatek.Add(48 * time.Hour)},
		{"przycięcie do maksimum", poczatek.Add(2 * konfiguracja.MaksWaznoscLinku), poczatek.Add(konfiguracja.MaksWaznoscLinku)},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			l := utworz(t, u, Link{Sciezka: "zdjecia//a.jpg", Wygasa: p.wygasa, Pobrano: 99, ID: "podane"}, "")
			if !l.Wygasa.Equal(p.chce) {
				t.Fatalf("Wygasa %v, oczekiwano %v", l.Wygasa, p.chce)
			}
			if l.Sciezka != "/zdjecia/a.jpg" || l.Pobrano != 0 || l.ID == "podane" || !l.Utworzono.Equal(poczatek) {
				t.Fatalf("niepoprawne pola: %+v", l)
			}
			if len(l.ID) != 22 { // 16 B w base64url bez dopełnienia
				t.Fatalf("długość ID %d", len(l.ID))
			}
		})
	}
}

func TestWygasanie(t *testing.T) {
	u, z, plik := nowe(t)
	krotki := utworz(t, u, Link{Sciezka: "/a", Wygasa: poczatek.Add(time.Hour)}, "")
	dlugi := utworz(t, u, Link{Sciezka: "/b", Wygasa: poczatek.Add(10 * time.Hour)}, "")
	if _, err := u.Sprawdz(krotki.ID, ""); err != nil {
		t.Fatalf("przed wygaśnięciem: %v", err)
	}
	z.przesun(time.Hour)
	if _, err := u.Sprawdz(krotki.ID, ""); !errors.Is(err, rdzen.ErrWygaslo) {
		t.Fatalf("po wygaśnięciu: %v", err)
	}
	if err := u.ZaliczPobranie(krotki.ID); !errors.Is(err, rdzen.ErrWygaslo) {
		t.Fatalf("pobranie po wygaśnięciu: %v", err)
	}
	if l := u.Lista(""); len(l) != 1 || l[0].ID != dlugi.ID {
		t.Fatalf("lista po wygaśnięciu: %+v", l)
	}
	if n, err := u.Sprzataj(); err != nil || n != 1 {
		t.Fatalf("Sprzataj: %d, %v", n, err)
	}
	if _, err := u.Sprawdz(krotki.ID, ""); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("po sprzątnięciu: %v", err)
	}
	dane, err := os.ReadFile(plik)
	if err != nil || strings.Contains(string(dane), krotki.ID) {
		t.Fatalf("wygasły link nadal w pliku (%v)", err)
	}
}

func TestOtworzSprzataWygasle(t *testing.T) {
	plik := filepath.Join(t.TempDir(), konfiguracja.PlikUdostepnienNazwa)
	// Otworz działa na prawdziwym czasie, więc daty liczymy od time.Now.
	stary := Link{ID: "stary", Udzial: "d", Sciezka: "/a", Wygasa: time.Now().Add(-time.Hour)}
	nowy := Link{ID: "nowy", Udzial: "d", Sciezka: "/b", Wygasa: time.Now().Add(time.Hour)}
	if err := zapisz(plik, map[string]Link{stary.ID: stary, nowy.ID: nowy}); err != nil {
		t.Fatal(err)
	}
	u, err := Otworz(plik)
	if err != nil {
		t.Fatalf("Otworz: %v", err)
	}
	if _, err := u.Pobierz("stary"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("wygasły po otwarciu: %v", err)
	}
	if _, err := u.Pobierz("nowy"); err != nil {
		t.Fatalf("ważny po otwarciu: %v", err)
	}
	dane, err := os.ReadFile(plik)
	if err != nil || strings.Contains(string(dane), "stary") {
		t.Fatalf("wygasły nadal w pliku (%v)", err)
	}
}

func TestLimitPobran(t *testing.T) {
	u, _, _ := nowe(t)
	l := utworz(t, u, Link{Sciezka: "/plik.txt", LimitPobran: 2}, "")
	przypadki := []struct {
		nazwa string
		blad  error
	}{
		{"pierwsze", nil},
		{"drugie", nil},
		{"trzecie ponad limit", rdzen.ErrWygaslo},
	}
	for _, p := range przypadki {
		if err := u.ZaliczPobranie(l.ID); !errors.Is(err, p.blad) {
			t.Fatalf("%s: %v, oczekiwano %v", p.nazwa, err, p.blad)
		}
	}
	if _, err := u.Sprawdz(l.ID, ""); !errors.Is(err, rdzen.ErrWygaslo) {
		t.Fatalf("Sprawdz po limicie: %v", err)
	}
	if err := u.ZaliczPobranie("brak"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("nieznany link: %v", err)
	}
	bez := utworz(t, u, Link{Sciezka: "/b"}, "")
	for i := 0; i < 10; i++ {
		if err := u.ZaliczPobranie(bez.ID); err != nil {
			t.Fatalf("bez limitu, pobranie %d: %v", i, err)
		}
	}
}

func TestHasloLinku(t *testing.T) {
	u, _, _ := nowe(t)
	z := utworz(t, u, Link{Sciezka: "/a"}, "tajne hasło")
	bez := utworz(t, u, Link{Sciezka: "/b"}, "")
	if !z.MaHaslo || z.HashHasla != "" || bez.MaHaslo {
		t.Fatalf("pola hasła: %+v / %+v", z, bez)
	}
	przypadki := []struct {
		nazwa     string
		id, haslo string
		blad      error
	}{
		{"dobre hasło", z.ID, "tajne hasło", nil},
		{"złe hasło", z.ID, "tajne haslo", rdzen.ErrZleDane},
		{"puste hasło", z.ID, "", rdzen.ErrZleDane},
		{"za długie hasło", z.ID, strings.Repeat("x", konfiguracja.MaksDlugoscHaslaLinku+1), rdzen.ErrZleDane},
		{"link bez hasła", bez.ID, "cokolwiek", nil},
		{"nieznany link", "nie-ma", "", rdzen.ErrNieZnaleziono},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			l, err := u.Sprawdz(p.id, p.haslo)
			if !errors.Is(err, p.blad) {
				t.Fatalf("błąd %v, oczekiwano %v", err, p.blad)
			}
			if err == nil && l.HashHasla != "" {
				t.Fatal("Sprawdz oddał hash hasła")
			}
		})
	}
}

func TestBlokadaProbHasla(t *testing.T) {
	u, z, _ := nowe(t)
	l := utworz(t, u, Link{Sciezka: "/a"}, "dobre")
	inny := utworz(t, u, Link{Sciezka: "/b"}, "dobre")
	for i := 0; i < konfiguracja.LimitProbHaslaLinku; i++ {
		if _, err := u.Sprawdz(l.ID, "złe"); !errors.Is(err, rdzen.ErrZleDane) {
			t.Fatalf("próba %d: %v", i, err)
		}
	}
	// Po limicie nawet dobre hasło jest odrzucane (inaczej zgadywanie trwa dalej).
	if _, err := u.Sprawdz(l.ID, "dobre"); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("po limicie: %v", err)
	}
	if _, err := u.Sprawdz(inny.ID, "dobre"); err != nil {
		t.Fatalf("blokada objęła inny link: %v", err)
	}
	z.przesun(konfiguracja.OknoProbHaslaLinku)
	if _, err := u.Sprawdz(l.ID, "dobre"); err != nil {
		t.Fatalf("po oknie: %v", err)
	}
}

func TestDobreHasloZerujeProby(t *testing.T) {
	u, _, _ := nowe(t)
	l := utworz(t, u, Link{Sciezka: "/a"}, "dobre")
	for runda := 0; runda < 3; runda++ {
		for i := 0; i < konfiguracja.LimitProbHaslaLinku-1; i++ {
			if _, err := u.Sprawdz(l.ID, "złe"); !errors.Is(err, rdzen.ErrZleDane) {
				t.Fatalf("runda %d próba %d: %v", runda, i, err)
			}
		}
		if _, err := u.Sprawdz(l.ID, "dobre"); err != nil {
			t.Fatalf("runda %d: dobre hasło: %v", runda, err)
		}
	}
}

func TestBlokadaRownolegla(t *testing.T) {
	u, _, _ := nowe(t)
	l := utworz(t, u, Link{Sciezka: "/a"}, "dobre")
	const proby = 4 * konfiguracja.LimitProbHaslaLinku
	var wg sync.WaitGroup
	var mu sync.Mutex
	zle := 0
	for i := 0; i < proby; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := u.Sprawdz(l.ID, "złe")
			if errors.Is(err, rdzen.ErrZleDane) {
				mu.Lock()
				zle++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if zle != konfiguracja.LimitProbHaslaLinku {
		t.Fatalf("sprawdzono %d haseł, limit %d", zle, konfiguracja.LimitProbHaslaLinku)
	}
}

func TestPlikLinkow(t *testing.T) {
	u, _, plik := nowe(t)
	z := utworz(t, u, Link{Sciezka: "/a", Autor: "ala", LimitPobran: 3}, "hasło")
	b := utworz(t, u, Link{Sciezka: "/b", Autor: "ola"}, "")
	if err := u.ZaliczPobranie(z.ID); err != nil {
		t.Fatal(err)
	}
	dane, err := os.ReadFile(plik)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dane), "$argon2id$") || strings.Contains(string(dane), "hasło") {
		t.Fatal("plik bez hasha argon2id albo z jawnym hasłem")
	}
	u2, err := Otworz(plik)
	if err != nil {
		t.Fatalf("ponowne Otworz: %v", err)
	}
	ali := u2.Lista("ala")
	if len(ali) != 1 || ali[0].ID != z.ID || ali[0].Pobrano != 1 || !ali[0].MaHaslo || ali[0].HashHasla != "" {
		t.Fatalf("lista po wczytaniu: %+v", ali)
	}
	if _, err := u2.Sprawdz(z.ID, "hasło"); err != nil {
		t.Fatalf("hasło po wczytaniu: %v", err)
	}
	if err := u2.Usun(b.ID); err != nil {
		t.Fatal(err)
	}
	if err := u2.Usun(b.ID); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("ponowne Usun: %v", err)
	}
	u3, err := Otworz(plik)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(u3.Lista("")); n != 1 {
		t.Fatalf("po usunięciu w pliku %d linków", n)
	}
}

func TestOtworzBledy(t *testing.T) {
	katalog := t.TempDir()
	zepsuty := filepath.Join(katalog, "zepsuty.json")
	if err := os.WriteFile(zepsuty, []byte("{nie json"), 0o600); err != nil {
		t.Fatal(err)
	}
	niespojny := filepath.Join(katalog, "niespojny.json")
	if err := os.WriteFile(niespojny, []byte(`[{"id":"x","udzial":"d","sciezka":"/","ma_haslo":true}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Otworz(zepsuty); err == nil {
		t.Fatal("zepsuty plik przyjęty")
	}
	if u, err := Otworz(filepath.Join(katalog, "brak.json")); err != nil || len(u.Lista("")) != 0 {
		t.Fatalf("brak pliku: %v", err)
	}
	u, err := Otworz(niespojny)
	if err != nil {
		t.Fatal(err)
	}
	// Wpis „ma hasło” bez hasha nie może otwierać się bez hasła.
	if _, err := u.Sprawdz("x", ""); !errors.Is(err, rdzen.ErrZleDane) {
		t.Fatalf("niespójny wpis: %v", err)
	}
}

func TestRownolegleZaliczPobranie(t *testing.T) {
	u, _, plik := nowe(t)
	const limit, chetni = 30, 80
	l := utworz(t, u, Link{Sciezka: "/a", LimitPobran: limit}, "")
	var wg sync.WaitGroup
	var mu sync.Mutex
	udane := 0
	for i := 0; i < chetni; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if u.ZaliczPobranie(l.ID) == nil {
				mu.Lock()
				udane++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if udane != limit {
		t.Fatalf("udanych pobrań %d, limit %d", udane, limit)
	}
	u2, err := Otworz(plik)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := u2.Pobierz(l.ID); got.Pobrano != limit {
		t.Fatalf("w pliku Pobrano=%d", got.Pobrano)
	}
}
