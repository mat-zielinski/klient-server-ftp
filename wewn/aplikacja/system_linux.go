//go:build linux

package aplikacja

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"

	"skrytka/wewn/konfiguracja"
)

// identyfikatorInstancji: plik blokady w katalogu danych.
func identyfikatorInstancji() string {
	return filepath.Join(konfiguracja.KatalogDanych(), konfiguracja.PlikBlokadyInstancji)
}

// jednaInstancja zakłada blokadę flock na pliku; zajęta = errDrugaInstancja. Blokada znika z końcem
// procesu (także po awarii), plik zostaje.
func jednaInstancja(plik string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(plik), konfiguracja.UprawnieniaKatalogu); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(plik, os.O_RDWR|os.O_CREATE, konfiguracja.UprawnieniaPlikuDanych)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close() // celowo: plik tylko do blokady, błąd zamknięcia bez skutków
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, errDrugaInstancja
		}
		return nil, err
	}
	return func() {
		_ = f.Close() // celowo: zamknięcie zwalnia blokadę, koniec procesu i tak ją zwalnia
	}, nil
}

// pokazKomunikat: na standardowe wyjście błędów (pod systemd trafia do journalctl).
func pokazKomunikat(tekst string) {
	_, _ = os.Stderr.WriteString(tekst + "\n") // celowo: brak innego kanału komunikatu
}

// otworzPrzegladarke: xdg-open, gdy jest ekran i program; inaczej pomijamy bez błędu.
func otworzPrzegladarke(adres string) error {
	if os.Getenv(konfiguracja.ZmiennaEkranuX11) == "" && os.Getenv(konfiguracja.ZmiennaEkranuWayland) == "" {
		return nil
	}
	program, err := exec.LookPath(konfiguracja.ProgramPrzegladarkiLinux)
	if err != nil {
		return nil // celowo: brak xdg-open (serwer bez pulpitu) to nie błąd
	}
	polecenie := exec.Command(program, adres)
	if err := polecenie.Start(); err != nil {
		return err
	}
	go func() {
		_ = polecenie.Wait() // celowo: tylko zbieramy proces, wynik bez znaczenia
	}()
	return nil
}
