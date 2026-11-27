package http

import (
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

func (s *serwer) trasyTransferow(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/transfery", s.zalogowany(s.transferyAktywne))
	m.HandleFunc("GET /api/transfery/strumien", s.zalogowany(s.strumienTransferow))
	m.HandleFunc("DELETE /api/transfery", s.tylkoAdmin(s.przerwijTransfer))
}

// ktoWidzi: admin wszystkie (""), inni tylko własne.
func ktoWidzi(u rdzen.Uzytkownik) string {
	if czyAdmin(u) {
		return ""
	}
	return u.Login
}

func (s *serwer) transferyAktywne(w nethttp.ResponseWriter, _ *nethttp.Request, u rdzen.Uzytkownik) {
	stany := s.transfery.Aktywne(ktoWidzi(u))
	if stany == nil {
		stany = []transfery.Stan{}
	}
	piszJSON(w, nethttp.StatusOK, stany)
}

// suma: zdarzenie „suma” co OkresSumySSE.
type suma struct {
	PredkoscBps int64 `json:"predkosc_bps"`
	Aktywne     int   `json:"aktywne"`
}

func (s *serwer) sumaDla(kto string) suma {
	aktywne := s.transfery.Aktywne(kto)
	if kto == "" {
		return suma{PredkoscBps: s.transfery.PredkoscLaczna(), Aktywne: len(aktywne)}
	}
	var p int64
	for _, a := range aktywne {
		p += a.PredkoscBps
	}
	return suma{PredkoscBps: p, Aktywne: len(aktywne)}
}

// piszZdarzenieSSE: "event: <typ>\ndata: <json>\n\n".
func piszZdarzenieSSE(w io.Writer, typ string, dane any) error {
	j, err := json.Marshal(dane)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", typ, j)
	return err
}

// strumienTransferow: SSE ze zdarzeniami start/postep/koniec, „suma” co sekundę i ping co 15 s.
func (s *serwer) strumienTransferow(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	f, ok := w.(nethttp.Flusher)
	if !ok {
		s.blad(w, r, errNieobslugiwane)
		return
	}
	// Subskrypcja przed nagłówkami: gdy klient dostanie odpowiedź, żadne zdarzenie już nie przepadnie.
	kto := ktoWidzi(u)
	zdarzenia, wypisz := s.transfery.Sluchaj(kto)
	defer wypisz()
	ping, stopPing := s.tykacz(konfiguracja.OkresPingSSE)
	defer stopPing()
	sumy, stopSumy := s.tykacz(konfiguracja.OkresSumySSE)
	defer stopSumy()
	rozpocznijSSE(w, f)
	for {
		var err error
		select {
		case <-r.Context().Done():
			return
		case z, otwarty := <-zdarzenia:
			if !otwarty {
				return
			}
			err = piszZdarzenieSSE(w, z.Typ, z)
		case <-sumy:
			err = piszZdarzenieSSE(w, "suma", s.sumaDla(kto))
		case <-ping:
			_, err = io.WriteString(w, ": ping\n\n")
		}
		if err != nil {
			return
		}
		f.Flush()
	}
}

// przerwijTransfer: DELETE /api/transfery?id (admin).
func (s *serwer) przerwijTransfer(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	id := r.URL.Query().Get("id")
	if err := s.transfery.Przerwij(id); err != nil {
		s.blad(w, r, err)
		return
	}
	wpis := s.wpisDziennika(r, dziennik.RodzajAdministracja, u.Login)
	wpis.Wynik, wpis.Opis = dziennik.WynikOK, "Przerwano transfer "+przytnij(id, konfiguracja.MaksDlugoscKtoWDzienniku)+"."
	s.dziennik.Zapisz(wpis)
	pusto(w)
}

// rozpocznijSSE wysyła nagłówki strumienia zdarzeń (no-cache, bez buforowania pośredników).
func rozpocznijSSE(w nethttp.ResponseWriter, f nethttp.Flusher) {
	// Strumień trwa długo: zdejmujemy limit zapisu serwera, jeśli ustawiony.
	_ = nethttp.NewResponseController(w).SetWriteDeadline(time.Time{}) // celowo: brak wsparcia = zostaje limit serwera
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(nethttp.StatusOK)
	f.Flush()
}
