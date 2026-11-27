package dziennik

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Czytaj zwraca wpisy od najnowszych wg filtru.
func (d *Dziennik) Czytaj(f Filtr) ([]Wpis, error) {
	limit := f.Limit
	if limit <= 0 || limit > konfiguracja.DziennikLimitMaks {
		limit = konfiguracja.DziennikLimitMaks
	}
	szukaj := strings.ToLower(f.Szukaj)
	wynik := []Wpis{}
	err := d.przegladaj(f.Od, f.Do, func(w Wpis) bool {
		if pasuje(w, f, szukaj) {
			wynik = append(wynik, w)
		}
		return len(wynik) < limit
	})
	return wynik, err
}

// pasuje sprawdza pola filtru poza zakresem czasu (szukaj już małymi literami).
func pasuje(w Wpis, f Filtr, szukaj string) bool {
	if f.Rodzaj != "" && w.Rodzaj != f.Rodzaj {
		return false
	}
	if f.Kto != "" && w.Kto != f.Kto {
		return false
	}
	if szukaj == "" {
		return true
	}
	return strings.Contains(strings.ToLower(w.Sciezka), szukaj) ||
		strings.Contains(strings.ToLower(w.Opis), szukaj)
}

// wZakresie: od włącznie, do wyłącznie, puste = bez granicy.
func wZakresie(t, od, do time.Time) bool {
	return (od.IsZero() || !t.Before(od)) && (do.IsZero() || t.Before(do))
}

// przegladaj podaje wpisy z zakresu od najnowszych; fn zwraca false = koniec.
func (d *Dziennik) przegladaj(od, do time.Time, fn func(Wpis) bool) error {
	d.oproznij()
	klucze, err := d.miesiaceWZakresie(od, do)
	if err != nil {
		return err
	}
	biezacy := kluczMiesiaca(d.teraz())
	for _, k := range klucze {
		wpisy, err := d.wpisyMiesiaca(k, k < biezacy)
		if err != nil {
			return err
		}
		for _, w := range wpisy {
			if wZakresie(w.Czas, od, do) && !fn(w) {
				return nil
			}
		}
	}
	return nil
}

// miesiaceWZakresie: klucze RRRR-MM istniejących plików, które mogą mieć wpisy z zakresu, od najnowszego.
func (d *Dziennik) miesiaceWZakresie(od, do time.Time) ([]string, error) {
	pozycje, err := os.ReadDir(d.katalog)
	if err != nil {
		return nil, err
	}
	var klucze []string
	for _, p := range pozycje {
		k, ok := kluczZNazwy(p.Name())
		if !ok || p.IsDir() {
			continue
		}
		if miesiacPrzecinaZakres(k, od, do) {
			klucze = append(klucze, k)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(klucze)))
	return klucze, nil
}

// kluczZNazwy: "dziennik_2026-11.jsonl" → "2026-11"; obca nazwa → false.
func kluczZNazwy(nazwa string) (string, bool) {
	if !strings.HasPrefix(nazwa, konfiguracja.DziennikPrefiks) ||
		!strings.HasSuffix(nazwa, konfiguracja.DziennikRozszerzenie) {
		return "", false
	}
	k := strings.TrimSuffix(strings.TrimPrefix(nazwa, konfiguracja.DziennikPrefiks), konfiguracja.DziennikRozszerzenie)
	t, err := time.ParseInLocation(konfiguracja.DziennikFormatMiesiaca, k, time.Local)
	if err != nil || t.Format(konfiguracja.DziennikFormatMiesiaca) != k {
		return "", false
	}
	return k, true
}

func miesiacPrzecinaZakres(klucz string, od, do time.Time) bool {
	poczatek, err := time.ParseInLocation(konfiguracja.DziennikFormatMiesiaca, klucz, time.Local)
	if err != nil {
		return false
	}
	koniec := poczatek.AddDate(0, 1, 0)
	return (od.IsZero() || koniec.After(od)) && (do.IsZero() || poczatek.Before(do))
}

// wpisyMiesiaca: wpisy pliku posortowane od najnowszych; zamknięte miesiące z pamięci podręcznej.
func (d *Dziennik) wpisyMiesiaca(klucz string, zamkniety bool) ([]Wpis, error) {
	sciezka := filepath.Join(d.katalog, nazwaPliku(klucz))
	info, err := os.Stat(sciezka)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if zamkniety {
		if w, ok := d.pamiec.pobierz(klucz, info); ok {
			return w, nil
		}
	}
	wpisy, err := wczytajPlik(sciezka)
	if err != nil {
		return nil, err
	}
	if zamkniety {
		d.pamiec.wstaw(klucz, info, wpisy)
	}
	return wpisy, nil
}

// wczytajPlik czyta linie JSON; uszkodzone linie są pomijane. Wynik od najnowszych.
func wczytajPlik(sciezka string) ([]Wpis, error) {
	f, err := os.Open(sciezka)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close() // celowo: plik tylko do odczytu, błąd zamknięcia nie zmienia wyniku
	r := bufio.NewReader(f)
	var wpisy []Wpis
	for {
		linia, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(linia)) > 0 {
			var w Wpis
			if json.Unmarshal(linia, &w) == nil { // celowo: uszkodzona linia (np. urwany zapis) jest pomijana
				wpisy = append(wpisy, w)
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	// Odwrócenie + stabilne sortowanie: przy równym czasie później zapisany wpis jest wyżej.
	for i, j := 0, len(wpisy)-1; i < j; i, j = i+1, j-1 {
		wpisy[i], wpisy[j] = wpisy[j], wpisy[i]
	}
	sort.SliceStable(wpisy, func(i, j int) bool { return wpisy[i].Czas.After(wpisy[j].Czas) })
	return wpisy, nil
}

// pamiecMiesiecy: wpisy zamkniętych miesięcy; ważne, póki plik ma ten sam rozmiar i czas zmiany.
type pamiecMiesiecy struct {
	mu        sync.Mutex
	maks      int
	pozycje   map[string]pozycjaPamieci
	kolejnosc []string // od najstarszej wstawionej
}

type pozycjaPamieci struct {
	rozmiar int64
	zmiana  time.Time
	wpisy   []Wpis
}

func nowaPamiec(maks int) pamiecMiesiecy {
	return pamiecMiesiecy{maks: maks, pozycje: map[string]pozycjaPamieci{}}
}

func (p *pamiecMiesiecy) pobierz(klucz string, info os.FileInfo) ([]Wpis, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	poz, ok := p.pozycje[klucz]
	if !ok || poz.rozmiar != info.Size() || !poz.zmiana.Equal(info.ModTime()) {
		return nil, false
	}
	return poz.wpisy, true
}

func (p *pamiecMiesiecy) wstaw(klucz string, info os.FileInfo, wpisy []Wpis) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, jest := p.pozycje[klucz]; !jest {
		p.kolejnosc = append(p.kolejnosc, klucz)
	}
	p.pozycje[klucz] = pozycjaPamieci{rozmiar: info.Size(), zmiana: info.ModTime(), wpisy: wpisy}
	for len(p.kolejnosc) > p.maks {
		delete(p.pozycje, p.kolejnosc[0])
		p.kolejnosc = p.kolejnosc[1:]
	}
}
