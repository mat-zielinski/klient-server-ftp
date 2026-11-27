package http

import (
	"errors"
	nethttp "net/http"
	"strconv"
	"time"

	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

// uslugaKosza: kosz aplikacji w magazynie (sprawdzane przez asercję typu, żeby nie zmieniać uslugaMagazynu).
type uslugaKosza interface {
	ListaKosza(udzial string) ([]magazyn.WpisKosza, error)
	Przywroc(udzial, id string, nadpisz bool) error
	UsunTrwale(udzial, id string) error
	Oproznij(udzial string, starszeNiz time.Duration) (int, error)
}

var errZlyIDKosza = nowyBlad(nethttp.StatusBadRequest, "Nieprawidłowy identyfikator wpisu kosza.")

func (s *serwer) trasyKosza(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/kosz", s.zalogowany(s.listaKosza))
	m.HandleFunc("POST /api/kosz/przywroc", s.zalogowany(s.przywrocZKosza))
	m.HandleFunc("DELETE /api/kosz", s.zalogowany(s.usunZKosza))
	m.HandleFunc("DELETE /api/kosz/wszystko", s.tylkoAdmin(s.oproznijKosz))
}

// widokWpisuKosza: GET /api/kosz.
type widokWpisuKosza struct {
	ID       string    `json:"id"`
	Nazwa    string    `json:"nazwa"`
	Sciezka  string    `json:"sciezka"`
	Usunieto time.Time `json:"usunieto"`
	Rozmiar  int64     `json:"rozmiar"`
	Folder   bool      `json:"folder"`
	Rodzaj   string    `json:"rodzaj"`
}

// kosz: bramka (prawo zapisu w udziale) i usługa kosza; brak usługi = 503.
func (s *serwer) kosz(r *nethttp.Request, u rdzen.Uzytkownik) (uslugaKosza, string, error) {
	udzial := r.URL.Query().Get("udzial")
	if _, err := s.dostepDoUdzialu(u, rdzen.OpUsuniecie, udzial); err != nil {
		return nil, "", err
	}
	k, ok := s.magazyn.(uslugaKosza)
	if !ok {
		return nil, "", errWylaczone
	}
	return k, udzial, nil
}

// bladKosza: zły identyfikator = 400, reszta jak zwykle.
func (s *serwer) bladKosza(w nethttp.ResponseWriter, r *nethttp.Request, err error) {
	if errors.Is(err, magazyn.ErrZlyIDKosza) {
		err = errZlyIDKosza
	}
	s.blad(w, r, err)
}

func (s *serwer) listaKosza(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	k, udzial, err := s.kosz(r, u)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	wpisy, err := k.ListaKosza(udzial)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	wynik := make([]widokWpisuKosza, 0, len(wpisy))
	for _, e := range wpisy {
		wynik = append(wynik, widokWpisuKosza(e))
	}
	piszJSON(w, nethttp.StatusOK, wynik)
}

func (s *serwer) przywrocZKosza(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d struct {
		ID      string `json:"id"`
		Nadpisz bool   `json:"nadpisz"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	k, udzial, err := s.kosz(r, u)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	sciezka, err := magazyn.SciezkaWpisuKosza(d.ID)
	if err == nil {
		err = k.Przywroc(udzial, d.ID, d.Nadpisz)
	}
	if err != nil {
		s.bladKosza(w, r, err)
		return
	}
	opis := "Przywrócono z kosza."
	if d.Nadpisz {
		opis = "Przywrócono z kosza z nadpisaniem (poprzedni element trafił do kosza)."
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, sciezka, opis)
	pusto(w)
}

func (s *serwer) usunZKosza(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	k, udzial, err := s.kosz(r, u)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	id := r.URL.Query().Get("id")
	sciezka, err := magazyn.SciezkaWpisuKosza(id)
	if err == nil {
		err = k.UsunTrwale(udzial, id)
	}
	if err != nil {
		s.bladKosza(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, sciezka, "Usunięto trwale z kosza.")
	pusto(w)
}

func (s *serwer) oproznijKosz(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	k, udzial, err := s.kosz(r, u)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	n, err := k.Oproznij(udzial, 0)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	s.zapiszZmiane(r, u.Login, dziennik.RodzajZmiana, udzial, "/", "Opróżniono kosz ("+strconv.Itoa(n)+" el.).")
	piszJSON(w, nethttp.StatusOK, map[string]int{"usuniete": n})
}
