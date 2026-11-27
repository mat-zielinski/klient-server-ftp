package http

import (
	"errors"
	nethttp "net/http"
	"path"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/transfery"
)

// parametryWgrania: ?przesuniecie=N&koniec=0|1&nadpisz=0|1 i nagłówek X-Rozmiar-Calkowity.
type parametryWgrania struct {
	przesuniecie int64
	koniec       bool
	nadpisz      bool
	// calkowity: rozmiar całego pliku z nagłówka; 0 = nieznany.
	calkowity int64
}

func czytajParametryWgrania(r *nethttp.Request) (parametryWgrania, error) {
	q := r.URL.Query()
	p := parametryWgrania{koniec: q.Get("koniec") == "1", nadpisz: q.Get("nadpisz") == "1"}
	var err error
	if p.przesuniecie, err = liczbaNieujemna(q.Get("przesuniecie")); err != nil {
		return p, err
	}
	p.calkowity, err = liczbaNieujemna(r.Header.Get(konfiguracja.NaglowekRozmiaruCalkowitego))
	return p, err
}

// liczbaNieujemna: pusty tekst = 0, ujemna albo nie-liczba = errZlaLiczba.
func liczbaNieujemna(t string) (int64, error) {
	if t == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(t, 10, 64)
	if err != nil || n < 0 {
		return 0, errZlaLiczba
	}
	return n, nil
}

// wgraj: PUT /api/wgraj, treść = kawałek (do MaksKawalekWgrania); koniec=1 zatwierdza plik.
// Cały plik to jeden transfer w rejestrze (od pierwszego kawałka do koniec=1 albo bezczynności);
// wpis dziennika „wgranie” robi wyłącznie callback naKoniec rejestru (jak przy pobraniu).
func (s *serwer) wgraj(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZmiany(r, u, rdzen.OpWgranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	p, err := czytajParametryWgrania(r)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if p.przesuniecie == 0 {
		if err := s.przygotujWgranie(u, udzial, sciezka, p.nadpisz); err != nil {
			s.blad(w, r, err)
			return
		}
	}
	k, a := s.zacznijKawalek(r, u, udzial, sciezka, p)
	tresc := a.t.CzytajZ(nethttp.MaxBytesReader(w, r.Body, konfiguracja.MaksKawalekWgrania))
	n, err := s.magazyn.DopiszKawalek(udzial, sciezka, p.przesuniecie, tresc)
	if err != nil {
		s.koniecKawalka(k, a, wynikBleduWgrania(r, err))
		s.blad(w, r, err)
		return
	}
	razem := p.przesuniecie + n
	if !p.koniec {
		s.koniecKawalka(k, a, "")
		piszJSON(w, nethttp.StatusOK, map[string]int64{"przeslano": razem})
		return
	}
	if err := s.magazyn.Zatwierdz(udzial, sciezka, p.nadpisz); err != nil {
		s.koniecKawalka(k, a, transfery.WynikBlad)
		s.blad(w, r, err)
		return
	}
	s.koniecKawalka(k, a, transfery.WynikOk)
	s.odswiezIndeks(udzial)
	piszJSON(w, nethttp.StatusOK, map[string]int64{"przeslano": razem})
}

// przygotujWgranie (pierwszy kawałek): foldery nadrzędne i wczesna odmowa nadpisania (nie przyjmujemy
// całego pliku, żeby na końcu odmówić).
func (s *serwer) przygotujWgranie(u rdzen.Uzytkownik, udzial, sciezka string, nadpisz bool) error {
	if err := s.folderyNadrzedne(u, udzial, sciezka); err != nil {
		return err
	}
	if nadpisz {
		return nil
	}
	if _, err := s.magazyn.Info(udzial, sciezka); err == nil {
		return rdzen.ErrJuzIstnieje
	} else if !errors.Is(err, rdzen.ErrNieZnaleziono) {
		return err
	}
	return nil
}

// stanWgrania: GET /api/wgraj, ile bajtów już zapisano (wznowienie).
func (s *serwer) stanWgrania(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZmiany(r, u, rdzen.OpWgranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	n, err := s.magazyn.RozmiarKawalkow(udzial, sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]int64{"przeslano": n})
}

// folderyNadrzedne tworzy brakujące foldery nad plikiem (wgrywanie całych folderów); każdy przez
// bramkę i sprawdzenie nazwy, istniejący = w porządku.
func (s *serwer) folderyNadrzedne(u rdzen.Uzytkownik, udzial, sciezka string) error {
	czlony := strings.Split(strings.Trim(path.Dir(sciezka), "/"), "/")
	biezaca := ""
	for _, c := range czlony {
		if c == "" {
			continue
		}
		biezaca += "/" + c
		if err := rdzen.SprawdzNazwe(c); err != nil {
			return err
		}
		if _, err := s.dostepDoUdzialu(u, rdzen.OpWgranie, udzial); err != nil {
			return err
		}
		if err := s.magazyn.NowyFolder(udzial, biezaca); err != nil && !errors.Is(err, rdzen.ErrJuzIstnieje) {
			return err
		}
	}
	return nil
}
