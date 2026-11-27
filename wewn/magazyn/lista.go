package magazyn

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"skrytka/wewn/rdzen"
)

// nowyWpis buduje Wpis z informacji o pliku.
func nowyWpis(nazwa, czysta string, fi fs.FileInfo) rdzen.Wpis {
	w := rdzen.Wpis{
		Nazwa:     nazwa,
		Sciezka:   czysta,
		Folder:    fi.IsDir(),
		Zmieniono: fi.ModTime(),
		Rodzaj:    rdzen.RodzajPliku(nazwa, fi.IsDir()),
	}
	if !w.Folder {
		w.Rozmiar = fi.Size()
	}
	return w
}

// nazwaWpisu: dla korzenia nazwa udziału, inaczej ostatni człon.
func nazwaWpisu(c cel) string {
	if c.czysta == "/" {
		return c.u.Nazwa
	}
	return path.Base(c.czysta)
}

// Lista zawartości folderu (bez kosza i plików ukrytych systemu), nieposortowana.
// Dowiązania prowadzące poza udział (albo zerwane) są pomijane.
func (m *Magazyn) Lista(udzial, sciezka string) ([]rdzen.Wpis, error) {
	c, err := m.pelna(udzial, sciezka, false)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(c.dysk)
	if err != nil {
		return nil, mapuj(err)
	}
	defer f.Close() // celowo: odczyt katalogu, błąd zamknięcia bez znaczenia
	wpisy, err := f.ReadDir(-1)
	if err != nil {
		return nil, rdzen.ErrNieZnaleziono // plik zamiast folderu albo brak dostępu
	}
	wynik := make([]rdzen.Wpis, 0, len(wpisy))
	for _, d := range wpisy {
		if czyUkryta(d.Name()) {
			continue
		}
		fi, ok := infoWpisu(c, d)
		if !ok {
			continue
		}
		wynik = append(wynik, nowyWpis(d.Name(), path.Join(c.czysta, d.Name()), fi))
	}
	return wynik, nil
}

// infoWpisu: informacja o pozycji katalogu; dowiązanie rozwijane z kontrolą korzenia.
func infoWpisu(c cel, d fs.DirEntry) (fs.FileInfo, bool) {
	pelnaSciezka := filepath.Join(c.dysk, d.Name())
	if !czyDowiazanie(d.Type()) {
		fi, err := d.Info()
		return fi, err == nil
	}
	real, err := rozwiazWUdziale(c.u.korzen, pelnaSciezka)
	if err != nil || sprawdzUkryteNaDysku(c.u.korzen, real) != nil {
		return nil, false
	}
	fi, err := os.Stat(real)
	return fi, err == nil
}

// Info o jednym pliku/folderze.
func (m *Magazyn) Info(udzial, sciezka string) (rdzen.Wpis, error) {
	c, err := m.pelna(udzial, sciezka, false)
	if err != nil {
		return rdzen.Wpis{}, err
	}
	fi, err := os.Stat(c.dysk)
	if err != nil {
		return rdzen.Wpis{}, mapuj(err)
	}
	return nowyWpis(nazwaWpisu(c), c.czysta, fi), nil
}

// Otworz plik do odczytu (wołający zamyka); folder = rdzen.ErrNieZnaleziono.
func (m *Magazyn) Otworz(udzial, sciezka string) (*os.File, rdzen.Wpis, error) {
	c, err := m.pelna(udzial, sciezka, false)
	if err != nil {
		return nil, rdzen.Wpis{}, err
	}
	f, err := os.Open(c.dysk)
	if err != nil {
		return nil, rdzen.Wpis{}, mapuj(err)
	}
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		_ = f.Close() // celowo: zwracamy ważniejszy błąd
		return nil, rdzen.Wpis{}, rdzen.ErrNieZnaleziono
	}
	return f, nowyWpis(nazwaWpisu(c), c.czysta, fi), nil
}
