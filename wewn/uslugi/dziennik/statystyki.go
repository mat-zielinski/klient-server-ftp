package dziennik

import (
	"errors"
	"path"
	"sort"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Błędy argumentów statystyk.
var (
	ErrZlyZakres     = errors.New("Nieprawidłowy zakres czasu statystyk.")
	ErrZlyKrok       = errors.New("Krok wykresu musi być dodatni.")
	ErrZaDuzoPunktow = errors.New("Za drobny krok wykresu dla tego zakresu.")
)

// Statystyki pobrań w zakresie, przedział wykresu: krok (np. godzina, dzień).
// Liczone tylko z wpisów RodzajPobranie (każdy wynik: bajty przerwanych też obciążyły łącze).
// Zakres: od włącznie, do wyłącznie; do puste = teraz.
func (d *Dziennik) Statystyki(od, do time.Time, krok time.Duration) (Statystyki, error) {
	if do.IsZero() {
		do = d.teraz()
	}
	if od.IsZero() || !od.Before(do) {
		return Statystyki{}, ErrZlyZakres
	}
	if krok <= 0 {
		return Statystyki{}, ErrZlyKrok
	}
	n := (do.Sub(od) + krok - 1) / krok
	if n > konfiguracja.DziennikMaksPunktowWykresu {
		return Statystyki{}, ErrZaDuzoPunktow
	}
	l := nowyLicznik(od, krok, int(n))
	err := d.przegladaj(od, do, func(w Wpis) bool {
		if w.Rodzaj == RodzajPobranie {
			l.dodaj(w)
		}
		return true
	})
	if err != nil {
		return Statystyki{}, err
	}
	return l.wynik(), nil
}

// licznik zbiera statystyki w jednym przebiegu.
type licznik struct {
	od      time.Time
	krok    time.Duration
	wykres  []PunktWykresu
	czasMs  []int64 // suma czasu pobrań w przedziale (do średniej prędkości)
	bajtyMs []int64 // bajty pobrań, które mają czas (żeby średnia nie liczyła bajtów bez czasu)
	pliki   map[string]*PozycjaRankingu
	klienci map[string]*PozycjaRankingu
	st      Statystyki
}

func nowyLicznik(od time.Time, krok time.Duration, n int) *licznik {
	l := &licznik{
		od: od, krok: krok,
		wykres:  make([]PunktWykresu, n),
		czasMs:  make([]int64, n),
		bajtyMs: make([]int64, n),
		pliki:   map[string]*PozycjaRankingu{},
		klienci: map[string]*PozycjaRankingu{},
		st:      Statystyki{Protokoly: map[string]int64{}},
	}
	for i := range l.wykres {
		l.wykres[i].Czas = od.Add(time.Duration(i) * krok)
	}
	return l
}

func (l *licznik) dodaj(w Wpis) {
	i := int(w.Czas.Sub(l.od) / l.krok)
	if i < 0 || i >= len(l.wykres) {
		return
	}
	l.wykres[i].Bajty += w.Bajty
	l.wykres[i].Pobrania++
	if w.CzasMs > 0 {
		l.czasMs[i] += w.CzasMs
		l.bajtyMs[i] += w.Bajty
	}
	dodajDoRankingu(l.pliki, path.Join(w.Udzial, w.Sciezka), w.Bajty)
	klient := w.Kto
	if klient == "" {
		klient = w.IP
	}
	dodajDoRankingu(l.klienci, klient, w.Bajty)
	protokol := w.Protokol
	if protokol == "" {
		protokol = konfiguracja.DziennikProtokolNieznany
	}
	l.st.Protokoly[protokol] += w.Bajty
	l.st.SumaBajtow += w.Bajty
	l.st.SumaPobran++
}

func dodajDoRankingu(m map[string]*PozycjaRankingu, nazwa string, bajty int64) {
	p, ok := m[nazwa]
	if !ok {
		p = &PozycjaRankingu{Nazwa: nazwa}
		m[nazwa] = p
	}
	p.Bajty += bajty
	p.Liczba++
}

func (l *licznik) wynik() Statystyki {
	for i := range l.wykres {
		if l.czasMs[i] > 0 {
			l.wykres[i].SredniaBps = l.bajtyMs[i] * 1000 / l.czasMs[i]
		}
	}
	l.st.Wykres = l.wykres
	l.st.NajczestszePliki = top(l.pliki, poLiczbie)
	l.st.NajaktywniejsiKlienci = top(l.klienci, poBajtach)
	return l.st
}

// poLiczbie: najpierw liczba, potem bajty, potem nazwa (kolejność stała).
func poLiczbie(a, b PozycjaRankingu) bool {
	if a.Liczba != b.Liczba {
		return a.Liczba > b.Liczba
	}
	if a.Bajty != b.Bajty {
		return a.Bajty > b.Bajty
	}
	return a.Nazwa < b.Nazwa
}

// poBajtach: najpierw bajty, potem liczba, potem nazwa.
func poBajtach(a, b PozycjaRankingu) bool {
	if a.Bajty != b.Bajty {
		return a.Bajty > b.Bajty
	}
	if a.Liczba != b.Liczba {
		return a.Liczba > b.Liczba
	}
	return a.Nazwa < b.Nazwa
}

func top(m map[string]*PozycjaRankingu, mniej func(a, b PozycjaRankingu) bool) []PozycjaRankingu {
	lista := make([]PozycjaRankingu, 0, len(m))
	for _, p := range m {
		lista = append(lista, *p)
	}
	sort.Slice(lista, func(i, j int) bool { return mniej(lista[i], lista[j]) })
	if len(lista) > konfiguracja.DziennikTop {
		lista = lista[:konfiguracja.DziennikTop]
	}
	return lista
}
