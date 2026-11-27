package konta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"skrytka/wewn/rdzen"
)

func TestOtworzBrakPliku(t *testing.T) {
	k, _, _ := noweKonta(t)
	if !k.CzyPusto() || len(k.Lista()) != 0 {
		t.Fatal("nowe konta nie są puste")
	}
}

func TestOtworzZlyPlik(t *testing.T) {
	przypadki := map[string]string{
		"zły JSON":        "{",
		"zły login":       `{"konta":[{"login":"a b"}]}`,
		"powtórka loginu": `{"konta":[{"login":"a"},{"login":"a"}]}`,
	}
	for nazwa, tresc := range przypadki {
		t.Run(nazwa, func(t *testing.T) {
			plik := filepath.Join(t.TempDir(), "konta.json")
			if err := os.WriteFile(plik, []byte(tresc), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Otworz(plik); err == nil {
				t.Fatal("brak błędu dla złego pliku")
			}
		})
	}
}

func TestPlikZapisIOdczyt(t *testing.T) {
	k, _, plik := kontaZAdminemIJanem(t)
	dane, err := os.ReadFile(plik)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dane), hasloJana) || !strings.Contains(string(dane), prefiksArgon) {
		t.Fatal("plik zawiera hasło jawnie albo nie ma hasha")
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(plik); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("uprawnienia pliku: %v %v", info.Mode(), err)
		}
	}
	if _, err := os.Stat(plik + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("został plik tymczasowy")
	}
	k2, err := Otworz(plik)
	if err != nil {
		t.Fatal(err)
	}
	k2.param = tanieParametry
	if k2.CzyPusto() {
		t.Fatal("po ponownym otwarciu brak kont")
	}
	u, err := k2.Zaloguj("jan", hasloJana, "10.0.0.1")
	if err != nil || u.Prawa["dom"] != rdzen.PrawoOdczyt {
		t.Fatalf("po ponownym otwarciu: %+v %v", u, err)
	}
	_ = k // celowo: k tylko tworzy plik
}

func TestBladZapisuNieZmieniaStanu(t *testing.T) {
	k, _, plik := kontaZAdminemIJanem(t)
	// Katalog w miejscu pliku tymczasowego: zapis się nie uda.
	if err := os.Mkdir(plik+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "ewa"}, "haslo-ewy-123"); err == nil {
		t.Fatal("brak błędu zapisu")
	}
	if _, err := k.Uzytkownik("ewa"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatal("konto zostało w pamięci mimo błędu zapisu")
	}
}

func TestDodaj(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	przypadki := []struct {
		nazwa, login, haslo string
		chce                error
	}{
		{"poprawne", "ewa", "haslo-ewy-123", nil},
		{"istnieje", "jan", "haslo-jana-xyz", rdzen.ErrJuzIstnieje},
		{"krótkie hasło", "ala", "krotkie", ErrSlabeHaslo},
		{"pusty login", "", "haslo-ewy-123", rdzen.ErrZlaNazwa},
		{"spacja w loginie", "a b", "haslo-ewy-123", rdzen.ErrZlaNazwa},
		{"ukośnik w loginie", "a/b", "haslo-ewy-123", rdzen.ErrZlaNazwa},
		{"kropka na początku", ".a", "haslo-ewy-123", rdzen.ErrZlaNazwa},
		{"ogonki w loginie", "żaba", "haslo-ewy-123", rdzen.ErrZlaNazwa},
		{"za długi login", strings.Repeat("a", 65), "haslo-ewy-123", rdzen.ErrZlaNazwa},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if err := k.Dodaj(rdzen.Uzytkownik{Login: p.login}, p.haslo); !errors.Is(err, p.chce) {
				t.Fatalf("błąd %v, chcę %v", err, p.chce)
			}
		})
	}
}

func TestDodajIgnorujeHashOdWywolujacego(t *testing.T) {
	k, _, _ := noweKonta(t)
	obcy, _ := zahaszuj("haslo-atakujacego", tanieParametry) // celowo: błąd nieistotny dla testu
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "ewa", HashHasla: obcy}, "haslo-ewy-123"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Zaloguj("ewa", "haslo-atakujacego", "10.0.0.1"); err == nil {
		t.Fatal("przyjęto hash podany przez wywołującego")
	}
}

func TestListaBezHasha(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	lista := k.Lista()
	if len(lista) != 2 || lista[0].Login != "admin" || lista[1].Login != "jan" {
		t.Fatalf("lista: %+v", lista)
	}
	for _, u := range lista {
		if u.HashHasla != "" {
			t.Fatalf("hash widoczny w liście: %s", u.Login)
		}
	}
	// Zmiana mapy praw z listy nie zmienia magazynu.
	lista[1].Prawa["dom"] = rdzen.PrawoZapis
	u, _ := k.Uzytkownik("jan") // celowo: istnienie sprawdzone wyżej
	if u.Prawa["dom"] != rdzen.PrawoOdczyt || u.HashHasla != "" {
		t.Fatalf("magazyn zmieniony przez kopię albo hash widoczny: %+v", u)
	}
}

func TestZmienIOstatniAdmin(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	przypadki := []struct {
		nazwa string
		u     rdzen.Uzytkownik
		chce  error
	}{
		{"odebranie admina ostatniemu", rdzen.Uzytkownik{Login: "admin"}, ErrOstatniAdmin},
		{"wyłączenie ostatniego admina", rdzen.Uzytkownik{Login: "admin", Admin: true, Wylaczony: true}, ErrOstatniAdmin},
		{"brak konta", rdzen.Uzytkownik{Login: "nikt"}, rdzen.ErrNieZnaleziono},
		{"nadanie admina", rdzen.Uzytkownik{Login: "jan", Admin: true}, nil},
		{"teraz można odebrać", rdzen.Uzytkownik{Login: "admin"}, nil},
		{"znów ostatni", rdzen.Uzytkownik{Login: "jan"}, ErrOstatniAdmin},
	}
	for _, p := range przypadki {
		if err := k.Zmien(p.u); !errors.Is(err, p.chce) {
			t.Fatalf("%s: %v, chcę %v", p.nazwa, err, p.chce)
		}
	}
	if _, err := k.Zaloguj("jan", hasloJana, "10.0.0.1"); err != nil {
		t.Fatalf("Zmien zepsuł hasło: %v", err)
	}
}

func TestUsun(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	if err := k.Usun("admin"); !errors.Is(err, ErrOstatniAdmin) {
		t.Fatalf("usunięcie ostatniego admina: %v", err)
	}
	if err := k.Usun("nikt"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("usunięcie nieistniejącego: %v", err)
	}
	if err := k.Usun("jan"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Uzytkownik("jan"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatal("konto nadal istnieje")
	}
	// Wyłączony admin nie liczy się jako aktywny: drugiego aktywnego brak.
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "admin2", Admin: true, Wylaczony: true}, "haslo-admin-2"); err != nil {
		t.Fatal(err)
	}
	if err := k.Usun("admin"); !errors.Is(err, ErrOstatniAdmin) {
		t.Fatalf("usunięto ostatniego aktywnego admina: %v", err)
	}
	if err := k.Usun("admin2"); err != nil {
		t.Fatalf("wyłączony admin powinien dać się usunąć: %v", err)
	}
}

func TestUstawHaslo(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	if err := k.UstawHaslo("jan", "krotkie"); !errors.Is(err, ErrSlabeHaslo) {
		t.Fatalf("krótkie hasło: %v", err)
	}
	if err := k.UstawHaslo("nikt", "nowe-haslo-123"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("brak konta: %v", err)
	}
	if err := k.UstawHaslo("jan", "nowe-haslo-123"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Zaloguj("jan", hasloJana, "10.0.0.1"); err == nil {
		t.Fatal("stare hasło nadal działa")
	}
	if _, err := k.Zaloguj("jan", "nowe-haslo-123", "10.0.0.2"); err != nil {
		t.Fatalf("nowe hasło: %v", err)
	}
}

func TestRownoleglosc(t *testing.T) {
	k, _, plik := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	var wg sync.WaitGroup
	const n = 16
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			login := fmt.Sprintf("u%d", i)
			if err := k.Dodaj(rdzen.Uzytkownik{Login: login}, "haslo-rownolegle"); err != nil {
				t.Errorf("Dodaj %s: %v", login, err)
				return
			}
			if _, err := k.Zaloguj(login, "haslo-rownolegle", fmt.Sprintf("10.5.0.%d", i)); err != nil {
				t.Errorf("Zaloguj %s: %v", login, err)
			}
			token, err := s.Utworz(login)
			if err != nil {
				t.Errorf("Utworz %s: %v", login, err)
				return
			}
			s.Sprawdz(token)
			k.Lista()
			if err := k.UstawHaslo(login, "haslo-rownolegle-2"); err != nil {
				t.Errorf("UstawHaslo %s: %v", login, err)
			}
			if _, ok := s.Sprawdz(token); ok {
				t.Errorf("sesja %s przetrwała zmianę hasła", login)
			}
		}(i)
	}
	wg.Wait()
	k2, err := Otworz(plik)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(k2.Lista()); got != n+2 {
		t.Fatalf("w pliku %d kont, chcę %d", got, n+2)
	}
}
