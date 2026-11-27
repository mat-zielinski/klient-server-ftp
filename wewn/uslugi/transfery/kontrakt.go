// Pakiet transfery: pobrania i wgrania NA ŻYWO (HTTP i FTPS) z prędkością, do podglądu serwera
// (admin: wszystkie) i klienta (tylko własne). Właściciel: agent „transfery”.
//
// Gorąca ścieżka (CzytajZ, PiszDo, Dodaj) to wyłącznie operacje atomowe; prędkość, historię
// i zdarzenia „postep” liczy jedna gorutyna próbkująca (co konfiguracja.TransferyOkresProbkowania).
package transfery

import (
	"time"
)

// Kierunki.
const (
	KierunekPobranie = "pobranie"
	KierunekWgranie  = "wgranie"
)

// Typy zdarzeń.
const (
	ZdarzenieStart  = "start"
	ZdarzeniePostep = "postep"
	ZdarzenieKoniec = "koniec"
)

// Wyniki transferu.
const (
	WynikOk        = "ok"
	WynikPrzerwane = "przerwane"
	WynikBlad      = "blad"
)

// Stan transferu (kopia do pokazania, bez wskaźników).
type Stan struct {
	ID          string    `json:"id"`
	Kto         string    `json:"kto"`
	IP          string    `json:"ip"`
	Protokol    string    `json:"protokol"` // https | http | ftps
	Kierunek    string    `json:"kierunek"`
	Udzial      string    `json:"udzial"`
	Sciezka     string    `json:"sciezka"`
	Rozmiar     int64     `json:"rozmiar"` // 0 = nieznany (ZIP)
	Przeslano   int64     `json:"przeslano"`
	PredkoscBps int64     `json:"predkosc_bps"` // średnia z ostatnich kilku sekund
	Start       time.Time `json:"start"`
	PozostaloS  int64     `json:"pozostalo_s"` // -1 = nieznane
	// Historia prędkości (B/s co sekundę, ostatnie 60 s) do wykresiku przy transferze.
	Historia []int64 `json:"historia"`
}

// Zdarzenie wysyłane słuchaczom (SSE): "start" | "postep" | "koniec".
type Zdarzenie struct {
	Typ   string `json:"typ"`
	Stan  Stan   `json:"stan"`
	Wynik string `json:"wynik,omitempty"` // przy "koniec": ok | przerwane | blad
}

// kopia zwraca stan z własną kopią historii (bezpieczny do oddania na zewnątrz).
func (s Stan) kopia() Stan {
	h := make([]int64, len(s.Historia))
	copy(h, s.Historia)
	s.Historia = h
	return s
}
