//go:build windows

package magazyn

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// krotkaNazwa: postać 8.3 ścieżki albo "" gdy system jej nie tworzy (8.3 wyłączone na dysku).
func krotkaNazwa(t *testing.T, sciezka string) string {
	t.Helper()
	wej, err := windows.UTF16PtrFromString(sciezka)
	if err != nil {
		t.Fatal(err)
	}
	bufor := make([]uint16, windows.MAX_PATH)
	n, err := windows.GetShortPathName(wej, &bufor[0], uint32(len(bufor)))
	if err != nil || n == 0 || int(n) >= len(bufor) {
		return ""
	}
	return windows.UTF16ToString(bufor[:n])
}

// Junction do wnętrza udziału zapisany krótką nazwą 8.3 (jak w folderze tymczasowym maszyn GitHub Actions)
// ma działać; bez rozwinięcia do pełnej nazwy był odrzucany jako „poza udziałem”.
func TestJunctionKrotkaNazwa(t *testing.T) {
	m, korzen := nowyTestowy(t)
	plik(t, filepath.Join(korzen, "folder z dluga nazwa", "a.txt"), "a")
	krotka := krotkaNazwa(t, filepath.Join(korzen, "folder z dluga nazwa"))
	if krotka == "" || krotka == filepath.Join(korzen, "folder z dluga nazwa") {
		t.Skip("system nie tworzy krótkich nazw 8.3 na tym dysku")
	}
	junction(t, filepath.Join(korzen, "skrot"), krotka)
	f, _, err := m.Otworz("dane", "/skrot/a.txt")
	if err != nil {
		t.Fatalf("junction z krótką nazwą celu wewnątrz udziału powinien działać: %v", err)
	}
	_ = f.Close() // celowo: test

	zewn := przygotujZewnetrzny(t, korzen)
	krotkaZewn := krotkaNazwa(t, zewn)
	if krotkaZewn != "" {
		junction(t, filepath.Join(korzen, "ucieczka"), krotkaZewn)
		sprawdzUcieczke(t, m, zewn, "/ucieczka")
	}
}

func TestDlugaNazwaBezZmianDlaNieistniejacej(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nie", "ma")
	if g := dlugaNazwa(p); g != p {
		t.Errorf("dlugaNazwa(%q) = %q, oczekiwano bez zmian", p, g)
	}
}
