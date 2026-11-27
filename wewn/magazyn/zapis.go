package magazyn

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// sprawdzCelZapisu: poprawna nazwa, istniejący folder nadrzędny, brak folderu pod tą nazwą,
// a przy nadpisz=false brak jakiegokolwiek pliku.
func sprawdzCelZapisu(c cel, nadpisz bool) error {
	if c.czysta == "/" {
		return rdzen.ErrZlaNazwa
	}
	if err := rdzen.SprawdzNazwe(path.Base(c.czysta)); err != nil {
		return err
	}
	fi, err := os.Stat(filepath.Dir(c.dysk))
	if err != nil || !fi.IsDir() {
		return rdzen.ErrNieZnaleziono
	}
	fi, err = os.Lstat(c.dysk)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return mapuj(err)
	case fi.IsDir() || !nadpisz:
		return rdzen.ErrJuzIstnieje
	}
	return nil
}

// umiesc zamienia gotowy plik tymczasowy w docelowy (wołający trzyma blokadę celu).
func umiesc(zrodlo, docelowy string, nadpisz bool) error {
	fi, err := os.Lstat(docelowy)
	if err == nil && (fi.IsDir() || !nadpisz) {
		return rdzen.ErrJuzIstnieje
	}
	if nadpisz {
		return mapuj(os.Rename(zrodlo, docelowy))
	}
	// Dowiązanie twarde nie nadpisuje istniejącego pliku: brak wyścigu z innym procesem.
	errL := os.Link(zrodlo, docelowy)
	if errL == nil {
		_ = os.Remove(zrodlo) // celowo: dane już są w celu, ukryty plik tymczasowy najwyżej zostanie
		return nil
	}
	if errors.Is(errL, fs.ErrExist) {
		return rdzen.ErrJuzIstnieje
	}
	// System plików bez dowiązań twardych (FAT, exFAT): sprawdzenie wyżej i zamiana pod blokadą.
	return mapuj(os.Rename(zrodlo, docelowy))
}

// Zapisz strumień jako plik: zapis do pliku tymczasowego w tym samym folderze i zamiana na końcu;
// nadpisz=false i plik istnieje = rdzen.ErrJuzIstnieje. Zwraca liczbę bajtów.
func (m *Magazyn) Zapisz(udzial, sciezka string, r io.Reader, nadpisz bool) (int64, error) {
	c, err := m.pelna(udzial, sciezka, true)
	if err != nil {
		return 0, err
	}
	if err := sprawdzCelZapisu(c, nadpisz); err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.dysk), konfiguracja.MagazynPrefiksTymczasowy+"*")
	if err != nil {
		return 0, mapuj(err)
	}
	n, err := kopiuj(tmp, r)
	if errZ := tmp.Close(); err == nil {
		err = errZ
	}
	if err == nil {
		blok := m.blokada(c.dysk)
		blok.Lock()
		err = umiesc(tmp.Name(), c.dysk, nadpisz)
		blok.Unlock()
	}
	if err != nil {
		_ = os.Remove(tmp.Name()) // celowo: sprzątanie po nieudanym zapisie, zwracamy pierwotny błąd
		return 0, mapuj(err)
	}
	return n, nil
}

// sciezkaCzesci: plik kawałków obok celu; dowiązanie pod tą nazwą jest odrzucane.
func sciezkaCzesci(c cel) (string, error) {
	cz := c.dysk + konfiguracja.MagazynPrzyrostekCzesci
	if fi, err := os.Lstat(cz); err == nil && (czyDowiazanie(fi.Mode()) || fi.IsDir()) {
		return "", rdzen.ErrPozaUdzialem
	}
	return cz, nil
}

// DopiszKawalek dla wgrywania w kawałkach (wznawianie): zapisuje dane od przesunięcia do pliku
// "<sciezka>.skrytka_czesc"; Zatwierdz zamienia go w docelowy plik. Przesunięcie mniejsze niż zapisana
// część obcina ją (ponowienie kawałka), większe = ErrZlePrzesuniecie. Zwraca liczbę zapisanych bajtów.
func (m *Magazyn) DopiszKawalek(udzial, sciezka string, przesuniecie int64, r io.Reader) (int64, error) {
	c, err := m.pelna(udzial, sciezka, true)
	if err != nil {
		return 0, err
	}
	if err := sprawdzCelZapisu(c, true); err != nil {
		return 0, err
	}
	cz, err := sciezkaCzesci(c)
	if err != nil {
		return 0, err
	}
	blok := m.blokada(c.dysk)
	blok.Lock()
	defer blok.Unlock()
	f, err := os.OpenFile(cz, os.O_CREATE|os.O_WRONLY, konfiguracja.MagazynUprawnieniaCzesci)
	if err != nil {
		return 0, mapuj(err)
	}
	n, err := dopisz(f, przesuniecie, r)
	if errZ := f.Close(); err == nil {
		err = errZ
	}
	return n, err
}

// dopisz: kontrola przesunięcia, obcięcie i zapis od przesunięcia.
func dopisz(f *os.File, przesuniecie int64, r io.Reader) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if przesuniecie < 0 || przesuniecie > fi.Size() {
		return 0, ErrZlePrzesuniecie
	}
	if przesuniecie < fi.Size() {
		if err := f.Truncate(przesuniecie); err != nil {
			return 0, err
		}
	}
	if _, err := f.Seek(przesuniecie, io.SeekStart); err != nil {
		return 0, err
	}
	return kopiuj(f, r)
}

// RozmiarKawalkow zwraca, ile bajtów niedokończonego wgrywania już zapisano (0, gdy nic),
// żeby klient wiedział, od którego przesunięcia wznowić.
func (m *Magazyn) RozmiarKawalkow(udzial, sciezka string) (int64, error) {
	c, err := m.pelna(udzial, sciezka, true)
	if err != nil {
		return 0, err
	}
	cz, err := sciezkaCzesci(c)
	if err != nil {
		return 0, err
	}
	fi, err := os.Stat(cz)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, mapuj(err)
	}
	return fi.Size(), nil
}

// Zatwierdz kończy wgrywanie w kawałkach (brak części = rdzen.ErrNieZnaleziono).
func (m *Magazyn) Zatwierdz(udzial, sciezka string, nadpisz bool) error {
	c, err := m.pelna(udzial, sciezka, true)
	if err != nil {
		return err
	}
	if err := sprawdzCelZapisu(c, nadpisz); err != nil {
		return err
	}
	cz, err := sciezkaCzesci(c)
	if err != nil {
		return err
	}
	blok := m.blokada(c.dysk)
	blok.Lock()
	defer blok.Unlock()
	if _, err := os.Lstat(cz); err != nil {
		return mapuj(err)
	}
	return umiesc(cz, c.dysk, nadpisz)
}

// NowyFolder tworzy folder (rodzic musi istnieć).
func (m *Magazyn) NowyFolder(udzial, sciezka string) error {
	c, err := m.pelna(udzial, sciezka, true)
	if err != nil {
		return err
	}
	if err := sprawdzCelZapisu(c, false); err != nil {
		return err
	}
	return mapuj(os.Mkdir(c.dysk, konfiguracja.MagazynUprawnieniaFolderu))
}

// ZmienNazwe albo przenieś w obrębie udziału. Cel zajęty = rdzen.ErrJuzIstnieje (poza zmianą samej
// wielkości liter), folder do własnego wnętrza = rdzen.ErrZlaNazwa.
func (m *Magazyn) ZmienNazwe(udzial, z, na string) error {
	cz, err := m.pelnaBezKonca(udzial, z)
	if err != nil {
		return err
	}
	if cz.czysta == "/" {
		return rdzen.ErrZlaNazwa
	}
	fiZ, err := os.Lstat(cz.dysk)
	if err != nil {
		return mapuj(err)
	}
	cn, err := m.pelnaBezKonca(udzial, na)
	if err != nil {
		return err
	}
	if err := sprawdzCelZapisu(cn, true); err != nil && !errors.Is(err, rdzen.ErrJuzIstnieje) {
		return err
	}
	if rel, ok := wKorzeniu(cz.dysk, cn.dysk); ok && rel != "." {
		return rdzen.ErrZlaNazwa
	}
	if fiNa, err := os.Lstat(cn.dysk); err == nil && !os.SameFile(fiZ, fiNa) {
		return rdzen.ErrJuzIstnieje
	}
	return mapuj(os.Rename(cz.dysk, cn.dysk))
}
