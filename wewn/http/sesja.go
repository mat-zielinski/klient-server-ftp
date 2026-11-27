package http

import (
	"errors"
	nethttp "net/http"
	"path/filepath"
	"strings"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

// widokUzytkownika: użytkownik bez hasha hasła (nigdy nie wysyłamy rdzen.Uzytkownik wprost).
type widokUzytkownika struct {
	Login     string                 `json:"login"`
	Admin     bool                   `json:"admin"`
	Prawa     map[string]rdzen.Prawo `json:"prawa"`
	Wylaczony bool                   `json:"wylaczony"`
	FTPS      bool                   `json:"ftps"`
}

func widok(u rdzen.Uzytkownik) widokUzytkownika {
	p := u.Prawa
	if p == nil {
		p = map[string]rdzen.Prawo{}
	}
	return widokUzytkownika{Login: u.Login, Admin: u.Admin, Prawa: p, Wylaczony: u.Wylaczony, FTPS: u.FTPS}
}

func (s *serwer) trasySesji(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/stan", s.stan)
	m.HandleFunc("GET /api/zdrowie", zdrowie)
	m.HandleFunc("POST /api/kreator", s.kreator)
	m.HandleFunc("POST /api/logowanie", s.logowanie)
	m.HandleFunc("POST /api/wylogowanie", s.wylogowanie)
}

// stan: GET /api/stan (bez sesji też działa).
func (s *serwer) stan(w nethttp.ResponseWriter, r *nethttp.Request) {
	odp := struct {
		Wersja     string            `json:"wersja"`
		Nazwa      string            `json:"nazwa"`
		Kreator    bool              `json:"kreator"`
		Zalogowany bool              `json:"zalogowany"`
		Uzytkownik *widokUzytkownika `json:"uzytkownik"`
		Motyw      string            `json:"motyw"`
	}{
		Wersja: wersja(), Nazwa: konfiguracja.NazwaAplikacji,
		Kreator: s.konta.CzyPusto(), Motyw: s.konfiguracja().Motyw,
	}
	if odp.Motyw == "" {
		odp.Motyw = konfiguracja.DomyslnyMotyw
	}
	if u, ok := uzytkownikZ(r); ok {
		v := widok(u)
		odp.Zalogowany, odp.Uzytkownik = true, &v
	}
	piszJSON(w, nethttp.StatusOK, odp)
}

// daneKreatora: POST /api/kreator.
type daneKreatora struct {
	Login  string              `json:"login"`
	Haslo  string              `json:"haslo"`
	Udzial konfiguracja.Udzial `json:"udzial"`
	Tryb   string              `json:"tryb_sieci"`
}

// sprawdzKreator: poprawność danych bez dotykania dysku (folder sprawdza magazyn przy starcie).
func sprawdzKreator(d daneKreatora) error {
	if strings.TrimSpace(d.Login) == "" {
		return nowyBlad(nethttp.StatusBadRequest, "Podaj login administratora.")
	}
	if err := rdzen.SprawdzNazwe(d.Udzial.Nazwa); err != nil {
		return nowyBlad(nethttp.StatusBadRequest, "Niedozwolona nazwa udziału.")
	}
	if !filepath.IsAbs(d.Udzial.Sciezka) {
		return nowyBlad(nethttp.StatusBadRequest, "Podaj pełną ścieżkę folderu udziału.")
	}
	if _, err := trybSieci(d.Tryb, konfiguracja.TrybVPN); err != nil {
		return err
	}
	return nil
}

// trybSieci: pusty = domyślny, inny niż vpn/publiczny = błąd.
func trybSieci(t string, domyslny konfiguracja.TrybSieci) (konfiguracja.TrybSieci, error) {
	switch konfiguracja.TrybSieci(t) {
	case "":
		return domyslny, nil
	case konfiguracja.TrybVPN, konfiguracja.TrybPubliczny:
		return konfiguracja.TrybSieci(t), nil
	}
	return "", nowyBlad(nethttp.StatusBadRequest, "Nieznany tryb sieci.")
}

// kreator: tylko gdy brak kont; zapisuje udział i tryb, zakłada admina i loguje.
func (s *serwer) kreator(w nethttp.ResponseWriter, r *nethttp.Request) {
	var d daneKreatora
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	s.kreatorMutex.Lock()
	defer s.kreatorMutex.Unlock()
	if !s.konta.CzyPusto() {
		s.blad(w, r, errKreatorZrobiony)
		return
	}
	if err := sprawdzKreator(d); err != nil {
		s.blad(w, r, err)
		return
	}
	k := s.konfiguracja()
	k.TrybSieci, _ = trybSieci(d.Tryb, k.TrybSieci) // celowo: tryb sprawdzony w sprawdzKreator
	k.Udzialy = []konfiguracja.Udzial{d.Udzial}
	if err := s.zapiszKonfig(k); err != nil {
		s.blad(w, r, err)
		return
	}
	u := rdzen.Uzytkownik{
		Login: d.Login, Admin: true, FTPS: true,
		Prawa: map[string]rdzen.Prawo{d.Udzial.Nazwa: rdzen.PrawoZapis},
	}
	if err := s.konta.Dodaj(u, d.Haslo); err != nil {
		s.blad(w, r, err)
		return
	}
	wpis := s.wpisDziennika(r, dziennik.RodzajAdministracja, d.Login)
	wpis.Wynik, wpis.Opis = dziennik.WynikOK, "Kreator: utworzono konto administratora i udział "+d.Udzial.Nazwa+"."
	s.dziennik.Zapisz(wpis)
	s.zalogujIOdpowiedz(w, r, u)
}

// zalogujIOdpowiedz tworzy sesję, ustawia ciasteczko i zwraca użytkownika.
func (s *serwer) zalogujIOdpowiedz(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	token, err := s.sesje.UtworzZOpisem(u.Login, adresIP(r), r.UserAgent())
	if err != nil {
		s.blad(w, r, err)
		return
	}
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: konfiguracja.NazwaCiasteczkaSesji, Value: token, Path: "/",
		HttpOnly: true, SameSite: nethttp.SameSiteStrictMode, Secure: r.TLS != nil,
	})
	piszJSON(w, nethttp.StatusOK, widok(u))
}

// logowanie: POST /api/logowanie; zła para = 401, blokada = 429; każda próba w dzienniku.
func (s *serwer) logowanie(w nethttp.ResponseWriter, r *nethttp.Request) {
	var d struct {
		Login string `json:"login"`
		Haslo string `json:"haslo"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	u, err := s.konta.Zaloguj(d.Login, d.Haslo, adresIP(r))
	if err != nil {
		wpis := s.wpisDziennika(r, dziennik.RodzajLogowanieBlad,
			przytnij(d.Login, konfiguracja.MaksDlugoscKtoWDzienniku))
		wpis.Wynik, wpis.Opis = dziennik.WynikBlad, opisBleduLogowania(err)
		s.dziennik.Zapisz(wpis)
		s.bladLogowania(w, r, err)
		return
	}
	wpis := s.wpisDziennika(r, dziennik.RodzajLogowanie, u.Login)
	wpis.Wynik = dziennik.WynikOK
	s.dziennik.Zapisz(wpis)
	s.zalogujIOdpowiedz(w, r, u)
}

func opisBleduLogowania(err error) string {
	if errors.Is(err, rdzen.ErrKontoZablokowane) {
		return "Konto albo adres chwilowo zablokowane."
	}
	return "Błędny login lub hasło."
}

// wylogowanie kończy sesję i czyści ciasteczko.
func (s *serwer) wylogowanie(w nethttp.ResponseWriter, r *nethttp.Request) {
	if t := tokenZ(r); t != "" {
		s.sesje.Zakoncz(t)
	}
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: konfiguracja.NazwaCiasteczkaSesji, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: nethttp.SameSiteStrictMode, Secure: r.TLS != nil,
	})
	pusto(w)
}

// zdrowie: GET /api/zdrowie (bez logowania; do monitoringu i testu dostępności).
func zdrowie(w nethttp.ResponseWriter, _ *nethttp.Request) {
	piszJSON(w, nethttp.StatusOK, map[string]any{"ok": true, "wersja": wersja()})
}

// wersja: wersja wydania (ldflags), a przy budowie lokalnej stała Wersja.
func wersja() string {
	if konfiguracja.WersjaBudowy != "" {
		return konfiguracja.WersjaBudowy
	}
	return konfiguracja.Wersja
}
