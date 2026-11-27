//go:build windows

package aplikacja

import (
	"errors"
	"os"
	"os/exec"

	"golang.org/x/sys/windows"

	"skrytka/wewn/konfiguracja"
)

// identyfikatorInstancji: nazwa mutexu Windows.
func identyfikatorInstancji() string { return konfiguracja.NazwaMuteksuInstancji }

// jednaInstancja zakłada nazwany mutex Windows; istnieje (albo należy do usługi i brak dostępu) =
// errDrugaInstancja. Zwraca funkcję zwalniającą.
func jednaInstancja(nazwa string) (func(), error) {
	n, err := windows.UTF16PtrFromString(nazwa)
	if err != nil {
		return nil, err
	}
	uchwyt, err := windows.CreateMutex(nil, false, n)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		if uchwyt != 0 {
			_ = windows.CloseHandle(uchwyt) // celowo: uchwyt cudzego mutexu, nic do zrobienia przy błędzie
		}
		return nil, errDrugaInstancja
	}
	if err != nil {
		return nil, err
	}
	return func() {
		_ = windows.CloseHandle(uchwyt) // celowo: koniec procesu i tak zwalnia uchwyt
	}, nil
}

// pokazKomunikat: na standardowe wyjście błędów (wersja konsolowa), a gdy go nie ma (wersja okienkowa)
// w okienku Windows.
func pokazKomunikat(tekst string) {
	if os.Stderr != nil {
		if _, err := os.Stderr.WriteString(tekst + "\r\n"); err == nil {
			return
		}
	}
	t, err1 := windows.UTF16PtrFromString(tekst)
	tytul, err2 := windows.UTF16PtrFromString(konfiguracja.NazwaAplikacji)
	if err1 != nil || err2 != nil {
		return // celowo: tekst z bajtem zerowym nie do pokazania, nie ma innego kanału
	}
	_, _ = windows.MessageBox(0, t, tytul, windows.MB_OK|windows.MB_ICONINFORMATION) // celowo: tylko informacja
}

// otworzPrzegladarke otwiera adres w domyślnej przeglądarce systemu.
func otworzPrzegladarke(adres string) error {
	polecenie := exec.Command(konfiguracja.ProgramPrzegladarki, konfiguracja.ArgumentPrzegladarki, adres)
	if err := polecenie.Start(); err != nil {
		return err
	}
	go func() {
		_ = polecenie.Wait() // celowo: tylko zbieramy proces, wynik bez znaczenia
	}()
	return nil
}
