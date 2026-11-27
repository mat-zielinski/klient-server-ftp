package http

import (
	"io"
	nethttp "net/http"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/monitor"
)

// trasyFali2: monitor, indeks, miniatury, sumy, urządzenia i 2FA, mapa dziennika.
func (s *serwer) trasyFali2(m *nethttp.ServeMux) {
	m.HandleFunc("GET /api/monitor", s.tylkoAdmin(s.stanMonitora))
	m.HandleFunc("GET /api/monitor/strumien", s.tylkoAdmin(s.strumienMonitora))
	m.HandleFunc("GET /api/indeks/szukaj", s.zalogowany(s.szukajWIndeksie))
	m.HandleFunc("GET /api/indeks/statystyki", s.tylkoAdmin(s.statystykiIndeksu))
	m.HandleFunc("GET /api/miniatura", s.zalogowany(s.miniatura))
	m.HandleFunc("GET /api/suma", s.zalogowany(s.suma))
	m.HandleFunc("GET /api/dziennik/mapa", s.zalogowany(s.mapaDziennika))
	m.HandleFunc("GET /api/konta/urzadzenia", s.zalogowany(s.urzadzenia))
	m.HandleFunc("DELETE /api/konta/urzadzenia", s.zalogowany(s.usunUrzadzenie))
	m.HandleFunc("POST /api/konta/2fa/start", s.zalogowany(s.start2FA))
	m.HandleFunc("POST /api/konta/2fa/wlacz", s.zalogowany(s.wlacz2FA))
	m.HandleFunc("POST /api/konta/2fa/wylacz", s.zalogowany(s.wylacz2FA))
}

// stanMonitora: GET /api/monitor (admin).
func (s *serwer) stanMonitora(w nethttp.ResponseWriter, r *nethttp.Request, _ rdzen.Uzytkownik) {
	if s.monitor == nil {
		s.blad(w, r, errWylaczone)
		return
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{
		"teraz":       s.monitor.Teraz(),
		"historia_1s": niePuste(s.monitor.Historia1s()),
		"historia_1m": niePuste(s.monitor.Historia1m()),
		"system":      s.monitor.System(),
	})
}

func niePuste(p []monitor.Probka) []monitor.Probka {
	if p == nil {
		return []monitor.Probka{}
	}
	return p
}

// strumienMonitora: SSE `event: probka` przy każdej próbce monitora, ping co 15 s.
func (s *serwer) strumienMonitora(w nethttp.ResponseWriter, r *nethttp.Request, _ rdzen.Uzytkownik) {
	if s.monitor == nil {
		s.blad(w, r, errWylaczone)
		return
	}
	f, ok := w.(nethttp.Flusher)
	if !ok {
		s.blad(w, r, errNieobslugiwane)
		return
	}
	probki, wypisz := s.monitor.Sluchaj()
	defer wypisz()
	ping, stopPing := s.tykacz(konfiguracja.OkresPingSSE)
	defer stopPing()
	rozpocznijSSE(w, f)
	for {
		var err error
		select {
		case <-r.Context().Done():
			return
		case p, otwarty := <-probki:
			if !otwarty {
				return
			}
			err = piszZdarzenieSSE(w, "probka", p)
		case <-ping:
			_, err = io.WriteString(w, ": ping\n\n")
		}
		if err != nil {
			return
		}
		f.Flush()
	}
}
