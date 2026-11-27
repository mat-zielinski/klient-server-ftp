//go:build linux

package aplikacja

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"skrytka/wewn/aplikacja/ikony"
	"skrytka/wewn/konfiguracja"
)

// etykietaAutostartu: pozycja menu z autostartem.
const etykietaAutostartu = konfiguracja.ZasobnikMenuAutostartLin

// errBrakSchowka: brak wl-copy (Wayland) i xclip (X11).
var errBrakSchowka = errors.New("Brak programu do schowka: zainstaluj wl-clipboard (Wayland) albo xclip (X11).")

// czyJasnyPasek: na Linuksie zawsze jasne ikony (motywu panelu nie da się pewnie odczytać).
func czyJasnyPasek() bool { return false }

// ikonaZasobnika: StatusNotifier czyta PNG, więc bierzemy PNG z wnętrza ICO; gdy go nie ma, ikonę aplikacji.
func ikonaZasobnika(aktywna, _ bool) []byte {
	ico := ikony.Zasobnik
	if aktywna {
		ico = ikony.ZasobnikAktywny
	}
	if png := pngZIco(ico); png != nil {
		return png
	}
	return ikony.Aplikacja
}

// plikAutostartu: ~/.config/autostart/skrytka.desktop.
func plikAutostartu() (string, error) {
	k, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(k, konfiguracja.KatalogAutostartuLinux, konfiguracja.PlikAutostartuLinux), nil
}

// czyAutostart: plik .desktop istnieje.
func czyAutostart() (bool, error) {
	p, err := plikAutostartu()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// ustawAutostart zapisuje albo usuwa plik .desktop w ~/.config/autostart.
func ustawAutostart(wlacz bool) error {
	p, err := plikAutostartu()
	if err != nil {
		return err
	}
	if !wlacz {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), konfiguracja.UprawnieniaKatalogu); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(trescAutostartuLinux(exe)), konfiguracja.UprawnieniaPlikuAutostartu)
}

// kopiujDoSchowka: wl-copy (Wayland) albo xclip (X11); oba zostają w tle, póki schowek jest ich.
func kopiujDoSchowka(tekst string) error {
	var polecenie *exec.Cmd
	if os.Getenv(konfiguracja.ZmiennaEkranuWayland) != "" {
		if p, err := exec.LookPath(konfiguracja.ProgramSchowkaWayland); err == nil {
			polecenie = exec.Command(p)
		}
	}
	if polecenie == nil && os.Getenv(konfiguracja.ZmiennaEkranuX11) != "" {
		if p, err := exec.LookPath(konfiguracja.ProgramSchowkaX11); err == nil {
			polecenie = exec.Command(p, konfiguracja.ArgumentySchowkaX11...)
		}
	}
	if polecenie == nil {
		return errBrakSchowka
	}
	polecenie.Stdin = strings.NewReader(tekst)
	if err := polecenie.Start(); err != nil {
		return err
	}
	go func() {
		_ = polecenie.Wait() // celowo: tylko zbieramy proces, wynik bez znaczenia
	}()
	return nil
}

// otworzFolder: xdg-open (menedżer plików pulpitu).
func otworzFolder(sciezka string) error {
	p, err := exec.LookPath(konfiguracja.ProgramPrzegladarkiLinux)
	if err != nil {
		return err
	}
	polecenie := exec.Command(p, sciezka)
	if err := polecenie.Start(); err != nil {
		return err
	}
	go func() {
		_ = polecenie.Wait() // celowo: tylko zbieramy proces, wynik bez znaczenia
	}()
	return nil
}
