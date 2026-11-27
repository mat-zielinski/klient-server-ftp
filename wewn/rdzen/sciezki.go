package rdzen

import (
	"path/filepath"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// OczyscSciezke normalizuje ścieżkę w udziale do postaci "/a/b" ("", "." i "/" = "/").
// Separator to wyłącznie "/". Odrzuca (ErrPozaUdzialem): "..", "\", UNC ("//serwer"), dyski ("C:").
// Odrzuca (ErrZlaNazwa): strumienie NTFS ("plik:strumien"), znaki sterujące i zakazane, nazwy urządzeń,
// człony z kropką/spacją na końcu, zbyt długie ścieżki. Puste człony i "." są pomijane.
// Ścieżka ma być już zdekodowana (URL) dokładnie raz; dowiązań symbolicznych rdzeń nie rozwiązuje.
func OczyscSciezke(wzgledna string) (string, error) {
	if len(wzgledna) > konfiguracja.MaksDlugoscSciezki {
		return "", ErrZlaNazwa
	}
	if strings.Contains(wzgledna, `\`) || strings.HasPrefix(wzgledna, "//") {
		return "", ErrPozaUdzialem
	}
	czlony := make([]string, 0, strings.Count(wzgledna, "/")+1)
	for _, c := range strings.Split(wzgledna, "/") {
		switch {
		case c == "" || c == ".":
			continue
		case c == "..":
			return "", ErrPozaUdzialem
		case czyDysk(c):
			return "", ErrPozaUdzialem
		}
		if err := sprawdzCzlon(c); err != nil {
			return "", err
		}
		czlony = append(czlony, c)
	}
	if len(czlony) > konfiguracja.MaksGlebokoscSciezki {
		return "", ErrZlaNazwa
	}
	return "/" + strings.Join(czlony, "/"), nil
}

// czyDysk: człon zaczyna się od litery dysku i dwukropka ("C:", "c:plik").
func czyDysk(c string) bool {
	if len(c) < 2 || c[1] != ':' {
		return false
	}
	l := c[0] | 0x20
	return l >= 'a' && l <= 'z'
}

// SciezkaNaDysku łączy bezwzględny korzeń udziału ze ścieżką po OczyscSciezke. Wynik (po filepath.Clean) to
// korzeń albo ścieżka zaczynająca się od "korzeń + separator" (korzeń "C:\a" nie przepuści "C:\ab").
// Dowiązania symboliczne i punkty połączenia NTFS sprawdza magazyn (EvalSymlinks), nie rdzeń.
func SciezkaNaDysku(korzen, wzgledna string) (string, error) {
	if korzen == "" || !filepath.IsAbs(korzen) {
		return "", ErrPozaUdzialem
	}
	czysta, err := OczyscSciezke(wzgledna)
	if err != nil {
		return "", err
	}
	return polaczZKorzeniem(korzen, czysta)
}

// polaczZKorzeniem: złączenie i druga linia obrony (nie ufa, że ścieżka przeszła przez OczyscSciezke).
func polaczZKorzeniem(korzen, czysta string) (string, error) {
	kor := filepath.Clean(korzen)
	wynik := filepath.Clean(filepath.Join(kor, filepath.FromSlash(strings.TrimPrefix(czysta, "/"))))
	if !wewnatrzKorzenia(kor, wynik) {
		return "", ErrPozaUdzialem
	}
	return wynik, nil
}

// wewnatrzKorzenia: druga linia obrony, porównanie z prefiksem korzenia zakończonym separatorem.
func wewnatrzKorzenia(kor, wynik string) bool {
	if wynik == kor {
		return true
	}
	prefiks := kor
	if !strings.HasSuffix(prefiks, string(filepath.Separator)) {
		prefiks += string(filepath.Separator)
	}
	return strings.HasPrefix(wynik, prefiks)
}
