//go:build linux

package monitor

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"skrytka/wewn/konfiguracja"
)

// zrodloLinux czyta /proc i /sys.
type zrodloLinux struct{}

// NoweZrodloSystemowe zwraca źródło odczytów prawdziwego systemu.
func NoweZrodloSystemowe() Zrodlo { return zrodloLinux{} }

// Odczytaj: CPU i RAM są wymagane (błąd = brak próbki systemowej); dysk i sieć najlepiej jak się da.
func (zrodloLinux) Odczytaj() (Odczyt, error) {
	var o Odczyt
	if err := czytajPlik(konfiguracja.MonitorPlikStatystykCPU, func(f *os.File) (err error) {
		o.CPURazem, o.CPUBezczynny, err = parsujStat(f)
		return err
	}); err != nil {
		return Odczyt{}, err
	}
	if err := czytajPlik(konfiguracja.MonitorPlikPamieci, func(f *os.File) (err error) {
		o.RAMUzyte, o.RAMRazem, err = parsujMeminfo(f)
		return err
	}); err != nil {
		return Odczyt{}, err
	}
	errDysk := czytajPlik(konfiguracja.MonitorPlikDyskow, func(f *os.File) (err error) {
		o.DyskOdczyt, o.DyskZapis, err = parsujDiskstats(f, czyDyskFizyczny)
		return err
	})
	o.DyskDostepny = errDysk == nil
	o.Siec = sieciLinux()
	return o, nil
}

func czytajPlik(sciezka string, f func(*os.File) error) error {
	plik, err := os.Open(sciezka)
	if err != nil {
		return err
	}
	defer plik.Close() // celowo: plik tylko do odczytu, błąd zamknięcia bez znaczenia
	return f(plik)
}

// czyDyskFizyczny: /sys/block/<nazwa>/device istnieje tylko dla dysków (nie partycji, loop, ram, dm).
func czyDyskFizyczny(nazwa string) bool {
	n := strings.ReplaceAll(nazwa, "/", "!")
	_, err := os.Stat(filepath.Join(konfiguracja.MonitorKatalogUrzadzenBlokowych, n, konfiguracja.MonitorPodkatalogUrzadzenia))
	return err == nil
}

// sieciLinux: liczniki z /proc/net/dev, tylko interfejsy w górze (gdy net.Interfaces działa).
func sieciLinux() []LicznikSieci {
	var lista []LicznikSieci
	if err := czytajPlik(konfiguracja.MonitorPlikSieci, func(f *os.File) (err error) {
		lista, err = parsujNetDev(f)
		return err
	}); err != nil {
		return []LicznikSieci{}
	}
	interfejsy, err := net.Interfaces()
	if err != nil {
		return lista // celowo: bez flag interfejsów pokazujemy wszystkie
	}
	flagi := make(map[string]net.Flags, len(interfejsy))
	for _, inf := range interfejsy {
		flagi[inf.Name] = inf.Flags
	}
	wynik := make([]LicznikSieci, 0, len(lista))
	for _, l := range lista {
		f, ok := flagi[l.Interfejs]
		if !ok || f&net.FlagUp == 0 {
			continue
		}
		l.Petla = l.Petla || f&net.FlagLoopback != 0
		wynik = append(wynik, l)
	}
	return wynik
}

// ZajetoscDysku: bajty razem i wolne (dla procesu) w systemie plików zawierającym ścieżkę.
func ZajetoscDysku(sciezka string) (Zajetosc, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(sciezka, &s); err != nil {
		return Zajetosc{}, err
	}
	blok := uint64(s.Bsize) //nolint:unconvert // typ Bsize zależy od architektury
	return Zajetosc{Razem: uint64(s.Blocks) * blok, Wolne: uint64(s.Bavail) * blok}, nil
}
