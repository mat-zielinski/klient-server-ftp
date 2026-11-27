package http

import (
	"context"
	nethttp "net/http"

	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/indeks"
	"skrytka/wewn/uslugi/monitor"
)

// Usługi fali 2 (monitor, indeks, miniatury, sumy): każda może być wyłączona (nil = 503).
type (
	uslugaMonitora interface {
		Teraz() monitor.Probka
		Historia1s() []monitor.Probka
		Historia1m() []monitor.Probka
		System() monitor.System
		Sluchaj() (<-chan monitor.Probka, func())
	}
	uslugaIndeksu interface {
		Gotowy(udzial string) bool
		Szukaj(udzial, sciezkaStartowa, zapytanie string, limit int) []rdzen.Wpis
		Statystyki() []indeks.StatystykaUdzialu
		Odswiez(udzial string)
	}
	uslugaMiniatur interface {
		Miniatura(ctx context.Context, udzial, sciezka, rozmiar string) (string, error)
	}
	uslugaSum interface {
		SHA256(ctx context.Context, udzial, sciezka string) (string, error)
	}
)

// dodatki: usługi opcjonalne (interfejs nil = wyłączona).
type dodatki struct {
	monitor   uslugaMonitora
	indeks    uslugaIndeksu
	miniatury uslugaMiniatur
	sumy      uslugaSum
}

// dodatkiZ przepisuje tylko niepuste wskaźniki (żeby nil w Zaleznosci dał nil interfejsu, nie „typowany nil”).
func dodatkiZ(z Zaleznosci) dodatki {
	var d dodatki
	if z.Monitor != nil {
		d.monitor = z.Monitor
	}
	if z.Indeks != nil {
		d.indeks = z.Indeks
	}
	if z.Miniatury != nil {
		d.miniatury = z.Miniatury
	}
	if z.Sumy != nil {
		d.sumy = z.Sumy
	}
	return d
}

var errWylaczone = nowyBlad(nethttp.StatusServiceUnavailable, "Ta funkcja jest wyłączona na tym serwerze.")

// odswiezIndeks po każdej zmianie plików (indeks sam łączy prośby).
func (s *serwer) odswiezIndeks(udzial string) {
	if s.indeks != nil {
		s.indeks.Odswiez(udzial)
	}
}
