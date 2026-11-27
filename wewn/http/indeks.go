package http

import (
	nethttp "net/http"
	"strconv"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/indeks"
)

// limitSzukania: ?limit (domyślnie i najwyżej LimitWynikowSzukania).
func limitSzukania(t string) (int, error) {
	if t == "" {
		return konfiguracja.LimitWynikowSzukania, nil
	}
	n, err := strconv.Atoi(t)
	if err != nil || n <= 0 {
		return 0, errZlaLiczba
	}
	return min(n, konfiguracja.LimitWynikowSzukania), nil
}

// szukajWIndeksie: GET /api/indeks/szukaj; gotowy indeks = szybko, inaczej przejście magazynu.
func (s *serwer) szukajWIndeksie(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
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
	limit, err := limitSzukania(r.URL.Query().Get("limit"))
	if err != nil {
		s.blad(w, r, err)
		return
	}
	var wpisy []rdzen.Wpis
	if s.indeks != nil && s.indeks.Gotowy(udzial) {
		wpisy = s.indeks.Szukaj(udzial, sciezka, q, limit)
	} else if wpisy, err = s.magazyn.Szukaj(udzial, sciezka, q, limit); err != nil {
		s.blad(w, r, err)
		return
	}
	if wpisy == nil {
		wpisy = []rdzen.Wpis{}
	}
	piszJSON(w, nethttp.StatusOK, wpisy)
}

// statystykiIndeksu: GET /api/indeks/statystyki (admin).
func (s *serwer) statystykiIndeksu(w nethttp.ResponseWriter, r *nethttp.Request, _ rdzen.Uzytkownik) {
	if s.indeks == nil {
		s.blad(w, r, errWylaczone)
		return
	}
	st := s.indeks.Statystyki()
	if st == nil {
		st = []indeks.StatystykaUdzialu{}
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{"udzialy": st})
}

// miniatura: GET /api/miniatura?udzial&sciezka&rozmiar=s|m; JPEG z pamięci podręcznej. To podgląd, nie
// pobranie: bez transferu i bez wpisu dziennika.
func (s *serwer) miniatura(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	if s.miniatury == nil {
		s.blad(w, r, errWylaczone)
		return
	}
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpOdczyt)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	plik, err := s.miniatury.Miniatura(r.Context(), udzial, sciezka, r.URL.Query().Get("rozmiar"))
	if err != nil {
		s.blad(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", konfiguracja.CacheMiniatur)
	nethttp.ServeFile(w, r, plik)
}

// suma: GET /api/suma?udzial&sciezka → {"sha256"} (zawartość pliku, więc prawo pobierania).
func (s *serwer) suma(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	if s.sumy == nil {
		s.blad(w, r, errWylaczone)
		return
	}
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpPobranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	h, err := s.sumy.SHA256(r.Context(), udzial, sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]string{"sha256": h})
}
