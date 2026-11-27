//go:build !windows && !linux

package aplikacja

// Zasobnik tylko na Windows i Linuksie (czyPokazacZasobnik zwraca tu false); zaślepki do kompilacji.

func petlaZasobnika(*zasobnik) error { return errNieobslugiwanySystem }

func zakonczPetleZasobnika() {}
