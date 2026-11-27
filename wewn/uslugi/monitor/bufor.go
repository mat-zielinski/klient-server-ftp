package monitor

import (
	"math"
	"sort"
	"time"
)

// pierscien: bufor kołowy o stałej pojemności; najstarszy element wypada przy dopisaniu do pełnego.
type pierscien struct {
	dane  []Probka
	start int
	n     int
}

func nowyPierscien(pojemnosc int) *pierscien {
	if pojemnosc < 1 {
		pojemnosc = 1
	}
	return &pierscien{dane: make([]Probka, pojemnosc)}
}

func (r *pierscien) dodaj(p Probka) {
	poj := len(r.dane)
	if r.n < poj {
		r.dane[(r.start+r.n)%poj] = p
		r.n++
		return
	}
	r.dane[r.start] = p
	r.start = (r.start + 1) % poj
}

// lista: kopia od najstarszej do najnowszej (nigdy nil).
func (r *pierscien) lista() []Probka {
	wynik := make([]Probka, r.n)
	for i := 0; i < r.n; i++ {
		wynik[i] = r.dane[(r.start+i)%len(r.dane)]
	}
	return wynik
}

// sumaInterfejsu: suma prędkości interfejsu w bieżącej minucie.
type sumaInterfejsu struct {
	rodzaj   string
	wys, odb float64
}

// sumator: zbiera próbki jednego okresu (minuty) i oddaje ich średnią.
type sumator struct {
	okres                                   time.Duration
	poczatek                                time.Time
	n                                       int
	cpu, ramU, ramR, dOdcz, dZap, wys, odb  float64
	polaczenia, transfery, gorutyny, pamiec float64
	siec                                    map[string]*sumaInterfejsu
}

func nowySumator(okres time.Duration) *sumator {
	return &sumator{okres: okres, siec: map[string]*sumaInterfejsu{}}
}

// dodaj dopisuje próbkę; gdy zaczyna się nowy okres, zwraca średnią poprzedniego (gotowa = true).
func (s *sumator) dodaj(p Probka) (Probka, bool) {
	poczatek := p.Czas.Truncate(s.okres)
	var srednia Probka
	gotowa := false
	if s.n > 0 && !poczatek.Equal(s.poczatek) {
		srednia, gotowa = s.srednia(), true
		s.zeruj()
	}
	if s.n == 0 {
		s.poczatek = poczatek
	}
	s.n++
	s.cpu += p.CPUProc
	s.ramU += float64(p.RAMUzyte)
	s.ramR += float64(p.RAMRazem)
	s.dOdcz += float64(p.DyskOdczytBps)
	s.dZap += float64(p.DyskZapisBps)
	s.wys += float64(p.WysylanieBps)
	s.odb += float64(p.OdbiorBps)
	s.polaczenia += float64(p.Polaczenia)
	s.transfery += float64(p.Transfery)
	s.gorutyny += float64(p.Gorutyny)
	s.pamiec += float64(p.PamiecProcesu)
	for _, si := range p.Siec {
		w := s.siec[si.Interfejs]
		if w == nil {
			w = &sumaInterfejsu{rodzaj: si.Rodzaj}
			s.siec[si.Interfejs] = w
		}
		w.wys += float64(si.WysylanieBps)
		w.odb += float64(si.OdbiorBps)
	}
	return srednia, gotowa
}

func (s *sumator) zeruj() {
	*s = sumator{okres: s.okres, siec: map[string]*sumaInterfejsu{}}
}

// srednia: średnia z próbek okresu; Czas = początek okresu. Interfejs nieobecny w części próbek liczy się
// w nich jako 0.
func (s *sumator) srednia() Probka {
	n := float64(s.n)
	sr := func(v float64) float64 { return v / n }
	p := Probka{
		Czas:          s.poczatek,
		CPUProc:       math.Round(sr(s.cpu)*10) / 10,
		RAMUzyte:      uint64(math.Round(sr(s.ramU))),
		RAMRazem:      uint64(math.Round(sr(s.ramR))),
		DyskOdczytBps: int64(math.Round(sr(s.dOdcz))),
		DyskZapisBps:  int64(math.Round(sr(s.dZap))),
		WysylanieBps:  int64(math.Round(sr(s.wys))),
		OdbiorBps:     int64(math.Round(sr(s.odb))),
		Polaczenia:    int(math.Round(sr(s.polaczenia))),
		Transfery:     int(math.Round(sr(s.transfery))),
		Gorutyny:      int(math.Round(sr(s.gorutyny))),
		PamiecProcesu: uint64(math.Round(sr(s.pamiec))),
		Siec:          make([]SiecInterfejsu, 0, len(s.siec)),
	}
	for nazwa, w := range s.siec {
		p.Siec = append(p.Siec, SiecInterfejsu{
			Interfejs:    nazwa,
			Rodzaj:       w.rodzaj,
			WysylanieBps: int64(math.Round(sr(w.wys))),
			OdbiorBps:    int64(math.Round(sr(w.odb))),
		})
	}
	sort.Slice(p.Siec, func(i, j int) bool { return p.Siec[i].Interfejs < p.Siec[j].Interfejs })
	return p
}
