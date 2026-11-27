package indeks

import (
	"time"
)

// LicznikRodzaju: pliki i bajty jednego rodzaju.
type LicznikRodzaju struct {
	Pliki int64 `json:"pliki"`
	Bajty int64 `json:"bajty"`
}

// StatystykaUdzialu do GET /api/indeks/statystyki (docs/api.md, Fala 2).
type StatystykaUdzialu struct {
	Nazwa   string `json:"nazwa"`
	Pliki   int64  `json:"pliki"`
	Foldery int64  `json:"foldery"`
	Bajty   int64  `json:"bajty"`
	// WgRodzaju: klucz = rdzen.RodzajObraz … (tylko pliki, bez folderów).
	WgRodzaju map[string]LicznikRodzaju `json:"wg_rodzaju"`
	// Zindeksowano: koniec ostatniego przejścia; zero = udział jeszcze niezindeksowany.
	Zindeksowano time.Time `json:"zindeksowano"`
}

// policzStatystyke zlicza zbudowany zbiór.
func policzStatystyke(udzial string, z *zbior, teraz time.Time) StatystykaUdzialu {
	s := StatystykaUdzialu{Nazwa: udzial, WgRodzaju: map[string]LicznikRodzaju{}, Zindeksowano: teraz}
	for i := range z.pozycje {
		p := &z.pozycje[i]
		if p.folder {
			s.Foldery++
			continue
		}
		s.Pliki++
		s.Bajty += p.rozmiar
		r := rodzaje[p.rodzaj]
		l := s.WgRodzaju[r]
		l.Pliki++
		l.Bajty += p.rozmiar
		s.WgRodzaju[r] = l
	}
	return s
}

// Statystyki wszystkich udziałów w kolejności z konfiguracji; niezindeksowany = same zera i pusta mapa.
func (x *Indeks) Statystyki() []StatystykaUdzialu {
	var wynik []StatystykaUdzialu
	for _, u := range x.m.Udzialy() {
		s := StatystykaUdzialu{Nazwa: u.Nazwa, WgRodzaju: map[string]LicznikRodzaju{}}
		if z := x.udzialy[u.Nazwa].zbior.Load(); z != nil {
			s = z.statystyka
			s.WgRodzaju = kopiaMapy(s.WgRodzaju)
		}
		wynik = append(wynik, s)
	}
	return wynik
}

// kopiaMapy: wołający może zmieniać wynik bez wpływu na indeks.
func kopiaMapy(m map[string]LicznikRodzaju) map[string]LicznikRodzaju {
	k := make(map[string]LicznikRodzaju, len(m))
	for r, l := range m {
		k[r] = l
	}
	return k
}
