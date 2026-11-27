package konta

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/rdzen"
)

// zegar testowy: czas przesuwany ręcznie, bez time.Sleep.
type zegar struct {
	mu sync.Mutex
	t  time.Time
}

func (z *zegar) Teraz() time.Time {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.t
}

func (z *zegar) Przesun(d time.Duration) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.t = z.t.Add(d)
}

// tanieParametry: argon2id o małym koszcie, by testy były szybkie (format i logika te same).
var tanieParametry = parametryArgon{czas: 1, pamiec: 64, watki: 1, sol: 16, klucz: 32}

const (
	hasloAdmina = "tajne-haslo-admina"
	hasloJana   = "haslo-jana-123"
)

// noweKonta otwiera puste konta w t.TempDir() z zegarem testowym i tanim haszowaniem.
func noweKonta(t *testing.T) (*Konta, *zegar, string) {
	t.Helper()
	plik := filepath.Join(t.TempDir(), "dane", "konta.json")
	k, err := Otworz(plik)
	if err != nil {
		t.Fatalf("Otworz: %v", err)
	}
	z := &zegar{t: time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)}
	k.teraz = z.Teraz
	k.param = tanieParametry
	return k, z, plik
}

// kontaZAdminemIJanem: admin (admin) i jan (zwykły, prawo odczytu do „dom”).
func kontaZAdminemIJanem(t *testing.T) (*Konta, *zegar, string) {
	t.Helper()
	k, z, plik := noweKonta(t)
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "admin", Admin: true}, hasloAdmina); err != nil {
		t.Fatalf("Dodaj admin: %v", err)
	}
	jan := rdzen.Uzytkownik{Login: "jan", Prawa: map[string]rdzen.Prawo{"dom": rdzen.PrawoOdczyt}}
	if err := k.Dodaj(jan, hasloJana); err != nil {
		t.Fatalf("Dodaj jan: %v", err)
	}
	return k, z, plik
}
