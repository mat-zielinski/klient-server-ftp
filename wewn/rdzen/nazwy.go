package rdzen

import (
	"strings"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
)

// SprawdzNazwe sprawdza nazwę pliku/folderu tworzonego przez użytkownika: bez separatorów, ":" (dyski, strumienie
// NTFS), znaków zakazanych w Windows i sterujących, bez nazw urządzeń (CON, NUL.txt …), bez kropki/spacji na
// końcu, nie "." / ".." i nie katalog kosza aplikacji. Błąd = ErrZlaNazwa.
func SprawdzNazwe(nazwa string) error {
	if err := sprawdzCzlon(nazwa); err != nil {
		return err
	}
	if strings.EqualFold(nazwa, konfiguracja.KatalogKosza) {
		return ErrZlaNazwa
	}
	return nil
}

// sprawdzCzlon to wspólna część SprawdzNazwe i OczyscSciezke (bez zakazu katalogu kosza).
func sprawdzCzlon(nazwa string) error {
	if nazwa == "" || len(nazwa) > konfiguracja.MaksDlugoscNazwy || !utf8.ValidString(nazwa) {
		return ErrZlaNazwa
	}
	if strings.ContainsAny(nazwa, konfiguracja.ZnakiZakazaneWNazwie) {
		return ErrZlaNazwa
	}
	for _, r := range nazwa {
		if czyZnakZakazany(r) {
			return ErrZlaNazwa
		}
	}
	// Windows po cichu obcina kropki i spacje na końcu (".. " = ".."), więc nie przepuszczamy ich wcale.
	ostatni := nazwa[len(nazwa)-1]
	if ostatni == '.' || ostatni == ' ' {
		return ErrZlaNazwa
	}
	if czyNazwaUrzadzenia(nazwa) {
		return ErrZlaNazwa
	}
	return nil
}

// czyZnakZakazany: znaki sterujące C0, DEL, C1 oraz sterowanie kierunkiem tekstu (podszywanie się pod rozszerzenie).
func czyZnakZakazany(r rune) bool {
	switch {
	case r < 0x20, r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}

// czyNazwaUrzadzenia: "NUL", "nul.txt", "Com1.tar.gz", "AUX .txt" to urządzenia Windows.
func czyNazwaUrzadzenia(nazwa string) bool {
	podstawa, _, _ := strings.Cut(nazwa, ".")
	podstawa = strings.TrimRight(podstawa, " ")
	for _, z := range konfiguracja.NazwyZarezerwowaneWindows {
		if strings.EqualFold(podstawa, z) {
			return true
		}
	}
	return false
}
