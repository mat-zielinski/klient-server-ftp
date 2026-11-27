package konta

import (
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// licznik złych prób dla jednego klucza (adres IP albo login).
type licznik struct {
	bledy     []time.Time
	blokadaDo time.Time
}

// blokady: liczniki złych haseł w oknie czasu, osobno dla IP i loginu.
type blokady struct {
	mu    sync.Mutex
	wpisy map[string]*licznik
}

func noweBlokady() *blokady { return &blokady{wpisy: map[string]*licznik{}} }

func kluczIP(ip string) string        { return "ip:" + ip }
func kluczLoginu(login string) string { return "login:" + login }

// czyZablokowane: true, gdy którykolwiek klucz ma trwającą blokadę.
func (b *blokady) czyZablokowane(teraz time.Time, klucze ...string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, k := range klucze {
		if l, ok := b.wpisy[k]; ok && teraz.Before(l.blokadaDo) {
			return true
		}
	}
	return false
}

// zarejestrujBlad dopisuje złą próbę; po MaksBlednychLogowan w oknie zakłada blokadę.
func (b *blokady) zarejestrujBlad(teraz time.Time, klucze ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.wpisy) >= konfiguracja.ProgSprzataniaBlokad {
		b.sprzataj(teraz)
	}
	for _, k := range klucze {
		l, ok := b.wpisy[k]
		if !ok {
			l = &licznik{}
			b.wpisy[k] = l
		}
		l.bledy = append(swieze(l.bledy, teraz), teraz)
		if len(l.bledy) >= konfiguracja.MaksBlednychLogowan {
			l.blokadaDo = teraz.Add(konfiguracja.CzasBlokady)
			l.bledy = nil
		}
	}
}

// wyczysc usuwa licznik klucza (np. po dobrym logowaniu).
func (b *blokady) wyczysc(klucz string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.wpisy, klucz)
}

// sprzataj usuwa wpisy bez trwającej blokady i bez prób w oknie. Wywołujący trzyma mu.
func (b *blokady) sprzataj(teraz time.Time) {
	for k, l := range b.wpisy {
		l.bledy = swieze(l.bledy, teraz)
		if len(l.bledy) == 0 && !teraz.Before(l.blokadaDo) {
			delete(b.wpisy, k)
		}
	}
}

// swieze zostawia próby nie starsze niż OknoBlednychLogowan.
func swieze(bledy []time.Time, teraz time.Time) []time.Time {
	granica := teraz.Add(-konfiguracja.OknoBlednychLogowan)
	wynik := bledy[:0]
	for _, t := range bledy {
		if t.After(granica) {
			wynik = append(wynik, t)
		}
	}
	return wynik
}
