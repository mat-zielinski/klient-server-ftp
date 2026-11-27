//go:build linux

package aplikacja

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// czyUsluga: pod systemd program działa jak zwykły proces (SIGTERM = łagodne zamknięcie).
func czyUsluga() bool { return false }

// uruchomJakoUsluge nie jest używane na Linuksie (czyUsluga zawsze false).
func uruchomJakoUsluge() int { return konfiguracja.KodBlad }

// errNieRoot: polecenia usługi wymagają roota; opis mówi, jak to zrobić ręcznie.
var errNieRoot = errors.New("Sterowanie usługą wymaga uprawnień roota. Uruchom: sudo ./skrytka --usluga " +
	"instaluj (albo skrypt skrypty/instaluj-linux.sh, który zakłada też użytkownika skrytka).")

// sterujUsluga: instaluj | usun | start | stop przez systemd. Zwraca opis wyniku.
func sterujUsluga(polecenie string) (string, error) {
	if os.Geteuid() != 0 {
		return "", errNieRoot
	}
	switch polecenie {
	case konfiguracja.PolecenieInstaluj:
		return instalujUslugeSystemd()
	case konfiguracja.PolecenieUsun:
		if err := systemctl("disable", "--now", konfiguracja.JednostkaSystemd); err != nil {
			return "", err
		}
		if err := os.Remove(konfiguracja.PlikUslugiSystemd); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("Nie można usunąć pliku usługi: %w", err)
		}
		if err := systemctl("daemon-reload"); err != nil {
			return "", err
		}
		return "Usługa usunięta.", nil
	case konfiguracja.PolecenieStart:
		return "Usługa uruchomiona.", systemctl("start", konfiguracja.JednostkaSystemd)
	default:
		return "Usługa zatrzymana.", systemctl("stop", konfiguracja.JednostkaSystemd)
	}
}

// instalujUslugeSystemd zapisuje plik jednostki, przeładowuje systemd i włącza start z systemem.
func instalujUslugeSystemd() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	uzytkownik := ""
	if _, err := user.Lookup(konfiguracja.UzytkownikUslugiLinux); err == nil {
		uzytkownik = konfiguracja.UzytkownikUslugiLinux
	}
	tresc := plikUslugiSystemd(exe, uzytkownik)
	if err := os.WriteFile(konfiguracja.PlikUslugiSystemd, []byte(tresc), konfiguracja.UprawnieniaPlikuUslugi); err != nil {
		return "", fmt.Errorf("Nie można zapisać %s: %w", konfiguracja.PlikUslugiSystemd, err)
	}
	if err := systemctl("daemon-reload"); err != nil {
		return "", err
	}
	if err := systemctl("enable", konfiguracja.JednostkaSystemd); err != nil {
		return "", err
	}
	opis := "Usługa zainstalowana (start z systemem). Uruchom: sudo systemctl start skrytka"
	if uzytkownik == "" {
		opis += ". Uwaga: brak użytkownika skrytka, usługa działa jako root."
	}
	return opis, nil
}

// systemctl woła systemctl z argumentami; błąd zawiera wyjście programu.
func systemctl(argumenty ...string) error {
	wyjscie, err := exec.Command(konfiguracja.ProgramSystemctl, argumenty...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w: %s", strings.Join(argumenty, " "), err, strings.TrimSpace(string(wyjscie)))
	}
	return nil
}
