package ftp

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

// klient: system plików jednego zalogowanego połączenia FTPS.
// "/" = wirtualny folder z udziałami użytkownika, "/<udzial>/<sciezka>" = magazyn.
type klient struct {
	u     uslugi
	login string
	ip    string
}

// cel: ścieżka FTP rozłożona na udział i ścieżkę w udziale.
type cel struct {
	udzial  string // "" = wirtualny korzeń
	sciezka string // zawsze "/..." po rdzen.OczyscSciezke
}

// rozloz zamienia ścieżkę FTP ("/zdjecia/a.jpg") na udział i ścieżkę w nim.
func rozloz(p string) (cel, error) {
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = path.Clean(p)
	if p == "/" {
		return cel{sciezka: "/"}, nil
	}
	czesci := strings.SplitN(strings.TrimPrefix(p, "/"), "/", 2)
	reszta := "/"
	if len(czesci) == 2 {
		reszta = "/" + czesci[1]
	}
	sc, err := rdzen.OczyscSciezke(reszta)
	if err != nil {
		return cel{}, err
	}
	return cel{udzial: czesci[0], sciezka: sc}, nil
}

// uzytkownik: aktualny stan konta (zmiana praw albo wyłączenie działa od razu, bez ponownego logowania).
func (k *klient) uzytkownik() (rdzen.Uzytkownik, error) {
	u, err := k.u.konta.Uzytkownik(k.login)
	if err != nil || !u.FTPS || u.Wylaczony {
		return rdzen.Uzytkownik{}, rdzen.ErrBrakPrawa
	}
	return u, nil
}

// wolno: bramka dla operacji w udziale; ukryty udział (brak odczytu) = nie znaleziono.
func (k *klient) wolno(op rdzen.Operacja, c cel) (rdzen.Uzytkownik, error) {
	u, err := k.uzytkownik()
	if err != nil {
		return u, err
	}
	if c.udzial == "" {
		if op == rdzen.OpOdczyt {
			return u, nil
		}
		return u, rdzen.ErrBrakPrawa
	}
	udz, err := k.u.magazyn.Udzial(c.udzial)
	if err != nil {
		return u, err
	}
	if k.u.czyWolno(u, rdzen.OpOdczyt, udz.Nazwa, udz.TylkoOdczyt) != nil {
		return u, rdzen.ErrNieZnaleziono
	}
	return u, k.u.czyWolno(u, op, udz.Nazwa, udz.TylkoOdczyt)
}

// wolnoZapis: jak wolno, ale cel nie może być korzeniem udziału, a nazwa musi być dozwolona.
func (k *klient) wolnoZapis(op rdzen.Operacja, c cel) error {
	if c.udzial == "" || c.sciezka == "/" {
		return rdzen.ErrBrakPrawa
	}
	if _, err := k.wolno(op, c); err != nil {
		return err
	}
	return rdzen.SprawdzNazwe(path.Base(c.sciezka))
}

// tlumacz dokleja błędy biblioteki, żeby klient dostał właściwy kod odpowiedzi.
func tlumacz(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, rdzen.ErrNieZnaleziono):
		return fmt.Errorf("%w: %w", err, os.ErrNotExist)
	case errors.Is(err, rdzen.ErrZlaNazwa), errors.Is(err, rdzen.ErrPozaUdzialem):
		return fmt.Errorf("%w: %w", err, ftpserver.ErrFileNameNotAllowed)
	case errors.Is(err, rdzen.ErrBrakPrawa), errors.Is(err, rdzen.ErrTylkoOdczyt):
		return fmt.Errorf("%w: %w", err, os.ErrPermission)
	}
	return err
}

// zmiana zapisuje w dzienniku operację na plikach (nowy folder, nazwa, kosz).
func (k *klient) zmiana(c cel, opis string, err error) {
	w := dziennik.Wpis{
		Rodzaj: dziennik.RodzajZmiana, Kto: k.login, IP: k.ip, Protokol: konfiguracja.FTPProtokol,
		Udzial: c.udzial, Sciezka: c.sciezka, Wynik: dziennik.WynikOK, Opis: opis,
	}
	if err != nil {
		w.Wynik = dziennik.WynikBlad
		w.Opis = opis + ": " + err.Error()
	}
	k.u.dziennik.Zapisz(w)
}

// Name: nazwa systemu plików (afero.Fs).
func (k *klient) Name() string { return konfiguracja.FTPBaner }

// Stat dla korzenia, udziału albo pliku.
func (k *klient) Stat(nazwa string) (os.FileInfo, error) {
	c, err := rozloz(nazwa)
	if err != nil {
		return nil, tlumacz(err)
	}
	if _, err := k.wolno(rdzen.OpOdczyt, c); err != nil {
		return nil, tlumacz(err)
	}
	if c.udzial == "" {
		return folderWirtualny("/"), nil
	}
	if c.sciezka == "/" {
		return folderWirtualny(c.udzial), nil
	}
	w, err := k.u.magazyn.Info(c.udzial, c.sciezka)
	if err != nil {
		return nil, tlumacz(err)
	}
	return infoPliku{w: w}, nil
}

// ReadDir (rozszerzenie ftpserverlib): korzeń = udziały z prawem odczytu, dalej magazyn.
func (k *klient) ReadDir(nazwa string) ([]os.FileInfo, error) {
	c, err := rozloz(nazwa)
	if err != nil {
		return nil, tlumacz(err)
	}
	u, err := k.wolno(rdzen.OpOdczyt, c)
	if err != nil {
		return nil, tlumacz(err)
	}
	if c.udzial == "" {
		return k.listaUdzialow(u), nil
	}
	wpisy, err := k.u.magazyn.Lista(c.udzial, c.sciezka)
	if err != nil {
		return nil, tlumacz(err)
	}
	wynik := make([]os.FileInfo, 0, len(wpisy))
	for _, w := range wpisy {
		wynik = append(wynik, infoPliku{w: w})
	}
	return wynik, nil
}

func (k *klient) listaUdzialow(u rdzen.Uzytkownik) []os.FileInfo {
	var wynik []os.FileInfo
	for _, udz := range k.u.magazyn.Udzialy() {
		if k.u.czyWolno(u, rdzen.OpOdczyt, udz.Nazwa, udz.TylkoOdczyt) == nil {
			wynik = append(wynik, folderWirtualny(udz.Nazwa))
		}
	}
	return wynik
}

// Mkdir: nowy folder (OpWgranie).
func (k *klient) Mkdir(nazwa string, _ os.FileMode) error {
	c, err := rozloz(nazwa)
	if err != nil {
		return tlumacz(err)
	}
	if err := k.wolnoZapis(rdzen.OpWgranie, c); err != nil {
		return tlumacz(err)
	}
	err = k.u.magazyn.NowyFolder(c.udzial, c.sciezka)
	k.zmiana(c, "Nowy folder", err)
	return tlumacz(err)
}

// MkdirAll: kolejne człony, istniejące foldery są pomijane.
func (k *klient) MkdirAll(nazwa string, tryb os.FileMode) error {
	c, err := rozloz(nazwa)
	if err != nil {
		return tlumacz(err)
	}
	if c.udzial == "" || c.sciezka == "/" {
		return tlumacz(rdzen.ErrBrakPrawa)
	}
	biezaca := "/" + c.udzial
	for _, czlon := range strings.Split(strings.TrimPrefix(c.sciezka, "/"), "/") {
		biezaca += "/" + czlon
		if info, err := k.Stat(biezaca); err == nil && info.IsDir() {
			continue
		}
		if err := k.Mkdir(biezaca, tryb); err != nil {
			return err
		}
	}
	return nil
}

// Remove (DELE): plik do kosza aplikacji; folder tylko przez RMD.
func (k *klient) Remove(nazwa string) error { return k.usun(nazwa, false) }

// RemoveDir (RMD, rozszerzenie ftpserverlib): pusty folder do kosza.
func (k *klient) RemoveDir(nazwa string) error { return k.usun(nazwa, true) }

// errFolderNiePusty: RMD na folderze z zawartością.
var errFolderNiePusty = errors.New("Folder nie jest pusty.")

// errToFolder / errToPlik: DELE na folderze albo RMD na pliku.
var (
	errToFolder = errors.New("To jest folder, użyj RMD.")
	errToPlik   = errors.New("To jest plik, użyj DELE.")
)

func (k *klient) usun(nazwa string, folder bool) error {
	c, err := rozloz(nazwa)
	if err != nil {
		return tlumacz(err)
	}
	if err := k.wolnoZapis(rdzen.OpUsuniecie, c); err != nil {
		return tlumacz(err)
	}
	w, err := k.u.magazyn.Info(c.udzial, c.sciezka)
	if err != nil {
		return tlumacz(err)
	}
	if err := sprawdzUsuwany(k, c, w, folder); err != nil {
		return tlumacz(err)
	}
	err = k.u.magazyn.DoKosza(c.udzial, c.sciezka)
	k.zmiana(c, "Do kosza", err)
	return tlumacz(err)
}

// sprawdzUsuwany: DELE tylko pliki, RMD tylko puste foldery.
func sprawdzUsuwany(k *klient, c cel, w rdzen.Wpis, folder bool) error {
	if w.Folder != folder {
		if w.Folder {
			return errToFolder
		}
		return errToPlik
	}
	if !folder {
		return nil
	}
	wpisy, err := k.u.magazyn.Lista(c.udzial, c.sciezka)
	if err != nil {
		return err
	}
	if len(wpisy) > 0 {
		return errFolderNiePusty
	}
	return nil
}

// RemoveAll: nieobsługiwane (rekurencyjne usuwanie tylko z WWW).
func (k *klient) RemoveAll(string) error { return errNieObslugiwane }

// errInnyUdzial: przeniesienie między udziałami.
var errInnyUdzial = errors.New("Nie można przenosić między udziałami.")

// Rename (RNFR/RNTO): w obrębie jednego udziału, OpZmiana.
func (k *klient) Rename(stara, nowa string) error {
	z, err := rozloz(stara)
	if err != nil {
		return tlumacz(err)
	}
	na, err := rozloz(nowa)
	if err != nil {
		return tlumacz(err)
	}
	if z.udzial != na.udzial {
		return errInnyUdzial
	}
	if err := k.wolnoZapis(rdzen.OpZmiana, z); err != nil {
		return tlumacz(err)
	}
	if err := k.wolnoZapis(rdzen.OpZmiana, na); err != nil {
		return tlumacz(err)
	}
	err = k.u.magazyn.ZmienNazwe(z.udzial, z.sciezka, na.sciezka)
	k.zmiana(z, "Zmiana nazwy na "+na.sciezka, err)
	return tlumacz(err)
}

// Create, Open, OpenFile: transfer idzie przez GetHandle, lista przez ReadDir.
func (k *klient) Create(string) (afero.File, error) { return nil, errNieObslugiwane }
func (k *klient) Open(string) (afero.File, error)   { return nil, errNieObslugiwane }
func (k *klient) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, errNieObslugiwane
}

// Chmod, Chown, Chtimes: nieobsługiwane (prawa nadaje bramka, nie system plików).
func (k *klient) Chmod(string, os.FileMode) error            { return errNieObslugiwane }
func (k *klient) Chown(string, int, int) error               { return errNieObslugiwane }
func (k *klient) Chtimes(string, time.Time, time.Time) error { return errNieObslugiwane }
