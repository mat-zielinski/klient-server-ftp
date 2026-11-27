package magazyn

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// Szukaj rekurencyjnie od folderu (rdzen.Pasuje), najwyżej limit wyników (0 = domyślny, górna granica
// z konfiguracji). Pomija ukryte nazwy, nie wchodzi w dowiązania ani junction, nieczytelne podfoldery pomija.
func (m *Magazyn) Szukaj(udzial, sciezka, zapytanie string, limit int) ([]rdzen.Wpis, error) {
	if limit <= 0 {
		limit = konfiguracja.MagazynDomyslnyLimitSzukania
	}
	limit = min(limit, konfiguracja.MagazynMaksLimitSzukania)
	c, err := m.pelna(udzial, sciezka, false)
	if err != nil {
		return nil, err
	}
	if fi, err := os.Stat(c.dysk); err != nil || !fi.IsDir() {
		return nil, rdzen.ErrNieZnaleziono
	}
	var wynik []rdzen.Wpis
	err = filepath.WalkDir(c.dysk, func(p string, d fs.DirEntry, err error) error {
		if p == c.dysk {
			return err
		}
		if err != nil || czyUkryta(d.Name()) {
			return pomin(d)
		}
		if czyDowiazanie(d.Type()) || !rdzen.Pasuje(d.Name(), zapytanie) {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil // celowo: plik zniknął w trakcie przeszukiwania
		}
		rel, err := filepath.Rel(c.dysk, p)
		if err != nil {
			return nil // celowo: nie wystąpi dla ścieżek z WalkDir
		}
		wynik = append(wynik, nowyWpis(d.Name(), path.Join(c.czysta, filepath.ToSlash(rel)), fi))
		if len(wynik) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	return wynik, mapuj(err)
}

// pomin: ukryty albo nieczytelny folder pomijamy w całości, plik po prostu.
func pomin(d fs.DirEntry) error {
	if d != nil && d.IsDir() {
		return fs.SkipDir
	}
	return nil
}

// RozmiarFolderu sumuje rozmiar plików (do paska postępu ZIP); dla pliku jego rozmiar.
// Liczy tak samo jak ZipDo: bez ukrytych i bez dowiązań.
func (m *Magazyn) RozmiarFolderu(udzial, sciezka string) (int64, error) {
	c, err := m.pelna(udzial, sciezka, false)
	if err != nil {
		return 0, err
	}
	return rozmiarDrzewa(c.dysk)
}

// rozmiarDrzewa: suma rozmiarów plików pod ścieżką dysku.
func rozmiarDrzewa(dysk string) (int64, error) {
	var suma int64
	err := filepath.WalkDir(dysk, func(p string, d fs.DirEntry, err error) error {
		if p == dysk && err != nil {
			return err
		}
		if p != dysk && (err != nil || czyUkryta(d.Name())) {
			return pomin(d)
		}
		if d.IsDir() || czyDowiazanie(d.Type()) {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			suma += fi.Size()
		}
		return nil
	})
	return suma, mapuj(err)
}
