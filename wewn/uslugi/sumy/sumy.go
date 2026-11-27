// Pakiet sumy: suma SHA-256 pliku z udziału (sprawdzenie pobrania), liczona strumieniowo przez magazyn,
// z pamięcią podręczną wg (udział, ścieżka, rozmiar, data zmiany) i jednym liczeniem danego pliku naraz.
package sumy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strconv"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

// Sumy liczy i pamięta sumy SHA-256. Bezpieczne dla wielu gorutyn.
type Sumy struct {
	m          *magazyn.Magazyn
	maksWpisow int

	mu    sync.Mutex
	cache map[string]zapamietana // klucz: udział + "\x00" + ścieżka
	wToku map[string]*liczenie   // klucz: jak cache + rozmiar i data (zmieniony plik = osobne liczenie)
}

// zapamietana suma z wersją pliku, dla której była liczona.
type zapamietana struct {
	rozmiar int64
	czas    time.Time
	suma    string
}

// liczenie: trwające liczenie; czekający dostają ten sam wynik.
type liczenie struct {
	gotowe chan struct{}
	suma   string
	err    error
}

// Nowe tworzy usługę sum dla magazynu.
func Nowe(m *magazyn.Magazyn) *Sumy {
	return &Sumy{
		m:          m,
		maksWpisow: konfiguracja.SumyMaksWpisowCache,
		cache:      map[string]zapamietana{},
		wToku:      map[string]*liczenie{},
	}
}

// SHA256 zwraca sumę pliku szesnastkowo (małe litery). Folder = rdzen.ErrNieZnaleziono, błędy magazynu bez zmian,
// przerwanie ctx przerywa czytanie pliku.
func (s *Sumy) SHA256(ctx context.Context, udzial, sciezka string) (string, error) {
	w, err := s.m.Info(udzial, sciezka)
	if err != nil {
		return "", err
	}
	if w.Folder {
		return "", rdzen.ErrNieZnaleziono
	}
	k := udzial + "\x00" + w.Sciezka
	s.mu.Lock()
	if z, jest := s.cache[k]; jest && z.rozmiar == w.Rozmiar && z.czas.Equal(w.Zmieniono) {
		s.mu.Unlock()
		return z.suma, nil
	}
	kl := k + "\x00" + wersja(w)
	l, trwa := s.wToku[kl]
	if !trwa {
		l = &liczenie{gotowe: make(chan struct{})}
		s.wToku[kl] = l
	}
	s.mu.Unlock()
	if trwa {
		select {
		case <-l.gotowe:
			return l.suma, l.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	l.suma, l.err = s.licz(ctx, udzial, k, w)
	s.mu.Lock()
	delete(s.wToku, kl)
	s.mu.Unlock()
	close(l.gotowe)
	return l.suma, l.err
}

// wersja pliku do klucza liczenia.
func wersja(w rdzen.Wpis) string {
	return strconv.FormatInt(w.Rozmiar, 10) + "\x00" + strconv.FormatInt(w.Zmieniono.UnixNano(), 10)
}

// licz czyta plik strumieniowo i zapamiętuje sumę, jeśli plik nie zmienił się w trakcie.
func (s *Sumy) licz(ctx context.Context, udzial, k string, w rdzen.Wpis) (string, error) {
	f, wo, err := s.m.Otworz(udzial, w.Sciezka)
	if err != nil {
		return "", err
	}
	defer f.Close() // celowo: tylko odczyt
	h := sha256.New()
	if _, err := io.Copy(h, czytnikZCtx{ctx, f}); err != nil {
		return "", err
	}
	suma := hex.EncodeToString(h.Sum(nil))
	po, err := s.m.Info(udzial, w.Sciezka)
	if err == nil && po.Rozmiar == wo.Rozmiar && po.Zmieniono.Equal(wo.Zmieniono) {
		s.zapamietaj(k, zapamietana{rozmiar: wo.Rozmiar, czas: wo.Zmieniono, suma: suma})
	}
	return suma, nil
}

// zapamietaj z limitem wpisów (pełna pamięć = usuwamy dowolne wpisy, kolejność map jest losowa).
func (s *Sumy) zapamietaj(k string, z zapamietana) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for kk := range s.cache {
		if len(s.cache) < s.maksWpisow {
			break
		}
		delete(s.cache, kk)
	}
	s.cache[k] = z
}

// czytnikZCtx przerywa odczyt, gdy ctx się skończy (rozłączony klient nie czyta dalej dużego pliku).
type czytnikZCtx struct {
	ctx context.Context
	r   io.Reader
}

func (c czytnikZCtx) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
