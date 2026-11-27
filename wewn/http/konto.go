package http

import (
	nethttp "net/http"
	"time"

	"skrytka/wewn/rdzen"
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
