package http

import (
	"errors"
	nethttp "net/http"
	"path"
	"strings"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/udostepnienia"
)

func (s *serwer) trasyUdostepnien(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/udostepnienia", s.zalogowany(s.listaLinkow))
	m.HandleFunc("POST /api/udostepnienia", s.zalogowany(s.nowyLink))
	m.HandleFunc("DELETE /api/udostepnienia", s.zalogowany(s.usunLink))
	m.HandleFunc("GET /s/{id}", s.plikStatyczny)
	m.HandleFunc("GET /api/publiczny/{id}", s.publicznyInfo)
	m.HandleFunc("GET /api/publiczny/{id}/pobierz", s.publicznyPobierz)
}

// widokLinku: link z gotowym adresem /s/<id>.
type widokLinku struct {
	udostepnienia.Link
	Adres string `json:"adres"`
}

func adresLinku(r *nethttp.Request, id string) string {
	return bazaAdresu(r) + konfiguracja.PrefiksLinkuPublicznego + id
}

func (s *serwer) listaLinkow(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	wynik := []widokLinku{}
	for _, l := range s.udostepnienia.Lista(ktoWidzi(u)) {
		l.HashHasla = ""
		wynik = append(wynik, widokLinku{Link: l, Adres: adresLinku(r, l.ID)})
	}
	piszJSON(w, nethttp.StatusOK, wynik)
}

// daneLinku: POST /api/udostepnienia.
type daneLinku struct {
	Udzial      string `json:"udzial"`
	Sciezka     string `json:"sciezka"`
	Wygasa      string `json:"wygasa"`
	Haslo       string `json:"haslo"`
	LimitPobran int    `json:"limit_pobran"`
}

func (s *serwer) nowyLink(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d daneLinku
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	sciezka, err := rdzen.OczyscSciezke(d.Sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if _, err := s.dostepDoUdzialu(u, rdzen.OpUdostepnienie, d.Udzial); err != nil {
		s.blad(w, r, err)
		return
	}
	var wygasa time.Time
	if d.Wygasa != "" {
		if wygasa, err = time.Parse(time.RFC3339, d.Wygasa); err != nil {
			s.blad(w, r, errZlaData)
			return
		}
	}
	if _, err := s.magazyn.Info(d.Udzial, sciezka); err != nil {
		s.blad(w, r, err)
		return
	}
	l, err := s.udostepnienia.Utworz(udostepnienia.Link{
		Udzial: d.Udzial, Sciezka: sciezka, Autor: u.Login, Wygasa: wygasa, LimitPobran: d.LimitPobran,
	}, d.Haslo)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	l.HashHasla = ""
	s.zapiszZmiane(r, u.Login, dziennik.RodzajUdostepnienie, l.Udzial, l.Sciezka, "Nowy link publiczny "+l.ID+".")
	piszJSON(w, nethttp.StatusOK, widokLinku{Link: l, Adres: adresLinku(r, l.ID)})
}

// usunLink: admin dowolny, inni tylko własne (autor sprawdzany przez Pobierz).
func (s *serwer) usunLink(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	id := r.URL.Query().Get("id")
	l, err := s.udostepnienia.Pobierz(id)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if !czyAdmin(u) && l.Autor != u.Login {
		s.blad(w, r, rdzen.ErrBrakPrawa)
		return
	}
	if err := s.udostepnienia.Usun(id); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajUdostepnienie, l.Udzial, l.Sciezka, "Usunięto link publiczny "+l.ID+".")
	pusto(w)
}

// haslo linku: nagłówek X-Haslo, awaryjnie parametr ?haslo.
func hasloLinku(r *nethttp.Request) string {
	if h := r.Header.Get(konfiguracja.NaglowekHaslaLinku); h != "" {
		return h
	}
	return r.URL.Query().Get("haslo")
}

var errAutorBezDostepu = nowyBlad(nethttp.StatusForbidden, "Autor linku nie ma już dostępu do tego elementu.")

// celLinku sprawdza link (ważność, hasło), czy autor nadal ma prawo pobierania, i zwraca pełną
// ścieżkę w udziale: ścieżka podrzędna z ?sciezka zawsze wewnątrz ścieżki linku.
func (s *serwer) celLinku(r *nethttp.Request) (udostepnienia.Link, string, string, error) {
	l, err := s.udostepnienia.Sprawdz(r.PathValue("id"), hasloLinku(r))
	if err != nil {
		return l, "", "", err
	}
	autor, err := s.konta.Uzytkownik(l.Autor)
	if err != nil {
		return l, "", "", errAutorBezDostepu
	}
	if _, err := s.dostepDoUdzialu(autor, rdzen.OpPobranie, l.Udzial); err != nil {
		return l, "", "", errAutorBezDostepu
	}
	pod, err := rdzen.OczyscSciezke(r.URL.Query().Get("sciezka"))
	if err != nil {
		return l, "", "", err
	}
	pelna, err := wewnatrzLinku(l.Sciezka, pod)
	return l, pod, pelna, err
}

// wewnatrzLinku łączy ścieżkę linku z podrzędną; wynik poza ścieżką linku = ErrPozaUdzialem.
func wewnatrzLinku(korzen, pod string) (string, error) {
	pelna := path.Join(korzen, pod)
	if pelna != korzen && !strings.HasPrefix(pelna, strings.TrimSuffix(korzen, "/")+"/") {
		return "", rdzen.ErrPozaUdzialem
	}
	return pelna, nil
}

// wzglednyWLinku: ścieżka widziana przez odbiorcę linku (bez ścieżek powyżej linku).
func wzglednyWLinku(korzen, pelna string) string {
	if korzen == "/" {
		return pelna
	}
	w := strings.TrimPrefix(pelna, korzen)
	if w == "" {
		return "/"
	}
	return w
}

// publicznyInfo: GET /api/publiczny/<id>; złe hasło = 401, wygasły = 410, blokada = 429.
func (s *serwer) publicznyInfo(w nethttp.ResponseWriter, r *nethttp.Request) {
	l, pod, pelna, err := s.celLinku(r)
	if err != nil {
		s.bladLinku(w, r, err)
		return
	}
	wpis, err := s.magazyn.Info(l.Udzial, pelna)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	odp := map[string]any{
		"id": l.ID, "nazwa": nazwaLinku(l), "folder": wpis.Folder, "rozmiar": wpis.Rozmiar,
		"zmieniono": wpis.Zmieniono, "wygasa": l.Wygasa, "sciezka": pod, "rodzaj": wpis.Rodzaj,
	}
	if wpis.Folder {
		wpisy, err := s.magazyn.Lista(l.Udzial, pelna)
		if err != nil {
			s.blad(w, r, err)
			return
		}
		rdzen.Sortuj(wpisy, r.URL.Query().Get("sort"), r.URL.Query().Get("malejaco") == "1")
		for i := range wpisy {
			wpisy[i].Sciezka = wzglednyWLinku(l.Sciezka, wpisy[i].Sciezka)
		}
		if wpisy == nil {
			wpisy = []rdzen.Wpis{}
		}
		odp["wpisy"] = wpisy
	}
	piszJSON(w, nethttp.StatusOK, odp)
}

// nazwaLinku: nazwa elementu linku (korzeń udziału = nazwa udziału).
func nazwaLinku(l udostepnienia.Link) string {
	if l.Sciezka == "/" {
		return l.Udzial
	}
	return path.Base(l.Sciezka)
}

// bladLinku: złe hasło linku = 401 z prośbą o hasło.
func (s *serwer) bladLinku(w nethttp.ResponseWriter, r *nethttp.Request, err error) {
	if errors.Is(err, rdzen.ErrZleDane) {
		piszBlad(w, nethttp.StatusUnauthorized, "Link wymaga poprawnego hasła.")
		return
	}
	s.blad(w, r, err)
}

// publicznyPobierz: GET /api/publiczny/<id>/pobierz[?sciezka]; folder = ZIP. Pobranie liczone
// (ZaliczPobranie) przed wysłaniem; wznowienie z Range od środka pliku nie jest liczone drugi raz.
func (s *serwer) publicznyPobierz(w nethttp.ResponseWriter, r *nethttp.Request) {
	l, _, pelna, err := s.celLinku(r)
	if err != nil {
		s.bladLinku(w, r, err)
		return
	}
	wpis, err := s.magazyn.Info(l.Udzial, pelna)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if czyPelnePobranie(r) {
		if err := s.udostepnienia.ZaliczPobranie(l.ID); err != nil {
			s.blad(w, r, err)
			return
		}
	}
	kto := konfiguracja.PrefiksKtoLinku + l.ID
	if wpis.Folder {
		s.wyslijZip(w, r, kto, l.Udzial, []string{pelna})
		return
	}
	s.wyslijPlik(w, r, zadaniePobrania{kto: kto, udzial: l.Udzial, sciezka: pelna,
		podglad: r.URL.Query().Get("podglad") == "1"})
}

// czyPelnePobranie: bez Range albo Range od bajtu 0.
func czyPelnePobranie(r *nethttp.Request) bool {
	z := r.Header.Get("Range")
	return z == "" || strings.HasPrefix(strings.ReplaceAll(z, " ", ""), "bytes=0-")
}
