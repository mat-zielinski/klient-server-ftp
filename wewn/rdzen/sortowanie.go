package rdzen

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	"skrytka/wewn/konfiguracja"
)

// alfabet: kolejność liter polskich (ą po a, ł po l …); litery spoza niego idą po nim.
const alfabet = "aąbcćdeęfghijklłmnńoópqrsśtuvwxyzźż"

// Wagi znaków przy porównaniu: znaki ASCII inne niż litery (cyfry, interpunkcja) najpierw, potem litery
// alfabetu, potem pozostałe znaki.
const (
	wagaLiter       = 0x10000
	krokLitery      = 4
	wariantAkcentu  = 2
	wagaPozostalych = wagaLiter + 64*krokLitery
)

var indeksAlfabetu = zbudujIndeksAlfabetu()

func zbudujIndeksAlfabetu() map[rune]int {
	m := make(map[rune]int)
	for _, r := range alfabet {
		if _, jest := m[r]; !jest {
			m[r] = len(m)
		}
	}
	return m
}

// waga znaku do porównania bez wielkości liter.
func waga(r rune) int {
	r = unicode.ToLower(r)
	if i, jest := indeksAlfabetu[r]; jest {
		return wagaLiter + i*krokLitery
	}
	if baza, jest := bezOgonkow[r]; jest {
		if i, jest := indeksAlfabetu[baza]; jest {
			return wagaLiter + i*krokLitery + wariantAkcentu
		}
	}
	if r < 'a' {
		return int(r)
	}
	return wagaPozostalych + int(r)
}

func czyCyfra(r rune) bool { return r >= '0' && r <= '9' }

// porownajNaturalnie: "a2" < "a10", bez wielkości liter, z polską kolejnością liter. 0 = równe dla użytkownika.
func porownajNaturalnie(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if czyCyfra(ra[i]) && czyCyfra(rb[j]) {
			si, sj := i, j
			for i < len(ra) && czyCyfra(ra[i]) {
				i++
			}
			for j < len(rb) && czyCyfra(rb[j]) {
				j++
			}
			if c := porownajLiczby(ra[si:i], rb[sj:j]); c != 0 {
				return c
			}
			continue
		}
		if c := cmp.Compare(waga(ra[i]), waga(rb[j])); c != 0 {
			return c
		}
		i++
		j++
	}
	return cmp.Compare(len(ra)-i, len(rb)-j)
}

// porownajLiczby porównuje ciągi cyfr dowolnej długości jako liczby (bez przepełnienia).
func porownajLiczby(a, b []rune) int {
	a, b = bezZerWiodacych(a), bezZerWiodacych(b)
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	return slices.Compare(a, b)
}

func bezZerWiodacych(c []rune) []rune {
	for len(c) > 1 && c[0] == '0' {
		c = c[1:]
	}
	return c
}

// porownajNazwy: naturalnie, a przy remisie dokładnie (stała kolejność), potem po ścieżce.
func porownajNazwy(a, b Wpis) int {
	if c := porownajNaturalnie(a.Nazwa, b.Nazwa); c != 0 {
		return c
	}
	if c := strings.Compare(a.Nazwa, b.Nazwa); c != 0 {
		return c
	}
	return strings.Compare(a.Sciezka, b.Sciezka)
}

func wybierzPorownanie(klucz string) func(a, b Wpis) int {
	switch klucz {
	case konfiguracja.KluczSortRozmiar:
		return func(a, b Wpis) int { return cmp.Compare(a.Rozmiar, b.Rozmiar) }
	case konfiguracja.KluczSortZmieniono:
		return func(a, b Wpis) int { return a.Zmieniono.Compare(b.Zmieniono) }
	}
	return func(a, b Wpis) int { return 0 }
}

// Sortuj porządkuje wpisy w miejscu: foldery zawsze pierwsze (także malejąco), potem wg klucza
// ("nazwa" | "rozmiar" | "zmieniono", nieznany = "nazwa"); remis rozstrzyga nazwa naturalnie bez wielkości liter.
func Sortuj(w []Wpis, klucz string, malejaco bool) {
	por := wybierzPorownanie(klucz)
	slices.SortStableFunc(w, func(a, b Wpis) int {
		if a.Folder != b.Folder {
			if a.Folder {
				return -1
			}
			return 1
		}
		c := por(a, b)
		if c == 0 {
			c = porownajNazwy(a, b)
		}
		if malejaco {
			return -c
		}
		return c
	})
}
