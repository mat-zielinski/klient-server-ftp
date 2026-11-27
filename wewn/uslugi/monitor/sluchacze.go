package monitor

import (
	"sync"

	"skrytka/wewn/konfiguracja"
)

// sluchacz: kanał próbek jednego odbiorcy (np. połączenia SSE).
type sluchacz struct {
	kanal chan Probka
}

// Sluchaj zwraca kanał nowych próbek i funkcję wypisania (idempotentna, zamyka kanał). Wysyłka nigdy
// nie blokuje: gdy odbiorca nie nadąża i kanał jest pełny, próbka dla niego przepada.
func (m *Monitor) Sluchaj() (<-chan Probka, func()) {
	s := &sluchacz{kanal: make(chan Probka, konfiguracja.MonitorBuforSluchacza)}
	m.mu.Lock()
	m.sluchacze[s] = struct{}{}
	m.mu.Unlock()
	var raz sync.Once
	wypisz := func() {
		raz.Do(func() {
			m.mu.Lock()
			delete(m.sluchacze, s)
			close(s.kanal)
			m.mu.Unlock()
		})
	}
	return s.kanal, wypisz
}

// rozeslij wysyła próbkę wszystkim słuchaczom bez czekania (wołane pod mu).
func (m *Monitor) rozeslij(p Probka) {
	for s := range m.sluchacze {
		select {
		case s.kanal <- p:
		default: // celowo: pełny kanał = wolny odbiorca, próbka przepada
		}
	}
}
