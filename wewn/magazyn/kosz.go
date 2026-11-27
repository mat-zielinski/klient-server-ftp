package magazyn

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// DoKosza przenosi do <udział>/.skrytka_kosz/<znacznik czasu>/<ścieżka>. Gdy miejsce jest zajęte,
// używa "<znacznik>_2", "_3" … Dowiązanie trafia do kosza samo (bez celu).
func (m *Magazyn) DoKosza(udzial, sciezka string) error {
	c, err := m.pelnaBezKonca(udzial, sciezka)
	if err != nil {
		return err
	}
	if c.czysta == "/" {
		return rdzen.ErrZlaNazwa
	}
	if _, err := os.Lstat(c.dysk); err != nil {
		return mapuj(err)
	}
	kosz := filepath.Join(c.u.korzen, konfiguracja.KatalogKosza)
	blok := m.blokada(kosz)
	blok.Lock()
	defer blok.Unlock()
	return przeniesDoKosza(c.dysk, kosz, m.zegar().Format(konfiguracja.MagazynFormatZnacznikaKosza), c.czysta)
}

// przeniesDoKosza próbuje kosz/<znacznik[_n]>/<czysta>, aż trafi na wolne miejsce. Wariant, w którym
// coś blokuje drogę (np. plik zamiast folderu), jest pomijany.
func przeniesDoKosza(zrodlo, kosz, znacznik, czysta string) error {
	wzgledna := filepath.FromSlash(czysta[1:])
	for i := 1; i <= konfiguracja.MagazynMaksProbKosza; i++ {
		nazwa := znacznik
		if i > 1 {
			nazwa += "_" + strconv.Itoa(i)
		}
		docelowy := filepath.Join(kosz, nazwa, wzgledna)
		if _, err := os.Lstat(docelowy); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(docelowy), konfiguracja.MagazynUprawnieniaFolderu); err != nil {
			continue // celowo: droga zajęta przez plik, próbujemy kolejny znacznik
		}
		err := os.Rename(zrodlo, docelowy)
		if err == nil {
			return nil
		}
		if _, errZ := os.Lstat(zrodlo); errZ != nil {
			return mapuj(errZ)
		}
		if !errors.Is(err, fs.ErrExist) {
			return mapuj(err)
		}
	}
	return rdzen.ErrJuzIstnieje
}
