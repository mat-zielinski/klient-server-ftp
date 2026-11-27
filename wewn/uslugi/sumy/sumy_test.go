package sumy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

func noweSumy(t *testing.T) (*Sumy, string) {
	t.Helper()
	kat := t.TempDir()
	m, err := magazyn.Nowy([]konfiguracja.Udzial{{Nazwa: "dom", Sciezka: kat}})
	if err != nil {
		t.Fatal(err)
	}
	return Nowe(m), kat
}

func zapisz(t *testing.T, kat, nazwa string, dane []byte) string {
	t.Helper()
	p := filepath.Join(kat, filepath.FromSlash(nazwa))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, dane, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func wzorzec(dane []byte) string {
	h := sha256.Sum256(dane)
	return hex.EncodeToString(h[:])
}

func TestZgodnoscZCryptoSHA256(t *testing.T) {
	s, kat := noweSumy(t)
	duzy := make([]byte, 3<<20+17)
	los := rand.New(rand.NewPCG(1, 2))
	for i := range duzy {
		duzy[i] = byte(los.Uint32())
	}
	przypadki := []struct {
		nazwa string
		dane  []byte
	}{
		{"pusty.bin", nil},
		{"abc.txt", []byte("abc")},
		{"zdjęcia/żółw.jpg", []byte("Żółw")},
		{"duzy.bin", duzy},
	}
	for _, p := range przypadki {
		zapisz(t, kat, p.nazwa, p.dane)
		suma, err := s.SHA256(context.Background(), "dom", "/"+p.nazwa)
		if err != nil || suma != wzorzec(p.dane) {
			t.Errorf("%s: %s %v, chcę %s", p.nazwa, suma, err, wzorzec(p.dane))
		}
	}
	// Znany wektor testowy.
	if suma, _ := s.SHA256(context.Background(), "dom", "abc.txt"); suma != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("abc: %s", suma)
	}
}

func TestBledy(t *testing.T) {
	s, kat := noweSumy(t)
	zapisz(t, kat, "folder/a.txt", []byte("a"))
	przypadki := []struct {
		udzial, sciezka string
		chce            error
	}{
		{"dom", "/folder", rdzen.ErrNieZnaleziono},
		{"dom", "/", rdzen.ErrNieZnaleziono},
		{"dom", "/brak.txt", rdzen.ErrNieZnaleziono},
		{"dom", "../poza.txt", rdzen.ErrPozaUdzialem},
		{"obcy", "/folder/a.txt", rdzen.ErrNieZnaleziono},
	}
	for _, p := range przypadki {
		if _, err := s.SHA256(context.Background(), p.udzial, p.sciezka); !errors.Is(err, p.chce) {
			t.Errorf("%s %s: %v, chcę %v", p.udzial, p.sciezka, err, p.chce)
		}
	}
}

func TestPamiecPodreczna(t *testing.T) {
	s, kat := noweSumy(t)
	p := zapisz(t, kat, "a.txt", []byte("pierwsza"))
	czas := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p, czas, czas); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s1, err := s.SHA256(ctx, "dom", "/a.txt")
	if err != nil || s1 != wzorzec([]byte("pierwsza")) {
		t.Fatal(s1, err)
	}
	// Ta sama długość i data: wynik z pamięci (dowód, że pliku nie czytano ponownie).
	if err := os.WriteFile(p, []byte("PIERWSZA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, czas, czas); err != nil {
		t.Fatal(err)
	}
	if s2, _ := s.SHA256(ctx, "dom", "/a.txt"); s2 != s1 {
		t.Errorf("brak trafienia w pamięci: %s", s2)
	}
	// Inna data = liczymy od nowa.
	if err := os.Chtimes(p, czas, czas.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if s3, _ := s.SHA256(ctx, "dom", "/a.txt"); s3 != wzorzec([]byte("PIERWSZA")) {
		t.Errorf("zmieniona data, stara suma: %s", s3)
	}
	// Inny rozmiar = liczymy od nowa.
	if err := os.WriteFile(p, []byte("dłuższa treść"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, czas, czas.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if s4, _ := s.SHA256(ctx, "dom", "/a.txt"); s4 != wzorzec([]byte("dłuższa treść")) {
		t.Errorf("zmieniony rozmiar, stara suma: %s", s4)
	}
}

func TestLimitPamieci(t *testing.T) {
	s, kat := noweSumy(t)
	s.maksWpisow = 2
	for _, n := range []string{"a", "b", "c", "d"} {
		zapisz(t, kat, n, []byte(n))
		if _, err := s.SHA256(context.Background(), "dom", n); err != nil {
			t.Fatal(err)
		}
		if len(s.cache) > 2 {
			t.Fatalf("pamięć ponad limit: %d", len(s.cache))
		}
	}
}

func TestRownoczesneLiczenie(t *testing.T) {
	s, kat := noweSumy(t)
	dane := make([]byte, 1<<20)
	zapisz(t, kat, "a.bin", dane)
	var wg sync.WaitGroup
	wyniki := make([]string, 8)
	for i := range wyniki {
		wg.Go(func() {
			w, err := s.SHA256(context.Background(), "dom", "/a.bin")
			if err != nil {
				t.Error(err)
			}
			wyniki[i] = w
		})
	}
	wg.Wait()
	for i, w := range wyniki {
		if w != wzorzec(dane) {
			t.Errorf("%d: %s", i, w)
		}
	}
	if len(s.wToku) != 0 {
		t.Errorf("zostały liczenia w toku: %d", len(s.wToku))
	}
}

func TestPrzerwanie(t *testing.T) {
	s, kat := noweSumy(t)
	zapisz(t, kat, "a.bin", make([]byte, 1<<20))
	ctx, anuluj := context.WithCancel(context.Background())
	anuluj()
	if _, err := s.SHA256(ctx, "dom", "/a.bin"); !errors.Is(err, context.Canceled) {
		t.Errorf("przerwane: %v", err)
	}
	if len(s.cache) != 0 {
		t.Error("przerwane liczenie zapamiętane")
	}
	if suma, err := s.SHA256(context.Background(), "dom", "/a.bin"); err != nil || suma != wzorzec(make([]byte, 1<<20)) {
		t.Errorf("po przerwaniu: %s %v", suma, err)
	}
}
