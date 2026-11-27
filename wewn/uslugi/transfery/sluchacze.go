package transfery

import (
	"sync"

	"skrytka/wewn/konfiguracja"
)

// sluchacz jednego strumienia SSE.
//
// Zdarzenia idą do kanału bez czekania. Gdy kanał jest pełny: „postep” przepada, a „start”
// i „koniec” trafiają do kolejki zaległych, którą opróżnia osobna gorutyna (tylko dla tego
// słuchacza), więc wolny słuchacz nie blokuje rejestru ani innych, a „koniec” zawsze dociera.
type sluchacz struct {
	kto string
	k   chan Zdarzenie

	mu        sync.Mutex
	zalegle   []Zdarzenie
	oprozniam bool
	zamkniety bool
	koniec    chan struct{}
	wg        sync.WaitGroup
}

// Sluchaj zwraca kanał zdarzeń (postęp najwyżej co 500 ms na transfer) i funkcję wypisania.
// kto == "" = wszystkie (admin). Po wypisaniu kanał zostaje zamknięty.
func (r *Rejestr) Sluchaj(kto string) (<-chan Zdarzenie, func()) {
	s := &sluchacz{
		kto:    kto,
		k:      make(chan Zdarzenie, konfiguracja.TransferyBuforSluchacza),
		koniec: make(chan struct{}),
	}
	r.mu.Lock()
	r.sluchacze[s] = struct{}{}
	r.mu.Unlock()

	var raz sync.Once
	wypisz := func() {
		raz.Do(func() {
			r.mu.Lock()
			delete(r.sluchacze, s)
			r.mu.Unlock()
			s.mu.Lock()
			s.zamkniety = true
			close(s.koniec)
			s.mu.Unlock()
			s.wg.Wait()
			close(s.k)
		})
	}
	return s.k, wypisz
}

// rozeslij (pod r.mu) przekazuje zdarzenie słuchaczom, którym wolno je widzieć.
func (r *Rejestr) rozeslij(z Zdarzenie) {
	for s := range r.sluchacze {
		if s.kto == "" || s.kto == z.Stan.Kto {
			s.wyslij(z)
		}
	}
}

// wyslij nigdy nie blokuje.
func (s *sluchacz) wyslij(z Zdarzenie) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.zamkniety {
		return
	}
	wazne := z.Typ != ZdarzeniePostep
	if s.oprozniam {
		// Są zaległe: zachowaj kolejność, postęp pomiń.
		if wazne {
			s.zalegle = append(s.zalegle, z)
		}
		return
	}
	select {
	case s.k <- z:
		return
	default:
	}
	if !wazne {
		return
	}
	s.zalegle = append(s.zalegle, z)
	s.oprozniam = true
	s.wg.Add(1)
	go s.oproznij()
}

// oproznij dostarcza zaległe „start”/„koniec” po kolei, czekając na odbiorcę.
func (s *sluchacz) oproznij() {
	defer s.wg.Done()
	for {
		s.mu.Lock()
		if len(s.zalegle) == 0 || s.zamkniety {
			s.oprozniam = false
			s.zalegle = nil
			s.mu.Unlock()
			return
		}
		z := s.zalegle[0]
		s.mu.Unlock()
		select {
		case s.k <- z:
			s.mu.Lock()
			s.zalegle = s.zalegle[1:]
			s.mu.Unlock()
		case <-s.koniec:
			return
		}
	}
}
