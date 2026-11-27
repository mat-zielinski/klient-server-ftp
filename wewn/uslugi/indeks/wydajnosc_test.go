package indeks

import (
	"fmt"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

// sztucznyIndeks: indeks z gotowym zbiorem n pozycji (bez dysku; przejście dysku testują inne testy).
func sztucznyIndeks(b testing.TB, n int) *Indeks {
	b.Helper()
	m, err := magazyn.Nowy([]konfiguracja.Udzial{{Nazwa: "dom", Sciezka: b.TempDir()}})
	if err != nil {
		b.Fatal(err)
	}
	x := Nowy(m)
	slowa := []string{"Wakacje", "Żółw", "raport", "faktura", "zdjęcie", "Projekt", "kopia", "nagranie", "umowa", "plan"}
	rozsz := []string{".jpg", ".pdf", ".txt", ".mp4", ".docx", ".zip"}
	z := &zbior{foldery: []string{"/"}}
	const naFolder = 100
	for i := 0; len(z.pozycje) < n; i++ {
		folder := fmt.Sprintf("/%s %d/%s", slowa[i%len(slowa)], i/len(slowa), slowa[(i*7)%len(slowa)])
		z.foldery = append(z.foldery, folder)
		for j := range naFolder {
			nazwa := fmt.Sprintf("%s %s %d_%d%s", slowa[(i+j)%len(slowa)], slowa[(j*3)%len(slowa)], i, j, rozsz[j%len(rozsz)])
			w := rdzen.Wpis{Nazwa: nazwa, Rozmiar: int64(j), Zmieniono: czasTestu, Rodzaj: rdzen.RodzajPliku(nazwa, false)}
			z.pozycje = append(z.pozycje, nowaPozycja(w, int32(len(z.foldery)-1)))
		}
	}
	z.uporzadkuj()
	x.udzialy["dom"].zbior.Store(z)
	return x
}

const pozycjiWydajnosci = 200_000

func BenchmarkSzukaj(b *testing.B) {
	x := sztucznyIndeks(b, pozycjiWydajnosci)
	przypadki := []struct{ nazwa, start, q string }{
		{"brak_trafien", "/", "nieistniejace"},
		{"rzadkie", "/", "zolw 1234_5"},
		{"czeste_limit", "/", "raport"},
		{"dwa_slowa", "/", "faktura umowa 99"},
		{"podfolder", "/Projekt 5", "plan"},
	}
	for _, p := range przypadki {
		b.Run(p.nazwa, func(b *testing.B) {
			for b.Loop() {
				x.Szukaj("dom", p.start, p.q, 0)
			}
		})
	}
}

// Wymóg: wynik < 50 ms dla 200 tys. plików. Mierzymy najgorszy przypadek (pełny przegląd bez trafień);
// próg testu ma zapas na wolną maszynę CI, dokładne liczby daje BenchmarkSzukaj.
func TestCzasSzukania200Tys(t *testing.T) {
	if testing.Short() {
		t.Skip("pomijam w trybie -short")
	}
	x := sztucznyIndeks(t, pozycjiWydajnosci)
	najlepszy := time.Hour
	for range 5 {
		start := time.Now()
		x.Szukaj("dom", "/", "nieistniejace slowo", 0)
		najlepszy = min(najlepszy, time.Since(start))
	}
	t.Logf("pełny przegląd 200 tys. pozycji: %v", najlepszy)
	if najlepszy > 50*time.Millisecond {
		t.Errorf("szukanie trwa %v (wymóg < 50 ms)", najlepszy)
	}
}
