package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// wynikTestow to wynik jednego `go test`.
type wynikTestow struct {
	Zielone   bool   // testy przeszły
	Kompiluje bool   // false = kod po mutacji się nie buduje (mutacja bez znaczenia)
	Wyjscie   string // ostatnie linie wyjścia go test
}

// uruchamiaczTestow uruchamia testy pakietu (podmieniany w testach).
type uruchamiaczTestow func(pakiet, wzorzec string) (wynikTestow, error)

// straz pilnuje przywrócenia pliku: po zwykłym końcu, po błędzie i po Ctrl+C.
type straz struct {
	mu        sync.Mutex
	sciezka   string
	oryginal  []byte
	zmutowany []byte
}

func (s *straz) ustaw(sciezka string, oryginal, zmutowany []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sciezka, s.oryginal, s.zmutowany = sciezka, oryginal, zmutowany
}

// przywroc zapisuje oryginał (jeśli jakiś plik jest zmutowany) i usuwa kopię awaryjną. Gdy ktoś zmienił plik
// w trakcie mutacji, niczego nie nadpisuje: oryginał zostaje obok jako *.mutacja_konflikt.
func (s *straz) przywroc() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sciezka == "" {
		return nil
	}
	sciezka := s.sciezka
	s.sciezka = ""
	if teraz, err := os.ReadFile(sciezka); err == nil && !bytes.Equal(teraz, s.zmutowany) {
		if err := os.Rename(sciezka+RozszerzenieKopii, sciezka+RozszerzenieKonfliktu); err != nil {
			return err
		}
		return fmt.Errorf("%s zmieniony w trakcie mutacji (edycja z zewnątrz): nie nadpisuję, oryginał sprzed mutacji w %s",
			sciezka, sciezka+RozszerzenieKonfliktu)
	}
	if err := zapiszBezpiecznie(sciezka, s.oryginal); err != nil {
		return fmt.Errorf("NIE PRZYWRÓCONO %s (kopia: %s): %w", sciezka, sciezka+RozszerzenieKopii, err)
	}
	if err := os.Remove(sciezka + RozszerzenieKopii); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// zapiszBezpiecznie: najpierw plik tymczasowy, potem zamiana (bez ryzyka pustego pliku).
func zapiszBezpiecznie(sciezka string, dane []byte) error {
	tymczasowy := sciezka + RozszerzenieTymczasowe
	if err := os.WriteFile(tymczasowy, dane, 0o644); err != nil {
		return err
	}
	return os.Rename(tymczasowy, sciezka)
}

// przywrocPozostaleKopie: kopia awaryjna po przerwanym przebiegu wraca na miejsce przed czymkolwiek innym.
func przywrocPozostaleKopie(korzen string, wpisy []Wpis, wyjscie io.Writer) error {
	for _, w := range wpisy {
		sciezka := filepath.Join(korzen, filepath.FromSlash(w.Plik))
		kopia, err := os.ReadFile(sciezka + RozszerzenieKopii)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if err := zapiszBezpiecznie(sciezka, kopia); err != nil {
			return err
		}
		if err := os.Remove(sciezka + RozszerzenieKopii); err != nil {
			return err
		}
		fmt.Fprintf(wyjscie, "UWAGA: przywrócono %s z kopii po przerwanym przebiegu\n", w.Plik)
	}
	return nil
}

// sprawdzMutacje: podmiana, testy, przywrócenie (zawsze). Zwraca wynik testów po mutacji.
func sprawdzMutacje(korzen string, w Wpis, uruchom uruchamiaczTestow, s *straz) (wynik wynikTestow, err error) {
	sciezka := filepath.Join(korzen, filepath.FromSlash(w.Plik))
	oryginal, err := os.ReadFile(sciezka)
	if err != nil {
		return wynik, err
	}
	zmutowany, err := mutuj(string(oryginal), w.Szukaj, w.Zamien)
	if err != nil {
		return wynik, err
	}
	if err := os.WriteFile(sciezka+RozszerzenieKopii, oryginal, 0o644); err != nil {
		return wynik, err
	}
	s.ustaw(sciezka, oryginal, []byte(zmutowany))
	defer func() {
		if blad := s.przywroc(); blad != nil {
			err = errors.Join(err, blad)
		}
	}()
	if err := zapiszBezpiecznie(sciezka, []byte(zmutowany)); err != nil {
		return wynik, err
	}
	return uruchom(w.Pakiet, w.Testy)
}

// przebieg to całe sprawdzenie listy; zwraca kod wyjścia.
func przebieg(korzen string, wpisy []Wpis, uruchom uruchamiaczTestow, s *straz, wyjscie io.Writer) int {
	var gotowe []Wpis
	for _, w := range wpisy {
		if w.DoUzupelnienia != "" {
			fmt.Fprintf(wyjscie, "UWAGA: %s do uzupełnienia: %s\n", w.ID, w.DoUzupelnienia)
			continue
		}
		gotowe = append(gotowe, w)
	}
	if err := przywrocPozostaleKopie(korzen, gotowe, wyjscie); err != nil {
		fmt.Fprintln(wyjscie, "MUTACJE: błąd przywracania kopii:", err)
		return KodBledu
	}
	if bledy := sprawdzAktualnosc(korzen, gotowe); len(bledy) > 0 {
		fmt.Fprintf(wyjscie, "MUTACJE: lista nieaktualna (%s):\n  %s\n", PlikListy, strings.Join(bledy, "\n  "))
		return KodBledu
	}
	if kod := sprawdzBezMutacji(gotowe, uruchom, wyjscie); kod != KodOK {
		return kod
	}
	return sprawdzWszystkie(korzen, gotowe, uruchom, s, wyjscie)
}

// sprawdzBezMutacji: testy bez mutacji muszą przejść, inaczej czerwony wynik mutacji nic nie znaczy.
func sprawdzBezMutacji(wpisy []Wpis, uruchom uruchamiaczTestow, wyjscie io.Writer) int {
	widziane := map[string]bool{}
	for _, w := range wpisy {
		klucz := w.Pakiet + "\x00" + w.Testy
		if widziane[klucz] {
			continue
		}
		widziane[klucz] = true
		wynik, err := uruchom(w.Pakiet, w.Testy)
		if err != nil || !wynik.Zielone {
			fmt.Fprintf(wyjscie, "MUTACJE: testy %s (%s) nie przechodzą BEZ mutacji, popraw je najpierw: %v\n%s\n",
				w.Pakiet, opisWzorca(w.Testy), err, wynik.Wyjscie)
			return KodBledu
		}
	}
	return KodOK
}

func sprawdzWszystkie(korzen string, wpisy []Wpis, uruchom uruchamiaczTestow, s *straz, wyjscie io.Writer) int {
	zielone, bledy := 0, 0
	for _, w := range wpisy {
		wynik, err := sprawdzMutacje(korzen, w, uruchom, s)
		switch {
		case err != nil:
			bledy++
			fmt.Fprintf(wyjscie, "BŁĄD     %s: %v\n", w.ID, err)
		case !wynik.Kompiluje:
			bledy++
			fmt.Fprintf(wyjscie, "BŁĄD     %s: kod po mutacji się nie buduje (popraw „zamien”):\n%s\n", w.ID, wynik.Wyjscie)
		case wynik.Zielone:
			zielone++
			fmt.Fprintf(wyjscie, "ZIELONE  %s: testy przechodzą BEZ blokady (%s): %s\n", w.ID, w.Opis, w.Pakiet)
		default:
			fmt.Fprintf(wyjscie, "OK       %s: bez blokady testy padają\n", w.ID)
		}
	}
	switch {
	case bledy > 0:
		fmt.Fprintf(wyjscie, "MUTACJE: błędy uruchomienia %d, fałszywie zielone %d\n", bledy, zielone)
		return KodBledu
	case zielone > 0:
		fmt.Fprintf(wyjscie, "MUTACJE: FAŁSZYWIE ZIELONE %d z %d: dopisz test, który pada bez blokady\n", zielone, len(wpisy))
		return KodZielone
	}
	fmt.Fprintf(wyjscie, "MUTACJE: OK (%d z %d wywraca swoje testy)\n", len(wpisy), len(wpisy))
	return KodOK
}

func opisWzorca(wzorzec string) string {
	if wzorzec == "" {
		return "wszystkie testy"
	}
	return "-run " + wzorzec
}
