package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// uruchamiaczNode uruchamia polecenie w katalogu i zwraca wyjście (podmieniany w testach).
type uruchamiaczNode func(katalog string, polecenie string, argumenty ...string) (string, error)

func uruchomPolecenie(katalog string, polecenie string, argumenty ...string) (string, error) {
	cmd := exec.Command(polecenie, argumenty...)
	cmd.Dir = katalog
	wyjscie, err := cmd.CombinedOutput()
	return string(wyjscie), err
}

// sprawdzStylWeb uruchamia `node web/sprawdz_styl.mjs`, jeśli plik istnieje. Drugi wynik: czy sprawdzono.
func sprawdzStylWeb(korzen string, uruchom uruchamiaczNode) ([]Naruszenie, bool) {
	if _, err := os.Stat(filepath.Join(korzen, filepath.FromSlash(PlikStyluWeb))); err != nil {
		return nil, false
	}
	wyjscie, err := uruchom(korzen, PolecenieNode, PlikStyluWeb)
	if err == nil {
		return nil, true
	}
	opis := strings.TrimSpace(wyjscie)
	var bladWyjscia *exec.ExitError
	if !errors.As(err, &bladWyjscia) {
		opis = "nie udało się uruchomić node (zainstalowany i w PATH?): " + err.Error()
	} else if opis == "" {
		opis = "strażnik stylu zwrócił błąd bez opisu: " + err.Error()
	}
	return []Naruszenie{{Regula: "styl.web", Plik: PlikStyluWeb, Opis: opis}}, true
}
