//go:build !windows && !linux

package aplikacja

import (
	"errors"
	"os"
)

// Skrytka działa na Windows i Linuksie; te zaślepki tylko pozwalają skompilować pakiet gdzie indziej.

var errNieobslugiwanySystem = errors.New("Ta funkcja nie działa w tym systemie (tylko Windows i Linux).")

func identyfikatorInstancji() string { return "" }

func jednaInstancja(string) (func(), error) { return func() {}, nil }

func pokazKomunikat(tekst string) {
	_, _ = os.Stderr.WriteString(tekst + "\n") // celowo: brak innego kanału
}

func otworzPrzegladarke(string) error { return nil }

func czyUsluga() bool { return false }

func uruchomJakoUsluge() int { return 1 }

func sterujUsluga(string) (string, error) { return "", errNieobslugiwanySystem }
