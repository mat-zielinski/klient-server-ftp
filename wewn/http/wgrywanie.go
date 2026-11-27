package http

import (
	"errors"
	nethttp "net/http"
	"path"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

// parametryWgrania: ?przesuniecie=N&koniec=0|1&nadpisz=0|1.
type parametryWgrania struct {
	przesuniecie int64
	koniec       bool
	nadpisz      bool
}

func czytajParametryWgrania(r *nethttp.Request) (parametryWgrania, error) {
	q := r.URL.Query()
	p := parametryWgrania{koniec: q.Get("koniec") == "1", nadpisz: q.Get("nadpisz") == "1"}
	if t := q.Get("przesuniecie"); t != "" {
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil || n < 0 {
			return p, errZlaLiczba
		}
		p.przesuniecie = n
	}
	return p, nil
}

// wgraj: PUT /api/wgraj, treść = kawałek (do MaksKawalekWgrania); koniec=1 zatwierdza plik.
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
		if err := s.folderyNadrzedne(u, udzial, sciezka); err != nil {
			s.blad(w, r, err)
			return
		}
	}
	if p.przesuniecie == 0 && !p.nadpisz {
		// Wcześnie: nie przyjmujemy całego pliku, żeby na końcu odmówić nadpisania.
		if _, err := s.magazyn.Info(udzial, sciezka); err == nil {
			s.blad(w, r, rdzen.ErrJuzIstnieje)
			return
		} else if !errors.Is(err, rdzen.ErrNieZnaleziono) {
			s.blad(w, r, err)
			return
		}
	}
	tresc := nethttp.MaxBytesReader(w, r.Body, konfiguracja.MaksKawalekWgrania)
	n, err := s.magazyn.DopiszKawalek(udzial, sciezka, p.przesuniecie, tresc)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	razem := p.przesuniecie + n
	if p.koniec {
		if err := s.magazyn.Zatwierdz(udzial, sciezka, p.nadpisz); err != nil {
			s.blad(w, r, err)
			return
		}
		wpis := s.wpisDziennika(r, dziennik.RodzajWgranie, u.Login)
		wpis.Udzial, wpis.Sciezka, wpis.Bajty, wpis.Wynik = udzial, sciezka, razem, dziennik.WynikOK
		s.dziennik.Zapisz(wpis)
		s.odswiezIndeks(udzial)
	}
	piszJSON(w, nethttp.StatusOK, map[string]int64{"przeslano": razem})
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
