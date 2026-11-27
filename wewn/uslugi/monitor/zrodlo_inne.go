//go:build !windows && !linux

package monitor

import "errors"

// errNieobslugiwany: system bez odczytów monitora (obsługiwane: Windows i Linux).
var errNieobslugiwany = errors.New("Monitor nie obsługuje tego systemu.")

type zrodloNieobslugiwane struct{}

// NoweZrodloSystemowe na nieobsługiwanym systemie zawsze zwraca błąd (próbki bez pól systemowych).
func NoweZrodloSystemowe() Zrodlo { return zrodloNieobslugiwane{} }

func (zrodloNieobslugiwane) Odczytaj() (Odczyt, error) { return Odczyt{}, errNieobslugiwany }

// ZajetoscDysku na nieobsługiwanym systemie zwraca błąd.
func ZajetoscDysku(string) (Zajetosc, error) { return Zajetosc{}, errNieobslugiwany }
