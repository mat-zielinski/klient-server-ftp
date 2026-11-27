package http

import (
	nethttp "net/http"
	"net/url"
	"strconv"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

func (s *serwer) trasyDziennika(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/dziennik", s.zalogowany(s.wpisyDziennika))
	m.HandleFunc("GET /api/dziennik/statystyki", s.tylkoAdmin(s.statystykiDziennika))
	m.HandleFunc("GET /api/dziennik/eksport", s.zalogowany(s.eksportDziennika))
}

// czas z parametru (RFC 3339); pusty = zero.
func czasZ(q url.Values, klucz string) (time.Time, error) {
	t := q.Get(klucz)
	if t == "" {
		return time.Time{}, nil
	}
	c, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return time.Time{}, errZlaData
	}
	return c, nil
}

// zakres: od/do z parametrów; brak „do” = teraz, brak „od” = do - DomyslnyZakresDziennika.
func (s *serwer) zakres(q url.Values) (time.Time, time.Time, error) {
	od, err := czasZ(q, "od")
	if err != nil {
		return od, od, err
	}
	do, err := czasZ(q, "do")
	if err != nil {
		return od, do, err
	}
	if do.IsZero() {
		do = s.teraz()
	}
	if od.IsZero() {
		od = do.Add(-konfiguracja.DomyslnyZakresDziennika)
	}
	return od, do, nil
}

// filtrDziennika: parametry zapytania; zwykły użytkownik widzi tylko własne pobrania (wymuszone).
func (s *serwer) filtrDziennika(q url.Values, u rdzen.Uzytkownik, domyslnyLimit int) (dziennik.Filtr, error) {
	od, err := czasZ(q, "od")
	if err != nil {
		return dziennik.Filtr{}, err
	}
	do, err := czasZ(q, "do")
	if err != nil {
		return dziennik.Filtr{}, err
	}
	f := dziennik.Filtr{Od: od, Do: do, Rodzaj: q.Get("rodzaj"), Kto: q.Get("kto"), Szukaj: q.Get("szukaj"),
		Limit: domyslnyLimit}
	if t := q.Get("limit"); t != "" {
		n, err := strconv.Atoi(t)
		if err != nil || n <= 0 {
			return f, errZlaLiczba
		}
		f.Limit = min(n, max(domyslnyLimit, konfiguracja.MaksLimitDziennika))
	}
	if !czyAdmin(u) {
		f.Kto, f.Rodzaj = u.Login, dziennik.RodzajPobranie
	}
	return f, nil
}

func (s *serwer) wpisyDziennika(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	f, err := s.filtrDziennika(r.URL.Query(), u, konfiguracja.DomyslnyLimitDziennika)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	wpisy, err := s.dziennik.Czytaj(f)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	if wpisy == nil {
		wpisy = []dziennik.Wpis{}
	}
	piszJSON(w, nethttp.StatusOK, wpisy)
}

// krokWykresu: godzina | dzien (domyślnie dzień).
func krokWykresu(t string) (time.Duration, error) {
	switch t {
	case konfiguracja.KrokStatystykGodzina:
		return konfiguracja.KrokGodzina, nil
	case konfiguracja.KrokStatystykDzien, "":
		return konfiguracja.KrokDzien, nil
	}
	return 0, nowyBlad(nethttp.StatusBadRequest, "Krok wykresu: godzina albo dzien.")
}

// statystykiDziennika: tylko admin (Statystyki nie mają filtra użytkownika).
func (s *serwer) statystykiDziennika(w nethttp.ResponseWriter, r *nethttp.Request, _ rdzen.Uzytkownik) {
	q := r.URL.Query()
	od, do, err := s.zakres(q)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	krok, err := krokWykresu(q.Get("krok"))
	if err != nil {
		s.blad(w, r, err)
		return
	}
	st, err := s.dziennik.Statystyki(od, do, krok)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	piszJSON(w, nethttp.StatusOK, st)
}

// eksportDziennika: CSV dziennik_<od>_<do>.csv (nie-admin: własne pobrania).
func (s *serwer) eksportDziennika(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	q := r.URL.Query()
	od, do, err := s.zakres(q)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	f, err := s.filtrDziennika(q, u, konfiguracja.LimitEksportuDziennika)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	f.Od, f.Do = od, do
	nazwa := konfiguracja.PrefiksEksportuDziennika + od.Format(konfiguracja.FormatDatyEksportu) + "_" +
		do.Format(konfiguracja.FormatDatyEksportu) + ".csv"
	h := w.Header()
	h.Set("Content-Type", "text/csv; charset=utf-8")
	h.Set("Content-Disposition", naglowekNazwy(true, nazwa))
	h.Set("Cache-Control", "no-store")
	if err := s.dziennik.EksportCSV(w, f); err != nil {
		// Nagłówki mogły już pójść: tylko wpis w dzienniku.
		wpis := s.wpisDziennika(r, dziennik.RodzajSerwer, u.Login)
		wpis.Wynik, wpis.Opis = dziennik.WynikBlad, "Eksport CSV dziennika: "+err.Error()
		s.dziennik.Zapisz(wpis)
	}
}

// mapaDziennika: GET /api/dziennik/mapa?od&do → bajty pobrań wg dnia tygodnia (pon=0) i godziny, czas lokalny.
// Nie-admin: tylko własne pobrania.
func (s *serwer) mapaDziennika(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	od, do, err := s.zakres(r.URL.Query())
	if err != nil {
		s.blad(w, r, err)
		return
	}
	f := dziennik.Filtr{Od: od, Do: do, Rodzaj: dziennik.RodzajPobranie}
	if !czyAdmin(u) {
		f.Kto = u.Login
	}
	wpisy, err := s.dziennik.Czytaj(f)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{"komorki": policzMape(wpisy)})
}

// policzMape: komórki [dzień][godzina] z bajtami; poniedziałek = 0.
func policzMape(wpisy []dziennik.Wpis) [][]int64 {
	k := make([][]int64, konfiguracja.MapaDni)
	for i := range k {
		k[i] = make([]int64, konfiguracja.MapaGodzin)
	}
	for _, w := range wpisy {
		c := w.Czas.Local()
		dzien := (int(c.Weekday()) + konfiguracja.MapaDni - 1) % konfiguracja.MapaDni
		k[dzien][c.Hour()] += w.Bajty
	}
	return k
}
