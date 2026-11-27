package transfery

import (
	"errors"
	"io"
	"sync/atomic"
	"time"
)

// ErrPrzerwany zwraca odczyt/zapis transferu przerwanego przez administratora (Rejestr.Przerwij).
var ErrPrzerwany = errors.New("Transfer przerwany przez administratora.")

// Transfer to uchwyt jednego przesyłu.
type Transfer struct {
	r *Rejestr
	// Gorąca ścieżka: tylko atomowe.
	przeslano  atomic.Int64
	przerwany  atomic.Bool
	zakonczony atomic.Bool

	// Pod r.mu (gorutyna próbkująca, Aktywne, usun).
	stan          Stan
	probki        []probka
	ostatniPostep time.Time
}

// CzytajZ opakowuje strumień: każdy odczyt dolicza bajty.
func (t *Transfer) CzytajZ(src io.Reader) io.Reader { return &czytnik{t: t, src: src} }

// PiszDo opakowuje strumień: każdy zapis dolicza bajty.
func (t *Transfer) PiszDo(dst io.Writer) io.Writer { return &pisarz{t: t, dst: dst} }

// Dodaj dolicza bajty ręcznie (np. FTPS).
func (t *Transfer) Dodaj(n int64) {
	if n > 0 {
		t.przeslano.Add(n)
	}
}

// Przerwany (rozszerzenie kontraktu): true po Rejestr.Przerwij; dla kodu liczącego ręcznie przez Dodaj.
func (t *Transfer) Przerwany() bool { return t.przerwany.Load() }

// Zakoncz zamyka transfer z wynikiem (ok | przerwane | blad); drugie wywołanie nic nie robi.
// Transfer przerwany przez administratora kończy się zawsze wynikiem „przerwane”.
func (t *Transfer) Zakoncz(wynik string) {
	if !t.zakonczony.CompareAndSwap(false, true) {
		return
	}
	if t.przerwany.Load() {
		wynik = WynikPrzerwane
	}
	s := t.r.usun(t, wynik)
	if t.r.naKoniec != nil {
		t.r.naKoniec(s, wynik)
	}
}

// migawka: kopia stanu z bieżącym licznikiem; wołać pod r.mu.
func (t *Transfer) migawka() Stan {
	s := t.stan.kopia()
	s.Przeslano = t.przeslano.Load()
	return s
}

type czytnik struct {
	t   *Transfer
	src io.Reader
}

func (c *czytnik) Read(p []byte) (int, error) {
	if c.t.przerwany.Load() {
		return 0, ErrPrzerwany
	}
	n, err := c.src.Read(p)
	if n > 0 {
		c.t.przeslano.Add(int64(n))
	}
	return n, err
}

type pisarz struct {
	t   *Transfer
	dst io.Writer
}

func (w *pisarz) Write(p []byte) (int, error) {
	if w.t.przerwany.Load() {
		return 0, ErrPrzerwany
	}
	n, err := w.dst.Write(p)
	if n > 0 {
		w.t.przeslano.Add(int64(n))
	}
	return n, err
}
