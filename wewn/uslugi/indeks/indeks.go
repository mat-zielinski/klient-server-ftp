// Pakiet indeks: indeks plików udziałów w pamięci, budowany w tle przez magazyn (jak FileBrowser Quantum).
// Wyszukiwanie po indeksie trwa milisekundy; dopóki udział nie jest zindeksowany, HTTP używa magazyn.Szukaj.
package indeks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
)

// Indeks trzyma zwartą strukturę plików każdego udziału. Bezpieczny dla wielu gorutyn.
type Indeks struct {
	m            *magazyn.Magazyn
	okres        time.Duration
	opoznienie   time.Duration
	przerwa      time.Duration
	rownoleglosc int
	zegar        func() time.Time

	udzialy map[string]*stanUdzialu // klucz: nazwa udziału; mapa stała po Nowy
	// poPrzejsciu: wołane po każdym przejściu w pętli tła (testy czekają na nie zamiast spać).
	poPrzejsciu func(udzial string)
}

// stanUdzialu: ostatni gotowy zbiór (podmieniany atomowo) i kanał próśb o odświeżenie.
type stanUdzialu struct {
	zbior  atomic.Pointer[zbior]
	prosba chan struct{} // bufor 1: wiele próśb = jedno przejście
	budowa sync.Mutex    // jedno przejście udziału naraz
}

// Opcja zmienia ustawienia indeksu (testy, konfiguracja).
type Opcja func(*Indeks)

// ZOkresem: co ile pełne odświeżenie wszystkich udziałów.
func ZOkresem(d time.Duration) Opcja { return func(x *Indeks) { x.okres = d } }

// ZOpoznieniem: zwłoka między Odswiez a przejściem udziału.
func ZOpoznieniem(d time.Duration) Opcja { return func(x *Indeks) { x.opoznienie = d } }

// ZPrzerwa: przerwa po każdym folderze (0 = bez przerw).
func ZPrzerwa(d time.Duration) Opcja { return func(x *Indeks) { x.przerwa = d } }

// ZRownolegloscia: ile folderów naraz (najmniej 1).
func ZRownolegloscia(n int) Opcja { return func(x *Indeks) { x.rownoleglosc = max(1, n) } }

// ZZegarem: źródło czasu do znacznika „zindeksowano”.
func ZZegarem(z func() time.Time) Opcja { return func(x *Indeks) { x.zegar = z } }

// Nowy tworzy indeks udziałów magazynu (pusty; przejścia rusza Uruchom albo Przeindeksuj).
func Nowy(m *magazyn.Magazyn, opcje ...Opcja) *Indeks {
	x := &Indeks{
		m:            m,
		okres:        konfiguracja.IndeksOkresOdswiezania,
		opoznienie:   konfiguracja.IndeksOpoznienieOdswiezenia,
		przerwa:      konfiguracja.IndeksPrzerwaMiedzyFolderami,
		rownoleglosc: konfiguracja.IndeksRownoleglosc,
		zegar:        time.Now,
		udzialy:      map[string]*stanUdzialu{},
	}
	for _, o := range opcje {
		o(x)
	}
	for _, u := range m.Udzialy() {
		x.udzialy[u.Nazwa] = &stanUdzialu{prosba: make(chan struct{}, 1)}
	}
	return x
}

// Uruchom startuje pracę w tle i od razu wraca: pierwsze przejście wszystkich udziałów, potem co okres
// i po każdym Odswiez. Kończy się z ctx.
func (x *Indeks) Uruchom(ctx context.Context) {
	for nazwa, s := range x.udzialy {
		go x.petla(ctx, nazwa, s)
	}
}

// petla jednego udziału: przejście, potem czekanie na okres albo prośbę.
func (x *Indeks) petla(ctx context.Context, nazwa string, s *stanUdzialu) {
	tykacz := time.NewTicker(x.okres)
	defer tykacz.Stop()
	for {
		_ = x.Przeindeksuj(ctx, nazwa) // celowo: błąd przejścia = zostaje poprzedni zbiór, następna próba za okres
		if x.poPrzejsciu != nil {
			x.poPrzejsciu(nazwa)
		}
		select {
		case <-ctx.Done():
			return
		case <-tykacz.C:
		case <-s.prosba:
			if !czekaj(ctx, x.opoznienie) {
				return
			}
		}
	}
}

// Odswiez prosi o ponowne przejście udziału (HTTP woła po zapisie, zmianie nazwy, usunięciu).
// Nie blokuje; wiele próśb przed przejściem daje jedno przejście. Nieznany udział = nic.
func (x *Indeks) Odswiez(udzial string) {
	s, ok := x.udzialy[udzial]
	if !ok {
		return
	}
	select {
	case s.prosba <- struct{}{}:
	default: // prośba już czeka
	}
}

// Przeindeksuj przechodzi udział teraz (synchronicznie) i podmienia zbiór. Przerwane przez ctx = stary zbiór zostaje.
func (x *Indeks) Przeindeksuj(ctx context.Context, udzial string) error {
	s, ok := x.udzialy[udzial]
	if !ok {
		return errNieznanyUdzial
	}
	s.budowa.Lock()
	defer s.budowa.Unlock()
	z, err := x.przejdz(ctx, udzial)
	if err != nil {
		return err
	}
	s.zbior.Store(z)
	return nil
}

// Gotowy: czy udział ma już zbudowany indeks.
func (x *Indeks) Gotowy(udzial string) bool {
	s, ok := x.udzialy[udzial]
	return ok && s.zbior.Load() != nil
}

// czekaj d albo do końca ctx; false = ctx zakończony.
func czekaj(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
