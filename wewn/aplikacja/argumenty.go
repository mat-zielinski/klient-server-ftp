package aplikacja

import (
	"errors"
	"fmt"

	"skrytka/wewn/konfiguracja"
)

// Polecenie z wiersza poleceń.
type polecenie struct {
	// Usluga: "" = serwer; inaczej instaluj | usun | start | stop.
	Usluga  string
	Konsola bool
	// BezZasobnika: zwykły serwer bez ikony w zasobniku systemowym.
	BezZasobnika bool
}

// errZleArgumenty: opis poprawnego użycia.
var errZleArgumenty = errors.New("Użycie: skrytka.exe [--konsola] [--bez-zasobnika] | --usluga instaluj|usun|start|stop")

// czytajArgumenty zamienia argumenty na polecenie; nieznany argument = błąd z opisem użycia.
func czytajArgumenty(argumenty []string) (polecenie, error) {
	var p polecenie
	for i := 0; i < len(argumenty); i++ {
		switch argumenty[i] {
		case konfiguracja.ArgumentKonsola:
			p.Konsola = true
		case konfiguracja.ArgumentBezZasobnika:
			p.BezZasobnika = true
		case konfiguracja.ArgumentUsluga:
			if i+1 >= len(argumenty) || !znanePolecenieUslugi(argumenty[i+1]) || p.Usluga != "" {
				return polecenie{}, errZleArgumenty
			}
			p.Usluga = argumenty[i+1]
			i++
		default:
			return polecenie{}, fmt.Errorf("Nieznany argument %q. %w", argumenty[i], errZleArgumenty)
		}
	}
	if p.Usluga != "" && (p.Konsola || p.BezZasobnika) {
		return polecenie{}, errZleArgumenty
	}
	return p, nil
}

// znanePolecenieUslugi: instaluj | usun | start | stop.
func znanePolecenieUslugi(s string) bool {
	switch s {
	case konfiguracja.PolecenieInstaluj, konfiguracja.PolecenieUsun,
		konfiguracja.PolecenieStart, konfiguracja.PolecenieStop:
		return true
	}
	return false
}
