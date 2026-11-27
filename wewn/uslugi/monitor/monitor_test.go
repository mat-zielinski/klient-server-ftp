package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

// atrapa: źródło odczytów sterowane z testu.
type atrapa struct {
	mu     sync.Mutex
	odczyt Odczyt
	blad   error
}

func (a *atrapa) Odczytaj() (Odczyt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.odczyt, a.blad
}

func (a *atrapa) ustaw(o Odczyt) {
	a.mu.Lock()
	a.odczyt = o
	a.mu.Unlock()
}

// zegarTestowy: czas przesuwany ręcznie.
type zegarTestowy struct {
	mu sync.Mutex
	t  time.Time
}

func (z *zegarTestowy) teraz() time.Time {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.t
}

func (z *zegarTestowy) przesun(d time.Duration) {
	z.mu.Lock()
	z.t = z.t.Add(d)
	z.mu.Unlock()
}

var poczatekTestu = time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)

func nowyTestowy(a *atrapa, opcje ...Opcja) (*Monitor, *zegarTestowy) {
	z := &zegarTestowy{t: poczatekTestu}
	opcje = append([]Opcja{ZZrodlem(a), ZZegarem(z.teraz)}, opcje...)
	return Nowy(opcje...), z
}

func odczytTestowy(cpuRazem, cpuBezcz, dOdcz, dZap, wys, odb uint64) Odczyt {
	return Odczyt{
		CPURazem: cpuRazem, CPUBezczynny: cpuBezcz,
		RAMUzyte: 3 << 30, RAMRazem: 8 << 30,
		DyskOdczyt: dOdcz, DyskZapis: dZap, DyskDostepny: true,
		Siec: []LicznikSieci{
			{Interfejs: "Ethernet", Wyslane: wys, Odebrane: odb},
			{Interfejs: "wg0", Wyslane: wys / 2, Odebrane: odb / 2},
			{Interfejs: "Loopback", Petla: true, Wyslane: wys, Odebrane: odb},
		},
	}
}

func TestRoznicaLicznika(t *testing.T) {
	przypadki := []struct {
		nazwa       string
		poprz, biez uint64
		oczekiwane  uint64
	}{
		{"wzrost", 100, 250, 150},
		{"bez zmian", 7, 7, 0},
		{"reset licznika", 1 << 40, 500, 0},
		{"reset niskiego licznika", 1000, 10, 0},
		{"przekręcenie 32 bitów", konfiguracja.MonitorGranicaLicznika32 - 99, 50, 150},
	}
	for _, p := range przypadki {
		if w := roznicaLicznika(p.poprz, p.biez); w != p.oczekiwane {
			t.Errorf("%s: %d, oczekiwano %d", p.nazwa, w, p.oczekiwane)
		}
	}
}

func TestPredkosc(t *testing.T) {
	przypadki := []struct {
		poprz, biez uint64
		dt          float64
		oczekiwane  int64
	}{
		{0, 1000, 1, 1000},
		{0, 1000, 2, 500},
		{0, 1000, 0, 0},
		{0, 1000, -1, 0},
		{1000, 0, 1, 0},
	}
	for _, p := range przypadki {
		if w := predkosc(p.poprz, p.biez, p.dt); w != p.oczekiwane {
			t.Errorf("predkosc(%d,%d,%v) = %d, oczekiwano %d", p.poprz, p.biez, p.dt, w, p.oczekiwane)
		}
	}
}

func TestProcentCPU(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		a, b       Odczyt
		oczekiwane float64
	}{
		{"połowa", Odczyt{CPURazem: 1000, CPUBezczynny: 500}, Odczyt{CPURazem: 2000, CPUBezczynny: 1000}, 50},
		{"pełne", Odczyt{CPURazem: 1000, CPUBezczynny: 500}, Odczyt{CPURazem: 2000, CPUBezczynny: 500}, 100},
		{"bezczynny", Odczyt{CPURazem: 1000, CPUBezczynny: 500}, Odczyt{CPURazem: 2000, CPUBezczynny: 1500}, 0},
		{"zaokrąglenie", Odczyt{}, Odczyt{CPURazem: 3, CPUBezczynny: 2}, 33.3},
		{"brak przyrostu", Odczyt{CPURazem: 1000}, Odczyt{CPURazem: 1000}, 0},
		{"reset", Odczyt{CPURazem: 1000, CPUBezczynny: 500}, Odczyt{CPURazem: 10, CPUBezczynny: 5}, 0},
		{"bezczynny większy niż razem", Odczyt{}, Odczyt{CPURazem: 10, CPUBezczynny: 20}, 0},
	}
	for _, p := range przypadki {
		if w := procentCPU(p.a, p.b); w != p.oczekiwane {
			t.Errorf("%s: %v, oczekiwano %v", p.nazwa, w, p.oczekiwane)
		}
	}
}

func TestRodzajInterfejsu(t *testing.T) {
	przypadki := []struct {
		l          LicznikSieci
		oczekiwane string
	}{
		{LicznikSieci{Interfejs: "Ethernet"}, konfiguracja.RodzajLAN},
		{LicznikSieci{Interfejs: "eth0"}, konfiguracja.RodzajLAN},
		{LicznikSieci{Interfejs: "wg0"}, konfiguracja.RodzajWireGuard},
		{LicznikSieci{Interfejs: "Dom", Opis: "WireGuard Tunnel"}, konfiguracja.RodzajWireGuard},
		{LicznikSieci{Interfejs: "lo"}, konfiguracja.RodzajPetla},
		{LicznikSieci{Interfejs: "Loopback Pseudo-Interface 1", Petla: true}, konfiguracja.RodzajPetla},
	}
	for _, p := range przypadki {
		if w := rodzajInterfejsu(p.l); w != p.oczekiwane {
			t.Errorf("%+v: %s, oczekiwano %s", p.l, w, p.oczekiwane)
		}
	}
}

func TestPredkosciSieci(t *testing.T) {
	poprz := []LicznikSieci{{Interfejs: "eth0", Wyslane: 1000, Odebrane: 2000}, {Interfejs: "lo", Wyslane: 0}}
	biez := []LicznikSieci{
		{Interfejs: "wg0", Wyslane: 999, Odebrane: 999},
		{Interfejs: "lo", Wyslane: 5000, Odebrane: 5000},
		{Interfejs: "eth0", Wyslane: 3000, Odebrane: 6000},
	}
	lista, wys, odb := predkosciSieci(poprz, biez, 2)
	if len(lista) != 3 || lista[0].Interfejs != "eth0" || lista[1].Interfejs != "lo" || lista[2].Interfejs != "wg0" {
		t.Fatalf("zła lista lub kolejność: %+v", lista)
	}
	if lista[0].WysylanieBps != 1000 || lista[0].OdbiorBps != 2000 {
		t.Errorf("eth0: %+v", lista[0])
	}
	if lista[2].WysylanieBps != 0 || lista[2].OdbiorBps != 0 {
		t.Errorf("nowy interfejs powinien mieć 0: %+v", lista[2])
	}
	if lista[1].WysylanieBps != 2500 {
		t.Errorf("pętla: %+v", lista[1])
	}
	if wys != 1000 || odb != 2000 {
		t.Errorf("sumy bez pętli: %d %d", wys, odb)
	}
}

func TestMonitorPrzedkosciIProcent(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(1000, 500, 0, 0, 10_000, 20_000)}
	m, z := nowyTestowy(a, ZLicznikami(func() (int, int) { return 3, 2 }))
	m.probkuj()
	p := m.Teraz()
	if p.CPUProc != 0 || p.DyskOdczytBps != 0 || p.WysylanieBps != 0 {
		t.Errorf("pierwsza próbka powinna mieć zerowe prędkości: %+v", p)
	}
	z.przesun(2 * time.Second)
	a.ustaw(odczytTestowy(2000, 750, 4_000_000, 2_000_000, 30_000, 60_000))
	m.probkuj()
	p = m.Teraz()
	if p.CPUProc != 75 {
		t.Errorf("CPU %v, oczekiwano 75", p.CPUProc)
	}
	if p.DyskOdczytBps != 2_000_000 || p.DyskZapisBps != 1_000_000 {
		t.Errorf("dysk %d/%d", p.DyskOdczytBps, p.DyskZapisBps)
	}
	// Ethernet 10000 B/s wys + wg0 5000 B/s; pętla poza sumą.
	if p.WysylanieBps != 15_000 || p.OdbiorBps != 30_000 {
		t.Errorf("sieć razem %d/%d", p.WysylanieBps, p.OdbiorBps)
	}
	if p.Polaczenia != 3 || p.Transfery != 2 || p.Gorutyny < 1 || p.PamiecProcesu == 0 {
		t.Errorf("liczniki: %+v", p)
	}
	if p.RAMUzyte != 3<<30 || p.RAMRazem != 8<<30 || !p.Czas.Equal(poczatekTestu.Add(2*time.Second)) {
		t.Errorf("RAM lub czas: %+v", p)
	}
	if len(p.Siec) != 3 || p.Siec[2].Interfejs != "wg0" || p.Siec[2].Rodzaj != konfiguracja.RodzajWireGuard {
		t.Errorf("interfejsy: %+v", p.Siec)
	}
}

func TestMonitorResetLicznikow(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(5000, 1000, 9_000_000, 9_000_000, 1<<40, 1<<40)}
	m, z := nowyTestowy(a)
	m.probkuj()
	z.przesun(time.Second)
	a.ustaw(odczytTestowy(100, 50, 1000, 1000, 100, 100))
	m.probkuj()
	p := m.Teraz()
	if p.CPUProc != 0 || p.DyskOdczytBps != 0 || p.DyskZapisBps != 0 || p.WysylanieBps != 0 || p.OdbiorBps != 0 {
		t.Errorf("po resecie liczników prędkości powinny być 0: %+v", p)
	}
	z.przesun(time.Second)
	a.ustaw(odczytTestowy(200, 100, 2000, 3000, 1100, 2100))
	m.probkuj()
	p = m.Teraz()
	if p.CPUProc != 50 || p.DyskOdczytBps != 1000 || p.DyskZapisBps != 2000 || p.WysylanieBps != 1500 {
		t.Errorf("po resecie liczenie od nowa: %+v", p)
	}
}

func TestMonitorDyskNiedostepny(t *testing.T) {
	o1 := odczytTestowy(0, 0, 0, 0, 0, 0)
	o1.DyskDostepny = false
	a := &atrapa{odczyt: o1}
	m, z := nowyTestowy(a)
	m.probkuj()
	z.przesun(time.Second)
	o2 := odczytTestowy(0, 0, 5000, 5000, 0, 0)
	o2.DyskDostepny = false
	a.ustaw(o2)
	m.probkuj()
	if p := m.Teraz(); p.DyskOdczytBps != 0 || p.DyskZapisBps != 0 {
		t.Errorf("niedostępny dysk powinien dać 0: %+v", p)
	}
	if m.System().DyskIODostepny {
		t.Error("System powinien zgłosić brak liczników dysku")
	}
}

func TestMonitorBladZrodla(t *testing.T) {
	a := &atrapa{blad: errors.New("brak")}
	m, _ := nowyTestowy(a, ZLicznikami(func() (int, int) { return 1, 1 }))
	m.probkuj()
	p := m.Teraz()
	if m.OstatniBlad() == nil || p.RAMRazem != 0 || p.Polaczenia != 1 || p.Siec == nil {
		t.Errorf("próbka przy błędzie źródła: %+v, błąd %v", p, m.OstatniBlad())
	}
	if len(m.Historia1s()) != 1 {
		t.Error("próbka przy błędzie i tak trafia do historii")
	}
}

func TestPierscien(t *testing.T) {
	przypadki := []struct {
		pojemnosc, dodane int
		oczekPierwsza     int
		oczekDlugosc      int
	}{
		{3, 0, -1, 0},
		{3, 2, 0, 2},
		{3, 3, 0, 3},
		{3, 7, 4, 3},
		{0, 5, 4, 1},
	}
	for _, p := range przypadki {
		r := nowyPierscien(p.pojemnosc)
		for i := 0; i < p.dodane; i++ {
			r.dodaj(Probka{Gorutyny: i})
		}
		l := r.lista()
		if l == nil || len(l) != p.oczekDlugosc {
			t.Fatalf("%+v: długość %d", p, len(l))
		}
		for i := range l {
			if l[i].Gorutyny != p.oczekPierwsza+i {
				t.Errorf("%+v: zła kolejność %v", p, l)
			}
		}
	}
}

func TestHistoria1sOgraniczona(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)}
	m, z := nowyTestowy(a)
	ile := konfiguracja.MonitorProbkiHistorii1s + 5
	for i := 0; i < ile; i++ {
		m.probkuj()
		z.przesun(time.Second)
	}
	h := m.Historia1s()
	if len(h) != konfiguracja.MonitorProbkiHistorii1s {
		t.Fatalf("długość %d", len(h))
	}
	if !h[0].Czas.Equal(poczatekTestu.Add(5*time.Second)) || !h[len(h)-1].Czas.Equal(poczatekTestu.Add(time.Duration(ile-1)*time.Second)) {
		t.Errorf("zakres historii %v .. %v", h[0].Czas, h[len(h)-1].Czas)
	}
}

func TestUsrednianieMinutowe(t *testing.T) {
	s := nowySumator(time.Minute)
	p1 := Probka{Czas: poczatekTestu, CPUProc: 10, RAMUzyte: 100, WysylanieBps: 100, Polaczenia: 1,
		Siec: []SiecInterfejsu{{Interfejs: "eth0", Rodzaj: "lan", WysylanieBps: 100}}}
	p2 := Probka{Czas: poczatekTestu.Add(30 * time.Second), CPUProc: 30, RAMUzyte: 300, WysylanieBps: 300, Polaczenia: 2,
		Siec: []SiecInterfejsu{{Interfejs: "eth0", Rodzaj: "lan", WysylanieBps: 300}, {Interfejs: "wg0", Rodzaj: "wireguard", OdbiorBps: 50}}}
	if _, gotowa := s.dodaj(p1); gotowa {
		t.Fatal("pierwsza próbka nie zamyka minuty")
	}
	if _, gotowa := s.dodaj(p2); gotowa {
		t.Fatal("ta sama minuta")
	}
	sr, gotowa := s.dodaj(Probka{Czas: poczatekTestu.Add(time.Minute), CPUProc: 99})
	if !gotowa {
		t.Fatal("nowa minuta powinna zamknąć poprzednią")
	}
	if !sr.Czas.Equal(poczatekTestu) || sr.CPUProc != 20 || sr.RAMUzyte != 200 || sr.WysylanieBps != 200 || sr.Polaczenia != 2 {
		t.Errorf("średnia: %+v", sr)
	}
	if len(sr.Siec) != 2 || sr.Siec[0].WysylanieBps != 200 || sr.Siec[1].Interfejs != "wg0" || sr.Siec[1].OdbiorBps != 25 {
		t.Errorf("średnia interfejsów (brak = 0): %+v", sr.Siec)
	}
	if s.n != 1 || s.cpu != 99 {
		t.Errorf("sumator po zamknięciu minuty: n=%d cpu=%v", s.n, s.cpu)
	}
}

func TestHistoria1mZMonitora(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)}
	m, z := nowyTestowy(a)
	for i := 0; i < 125; i++ { // 2 pełne minuty i początek trzeciej
		m.probkuj()
		z.przesun(time.Second)
	}
	h := m.Historia1m()
	if len(h) != 2 || !h[0].Czas.Equal(poczatekTestu) || !h[1].Czas.Equal(poczatekTestu.Add(time.Minute)) {
		t.Errorf("historia minutowa: %d %+v", len(h), h)
	}
}

func TestSluchaczNieblokujacy(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)}
	m, z := nowyTestowy(a)
	kanal, wypisz := m.Sluchaj()
	gotowe := make(chan struct{})
	go func() {
		for i := 0; i < konfiguracja.MonitorBuforSluchacza*3; i++ {
			m.probkuj()
			z.przesun(time.Second)
		}
		close(gotowe)
	}()
	select {
	case <-gotowe:
	case <-time.After(5 * time.Second):
		t.Fatal("próbkowanie zablokowane przez słuchacza, który nie czyta")
	}
	if len(kanal) != konfiguracja.MonitorBuforSluchacza {
		t.Errorf("w kanale %d próbek", len(kanal))
	}
	pierwsza := <-kanal
	if !pierwsza.Czas.Equal(poczatekTestu) {
		t.Errorf("pierwsza próbka %v", pierwsza.Czas)
	}
	wypisz()
	wypisz() // idempotentne
	for range kanal {
	}
	m.probkuj() // po wypisaniu nic nie wysyła do zamkniętego kanału
}

func TestUruchomZTykaczem(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)}
	takty := make(chan time.Time)
	m, z := nowyTestowy(a, ZTykaczem(takty))
	kanal, wypisz := m.Sluchaj()
	defer wypisz()
	ctx, anuluj := context.WithCancel(context.Background())
	defer anuluj()
	m.Uruchom(ctx)
	m.Uruchom(ctx) // drugie wywołanie nic nie robi
	if p := <-kanal; !p.Czas.Equal(poczatekTestu) {
		t.Errorf("próbka startowa %v", p.Czas)
	}
	z.przesun(time.Second)
	takty <- time.Time{}
	if p := <-kanal; !p.Czas.Equal(poczatekTestu.Add(time.Second)) {
		t.Errorf("próbka po takcie %v", p.Czas)
	}
	if len(m.Historia1s()) != 2 {
		t.Errorf("historia %d", len(m.Historia1s()))
	}
}

func TestSystem(t *testing.T) {
	a := &atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)}
	m, z := nowyTestowy(a)
	m.probkuj()
	z.przesun(90 * time.Second)
	s := m.System()
	if s.CzasPracyS != 90 || s.RAMRazem != 8<<30 || s.CPURdzenie < 1 || s.OS == "" || s.Arch == "" || !s.DyskIODostepny {
		t.Errorf("System: %+v", s)
	}
}

func TestJSONProbki(t *testing.T) {
	m, _ := nowyTestowy(&atrapa{odczyt: odczytTestowy(0, 0, 0, 0, 0, 0)})
	m.probkuj()
	b, err := json.Marshal(m.Teraz())
	if err != nil {
		t.Fatal(err)
	}
	for _, klucz := range []string{"czas", "cpu_proc", "ram_uzyte", "ram_razem", "dysk_odczyt_bps", "dysk_zapis_bps",
		"siec", "interfejs", "rodzaj", "wysylanie_bps", "odbior_bps", "polaczenia", "transfery", "gorutyny", "pamiec_procesu"} {
		if !strings.Contains(string(b), `"`+klucz+`":`) {
			t.Errorf("brak klucza %s w %s", klucz, b)
		}
	}
	pusty, err := json.Marshal(Nowy(ZZrodlem(&atrapa{})).Teraz())
	if err != nil || !strings.Contains(string(pusty), `"siec":[]`) {
		t.Errorf("pusta próbka: siec powinna być [] (%s)", pusty)
	}
}
