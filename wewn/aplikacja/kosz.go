package aplikacja

import (
	"context"
	"fmt"
	"time"

	"skrytka/wewn/konfiguracja"
)

// uruchomOproznianieKosza: raz przy starcie i potem co okres trwale usuwa z koszy udziałów wpisy starsze niż
// KoszOproznijPoDniach, do końca ctx.
func (a *Aplikacja) uruchomOproznianieKosza(ctx context.Context, okres time.Duration) {
	a.serwery.wg.Add(1)
	go func() {
		defer a.serwery.wg.Done()
		a.oproznijKosze()
		zegar := time.NewTicker(okres)
		defer zegar.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-zegar.C:
				a.oproznijKosze()
			}
		}
	}()
}

// oproznijKosze: jedno przejście po udziałach (tylko do odczytu pomijane), wpis w dzienniku gdy coś usunięto
// albo był błąd.
func (a *Aplikacja) oproznijKosze() {
	m := a.magazyn.Load()
	if m == nil {
		return
	}
	for _, u := range m.Udzialy() {
		if u.TylkoOdczyt {
			continue
		}
		n, err := m.Oproznij(u.Nazwa, konfiguracja.KoszWiekAutomatycznegoOprozniania)
		if err != nil {
			a.zglosBlad("Opróżnianie kosza udziału "+u.Nazwa, err)
			continue
		}
		if n > 0 {
			a.zglos(fmt.Sprintf("Kosz udziału %s: usunięto trwale %d el. starszych niż %d dni.",
				u.Nazwa, n, konfiguracja.KoszOproznijPoDniach))
		}
	}
}
