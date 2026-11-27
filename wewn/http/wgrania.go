package http

import (
	"errors"
	nethttp "net/http"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/transfery"
)

// aktywneWgranie: jeden transfer na cały plik wgrywany w kawałkach (PUT /api/wgraj).
type aktywneWgranie struct {
	t transfer
	// ostatnio: koniec ostatniego kawałka (albo start); od niego liczona bezczynność.
	ostatnio time.Time
	// wTrakcie: ile kawałków właśnie się wczytuje (takiego wgrania sprzątanie nie rusza).
	wTrakcie int
}

// wgraniaHTTP: aktywne wgrania wg klucza login+udział+ścieżka.
type wgraniaHTTP struct {
	mu      sync.Mutex
	aktywne map[string]*aktywneWgranie
	// sprzatacz: czy działa gorutyna z tykaczem (tylko gdy są aktywne wgrania).
	sprzatacz bool
}

func kluczWgrania(login, udzial, sciezka string) string {
	return login + "\x00" + udzial + "\x00" + sciezka
}

// rozmiarDoWyslania: ile bajtów zostało do wgrania (całość minus przesunięcie startu); 0 = nieznane.
func rozmiarDoWyslania(calkowity, przesuniecie int64) int64 {
	if calkowity <= 0 || calkowity < przesuniecie {
		return 0
	}
	return calkowity - przesuniecie
}

// zacznijKawalek zwraca transfer dla kawałka: istniejący albo nowy (pierwszy kawałek albo brak aktywnego).
// Kawałek od zera zamyka poprzedni transfer tego pliku jako przerwany (wgrywanie zaczęte od nowa).
func (s *serwer) zacznijKawalek(r *nethttp.Request, u rdzen.Uzytkownik, udzial, sciezka string,
	p parametryWgrania) (string, *aktywneWgranie) {
	teraz := s.teraz()
	s.sprzatajWgrania(teraz, false)
	k := kluczWgrania(u.Login, udzial, sciezka)
	s.wgrania.mu.Lock()
	defer s.wgrania.mu.Unlock()
	if s.wgrania.aktywne == nil {
		s.wgrania.aktywne = map[string]*aktywneWgranie{}
	}
	a := s.wgrania.aktywne[k]
	if a != nil && p.przesuniecie == 0 {
		delete(s.wgrania.aktywne, k)
		a.t.Zakoncz(transfery.WynikPrzerwane)
		a = nil
	}
	if a == nil {
		a = &aktywneWgranie{t: s.transfery.Rozpocznij(transfery.Stan{
			Kto: u.Login, IP: adresIP(r), Protokol: protokol(r), Kierunek: transfery.KierunekWgranie,
			Udzial: udzial, Sciezka: sciezka, Rozmiar: rozmiarDoWyslania(p.calkowity, p.przesuniecie),
		})}
		s.wgrania.aktywne[k] = a
		s.uruchomSprzatacz()
	}
	a.wTrakcie++
	a.ostatnio = teraz
	return k, a
}

// koniecKawalka: wynik "" = wgranie trwa dalej; inny wynik kończy transfer (callback rejestru pisze dziennik).
func (s *serwer) koniecKawalka(k string, a *aktywneWgranie, wynik string) {
	s.wgrania.mu.Lock()
	defer s.wgrania.mu.Unlock()
	a.wTrakcie--
	a.ostatnio = s.teraz()
	if wynik == "" {
		return
	}
	if s.wgrania.aktywne[k] == a {
		delete(s.wgrania.aktywne, k)
	}
	a.t.Zakoncz(wynik)
}

// wynikBleduWgrania: przerwanie przez admina albo rozłączenie = przerwane, reszta = blad.
func wynikBleduWgrania(r *nethttp.Request, err error) string {
	if errors.Is(err, transfery.ErrPrzerwany) || r.Context().Err() != nil {
		return transfery.WynikPrzerwane
	}
	return transfery.WynikBlad
}

// sprzatajWgrania kończy jako przerwane wgrania bez kawałka od CzasBezczynnosciWgrania; zwraca, ile
// zostało. zSprzatacza: przy zerze gorutyna sprzątająca kończy pracę (flaga zdejmowana pod blokadą).
func (s *serwer) sprzatajWgrania(teraz time.Time, zSprzatacza bool) int {
	s.wgrania.mu.Lock()
	defer s.wgrania.mu.Unlock()
	for k, a := range s.wgrania.aktywne {
		if a.wTrakcie == 0 && teraz.Sub(a.ostatnio) >= konfiguracja.CzasBezczynnosciWgrania {
			delete(s.wgrania.aktywne, k)
			a.t.Zakoncz(transfery.WynikPrzerwane)
		}
	}
	n := len(s.wgrania.aktywne)
	if zSprzatacza && n == 0 {
		s.wgrania.sprzatacz = false
	}
	return n
}

// uruchomSprzatacz startuje gorutynę z tykaczem, jeśli nie działa (wołający trzyma s.wgrania.mu).
func (s *serwer) uruchomSprzatacz() {
	if s.wgrania.sprzatacz {
		return
	}
	s.wgrania.sprzatacz = true
	takty, stop := s.tykacz(konfiguracja.OkresSprzataniaWgran)
	go func() {
		defer stop()
		for range takty {
			if s.sprzatajWgrania(s.teraz(), true) == 0 {
				return
			}
		}
	}()
}
