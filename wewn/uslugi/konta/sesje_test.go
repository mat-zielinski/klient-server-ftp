package konta

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

func TestUtworzSesje(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "wyl", Wylaczony: true}, "haslo-wyl-123"); err != nil {
		t.Fatal(err)
	}
	s := NoweSesje(k)
	przypadki := []struct {
		login string
		chce  error
	}{
		{"jan", nil},
		{"nikt", rdzen.ErrNieZnaleziono},
		{"wyl", rdzen.ErrZleDane},
	}
	for _, p := range przypadki {
		token, err := s.Utworz(p.login)
		if !errors.Is(err, p.chce) {
			t.Fatalf("%s: %v, chcę %v", p.login, err, p.chce)
		}
		if err != nil {
			continue
		}
		surowy, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(surowy) != konfiguracja.DlugoscTokenuSesji {
			t.Fatalf("token nie jest base64url z %d B: %q", konfiguracja.DlugoscTokenuSesji, token)
		}
	}
}

func TestTokenNiePrzechowywanyJawnie(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	token, err := s.Utworz("jan")
	if err != nil {
		t.Fatal(err)
	}
	t2, _ := s.Utworz("jan") // celowo: drugi token tylko do porównania
	if token == t2 {
		t.Fatal("dwa takie same tokeny")
	}
	for klucz := range s.wpisy {
		if string(klucz[:]) == token {
			t.Fatal("token przechowywany jawnie")
		}
	}
}

func TestSprawdzSesje(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	token, err := s.Utworz("jan")
	if err != nil {
		t.Fatal(err)
	}
	przypadki := []struct {
		nazwa, token string
		chce         bool
	}{
		{"dobry", token, true},
		{"pusty", "", false},
		{"zmieniony znak", zmienPierwszy(token), false},
		{"za krótki", token[:20], false},
	}
	for _, p := range przypadki {
		u, ok := s.Sprawdz(p.token)
		if ok != p.chce {
			t.Fatalf("%s: %v, chcę %v", p.nazwa, ok, p.chce)
		}
		if ok && (u.Login != "jan" || u.HashHasla != "") {
			t.Fatalf("%s: zły użytkownik albo widoczny hash: %+v", p.nazwa, u)
		}
	}
}

func TestWygasanieIPrzedluzanieSesji(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	token, _ := s.Utworz("jan") // celowo: konto istnieje
	z.Przesun(konfiguracja.CzasSesji - time.Second)
	if _, ok := s.Sprawdz(token); !ok {
		t.Fatal("sesja wygasła za wcześnie")
	}
	// Użycie przedłużyło sesję o pełny CzasSesji.
	z.Przesun(konfiguracja.CzasSesji - time.Second)
	if _, ok := s.Sprawdz(token); !ok {
		t.Fatal("sesja nie została przedłużona")
	}
	z.Przesun(konfiguracja.CzasSesji)
	if _, ok := s.Sprawdz(token); ok {
		t.Fatal("wygasła sesja nadal ważna")
	}
}

func TestSprzatanieWygaslychSesji(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	for i := 0; i < 3; i++ {
		if _, err := s.Utworz("jan"); err != nil {
			t.Fatal(err)
		}
	}
	z.Przesun(konfiguracja.CzasSesji)
	if _, err := s.Utworz("admin"); err != nil {
		t.Fatal(err)
	}
	if len(s.wpisy) != 1 {
		t.Fatalf("po sprzątaniu %d sesji, chcę 1", len(s.wpisy))
	}
}

func TestKoniecSesji(t *testing.T) {
	przypadki := []struct {
		nazwa   string
		akcja   func(k *Konta, s *Sesje, tokenJana string) error
		janOK   bool
		adminOK bool
	}{
		{"Zakoncz", func(_ *Konta, s *Sesje, tok string) error { s.Zakoncz(tok); return nil }, false, true},
		{"ZakonczWszystkie", func(_ *Konta, s *Sesje, _ string) error { s.ZakonczWszystkie("jan"); return nil }, false, true},
		{"UstawHaslo", func(k *Konta, _ *Sesje, _ string) error { return k.UstawHaslo("jan", "nowe-haslo-123") }, false, true},
		{"wyłączenie konta", func(k *Konta, _ *Sesje, _ string) error {
			return k.Zmien(rdzen.Uzytkownik{Login: "jan", Wylaczony: true})
		}, false, true},
		{"usunięcie konta", func(k *Konta, _ *Sesje, _ string) error { return k.Usun("jan") }, false, true},
		{"zmiana praw", func(k *Konta, _ *Sesje, _ string) error {
			return k.Zmien(rdzen.Uzytkownik{Login: "jan", Prawa: map[string]rdzen.Prawo{"dom": rdzen.PrawoZapis}})
		}, true, true},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			k, _, _ := kontaZAdminemIJanem(t)
			s := NoweSesje(k)
			tj1, _ := s.Utworz("jan")  // celowo: konto istnieje
			tj2, _ := s.Utworz("jan")  // celowo: konto istnieje
			ta, _ := s.Utworz("admin") // celowo: konto istnieje
			if err := p.akcja(k, s, tj1); err != nil {
				t.Fatal(err)
			}
			if _, ok := s.Sprawdz(tj1); ok != p.janOK {
				t.Fatalf("sesja jana: %v, chcę %v", ok, p.janOK)
			}
			if p.nazwa != "Zakoncz" {
				if _, ok := s.Sprawdz(tj2); ok != p.janOK {
					t.Fatalf("druga sesja jana: %v, chcę %v", ok, p.janOK)
				}
			}
			if _, ok := s.Sprawdz(ta); ok != p.adminOK {
				t.Fatalf("sesja admina: %v, chcę %v", ok, p.adminOK)
			}
		})
	}
}

func TestUsunieteIPonownieZalozoneKontoNieOzywiaSesji(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	token, _ := s.Utworz("jan") // celowo: konto istnieje
	if err := k.Usun("jan"); err != nil {
		t.Fatal(err)
	}
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "jan"}, "inne-haslo-123"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Sprawdz(token); ok {
		t.Fatal("stara sesja ożyła po ponownym założeniu konta")
	}
}

// zmienPierwszy podmienia pierwszy znak tokenu na inny (zawsze różny od oryginału).
func zmienPierwszy(token string) string {
	if token[0] == 'A' {
		return "B" + token[1:]
	}
	return "A" + token[1:]
}

// Sprawdz sam pilnuje stanu konta (niezależnie od kończenia sesji w Zmien).
func TestSprawdzOdrzucaWylaczoneKontoBezZakonczeniaSesji(t *testing.T) {
	k, _, _ := kontaZAdminemIJanem(t)
	s := NoweSesje(k)
	token, _ := s.Utworz("jan") // celowo: konto istnieje
	k.mu.Lock()
	u := k.konta["jan"]
	u.Wylaczony = true
	k.konta["jan"] = u
	k.mu.Unlock()
	if _, ok := s.Sprawdz(token); ok {
		t.Fatal("sesja wyłączonego konta przyjęta")
	}
}
