// Strażnik projektu Skrytka: architektura warstw, jakość kodu, testy i styl web.
//
// Użycie (z dowolnego katalogu w projekcie):
//
//	go run ./narzedzia_dev/straznik [-katalog KATALOG] [-tylko architektura,jakosc,testy,styl]
//
// Kod wyjścia: 0 = OK, 1 = naruszenia, 2 = błąd uruchomienia. Opis reguł: narzedzia_dev/README.md.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// grupaRegul to jedna grupa sprawdzeń strażnika działająca na sparsowanym projekcie.
type grupaRegul struct {
	Nazwa   string
	Sprawdz func(p *projekt) []Naruszenie
}

var grupyRegul = []grupaRegul{
	{"architektura", sprawdzArchitekture},
	{"jakosc", sprawdzJakosc},
	{"testy", sprawdzTesty},
}

const grupaStylu = "styl"

// kolejnoscGrup: kolejność grup w podsumowaniu.
var kolejnoscGrup = []string{"architektura", "jakosc", "testy", grupaStylu, "wyjatki"}

func main() {
	os.Exit(uruchom(os.Args[1:], os.Stdout, uruchomPolecenie))
}

// uruchom to całe działanie programu (osobno od main dla testów).
func uruchom(argumenty []string, wyjscie io.Writer, node uruchamiaczNode) int {
	flagi := flag.NewFlagSet("straznik", flag.ContinueOnError)
	flagi.SetOutput(wyjscie)
	katalog := flagi.String("katalog", "", "korzeń projektu (domyślnie: najbliższy katalog z go.mod w górę)")
	tylko := flagi.String("tylko", "", "grupy reguł po przecinku: architektura,jakosc,testy,styl (domyślnie wszystkie)")
	if err := flagi.Parse(argumenty); err != nil {
		return KodBladUruchomienia
	}
	naruszenia, opis, err := przebieg(*katalog, *tylko, node)
	if err != nil {
		fmt.Fprintln(wyjscie, "STRAŻNIK: błąd uruchomienia:", err)
		return KodBladUruchomienia
	}
	return wypisz(wyjscie, naruszenia, opis)
}

func przebieg(katalog, tylko string, node uruchamiaczNode) ([]Naruszenie, string, error) {
	korzen, err := znajdzKorzen(katalog)
	if err != nil {
		return nil, "", err
	}
	grupy, err := wybierzGrupy(tylko)
	if err != nil {
		return nil, "", err
	}
	return zbierz(korzen, grupy, node)
}

func wybierzGrupy(tylko string) (map[string]bool, error) {
	grupy := map[string]bool{grupaStylu: true}
	for _, g := range grupyRegul {
		grupy[g.Nazwa] = true
	}
	if strings.TrimSpace(tylko) == "" {
		return grupy, nil
	}
	wybrane := map[string]bool{}
	for _, g := range strings.Split(tylko, ",") {
		g = strings.TrimSpace(g)
		if !grupy[g] {
			return nil, fmt.Errorf("nieznana grupa reguł %q (są: architektura, jakosc, testy, styl)", g)
		}
		wybrane[g] = true
	}
	return wybrane, nil
}

// zbierz uruchamia wybrane grupy i nakłada wyjątki; zwraca też jednolinijkowy opis zakresu.
func zbierz(korzen string, grupy map[string]bool, node uruchamiaczNode) ([]Naruszenie, string, error) {
	p, err := wczytajProjekt(korzen)
	if err != nil {
		return nil, "", err
	}
	wyjatki, err := wczytajWyjatki(korzen)
	if err != nil {
		return nil, "", err
	}
	var wszystkie []Naruszenie
	for _, g := range grupyRegul {
		if grupy[g.Nazwa] {
			wszystkie = append(wszystkie, g.Sprawdz(p)...)
		}
	}
	styl := "styl web nie wybrany"
	if grupy[grupaStylu] {
		n, sprawdzono := sprawdzStylWeb(korzen, node)
		wszystkie = append(wszystkie, n...)
		styl = "styl web pominięty (brak " + PlikStyluWeb + ")"
		if sprawdzono {
			styl = "styl web sprawdzony"
		}
	}
	plikow := 0
	for _, pk := range p.Pakiety {
		plikow += len(pk.Pliki)
	}
	opis := fmt.Sprintf("pakietów %d, plików %d, %s", len(p.Pakiety), plikow, styl)
	return zastosujWyjatki(wszystkie, wyjatki, grupy), opis, nil
}

func wypisz(wyjscie io.Writer, naruszenia []Naruszenie, opis string) int {
	sort.SliceStable(naruszenia, func(i, j int) bool {
		if naruszenia[i].Plik != naruszenia[j].Plik {
			return naruszenia[i].Plik < naruszenia[j].Plik
		}
		return naruszenia[i].Linia < naruszenia[j].Linia
	})
	for _, n := range naruszenia {
		fmt.Fprintln(wyjscie, n.Tekst())
	}
	if len(naruszenia) == 0 {
		fmt.Fprintf(wyjscie, "STRAŻNIK: OK (%s)\n", opis)
		return KodOK
	}
	ile := map[string]int{}
	for _, n := range naruszenia {
		grupa, _, _ := strings.Cut(n.Regula, ".")
		ile[grupa]++
	}
	var czesci []string
	for _, g := range kolejnoscGrup {
		if ile[g] > 0 {
			czesci = append(czesci, fmt.Sprintf("%s %d", g, ile[g]))
		}
	}
	fmt.Fprintf(wyjscie, "STRAŻNIK: naruszeń %d (%s); %s\n", len(naruszenia), strings.Join(czesci, ", "), opis)
	fmt.Fprintf(wyjscie, "Popraw kod; wyjątek tylko z uzasadnieniem w %s.\n", PlikWyjatkow)
	return KodNaruszenia
}

// znajdzKorzen: podany katalog albo najbliższy w górę od katalogu bieżącego, który ma go.mod.
func znajdzKorzen(podany string) (string, error) {
	if podany != "" {
		return filepath.Abs(podany)
	}
	katalog, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(katalog, "go.mod")); err == nil {
			return katalog, nil
		}
		wyzej := filepath.Dir(katalog)
		if wyzej == katalog {
			return "", errors.New("nie znaleziono go.mod (uruchom w projekcie albo podaj -katalog)")
		}
		katalog = wyzej
	}
}
