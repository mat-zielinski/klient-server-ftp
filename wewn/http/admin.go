package http

import (
	nethttp "net/http"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/siec"
	"skrytka/wewn/uslugi/dziennik"
)

func (s *serwer) trasyAdmina(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/konta", s.tylkoAdmin(s.listaKont))
	m.HandleFunc("POST /api/konta", s.tylkoAdmin(s.dodajKonto))
	m.HandleFunc("PUT /api/konta", s.tylkoAdmin(s.zmienKonto))
	m.HandleFunc("DELETE /api/konta", s.tylkoAdmin(s.usunKonto))
	m.HandleFunc("POST /api/konta/haslo", s.tylkoAdmin(s.hasloKonta))
	m.HandleFunc("GET /api/ustawienia", s.tylkoAdmin(s.ustawienia))
	m.HandleFunc("PUT /api/ustawienia", s.tylkoAdmin(s.zapiszUstawienia))
	m.HandleFunc("GET /api/siec", s.tylkoAdmin(s.stanSieci))
	m.HandleFunc("POST /api/siec/zapora", s.tylkoAdmin(s.zapora))
	m.HandleFunc("GET /api/pulpit", s.tylkoAdmin(s.pulpit))
}

// administracja: wpis dziennika o zmianie wykonanej przez admina.
func (s *serwer) administracja(r *nethttp.Request, kto, opis string) {
	wpis := s.wpisDziennika(r, dziennik.RodzajAdministracja, kto)
	wpis.Wynik, wpis.Opis = dziennik.WynikOK, opis
	s.dziennik.Zapisz(wpis)
}

func (s *serwer) listaKont(w nethttp.ResponseWriter, _ *nethttp.Request, _ rdzen.Uzytkownik) {
	wynik := []widokUzytkownika{}
	for _, u := range s.konta.Lista() {
		wynik = append(wynik, widok(u))
	}
	piszJSON(w, nethttp.StatusOK, wynik)
}

// daneKonta: Uzytkownik + haslo (bez hasha).
type daneKonta struct {
	Login     string                 `json:"login"`
	Admin     bool                   `json:"admin"`
	Prawa     map[string]rdzen.Prawo `json:"prawa"`
	Wylaczony bool                   `json:"wylaczony"`
	FTPS      bool                   `json:"ftps"`
	Haslo     string                 `json:"haslo"`
}

func (d daneKonta) uzytkownik() (rdzen.Uzytkownik, error) {
	for _, p := range d.Prawa {
		if p != rdzen.PrawoBrak && p != rdzen.PrawoOdczyt && p != rdzen.PrawoZapis {
			return rdzen.Uzytkownik{}, nowyBlad(nethttp.StatusBadRequest, "Nieznane prawo do udziału.")
		}
	}
	return rdzen.Uzytkownik{Login: d.Login, Admin: d.Admin, Prawa: d.Prawa, Wylaczony: d.Wylaczony, FTPS: d.FTPS}, nil
}

func (s *serwer) dodajKonto(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	var d daneKonta
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	u, err := d.uzytkownik()
	if err == nil {
		err = s.konta.Dodaj(u, d.Haslo)
	}
	if err != nil {
		s.blad(w, r, err)
		return
	}
	s.administracja(r, a.Login, "Dodano konto "+u.Login+".")
	pusto(w)
}

func (s *serwer) zmienKonto(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	var d daneKonta
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	d.Login = r.URL.Query().Get("login")
	u, err := d.uzytkownik()
	if err == nil {
		err = s.konta.Zmien(u)
	}
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if u.Wylaczony {
		s.sesje.ZakonczWszystkie(u.Login)
	}
	s.administracja(r, a.Login, "Zmieniono konto "+u.Login+".")
	pusto(w)
}

func (s *serwer) usunKonto(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	login := r.URL.Query().Get("login")
	if login == a.Login {
		s.blad(w, r, nowyBlad(nethttp.StatusBadRequest, "Nie możesz usunąć własnego konta."))
		return
	}
	if err := s.konta.Usun(login); err != nil {
		s.blad(w, r, err)
		return
	}
	s.sesje.ZakonczWszystkie(login)
	s.administracja(r, a.Login, "Usunięto konto "+login+".")
	pusto(w)
}

func (s *serwer) hasloKonta(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	var d struct {
		Haslo string `json:"haslo"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	login := r.URL.Query().Get("login")
	if err := s.konta.UstawHaslo(login, d.Haslo); err != nil {
		s.blad(w, r, err)
		return
	}
	s.administracja(r, a.Login, "Zmieniono hasło konta "+login+".")
	pusto(w)
}

func (s *serwer) ustawienia(w nethttp.ResponseWriter, _ *nethttp.Request, _ rdzen.Uzytkownik) {
	piszJSON(w, nethttp.StatusOK, s.konfiguracja())
}

// sprawdzUstawienia: tryb, porty, nazwy i ścieżki udziałów (bez dotykania dysku).
func sprawdzUstawienia(k konfiguracja.Konfiguracja) error {
	if _, err := trybSieci(string(k.TrybSieci), ""); err != nil || k.TrybSieci == "" {
		return nowyBlad(nethttp.StatusBadRequest, "Nieznany tryb sieci.")
	}
	for _, p := range []int{k.PortHTTP, k.PortHTTPS, k.PortFTPS, k.PortyPasywne[0], k.PortyPasywne[1]} {
		if p < konfiguracja.MinPort || p > konfiguracja.MaksPort {
			return nowyBlad(nethttp.StatusBadRequest, "Port poza zakresem 1–65535.")
		}
	}
	if k.PortyPasywne[0] > k.PortyPasywne[1] {
		return nowyBlad(nethttp.StatusBadRequest, "Zakres portów pasywnych jest odwrócony.")
	}
	nazwy := map[string]bool{}
	for _, u := range k.Udzialy {
		if rdzen.SprawdzNazwe(u.Nazwa) != nil {
			return nowyBlad(nethttp.StatusBadRequest, "Niedozwolona nazwa udziału: "+u.Nazwa+".")
		}
		klucz := strings.ToLower(u.Nazwa)
		if nazwy[klucz] {
			return nowyBlad(nethttp.StatusConflict, "Powtórzona nazwa udziału: "+u.Nazwa+".")
		}
		nazwy[klucz] = true
		if strings.TrimSpace(u.Sciezka) == "" {
			return nowyBlad(nethttp.StatusBadRequest, "Podaj folder udziału "+u.Nazwa+".")
		}
	}
	return nil
}

// wymagaRestartu: zmiany, które działają dopiero po ponownym uruchomieniu serwera.
func wymagaRestartu(a, b konfiguracja.Konfiguracja) bool {
	return a.TrybSieci != b.TrybSieci || a.AdresNasluchu != b.AdresNasluchu || a.PortHTTP != b.PortHTTP ||
		a.PortHTTPS != b.PortHTTPS || a.PortFTPS != b.PortFTPS || a.PortyPasywne != b.PortyPasywne ||
		a.FTPSWlaczone != b.FTPSWlaczone || a.Certyfikat != b.Certyfikat || a.Klucz != b.Klucz ||
		a.DomenaLetsEncrypt != b.DomenaLetsEncrypt
}

func (s *serwer) zapiszUstawienia(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	stara := s.konfiguracja()
	nowa := stara
	nowa.Udzialy = nil
	if err := czytajJSON(w, r, &nowa); err != nil {
		s.blad(w, r, err)
		return
	}
	if nowa.Udzialy == nil {
		nowa.Udzialy = stara.Udzialy
	}
	if err := sprawdzUstawienia(nowa); err != nil {
		s.blad(w, r, err)
		return
	}
	if err := s.zapiszKonfig(nowa); err != nil {
		s.blad(w, r, err)
		return
	}
	s.administracja(r, a.Login, "Zmieniono ustawienia.")
	if wymagaRestartu(stara, nowa) {
		piszJSON(w, nethttp.StatusOK, map[string]any{"restart": true, "komunikat": "Wymaga restartu serwera."})
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{"restart": false})
}

// pamiecAdresuPublicznego: siec.AdresPubliczny pyta usługę w sieci za każdym razem, więc pamiętamy wynik.
type pamiecAdresuPublicznego struct {
	mu    sync.Mutex
	adres string
	kiedy time.Time
}

func (s *serwer) adresPubliczny() string {
	p := &s.publiczny
	p.mu.Lock()
	defer p.mu.Unlock()
	teraz := s.teraz()
	if p.kiedy.IsZero() || teraz.Sub(p.kiedy) > konfiguracja.CzasPamieciAdresuPublicznego {
		p.adres, p.kiedy = s.siec.adresPubliczny(), teraz
	}
	return p.adres
}

func (s *serwer) stanSieci(w nethttp.ResponseWriter, _ *nethttp.Request, _ rdzen.Uzytkownik) {
	k := s.konfiguracja()
	adresy := s.siec.adresyLokalne()
	if adresy == nil {
		adresy = []siec.Adres{}
	}
	otwarty := func(port int) bool { ok, _ := s.siec.sprawdzPort(port); return ok }
	piszJSON(w, nethttp.StatusOK, map[string]any{
		"adresy":    adresy,
		"publiczny": s.adresPubliczny(),
		"tryb":      k.TrybSieci,
		"porty":     map[string]int{"http": k.PortHTTP, "https": k.PortHTTPS, "ftps": k.PortFTPS},
		"porty_otwarte": map[string]bool{
			"http": otwarty(k.PortHTTP), "https": otwarty(k.PortHTTPS), "ftps": otwarty(k.PortFTPS),
		},
		"odcisk_certyfikatu": s.odcisk,
	})
}

// portyZapory: HTTP, HTTPS, a przy włączonym FTPS także port sterujący i pasywne.
func portyZapory(k konfiguracja.Konfiguracja) []int {
	porty := []int{k.PortHTTP, k.PortHTTPS}
	if k.FTPSWlaczone {
		porty = append(porty, k.PortFTPS)
		for p := k.PortyPasywne[0]; p > 0 && p <= k.PortyPasywne[1]; p++ {
			porty = append(porty, p)
		}
	}
	return porty
}

func (s *serwer) zapora(w nethttp.ResponseWriter, r *nethttp.Request, a rdzen.Uzytkownik) {
	if err := s.siec.regulaZapory(konfiguracja.DomyslnaNazwaRegulyZapory, portyZapory(s.konfiguracja())); err != nil {
		s.blad(w, r, err)
		return
	}
	s.administracja(r, a.Login, "Dodano regułę Zapory Windows.")
	pusto(w)
}

// miejsceUdzialu: wolne i razem (bajty) dysku pod folderem udziału; błąd odczytu = 0.
type miejsceUdzialu struct {
	Nazwa string `json:"nazwa"`
	Wolne uint64 `json:"wolne"`
	Razem uint64 `json:"razem"`
}

func (s *serwer) pulpit(w nethttp.ResponseWriter, _ *nethttp.Request, _ rdzen.Uzytkownik) {
	teraz := s.teraz()
	rok, mies, dzien := teraz.Date()
	poczatek := time.Date(rok, mies, dzien, 0, 0, 0, 0, teraz.Location())
	dzis := map[string]int64{"bajty": 0, "pobrania": 0}
	if st, err := s.dziennik.Statystyki(poczatek, teraz, konfiguracja.KrokDzien); err == nil {
		dzis["bajty"], dzis["pobrania"] = st.SumaBajtow, int64(st.SumaPobran)
	}
	udzialy := []miejsceUdzialu{}
	for _, u := range s.konfiguracja().Udzialy {
		m := miejsceUdzialu{Nazwa: u.Nazwa}
		if z, err := s.zajetosc(u.Sciezka); err == nil {
			m.Wolne, m.Razem = z.Wolne, z.Razem
		}
		udzialy = append(udzialy, m)
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{
		"aktywne":      len(s.transfery.Aktywne("")),
		"predkosc_bps": s.transfery.PredkoscLaczna(),
		"dzis":         dzis,
		"udzialy":      udzialy,
		"czas_pracy_s": int64(teraz.Sub(s.start).Seconds()),
	})
}
