package rdzen

import (
	"strings"
	"unicode"
)

// bezOgonkow: litera z ogonkiem / akcentem (małe litery) -> litera podstawowa. Polskie i najczęstsze łacińskie.
var bezOgonkow = zbudujBezOgonkow(map[rune]string{
	'a': "ąáàâäãåā",
	'c': "ćçč",
	'd': "ďđ",
	'e': "ęéèêëěē",
	'i': "íìîïī",
	'l': "łľĺ",
	'n': "ńñň",
	'o': "óòôöõőø",
	'r': "řŕ",
	's': "śšş",
	't': "ťţ",
	'u': "úùûüůűū",
	'y': "ýÿ",
	'z': "źżž",
})

func zbudujBezOgonkow(z map[rune]string) map[rune]rune {
	m := make(map[rune]rune)
	for baza, warianty := range z {
		for _, r := range warianty {
			m[r] = baza
		}
	}
	return m
}

// zwin sprowadza tekst do małych liter bez ogonków ("Żółw" -> "zolw").
func zwin(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		r = unicode.ToLower(r)
		if baza, jest := bezOgonkow[r]; jest {
			r = baza
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Pasuje: czy nazwa zawiera zapytanie, bez wielkości liter i bez ogonków ("zolw" pasuje do "Żółw",
// "ŻÓŁW" do "zolw.txt"). Zapytanie puste (po obcięciu spacji) pasuje do wszystkiego.
func Pasuje(nazwa, zapytanie string) bool {
	z := zwin(strings.TrimSpace(zapytanie))
	if z == "" {
		return true
	}
	return strings.Contains(zwin(nazwa), z)
}
