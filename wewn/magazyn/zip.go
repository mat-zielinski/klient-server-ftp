package magazyn

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// zipowanie: stan jednego strumienia ZIP.
type zipowanie struct {
	zw    *zip.Writer
	suma  int64
	uzyte map[string]bool
}

// ZipDo strumieniuje ZIP (Store dla obrazów/wideo/archiwów, Deflate dla reszty) wybranych ścieżek do w;
// zwraca liczbę bajtów danych źródłowych. Nie buforuje całości w pamięci. Foldery idą rekurencyjnie
// (bez ukrytych i bez dowiązań), korzeń udziału pod nazwą udziału, powtórzone nazwy dostają " (2)".
func (m *Magazyn) ZipDo(w io.Writer, udzial string, sciezki []string) (int64, error) {
	z := &zipowanie{zw: zip.NewWriter(w), uzyte: map[string]bool{}}
	for _, s := range sciezki {
		c, err := m.pelna(udzial, s, false)
		if err != nil {
			return z.suma, err
		}
		if err := z.dodaj(c.dysk, z.unikalna(nazwaWpisu(c))); err != nil {
			return z.suma, err
		}
	}
	return z.suma, z.zw.Close()
}

// unikalna: nazwa najwyższego poziomu bez powtórzeń.
func (z *zipowanie) unikalna(nazwa string) string {
	wynik := nazwa
	for i := 2; z.uzyte[strings.ToLower(wynik)]; i++ {
		ext := path.Ext(nazwa)
		wynik = strings.TrimSuffix(nazwa, ext) + " (" + strconv.Itoa(i) + ")" + ext
	}
	z.uzyte[strings.ToLower(wynik)] = true
	return wynik
}

// dodaj plik albo całe drzewo folderu pod nazwą w archiwum.
func (z *zipowanie) dodaj(dysk, nazwa string) error {
	fi, err := os.Stat(dysk)
	if err != nil {
		return mapuj(err)
	}
	if !fi.IsDir() {
		return z.plik(dysk, nazwa, fi)
	}
	return filepath.WalkDir(dysk, func(p string, d fs.DirEntry, err error) error {
		if p != dysk && (err != nil || czyUkryta(d.Name())) {
			return pomin(d)
		}
		if err != nil {
			return mapuj(err)
		}
		if czyDowiazanie(d.Type()) {
			return nil // celowo: nie podążamy za dowiązaniami (mogą wyjść poza udział)
		}
		rel, err := filepath.Rel(dysk, p)
		if err != nil {
			return err
		}
		wewn := path.Join(nazwa, filepath.ToSlash(rel))
		fi, err := d.Info()
		if err != nil {
			return nil // celowo: plik zniknął w trakcie pakowania
		}
		if d.IsDir() {
			return z.folder(wewn, fi)
		}
		return z.plik(p, wewn, fi)
	})
}

// folder dodaje pusty wpis "nazwa/" (zachowuje puste foldery).
func (z *zipowanie) folder(nazwa string, fi fs.FileInfo) error {
	h := &zip.FileHeader{Name: nazwa + "/", Method: zip.Store, Modified: fi.ModTime()}
	h.SetMode(fi.Mode())
	_, err := z.zw.CreateHeader(h)
	return err
}

// plik kopiuje dane pliku do archiwum buforem z puli.
func (z *zipowanie) plik(dysk, nazwa string, fi fs.FileInfo) error {
	f, err := os.Open(dysk)
	if err != nil {
		return nil // celowo: plik zablokowany albo usunięty w trakcie pakowania, pomijamy
	}
	defer f.Close() // celowo: tylko odczyt
	h := &zip.FileHeader{Name: nazwa, Method: metodaZip(nazwa), Modified: fi.ModTime()}
	h.SetMode(fi.Mode())
	w, err := z.zw.CreateHeader(h)
	if err != nil {
		return err
	}
	n, err := kopiuj(w, f)
	z.suma += n
	return err
}

// metodaZip: Store dla plików już skompresowanych, Deflate dla reszty.
func metodaZip(nazwa string) uint16 {
	ext := strings.ToLower(path.Ext(nazwa))
	for _, e := range konfiguracja.MagazynBezKompresji {
		if ext == e {
			return zip.Store
		}
	}
	return zip.Deflate
}
