package magazyn

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

// stalyCzas: zegar testów (bez time.Now i bez usypiania).
var stalyCzas = time.Date(2026, 11, 27, 10, 30, 0, 0, time.UTC)

// nowyTestowy tworzy udział "dane" (zapis) i "archiwum" (tylko odczyt) w t.TempDir().
func nowyTestowy(t *testing.T) (*Magazyn, string) {
	t.Helper()
	baza := t.TempDir()
	korzen := filepath.Join(baza, "udzial")
	ro := filepath.Join(baza, "ro")
	for _, k := range []string{korzen, ro} {
		if err := os.MkdirAll(k, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m, err := Nowy([]konfiguracja.Udzial{
		{Nazwa: "dane", Sciezka: korzen},
		{Nazwa: "archiwum", Sciezka: ro, TylkoOdczyt: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.zegar = func() time.Time { return stalyCzas }
	return m, korzen
}

// plik zapisuje plik testowy (tworzy foldery po drodze).
func plik(t *testing.T, sciezka, tresc string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(sciezka), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sciezka, []byte(tresc), 0o644); err != nil {
		t.Fatal(err)
	}
}

// czytaj zwraca treść pliku albo "" gdy go brak.
func czytaj(t *testing.T, sciezka string) string {
	t.Helper()
	b, err := os.ReadFile(sciezka)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// junction tworzy punkt połączenia NTFS (mklink /J nie wymaga uprawnień administratora).
func junction(t *testing.T, link, cel string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("junction istnieje tylko na Windows")
	}
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, cel).CombinedOutput()
	if err != nil {
		t.Skipf("nie udało się utworzyć junction (%v): %s", err, out)
	}
}

// symlink tworzy dowiązanie symboliczne albo pomija test, gdy brak uprawnień (Windows bez trybu dewelopera).
func symlink(t *testing.T, cel, link string) {
	t.Helper()
	if err := os.Symlink(cel, link); err != nil {
		t.Skipf("brak uprawnień do tworzenia dowiązań symbolicznych (tryb dewelopera albo administrator): %v", err)
	}
}
