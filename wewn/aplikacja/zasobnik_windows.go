//go:build windows

package aplikacja

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"skrytka/wewn/aplikacja/ikony"
	"skrytka/wewn/konfiguracja"
)

// etykietaAutostartu: pozycja menu z autostartem.
const etykietaAutostartu = konfiguracja.ZasobnikMenuAutostart

// ikonyZasobnika: ICO (Windows czyta ICO); ciemne na jasny pasek zadań.
var ikonyZasobnika = zestawIkon{
	jasna: ikony.Zasobnik, jasnaAktywna: ikony.ZasobnikAktywny,
	ciemna: ikony.ZasobnikCiemny, ciemnaAktywna: ikony.ZasobnikCiemnyAktywny,
}

// ikonaZasobnika: zwykła albo ze znacznikiem transferu, wg jasności paska zadań.
func ikonaZasobnika(aktywna, jasnyPasek bool) []byte {
	return wybierzIkone(ikonyZasobnika, aktywna, jasnyPasek)
}

// czyJasnyPasek: HKCU\...\Themes\Personalize SystemUsesLightTheme = 1 (brak wpisu = ciemny, jak Windows 10).
func czyJasnyPasek() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, konfiguracja.KluczMotywuWindows, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	w, _, err := k.GetIntegerValue(konfiguracja.WartoscJasnegoPaska)
	return err == nil && w == 1
}

// czyAutostart: w HKCU\...\Run jest wpis wskazujący ten program.
func czyAutostart() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, konfiguracja.KluczAutostartuWindows, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	w, _, err := k.GetStringValue(konfiguracja.NazwaWpisuAutostartu)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	return czyNaszAutostart(w, exe), nil
}

// ustawAutostart zapisuje albo usuwa wpis w HKCU\...\Run (bez uprawnień administratora).
func ustawAutostart(wlacz bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, konfiguracja.KluczAutostartuWindows, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !wlacz {
		if err := k.DeleteValue(konfiguracja.NazwaWpisuAutostartu); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(konfiguracja.NazwaWpisuAutostartu, wartoscAutostartu(exe))
}

// kopiujDoSchowka: clip.exe ze standardowego wejścia, bez okna konsoli (adres to sam ASCII).
func kopiujDoSchowka(tekst string) error {
	polecenie := exec.Command(konfiguracja.ProgramSchowkaWindows)
	polecenie.Stdin = strings.NewReader(tekst)
	polecenie.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return polecenie.Run()
}

// otworzFolder: Eksplorator Windows (kod wyjścia explorer.exe bywa 1 także przy sukcesie, więc tylko Start).
func otworzFolder(sciezka string) error {
	polecenie := exec.Command(konfiguracja.ProgramEksploratora, sciezka)
	if err := polecenie.Start(); err != nil {
		return err
	}
	go func() {
		_ = polecenie.Wait() // celowo: tylko zbieramy proces, wynik bez znaczenia
	}()
	return nil
}
