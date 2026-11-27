package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Wyjatek zwalnia z reguły jedno miejsce; bez uzasadnienia nie działa.
type Wyjatek struct {
	Regula       string `json:"regula"`
	Plik         string `json:"plik"`
	Nazwa        string `json:"nazwa,omitempty"` // puste = każde naruszenie tej reguły w pliku
	Uzasadnienie string `json:"uzasadnienie"`
}

type plikWyjatkow struct {
	Opis    string    `json:"_opis"`
	Wyjatki []Wyjatek `json:"wyjatki"`
}

// wczytajWyjatki czyta narzedzia_dev/wyjatki.json (brak pliku = brak wyjątków).
func wczytajWyjatki(korzen string) ([]Wyjatek, error) {
	dane, err := os.ReadFile(filepath.Join(korzen, filepath.FromSlash(PlikWyjatkow)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p plikWyjatkow
	if err := json.Unmarshal(dane, &p); err != nil {
		return nil, fmt.Errorf("%s: zły JSON: %w", PlikWyjatkow, err)
	}
	return p.Wyjatki, nil
}

// brakiWyjatku: nazwy pustych pól obowiązkowych (pusta lista = wpis poprawny).
func brakiWyjatku(w Wyjatek) []string {
	var braki []string
	if strings.TrimSpace(w.Regula) == "" {
		braki = append(braki, "regula")
	}
	if strings.TrimSpace(w.Plik) == "" {
		braki = append(braki, "plik")
	}
	if len([]rune(strings.TrimSpace(w.Uzasadnienie))) < MinDlugoscPowodu {
		braki = append(braki, "uzasadnienie")
	}
	return braki
}

func (w Wyjatek) pasuje(n Naruszenie) bool {
	return w.Regula == n.Regula && w.Plik == n.Plik && (w.Nazwa == "" || w.Nazwa == n.Nazwa)
}

// zastosujWyjatki usuwa naruszenia objęte poprawnymi wyjątkami, zgłasza złe wpisy i wyjątki nieużywane
// (nieużywane tylko dla grup reguł, które w tym przebiegu sprawdzano).
func zastosujWyjatki(naruszenia []Naruszenie, wyjatki []Wyjatek, grupy map[string]bool) []Naruszenie {
	var wynik []Naruszenie
	poprawny := make([]bool, len(wyjatki))
	for i, w := range wyjatki {
		braki := brakiWyjatku(w)
		poprawny[i] = len(braki) == 0
		if !poprawny[i] {
			wynik = append(wynik, Naruszenie{Regula: "wyjatki.zly_wpis", Plik: PlikWyjatkow,
				Opis: fmt.Sprintf("wyjątek nr %d: brak pola %s", i+1, strings.Join(braki, ", "))})
		}
	}
	uzyty := make([]bool, len(wyjatki))
	for _, n := range naruszenia {
		objete := false
		for i, w := range wyjatki {
			if poprawny[i] && w.pasuje(n) {
				uzyty[i], objete = true, true
			}
		}
		if !objete {
			wynik = append(wynik, n)
		}
	}
	for i, w := range wyjatki {
		grupa, _, _ := strings.Cut(w.Regula, ".")
		if poprawny[i] && !uzyty[i] && grupy[grupa] {
			wynik = append(wynik, Naruszenie{Regula: "wyjatki.nieuzywany", Plik: PlikWyjatkow,
				Opis: fmt.Sprintf("wyjątek %s dla %s już niepotrzebny: usuń wpis", w.Regula, w.Plik)})
		}
	}
	return wynik
}
