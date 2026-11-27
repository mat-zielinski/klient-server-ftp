package monitor

// Parsery plików /proc (Linux). Bez znacznika systemu, żeby testy szły na każdym systemie.

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// Liczba pól taktów w wierszu „cpu” liczonych do sumy: user nice system idle iowait irq softirq steal
// (guest i guest_nice są już wliczone w user i nice).
const (
	polaTaktowCPU    = 8
	poleIdle         = 3
	poleIowait       = 4
	polaSieciOdbior  = 8 // rx: bytes packets errs drop fifo frame compressed multicast
	minPolaSieci     = 9
	minPolaDyskow    = 10
	poleSektoryOdcz  = 5
	poleSektoryZapis = 9
	poleNazwyDysku   = 2
	naglowkiSieci    = 2
)

var errBrakDanych = errors.New("Brak oczekiwanych danych w pliku systemowym.")

// parsujStat czyta takty procesora z pierwszego wiersza „cpu ” w /proc/stat.
func parsujStat(r io.Reader) (razem, bezczynny uint64, err error) {
	sk := bufio.NewScanner(r)
	for sk.Scan() {
		pola := strings.Fields(sk.Text())
		if len(pola) < 2 || pola[0] != "cpu" {
			continue
		}
		for i, tekst := range pola[1:] {
			if i >= polaTaktowCPU {
				break
			}
			v, err := strconv.ParseUint(tekst, 10, 64)
			if err != nil {
				return 0, 0, err
			}
			razem += v
			if i == poleIdle || i == poleIowait {
				bezczynny += v
			}
		}
		return razem, bezczynny, nil
	}
	return 0, 0, errBrakDanych
}

// parsujMeminfo czyta MemTotal i MemAvailable (kB) z /proc/meminfo; zwraca bajty.
func parsujMeminfo(r io.Reader) (uzyte, razem uint64, err error) {
	var dostepne uint64
	maRazem, maDostepne := false, false
	sk := bufio.NewScanner(r)
	for sk.Scan() {
		pola := strings.Fields(sk.Text())
		if len(pola) < 2 {
			continue
		}
		v, err := strconv.ParseUint(pola[1], 10, 64)
		if err != nil {
			continue // celowo: wiersz w nieznanym formacie pomijamy
		}
		switch pola[0] {
		case "MemTotal:":
			razem, maRazem = v*konfiguracja.MonitorBajtyKB, true
		case "MemAvailable:":
			dostepne, maDostepne = v*konfiguracja.MonitorBajtyKB, true
		}
	}
	if !maRazem || !maDostepne {
		return 0, 0, errBrakDanych
	}
	if dostepne > razem {
		dostepne = razem
	}
	return razem - dostepne, razem, nil
}

// parsujNetDev czyta bajty odebrane i wysłane interfejsów z /proc/net/dev.
func parsujNetDev(r io.Reader) ([]LicznikSieci, error) {
	wynik := []LicznikSieci{}
	sk := bufio.NewScanner(r)
	for nr := 0; sk.Scan(); nr++ {
		if nr < naglowkiSieci {
			continue
		}
		nazwa, reszta, ok := strings.Cut(sk.Text(), ":")
		if !ok {
			continue
		}
		pola := strings.Fields(reszta)
		if len(pola) < minPolaSieci {
			continue
		}
		odb, err1 := strconv.ParseUint(pola[0], 10, 64)
		wys, err2 := strconv.ParseUint(pola[polaSieciOdbior], 10, 64)
		if err1 != nil || err2 != nil {
			continue // celowo: uszkodzony wiersz pomijamy
		}
		n := strings.TrimSpace(nazwa)
		wynik = append(wynik, LicznikSieci{
			Interfejs: n,
			Petla:     n == konfiguracja.MonitorNazwaPetliLinux,
			Wyslane:   wys,
			Odebrane:  odb,
		})
	}
	return wynik, sk.Err()
}

// parsujDiskstats sumuje bajty odczytane i zapisane dysków, dla których czyFizyczny zwraca true
// (partycje i urządzenia wirtualne pomijamy, żeby nie liczyć tych samych bajtów dwa razy).
func parsujDiskstats(r io.Reader, czyFizyczny func(nazwa string) bool) (odczyt, zapis uint64, err error) {
	sk := bufio.NewScanner(r)
	for sk.Scan() {
		pola := strings.Fields(sk.Text())
		if len(pola) < minPolaDyskow || !czyFizyczny(pola[poleNazwyDysku]) {
			continue
		}
		o, err1 := strconv.ParseUint(pola[poleSektoryOdcz], 10, 64)
		z, err2 := strconv.ParseUint(pola[poleSektoryZapis], 10, 64)
		if err1 != nil || err2 != nil {
			continue // celowo: uszkodzony wiersz pomijamy
		}
		odczyt += o * konfiguracja.MonitorBajtySektora
		zapis += z * konfiguracja.MonitorBajtySektora
	}
	return odczyt, zapis, sk.Err()
}
