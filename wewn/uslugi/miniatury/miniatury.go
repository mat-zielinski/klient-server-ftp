// Pakiet miniatury: pomniejszone obrazy (jpg, png, gif, webp) z pamięcią podręczną na dysku w dane/miniatury.
// Źródło czytane wyłącznie przez magazyn; nagłówek sprawdzany przed dekodowaniem (bomba dekompresyjna).
package miniatury

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

// Błędy (HTTP: ErrNieObraz i ErrNieObslugiwany = 404, ErrZaDuzy = 413 albo 404, ErrZlyRozmiar = 400).
var (
	ErrNieObraz       = errors.New("to nie jest obraz")
	ErrNieObslugiwany = errors.New("format obrazu bez miniatury")
	ErrZaDuzy         = errors.New("obraz za duży na miniaturę")
	ErrZlyRozmiar     = errors.New("nieznany rozmiar miniatury")
	ErrUszkodzony     = errors.New("uszkodzony obraz")
)

// Miniatury tworzy i przechowuje miniatury. Bezpieczne dla wielu gorutyn.
type Miniatury struct {
	m           *magazyn.Magazyn
	katalog     string
	semafor     chan struct{} // ile dekodowań naraz
	maksBajtow  int64
	maksPikseli int64
	limitCache  int64 // bajty

	mu         sync.Mutex
	wToku      map[string]*robota // jedno tworzenie danej miniatury naraz
	nowych     atomic.Int64       // licznik do sprzątania co MiniaturaSprzatanieCo
	sprzatanie sync.Mutex
}

// robota: trwające tworzenie miniatury; czekający dostają ten sam wynik.
type robota struct {
	gotowe chan struct{}
	err    error
}

// Opcja zmienia limity (testy).
type Opcja func(*Miniatury)

// ZLimitemCache ustawia limit pamięci podręcznej w bajtach.
func ZLimitemCache(bajty int64) Opcja { return func(x *Miniatury) { x.limitCache = bajty } }

// ZLimitemZrodla ustawia największy plik i największą liczbę pikseli źródła.
func ZLimitemZrodla(bajty, piksele int64) Opcja {
	return func(x *Miniatury) { x.maksBajtow, x.maksPikseli = bajty, piksele }
}

// Nowe tworzy usługę; katalog "" = konfiguracja.KatalogMiniatur(). Folder powstaje przy pierwszej miniaturze.
func Nowe(m *magazyn.Magazyn, katalog string, opcje ...Opcja) *Miniatury {
	if katalog == "" {
		katalog = konfiguracja.KatalogMiniatur()
	}
	x := &Miniatury{
		m:           m,
		katalog:     katalog,
		semafor:     make(chan struct{}, max(1, runtime.NumCPU()/2)),
		maksBajtow:  konfiguracja.MiniaturaMaksBajtowZrodla,
		maksPikseli: konfiguracja.MiniaturaMaksPikseliZrodla,
		limitCache:  konfiguracja.MiniaturaLimitCacheMB << 20,
		wToku:       map[string]*robota{},
	}
	for _, o := range opcje {
		o(x)
	}
	return x
}

// wymiar: "s" / "m" -> dłuższy bok w pikselach.
func wymiar(rozmiar string) (int, error) {
	switch rozmiar {
	case "s":
		return konfiguracja.MiniaturaRozmiarS, nil
	case "m":
		return konfiguracja.MiniaturaRozmiarM, nil
	}
	return 0, ErrZlyRozmiar
}

// Miniatura zwraca ścieżkę pliku JPEG w pamięci podręcznej (tworzy go, gdy brak). Klucz zawiera rozmiar i datę
// zmiany źródła, więc zmieniony plik dostaje nową miniaturę. Błędy magazynu (rdzen.Err…) przechodzą bez zmian.
func (x *Miniatury) Miniatura(ctx context.Context, udzial, sciezka, rozmiar string) (string, error) {
	bok, err := wymiar(rozmiar)
	if err != nil {
		return "", err
	}
	w, err := x.m.Info(udzial, sciezka)
	if err != nil {
		return "", err
	}
	if err := x.sprawdzZrodlo(w); err != nil {
		return "", err
	}
	plik := filepath.Join(x.katalog, klucz(udzial, w, bok)+".jpg")
	if trafienie(plik) {
		return plik, nil
	}
	if err := x.utworzRaz(ctx, plik, func() error { return x.utworz(ctx, udzial, w, bok, plik) }); err != nil {
		return "", err
	}
	if x.nowych.Add(1)%konfiguracja.MiniaturaSprzatanieCo == 0 {
		go func() { _ = x.Sprzataj() }() // celowo: sprzątanie w tle, błąd nie wpływa na odpowiedź
	}
	return plik, nil
}

// sprawdzZrodlo: obraz w obsługiwanym formacie, nie większy niż limit.
func (x *Miniatury) sprawdzZrodlo(w rdzen.Wpis) error {
	if w.Folder || w.Rodzaj != rdzen.RodzajObraz {
		return ErrNieObraz
	}
	if !slices.Contains(konfiguracja.MiniaturaRozszerzenia, strings.ToLower(path.Ext(w.Nazwa))) {
		return ErrNieObslugiwany
	}
	if w.Rozmiar > x.maksBajtow {
		return ErrZaDuzy
	}
	return nil
}

// klucz: sha256(udział, ścieżka, rozmiar, data, wymiar) szesnastkowo; separator \x00 nie występuje w nazwach.
func klucz(udzial string, w rdzen.Wpis, bok int) string {
	h := sha256.New()
	for _, s := range []string{udzial, w.Sciezka, strconv.FormatInt(w.Rozmiar, 10),
		strconv.FormatInt(w.Zmieniono.UnixNano(), 10), strconv.Itoa(bok)} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// trafienie: plik jest w cache; odświeżamy jego czas, żeby sprzątanie usuwało najdawniej używane.
func trafienie(plik string) bool {
	fi, err := os.Stat(plik)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	_ = touch(plik) // celowo: nieudane odświeżenie czasu = najwyżej wcześniejsze usunięcie z cache
	return true
}

// utworzRaz: równoległe prośby o tę samą miniaturę czekają na jedno tworzenie.
func (x *Miniatury) utworzRaz(ctx context.Context, plik string, f func() error) error {
	x.mu.Lock()
	if r, jest := x.wToku[plik]; jest {
		x.mu.Unlock()
		select {
		case <-r.gotowe:
			return r.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	r := &robota{gotowe: make(chan struct{})}
	x.wToku[plik] = r
	x.mu.Unlock()
	r.err = f()
	x.mu.Lock()
	delete(x.wToku, plik)
	x.mu.Unlock()
	close(r.gotowe)
	return r.err
}
