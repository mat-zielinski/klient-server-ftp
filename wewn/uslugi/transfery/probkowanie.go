package transfery

import (
	"time"

	"skrytka/wewn/konfiguracja"
)

// probka: licznik bajtów w chwili próbkowania.
type probka struct {
	czas  time.Time
	bajty int64
}

// petlaProbkowania: jedyna gorutyna licząca prędkość, historię i zdarzenia „postep”.
func (r *Rejestr) petlaProbkowania() {
	for {
		select {
		case <-r.stop:
			return
		case <-r.tykacz:
			r.probkuj(r.teraz())
			if r.poProbce != nil {
				r.poProbce()
			}
		}
	}
}

// probkuj robi jedną próbkę wszystkich aktywnych transferów w chwili teraz.
func (r *Rejestr) probkuj(teraz time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.aktywne {
		t.probkuj(teraz)
		if teraz.Sub(t.ostatniPostep) >= konfiguracja.TransferyOdstepPostepu {
			t.ostatniPostep = teraz
			r.rozeslij(Zdarzenie{Typ: ZdarzeniePostep, Stan: t.migawka()})
		}
	}
}

// probkuj (transfer, pod r.mu): dopisuje próbkę, liczy prędkość z okna, historię i pozostały czas.
func (t *Transfer) probkuj(teraz time.Time) {
	bajty := t.przeslano.Load()
	poprzednia := t.probki[len(t.probki)-1]
	t.probki = append(t.probki, probka{czas: teraz, bajty: bajty})
	if nadmiar := len(t.probki) - (konfiguracja.TransferyOknoPredkosciS + 1); nadmiar > 0 {
		t.probki = append(t.probki[:0], t.probki[nadmiar:]...)
	}

	t.stan.Historia = append(t.stan.Historia, tempo(poprzednia, probka{czas: teraz, bajty: bajty}))
	if nadmiar := len(t.stan.Historia) - konfiguracja.TransferyProbkiHistorii; nadmiar > 0 {
		t.stan.Historia = append(t.stan.Historia[:0], t.stan.Historia[nadmiar:]...)
	}

	t.stan.PredkoscBps = tempo(t.probki[0], t.probki[len(t.probki)-1])
	t.stan.PozostaloS = pozostalo(t.stan.Rozmiar, bajty, t.stan.PredkoscBps)
}

// tempo: B/s między dwiema próbkami (0, gdy czas nie płynie).
func tempo(od, do probka) int64 {
	dt := do.czas.Sub(od.czas)
	if dt <= 0 || do.bajty <= od.bajty {
		return 0
	}
	return int64(float64(do.bajty-od.bajty) / dt.Seconds())
}

// pozostalo: sekundy do końca w górę; -1 = nieznane (brak rozmiaru albo prędkości).
func pozostalo(rozmiar, przeslano, predkosc int64) int64 {
	if rozmiar <= 0 {
		return -1
	}
	if przeslano >= rozmiar {
		return 0
	}
	if predkosc <= 0 {
		return -1
	}
	return (rozmiar - przeslano + predkosc - 1) / predkosc
}
