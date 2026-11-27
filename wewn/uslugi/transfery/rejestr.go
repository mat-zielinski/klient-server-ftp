package transfery

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Rejestr aktywnych transferów; bezpieczny dla wielu gorutyn.
type Rejestr struct {
	mu        sync.Mutex
	aktywne   map[string]*Transfer
	sluchacze map[*sluchacz]struct{}
	naKoniec  func(s Stan, wynik string)

	teraz     func() time.Time
	tykacz    <-chan time.Time
	zatrzymaj func()
	stop      chan struct{}
	stopRaz   sync.Once
	// poProbce: zaczep dla testów, wołany przez gorutynę po każdej próbce.
	poProbce func()
}

// Opcja zmienia ustawienia rejestru (głównie do testów).
type Opcja func(*Rejestr)

// ZZegarem wstrzykuje zegar (domyślnie time.Now).
func ZZegarem(teraz func() time.Time) Opcja {
	return func(r *Rejestr) { r.teraz = teraz }
}

// ZTykaczem wstrzykuje kanał taktów próbkowania (domyślnie time.Ticker co TransferyOkresProbkowania).
// Kanał nil = brak gorutyny próbkującej.
func ZTykaczem(tykacz <-chan time.Time) Opcja {
	return func(r *Rejestr) {
		r.tykacz = tykacz
		r.zatrzymaj = func() {}
	}
}

// Nowy rejestr; koniec transferu sam zapisuje wpis w dzienniku przez funkcję naKoniec (może być nil).
func Nowy(naKoniec func(s Stan, wynik string)) *Rejestr {
	return NowyZOpcjami(naKoniec)
}

// NowyZOpcjami jak Nowy, z wstrzykniętym zegarem lub tykaczem.
func NowyZOpcjami(naKoniec func(s Stan, wynik string), opcje ...Opcja) *Rejestr {
	r := &Rejestr{
		aktywne:   map[string]*Transfer{},
		sluchacze: map[*sluchacz]struct{}{},
		naKoniec:  naKoniec,
		teraz:     time.Now,
		stop:      make(chan struct{}),
	}
	for _, o := range opcje {
		o(r)
	}
	if r.zatrzymaj == nil {
		t := time.NewTicker(konfiguracja.TransferyOkresProbkowania)
		r.tykacz = t.C
		r.zatrzymaj = t.Stop
	}
	if r.tykacz != nil {
		go r.petlaProbkowania()
	}
	return r
}

// Zamknij zatrzymuje gorutynę próbkującą (rozszerzenie kontraktu; wołane przy wyłączaniu i w testach).
func (r *Rejestr) Zamknij() {
	r.stopRaz.Do(func() {
		close(r.stop)
		r.zatrzymaj()
	})
}

// Rozpocznij rejestruje transfer (pola ID, Start, Przeslano, Predkosc wypełnia rejestr).
func (r *Rejestr) Rozpocznij(s Stan) *Transfer {
	teraz := r.teraz()
	s.Start = teraz
	s.Przeslano = 0
	s.PredkoscBps = 0
	s.PozostaloS = -1
	s.Historia = []int64{}
	t := &Transfer{r: r, stan: s, probki: []probka{{czas: teraz}}, ostatniPostep: teraz}

	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		t.stan.ID = noweID()
		if _, jest := r.aktywne[t.stan.ID]; !jest {
			break
		}
	}
	r.aktywne[t.stan.ID] = t
	r.rozeslij(Zdarzenie{Typ: ZdarzenieStart, Stan: t.migawka()})
	return t
}

// Aktywne: wszystkie (kto == "") albo tylko danego użytkownika.
func (r *Rejestr) Aktywne(kto string) []Stan {
	r.mu.Lock()
	wynik := make([]Stan, 0, len(r.aktywne))
	for _, t := range r.aktywne {
		if kto == "" || t.stan.Kto == kto {
			wynik = append(wynik, t.migawka())
		}
	}
	r.mu.Unlock()
	sort.Slice(wynik, func(i, j int) bool {
		if !wynik[i].Start.Equal(wynik[j].Start) {
			return wynik[i].Start.Before(wynik[j].Start)
		}
		return wynik[i].ID < wynik[j].ID
	})
	return wynik
}

// PredkoscLaczna B/s wszystkich aktywnych (do wykresu na Pulpicie).
func (r *Rejestr) PredkoscLaczna() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	var suma int64
	for _, t := range r.aktywne {
		suma += t.stan.PredkoscBps
	}
	return suma
}

// Przerwij (rozszerzenie kontraktu, admin): oznacza transfer jako przerwany; kolejny odczyt/zapis
// przez CzytajZ/PiszDo zwraca ErrPrzerwany, a Zakoncz zapisze wynik „przerwane”.
// false = brak aktywnego transferu o tym ID.
func (r *Rejestr) Przerwij(id string) bool {
	r.mu.Lock()
	t, jest := r.aktywne[id]
	r.mu.Unlock()
	if !jest {
		return false
	}
	t.przerwany.Store(true)
	return true
}

// usun wyjmuje transfer z rejestru i rozsyła „koniec”; zwraca stan końcowy.
func (r *Rejestr) usun(t *Transfer, wynik string) Stan {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.aktywne, t.stan.ID)
	s := t.migawka()
	r.rozeslij(Zdarzenie{Typ: ZdarzenieKoniec, Stan: s, Wynik: wynik})
	return s
}

// noweID zwraca losowy identyfikator szesnastkowy.
func noweID() string {
	b := make([]byte, konfiguracja.TransferyBajtyID)
	_, _ = rand.Read(b) // celowo: od Go 1.24 crypto/rand.Read nigdy nie zwraca błędu (przy awarii kończy program)
	return hex.EncodeToString(b)
}
