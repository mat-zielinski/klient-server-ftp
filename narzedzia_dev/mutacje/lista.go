package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Wpis to jedna mutacja blokady: fragment `szukaj` w pliku zamieniany na `zamien` MUSI wywrócić testy pakietu.
type Wpis struct {
	ID     string `json:"id"`
	Opis   string `json:"opis"`
	Plik   string `json:"plik"`   // względem korzenia, z "/"
	Szukaj string `json:"szukaj"` // dokładny fragment kodu (dokładnie jedno wystąpienie)
	Zamien string `json:"zamien"` // fragment po mutacji (blokada zdjęta)
	Pakiet string `json:"pakiet"` // np. "./wewn/rdzen"
	Testy  string `json:"testy"`  // opcjonalnie wzorzec -run (puste = wszystkie testy pakietu)
	// DoUzupelnienia: niepuste = wpis czeka na kod (opis, czego brakuje); pomijany z ostrzeżeniem.
	DoUzupelnienia string `json:"do_uzupelnienia,omitempty"`
}

type plikListy struct {
	Opis    string `json:"_opis"`
	Mutacje []Wpis `json:"mutacje"`
}

// wczytajListe czyta listę mutacji i sprawdza budowę wpisów (błąd = lista nieaktualna).
func wczytajListe(sciezka string) ([]Wpis, error) {
	dane, err := os.ReadFile(sciezka)
	if err != nil {
		return nil, fmt.Errorf("nie da się wczytać %s: %w", sciezka, err)
	}
	var p plikListy
	if err := json.Unmarshal(dane, &p); err != nil {
		return nil, fmt.Errorf("%s: zły JSON: %w", sciezka, err)
	}
	if len(p.Mutacje) == 0 {
		return nil, fmt.Errorf("%s: brak niepustej listy „mutacje”", sciezka)
	}
	var bledy []string
	widziane := map[string]bool{}
	for _, w := range p.Mutacje {
		bledy = append(bledy, bledyWpisu(w)...)
		if widziane[w.ID] {
			bledy = append(bledy, fmt.Sprintf("powtórzone id „%s”", w.ID))
		}
		widziane[w.ID] = true
	}
	if len(bledy) > 0 {
		return nil, fmt.Errorf("lista nieaktualna:\n  %s", strings.Join(bledy, "\n  "))
	}
	return p.Mutacje, nil
}

// bledyWpisu: błędy budowy jednego wpisu (pusta lista = poprawny). Wpis do uzupełnienia wymaga tylko id i opisu.
func bledyWpisu(w Wpis) []string {
	nazwa := w.ID
	if nazwa == "" {
		nazwa = "?"
	}
	var bledy []string
	wymagane := map[string]string{"id": w.ID, "opis": w.Opis}
	if w.DoUzupelnienia == "" {
		wymagane["plik"], wymagane["szukaj"], wymagane["pakiet"] = w.Plik, w.Szukaj, w.Pakiet
	}
	for _, pole := range []string{"id", "opis", "plik", "szukaj", "pakiet"} {
		if wartosc, ok := wymagane[pole]; ok && strings.TrimSpace(wartosc) == "" {
			bledy = append(bledy, fmt.Sprintf("%s: pole „%s” musi być niepustym napisem", nazwa, pole))
		}
	}
	if w.DoUzupelnienia != "" {
		return bledy
	}
	if w.Szukaj == w.Zamien {
		bledy = append(bledy, nazwa+": „zamien” jest takie samo jak „szukaj” (to nie mutacja)")
	}
	if w.Pakiet != "" && !strings.HasPrefix(w.Pakiet, "./") {
		bledy = append(bledy, nazwa+": „pakiet” musi zaczynać się od ./ (ścieżka względem korzenia)")
	}
	return bledy
}

// wKonczachLinii: fragment z listy (końce "\n") w końcach linii pliku (Windows może mieć "\r\n").
func wKonczachLinii(fragment, tekst string) string {
	if strings.Contains(tekst, "\r\n") {
		return strings.ReplaceAll(fragment, "\n", "\r\n")
	}
	return fragment
}

func ileWystapien(tekst, szukaj string) int {
	return strings.Count(tekst, wKonczachLinii(szukaj, tekst))
}

// mutuj zwraca tekst po mutacji; `szukaj` musi wystąpić dokładnie raz.
func mutuj(tekst, szukaj, zamien string) (string, error) {
	if n := ileWystapien(tekst, szukaj); n != 1 {
		return "", fmt.Errorf("fragment „szukaj” występuje %d razy (ma dokładnie 1)", n)
	}
	return strings.Replace(tekst, wKonczachLinii(szukaj, tekst), wKonczachLinii(zamien, tekst), 1), nil
}

// sprawdzAktualnosc: plik istnieje, `szukaj` dokładnie raz, katalog pakietu istnieje.
func sprawdzAktualnosc(korzen string, wpisy []Wpis) []string {
	var bledy []string
	for _, w := range wpisy {
		tresc, err := os.ReadFile(filepath.Join(korzen, filepath.FromSlash(w.Plik)))
		if err != nil {
			bledy = append(bledy, fmt.Sprintf("%s: brak pliku %s", w.ID, w.Plik))
		} else if n := ileWystapien(string(tresc), w.Szukaj); n != 1 {
			bledy = append(bledy, fmt.Sprintf("%s: fragment „szukaj” w %s występuje %d razy (ma dokładnie 1)", w.ID, w.Plik, n))
		}
		if info, err := os.Stat(filepath.Join(korzen, filepath.FromSlash(w.Pakiet))); err != nil || !info.IsDir() {
			bledy = append(bledy, fmt.Sprintf("%s: brak katalogu pakietu %s", w.ID, w.Pakiet))
		}
	}
	return bledy
}
