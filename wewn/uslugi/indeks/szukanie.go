package indeks

import (
	"strings"
	"unicode"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// bezOgonkow: ta sama tabela co w rdzen.Pasuje (rdzeń nie eksportuje zwijania; zgodność pilnuje test).
var bezOgonkow = zbudujBezOgonkow(map[rune]string{
	'a': "ąáàâäãåā", 'c': "ćçč", 'd': "ďđ", 'e': "ęéèêëěē", 'i': "íìîïī", 'l': "łľĺ", 'n': "ńñň",
	'o': "óòôöõőø", 'r': "řŕ", 's': "śšş", 't': "ťţ", 'u': "úùûüůűū", 'y': "ýÿ", 'z': "źżž",
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

// zwin: małe litery bez ogonków ("Żółw" -> "zolw"); czysty ASCII bez wielkich liter wraca bez kopii.
func zwin(s string) string {
	if czyProste(s) {
		return s
	}
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

// czyProste: same znaki ASCII bez wielkich liter.
func czyProste(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x80 || (c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

// slowaZapytania: zapytanie rozbite na zwinięte słowa, najdłuższe pierwsze (najszybciej odrzuca).
func slowaZapytania(zapytanie string) []string {
	slowa := strings.Fields(zwin(zapytanie))
	for i := 1; i < len(slowa); i++ {
		for j := i; j > 0 && len(slowa[j]) > len(slowa[j-1]); j-- {
			slowa[j], slowa[j-1] = slowa[j-1], slowa[j]
		}
	}
	return slowa
}

// pasujeWszystkie: każde słowo zawarte w zwiniętej nazwie.
func pasujeWszystkie(zwinieta string, slowa []string) bool {
	for _, s := range slowa {
		if !strings.Contains(zwinieta, s) {
			return false
		}
	}
	return true
}

// Szukaj w indeksie udziału od folderu sciezkaStartowa (bez samego folderu startowego). Wiele słów = każde musi
// być w nazwie (jak rdzen.Pasuje: bez wielkości liter i ogonków). Limit 0 = domyślny, górna granica z konfiguracji.
// Udział niegotowy, nieznany albo zła ścieżka = nil (HTTP wtedy używa magazyn.Szukaj).
func (x *Indeks) Szukaj(udzial, sciezkaStartowa, zapytanie string, limit int) []rdzen.Wpis {
	s, ok := x.udzialy[udzial]
	if !ok {
		return nil
	}
	z := s.zbior.Load()
	start, err := rdzen.OczyscSciezke(sciezkaStartowa)
	if z == nil || err != nil {
		return nil
	}
	if limit <= 0 {
		limit = konfiguracja.MagazynDomyslnyLimitSzukania
	}
	limit = min(limit, konfiguracja.MagazynMaksLimitSzukania)
	slowa := slowaZapytania(zapytanie)
	wFolderze := z.podFolderami(start)
	wynik := []rdzen.Wpis{}
	for i := range z.pozycje {
		p := &z.pozycje[i]
		if wFolderze != nil && !wFolderze[p.rodzic] {
			continue
		}
		if !pasujeWszystkie(p.zwinieta, slowa) {
			continue
		}
		wynik = append(wynik, z.wpis(p))
		if len(wynik) >= limit {
			break
		}
	}
	return wynik
}
