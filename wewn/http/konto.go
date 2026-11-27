package http

import (
	"errors"
	nethttp "net/http"
	"time"

	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/konta"
)

// widokUrzadzenia: sesja bieżącego konta (bez tokenu).
type widokUrzadzenia struct {
	ID           string    `json:"id"`
	IP           string    `json:"ip"`
	Przegladarka string    `json:"przegladarka"`
	Utworzono    time.Time `json:"utworzono"`
	Ostatnio     time.Time `json:"ostatnio"`
	Biezaca      bool      `json:"biezaca"`
}

// urzadzenia: GET /api/konta/urzadzenia (tylko własne sesje).
func (s *serwer) urzadzenia(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	biezaca := konta.IDSesji(tokenZ(r))
	wynik := []widokUrzadzenia{}
	for _, d := range s.sesje.Urzadzenia(u.Login) {
		wynik = append(wynik, widokUrzadzenia{ID: d.ID, IP: d.IP, Przegladarka: d.Przegladarka,
			Utworzono: d.Utworzono, Ostatnio: d.Ostatnio, Biezaca: d.ID == biezaca})
	}
	piszJSON(w, nethttp.StatusOK, wynik)
}

// usunUrzadzenie: DELETE /api/konta/urzadzenia?id (tylko własne; cudze = 404).
func (s *serwer) usunUrzadzenie(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	if err := s.sesje.ZakonczUrzadzenie(u.Login, r.URL.Query().Get("id")); err != nil {
		s.blad(w, r, err)
		return
	}
	pusto(w)
}

// zapisz2FA: wpis dziennika o zmianie drugiego składnika.
func (s *serwer) zapisz2FA(r *nethttp.Request, login, opis string) {
	wpis := s.wpisDziennika(r, dziennik.RodzajAdministracja, login)
	wpis.Wynik, wpis.Opis = dziennik.WynikOK, opis
	s.dziennik.Zapisz(wpis)
}

// start2FA: POST /api/konta/2fa/start → sekret i adres otpauth (czeka na potwierdzenie kodem).
func (s *serwer) start2FA(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	sekret, adres, err := s.konta.Start2FA(u.Login)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]string{"sekret": sekret, "adres_otpauth": adres})
}

// wlacz2FA: POST /api/konta/2fa/wlacz {"kod"}.
func (s *serwer) wlacz2FA(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d struct {
		Kod string `json:"kod"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	if err := s.konta.Wlacz2FA(u.Login, d.Kod); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapisz2FA(r, u.Login, "Włączono weryfikację dwuetapową.")
	pusto(w)
}

// wylacz2FA: POST /api/konta/2fa/wylacz {"haslo"}; złe hasło = 403 (401 wylogowałby interfejs).
func (s *serwer) wylacz2FA(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d struct {
		Haslo string `json:"haslo"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	if err := s.konta.Wylacz2FA(u.Login, d.Haslo); err != nil {
		if errors.Is(err, rdzen.ErrZleDane) {
			piszBlad(w, nethttp.StatusForbidden, "Błędne hasło.")
			return
		}
		s.blad(w, r, err)
		return
	}
	s.zapisz2FA(r, u.Login, "Wyłączono weryfikację dwuetapową.")
	pusto(w)
}
