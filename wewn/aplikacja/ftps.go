package aplikacja

import (
	"context"
	"sync"
	"time"

	"skrytka/wewn/ftp"
	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/siec"
)

// biegFTPS: serwery FTPS (po jednym na adres nasłuchu), restartowane po zmianie udziałów.
type biegFTPS struct {
	a      *Aplikacja
	ctx    context.Context // koniec programu
	adresy []string        // host:port

	mu     sync.Mutex
	anuluj context.CancelFunc
	koniec chan struct{} // zamykany, gdy wszystkie serwery bieżącego pokolenia skończą
}

// nowyBiegFTPS przygotowuje bieg (jeszcze nie uruchamia).
func nowyBiegFTPS(ctx context.Context, a *Aplikacja, adresy []string) *biegFTPS {
	return &biegFTPS{a: a, ctx: ctx, adresy: adresy}
}

// adresyFTPS: AdresNasluchu albo adresy z trybu sieci (publiczny = wszystkie interfejsy).
func adresyFTPS(k konfiguracja.Konfiguracja) []string {
	return siec.AdresyNasluchu(k)
}

// przelacz zatrzymuje działające serwery i uruchamia nowe na bieżącym magazynie
// (nic nie robi po końcu programu).
func (b *biegFTPS) przelacz() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.zatrzymajBezBlokady()
	if b.ctx.Err() != nil {
		return
	}
	ctx, anuluj := context.WithCancel(b.ctx)
	b.anuluj = anuluj
	k := b.a.Konfiguracja()
	var wg sync.WaitGroup
	for _, adres := range b.adresy {
		z := ftp.Zaleznosci{
			Konfiguracja: k,
			Adres:        adres,
			TLS:          b.a.tls,
			Magazyn:      b.a.magazyn.Load(),
			// Pusty w trybie VPN i do czasu odpowiedzi usługi IP (wtedy restart z adresem).
			AdresPubliczny: b.a.adresPubliczny.Load().(string),
			Konta:          b.a.konta,
			Dziennik:       b.a.dziennik,
			Transfery:      b.a.transfery,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ftp.Uruchom(ctx, z); err != nil && ctx.Err() == nil {
				b.a.zglosBlad("Serwer FTPS na "+z.Adres, err)
			}
		}()
	}
	koniec := make(chan struct{})
	b.koniec = koniec
	go func() {
		wg.Wait()
		close(koniec)
	}()
}

// zatrzymaj kończy serwery FTPS (czeka najwyżej CzasZamykania).
func (b *biegFTPS) zatrzymaj() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.zatrzymajBezBlokady()
}

// zatrzymajBezBlokady: wołający trzyma b.mu.
func (b *biegFTPS) zatrzymajBezBlokady() {
	if b.anuluj == nil {
		return
	}
	b.anuluj()
	b.anuluj = nil
	czasomierz := time.NewTimer(konfiguracja.CzasZamykania)
	defer czasomierz.Stop()
	select {
	case <-b.koniec:
	case <-czasomierz.C:
		b.a.zglosBlad("Zamykanie FTPS", errFTPSNieZamknal)
	}
}
