package miniatury

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"skrytka/wewn/konfiguracja"
)

// plikCache: miniatura z czasem ostatniego użycia.
type plikCache struct {
	sciezka string
	rozmiar int64
	czas    int64
}

// Sprzataj usuwa najdawniej używane miniatury, gdy cache przekracza limit (do MiniaturaCelSprzatania % limitu).
// Usuwa też porzucone pliki tymczasowe. Brak folderu = nic do sprzątania.
func (x *Miniatury) Sprzataj() error {
	x.sprzatanie.Lock()
	defer x.sprzatanie.Unlock()
	wpisy, err := os.ReadDir(x.katalog)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var pliki []plikCache
	var suma int64
	for _, d := range wpisy {
		if !d.Type().IsRegular() {
			continue
		}
		fi, err := d.Info()
		if err != nil {
			continue // celowo: plik zniknął w trakcie
		}
		p := plikCache{filepath.Join(x.katalog, d.Name()), fi.Size(), fi.ModTime().UnixNano()}
		pliki = append(pliki, p)
		suma += p.rozmiar
	}
	if suma <= x.limitCache {
		return nil
	}
	sort.Slice(pliki, func(i, j int) bool { return pliki[i].czas < pliki[j].czas })
	docelowo := x.limitCache * konfiguracja.MiniaturaCelSprzatania / 100
	for _, p := range pliki {
		if suma <= docelowo {
			break
		}
		if os.Remove(p.sciezka) == nil {
			suma -= p.rozmiar
		}
	}
	return nil
}
