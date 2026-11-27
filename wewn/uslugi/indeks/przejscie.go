package indeks

import (
	"context"
	"path"
	"sync"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// zadanie: folder do przeczytania.
type zadanie struct {
	sciezka   string
	glebokosc int
}

// przejscie: wspólny stan pracowników jednego przejścia udziału (wszystko pod mu).
type przejscie struct {
	mu           sync.Mutex
	cond         *sync.Cond
	kolejka      []zadanie
	aktywne      int
	koniec       bool
	z            *zbior
	bladKorzenia error
}

// przejdz czyta udział folder po folderze (magazyn.Lista: bez ukrytych, dowiązania tylko w udziale)
// w x.rownoleglosc pracownikach i buduje nowy zbiór.
func (x *Indeks) przejdz(ctx context.Context, udzial string) (*zbior, error) {
	p := &przejscie{z: &zbior{}, kolejka: []zadanie{{sciezka: "/"}}}
	p.cond = sync.NewCond(&p.mu)
	var wg sync.WaitGroup
	for range x.rownoleglosc {
		wg.Go(func() { x.pracownik(ctx, udzial, p) })
	}
	// Przerwanie ctx budzi czekających pracowników.
	stop := context.AfterFunc(ctx, func() {
		p.mu.Lock()
		p.koniec = true
		p.cond.Broadcast()
		p.mu.Unlock()
	})
	wg.Wait()
	stop()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.bladKorzenia != nil {
		return nil, p.bladKorzenia
	}
	p.z.uporzadkuj()
	p.z.statystyka = policzStatystyke(udzial, p.z, x.zegar())
	return p.z, nil
}

// pracownik bierze foldery z kolejki, aż kolejka pusta i nikt nie pracuje.
func (x *Indeks) pracownik(ctx context.Context, udzial string, p *przejscie) {
	for {
		zad, ok := p.wez()
		if !ok {
			return
		}
		wpisy, err := x.m.Lista(udzial, zad.sciezka)
		p.oddaj(zad, wpisy, err)
		if !czekaj(ctx, x.przerwa) {
			return
		}
	}
}

// wez czeka na folder; false = koniec przejścia.
func (p *przejscie) wez() (zadanie, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.kolejka) == 0 && p.aktywne > 0 && !p.koniec {
		p.cond.Wait()
	}
	if len(p.kolejka) == 0 || p.koniec {
		p.koniec = true
		p.cond.Broadcast()
		return zadanie{}, false
	}
	zad := p.kolejka[len(p.kolejka)-1]
	p.kolejka = p.kolejka[:len(p.kolejka)-1]
	p.aktywne++
	return zad, true
}

// oddaj dopisuje zawartość folderu do zbioru i jego podfoldery do kolejki.
func (p *przejscie) oddaj(zad zadanie, wpisy []rdzen.Wpis, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.aktywne--
	defer p.cond.Broadcast()
	if err != nil {
		if zad.sciezka == "/" {
			p.bladKorzenia = err
		}
		return // celowo: nieczytelny podfolder pomijamy, reszta udziału zostaje zindeksowana
	}
	z := p.z
	rodzic := int32(len(z.foldery))
	z.foldery = append(z.foldery, zad.sciezka)
	for _, w := range wpisy {
		if len(z.pozycje) >= konfiguracja.IndeksMaksPozycji {
			return
		}
		z.pozycje = append(z.pozycje, nowaPozycja(w, rodzic))
		if w.Folder && zad.glebokosc < konfiguracja.IndeksMaksGlebokosc {
			p.kolejka = append(p.kolejka, zadanie{sciezka: path.Join(zad.sciezka, w.Nazwa), glebokosc: zad.glebokosc + 1})
		}
	}
}
