package rdzen

import (
	"path"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// rodzajeRozszerzen: rozszerzenie (małe litery, z kropką) -> rodzaj; budowane raz z list w konfiguracji.
var rodzajeRozszerzen = zbudujRodzaje()

func zbudujRodzaje() map[string]string {
	listy := []struct {
		rodzaj string
		rozsz  []string
	}{
		{RodzajObraz, konfiguracja.RozszerzeniaObrazow},
		{RodzajWideo, konfiguracja.RozszerzeniaWideo},
		{RodzajDzwiek, konfiguracja.RozszerzeniaDzwiekow},
		{RodzajPDF, konfiguracja.RozszerzeniaPDF},
		{RodzajTekst, konfiguracja.RozszerzeniaTekstow},
		{RodzajArchiwum, konfiguracja.RozszerzeniaArchiwow},
	}
	m := make(map[string]string)
	for _, l := range listy {
		for _, r := range l.rozsz {
			if _, jest := m[r]; jest {
				panic("rdzen: rozszerzenie w dwóch rodzajach: " + r)
			}
			m[r] = l.rodzaj
		}
	}
	return m
}

// RodzajPliku zwraca rodzaj do ikon i podglądu: folder = RodzajFolder, dalej wg rozszerzenia bez względu na
// wielkość liter, nieznane albo brak rozszerzenia = RodzajInny.
func RodzajPliku(nazwa string, folder bool) string {
	if folder {
		return RodzajFolder
	}
	if r, jest := rodzajeRozszerzen[strings.ToLower(path.Ext(nazwa))]; jest {
		return r
	}
	return RodzajInny
}
