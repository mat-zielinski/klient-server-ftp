package http

import (
	nethttp "net/http"
	"path"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

func (s *serwer) trasyPlikow(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/udzialy", s.zalogowany(s.udzialy))
	m.HandleFunc("GET /api/lista", s.zalogowany(s.lista))
	m.HandleFunc("GET /api/szukaj", s.zalogowany(s.szukaj))
	m.HandleFunc("GET /api/pobierz", s.zalogowany(s.pobierz))
	m.HandleFunc("GET /api/zip", s.zalogowany(s.zipGet))
	m.HandleFunc("POST /api/zip", s.zalogowany(s.zipPost))
	m.HandleFunc("GET /api/protokoly", s.zalogowany(s.protokoly))
	m.HandleFunc("POST /api/folder", s.zalogowany(s.nowyFolder))
	m.HandleFunc("POST /api/zmien", s.zalogowany(s.zmienNazwe))
	m.HandleFunc("DELETE /api/plik", s.zalogowany(s.usunPlik))
	m.HandleFunc("PUT /api/wgraj", s.zalogowany(s.wgraj))
	m.HandleFunc("GET /api/wgraj", s.zalogowany(s.stanWgrania))
}

// dostepDoUdzialu: bramka rdzenia PRZED jakąkolwiek operacją magazynu. Nieznany udział dla kogoś
// bez prawa daje 403 (nie zdradzamy, które udziały istnieją).
func (s *serwer) dostepDoUdzialu(u rdzen.Uzytkownik, op rdzen.Operacja, nazwa string) (konfiguracja.Udzial, error) {
	ud, err := s.magazyn.Udzial(nazwa)
	if err != nil {
		if e := rdzen.CzyWolno(u, op, nazwa, false); e != nil {
			return konfiguracja.Udzial{}, e
		}
		return konfiguracja.Udzial{}, err
	}
	if err := rdzen.CzyWolno(u, op, nazwa, ud.TylkoOdczyt); err != nil {
		return konfiguracja.Udzial{}, err
	}
	return ud, nil
}

// celZadania: udział i oczyszczona ścieżka z parametrów ?udzial&sciezka, po sprawdzeniu prawa.
func (s *serwer) celZadania(r *nethttp.Request, u rdzen.Uzytkownik, op rdzen.Operacja) (string, string, error) {
	udzial := r.URL.Query().Get("udzial")
	sciezka, err := rdzen.OczyscSciezke(r.URL.Query().Get("sciezka"))
	if err != nil {
		return "", "", err
	}
	if _, err := s.dostepDoUdzialu(u, op, udzial); err != nil {
		return "", "", err
	}
	return udzial, sciezka, nil
}

// celZmiany: jak celZadania, ale ścieżka nie może być korzeniem, a ostatni człon musi być poprawną nazwą.
func (s *serwer) celZmiany(r *nethttp.Request, u rdzen.Uzytkownik, op rdzen.Operacja) (string, string, error) {
	udzial, sciezka, err := s.celZadania(r, u, op)
	if err != nil {
		return "", "", err
	}
	if err := sprawdzNieKorzen(sciezka); err != nil {
		return "", "", err
	}
	return udzial, sciezka, nil
}

func sprawdzNieKorzen(sciezka string) error {
	if sciezka == "/" {
		return errKorzen
	}
	return rdzen.SprawdzNazwe(path.Base(sciezka))
}

// infoUdzialu: GET /api/udzialy.
type infoUdzialu struct {
	Nazwa       string      `json:"nazwa"`
	TylkoOdczyt bool        `json:"tylko_odczyt"`
	Prawo       rdzen.Prawo `json:"prawo"`
}

func (s *serwer) udzialy(w nethttp.ResponseWriter, _ *nethttp.Request, u rdzen.Uzytkownik) {
	wynik := []infoUdzialu{}
	for _, ud := range s.magazyn.Udzialy() {
		if rdzen.CzyWolno(u, rdzen.OpOdczyt, ud.Nazwa, ud.TylkoOdczyt) != nil {
			continue
		}
		prawo := rdzen.PrawoOdczyt
		if rdzen.CzyWolno(u, rdzen.OpWgranie, ud.Nazwa, ud.TylkoOdczyt) == nil {
			prawo = rdzen.PrawoZapis
		}
		wynik = append(wynik, infoUdzialu{Nazwa: ud.Nazwa, TylkoOdczyt: ud.TylkoOdczyt, Prawo: prawo})
	}
	piszJSON(w, nethttp.StatusOK, wynik)
}

func (s *serwer) lista(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpOdczyt)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	wpisy, err := s.magazyn.Lista(udzial, sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if wpisy == nil {
		wpisy = []rdzen.Wpis{}
	}
	q := r.URL.Query()
	rdzen.Sortuj(wpisy, q.Get("sort"), q.Get("malejaco") == "1")
	piszJSON(w, nethttp.StatusOK, map[string]any{"sciezka": sciezka, "wpisy": wpisy})
}

func (s *serwer) szukaj(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpOdczyt)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		s.blad(w, r, errBrakZapytania)
		return
	}
	wpisy, err := s.magazyn.Szukaj(udzial, sciezka, q, konfiguracja.LimitWynikowSzukania)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if wpisy == nil {
		wpisy = []rdzen.Wpis{}
	}
	piszJSON(w, nethttp.StatusOK, wpisy)
}

// zapiszZmiane: wpis dziennika o operacji na plikach.
func (s *serwer) zapiszZmiane(r *nethttp.Request, kto, rodzaj, udzial, sciezka, opis string) {
	wpis := s.wpisDziennika(r, rodzaj, kto)
	wpis.Udzial, wpis.Sciezka, wpis.Wynik, wpis.Opis = udzial, sciezka, dziennik.WynikOK, opis
	s.dziennik.Zapisz(wpis)
	if rodzaj == dziennik.RodzajZmiana {
		s.odswiezIndeks(udzial)
	}
}

func (s *serwer) nowyFolder(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZmiany(r, u, rdzen.OpWgranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if err := s.magazyn.NowyFolder(udzial, sciezka); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, sciezka, "Nowy folder.")
	pusto(w)
}

func (s *serwer) zmienNazwe(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d struct {
		Z  string `json:"z"`
		Na string `json:"na"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	udzial := r.URL.Query().Get("udzial")
	if _, err := s.dostepDoUdzialu(u, rdzen.OpZmiana, udzial); err != nil {
		s.blad(w, r, err)
		return
	}
	// Źródło może mieć dowolną istniejącą nazwę; sprawdzamy tylko nową.
	z, err := rdzen.OczyscSciezke(d.Z)
	if err == nil && z == "/" {
		err = errKorzen
	}
	if err != nil {
		s.blad(w, r, err)
		return
	}
	na, err := oczyscNieKorzen(d.Na)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if err := s.magazyn.ZmienNazwe(udzial, z, na); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, z, "Zmiana nazwy na "+na+".")
	pusto(w)
}

func oczyscNieKorzen(s string) (string, error) {
	c, err := rdzen.OczyscSciezke(s)
	if err != nil {
		return "", err
	}
	return c, sprawdzNieKorzen(c)
}

func (s *serwer) usunPlik(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpUsuniecie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if sciezka == "/" {
		s.blad(w, r, errKorzen)
		return
	}
	if err := s.magazyn.DoKosza(udzial, sciezka); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, sciezka, "Przeniesiono do kosza.")
	pusto(w)
}
