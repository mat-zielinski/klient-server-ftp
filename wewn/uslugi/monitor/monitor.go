package monitor

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Monitor zbiera próbki co konfiguracja.MonitorOkresProbkowania i trzyma historię. Bezpieczny współbieżnie.
type Monitor struct {
	zegar    func() time.Time
	zrodlo   Zrodlo
	liczniki func() (polaczenia, transfery int)
	tykacz   func() (<-chan time.Time, func())
	start    time.Time

	mu          sync.Mutex
	uruchomiony bool
	poprzedni   Odczyt
	czasPoprz   time.Time
	maPoprz     bool
	teraz       Probka
	ostatniBlad error
	historia1s  *pierscien
	historia1m  *pierscien
	minuta      *sumator
	sluchacze   map[*sluchacz]struct{}
}

// Opcja zmienia ustawienia monitora przy tworzeniu.
type Opcja func(*Monitor)

// ZZegarem podaje zegar (testy).
func ZZegarem(z func() time.Time) Opcja { return func(m *Monitor) { m.zegar = z } }

// ZZrodlem podaje źródło odczytów systemowych (testy: atrapa).
func ZZrodlem(z Zrodlo) Opcja { return func(m *Monitor) { m.zrodlo = z } }

// ZLicznikami podaje funkcję zwracającą liczbę połączeń i aktywnych transferów (wołana co próbkę).
func ZLicznikami(f func() (polaczenia, transfery int)) Opcja {
	return func(m *Monitor) { m.liczniki = f }
}

// ZTykaczem podaje kanał taktów próbkowania zamiast time.Ticker (testy).
func ZTykaczem(c <-chan time.Time) Opcja {
	return func(m *Monitor) { m.tykacz = func() (<-chan time.Time, func()) { return c, func() {} } }
}

// Nowy tworzy monitor; domyślnie zegar time.Now, źródło systemowe, liczniki zerowe.
func Nowy(opcje ...Opcja) *Monitor {
	m := &Monitor{
		zegar:      time.Now,
		zrodlo:     NoweZrodloSystemowe(),
		liczniki:   func() (int, int) { return 0, 0 },
		historia1s: nowyPierscien(konfiguracja.MonitorProbkiHistorii1s),
		historia1m: nowyPierscien(konfiguracja.MonitorProbkiHistorii1m),
		minuta:     nowySumator(konfiguracja.MonitorOkresHistorii1m),
		sluchacze:  map[*sluchacz]struct{}{},
	}
	m.tykacz = func() (<-chan time.Time, func()) {
		t := time.NewTicker(konfiguracja.MonitorOkresProbkowania)
		return t.C, t.Stop
	}
	for _, o := range opcje {
		o(m)
	}
	m.start = m.zegar()
	m.teraz = Probka{Czas: m.start, Siec: []SiecInterfejsu{}}
	return m
}

// Uruchom zbiera pierwszą próbkę od razu i startuje gorutynę próbkującą (do końca ctx). Nie blokuje;
// kolejne wywołania nic nie robią.
func (m *Monitor) Uruchom(ctx context.Context) {
	m.mu.Lock()
	if m.uruchomiony {
		m.mu.Unlock()
		return
	}
	m.uruchomiony = true
	m.mu.Unlock()
	takty, zatrzymaj := m.tykacz()
	m.probkuj()
	go func() {
		defer zatrzymaj()
		for {
			select {
			case <-ctx.Done():
				return
			case <-takty:
				m.probkuj()
			}
		}
	}()
}

// probkuj: odczyt systemu, liczenie prędkości, zapis do historii i rozesłanie słuchaczom.
func (m *Monitor) probkuj() {
	odczyt, blad := m.zrodlo.Odczytaj()
	polaczenia, transfery := m.liczniki()
	var pamiec runtime.MemStats
	runtime.ReadMemStats(&pamiec)
	teraz := m.zegar()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.ostatniBlad = blad
	p := Probka{
		Czas:          teraz,
		Siec:          []SiecInterfejsu{},
		Polaczenia:    polaczenia,
		Transfery:     transfery,
		Gorutyny:      runtime.NumGoroutine(),
		PamiecProcesu: pamiec.Sys,
	}
	if blad == nil {
		m.uzupelnijZOdczytu(&p, odczyt, teraz)
	}
	m.teraz = p
	m.historia1s.dodaj(p)
	if sr, gotowa := m.minuta.dodaj(p); gotowa {
		m.historia1m.dodaj(sr)
	}
	m.rozeslij(p)
}

// uzupelnijZOdczytu liczy pola systemowe z różnicy względem poprzedniego odczytu (wołane pod mu).
func (m *Monitor) uzupelnijZOdczytu(p *Probka, o Odczyt, teraz time.Time) {
	p.RAMUzyte, p.RAMRazem = o.RAMUzyte, o.RAMRazem
	var poprz Odczyt
	dt := 0.0
	if m.maPoprz {
		poprz = m.poprzedni
		dt = teraz.Sub(m.czasPoprz).Seconds()
		p.CPUProc = procentCPU(poprz, o)
		if o.DyskDostepny && poprz.DyskDostepny {
			p.DyskOdczytBps = predkosc(poprz.DyskOdczyt, o.DyskOdczyt, dt)
			p.DyskZapisBps = predkosc(poprz.DyskZapis, o.DyskZapis, dt)
		}
	}
	p.Siec, p.WysylanieBps, p.OdbiorBps = predkosciSieci(poprz.Siec, o.Siec, dt)
	m.poprzedni, m.czasPoprz, m.maPoprz = o, teraz, true
}

// Teraz zwraca ostatnią próbkę (przed pierwszą: same zera z czasem utworzenia).
func (m *Monitor) Teraz() Probka {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.teraz
}

// OstatniBlad: błąd ostatniego odczytu systemu (nil = w porządku). Próbka i tak powstaje, bez pól systemowych.
func (m *Monitor) OstatniBlad() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ostatniBlad
}

// Historia1s: próbki co sekundę z ostatnich 10 min, od najstarszej. Nie modyfikuj pól Siec.
func (m *Monitor) Historia1s() []Probka {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.historia1s.lista()
}

// Historia1m: średnie z pełnych minut z ostatnich 24 h, od najstarszej (bieżąca minuta jeszcze niepełna).
func (m *Monitor) Historia1m() []Probka {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.historia1m.lista()
}

// System zwraca dane maszyny; RAM i dostępność dysku z ostatniego udanego odczytu.
func (m *Monitor) System() System {
	host, err := os.Hostname()
	if err != nil {
		host = "" // celowo: brak nazwy hosta nie psuje reszty danych
	}
	teraz := m.zegar()
	m.mu.Lock()
	defer m.mu.Unlock()
	return System{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		CPURdzenie:     runtime.NumCPU(),
		RAMRazem:       m.poprzedni.RAMRazem,
		Host:           host,
		CzasPracyS:     int64(teraz.Sub(m.start) / time.Second),
		DyskIODostepny: m.poprzedni.DyskDostepny,
	}
}
