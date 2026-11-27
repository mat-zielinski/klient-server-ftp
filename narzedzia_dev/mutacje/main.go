// Strażnik MUTACJI blokad bezpieczeństwa (wzór: Programator_Czarek/narzedzia_dev/mutacje_blokad.py).
//
// Dla wpisów z narzedzia_dev/mutacje_blokad.json tymczasowo zdejmuje blokadę (fragment `szukaj` → `zamien`),
// uruchamia testy pakietu i wymaga, żeby PADŁY. Plik wraca na miejsce zawsze: po zwykłym końcu, po błędzie,
// po Ctrl+C, a po twardym przerwaniu przy następnym uruchomieniu (kopia *.mutacja_kopia).
//
// Użycie (z dowolnego katalogu w projekcie):
//
//	go run ./narzedzia_dev/mutacje [-katalog KATALOG] [-id ID] [-lista]
//
// Kod wyjścia: 0 = każda mutacja wywraca testy, 1 = fałszywie zielone, 2 = błąd uruchomienia / lista nieaktualna.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	PlikListy              = "narzedzia_dev/mutacje_blokad.json"
	RozszerzenieKopii      = ".mutacja_kopia"
	RozszerzenieTymczasowe = ".mutacja_tmp"
	RozszerzenieKonfliktu  = ".mutacja_konflikt"
	GoPrzenosne            = "_go-portable/bin/go"
	OgonWyjscia            = 15 // ile ostatnich linii go test pokazać
	KodOK                  = 0
	KodZielone             = 1
	KodBledu               = 2
)

func main() {
	s := &straz{}
	sygnaly := make(chan os.Signal, 1)
	signal.Notify(sygnaly, os.Interrupt)
	go func() {
		<-sygnaly
		if err := s.przywroc(); err != nil {
			fmt.Fprintln(os.Stderr, "MUTACJE:", err)
		}
		fmt.Fprintln(os.Stderr, "MUTACJE: przerwano, plik przywrócony")
		os.Exit(KodBledu)
	}()
	os.Exit(uruchom(os.Args[1:], os.Stdout, s))
}

func uruchom(argumenty []string, wyjscie io.Writer, s *straz) int {
	flagi := flag.NewFlagSet("mutacje", flag.ContinueOnError)
	flagi.SetOutput(wyjscie)
	katalog := flagi.String("katalog", "", "korzeń projektu (domyślnie: najbliższy katalog z go.mod w górę)")
	id := flagi.String("id", "", "tylko wpis o tym id")
	lista := flagi.Bool("lista", false, "tylko wypisz listę mutacji")
	if err := flagi.Parse(argumenty); err != nil {
		return KodBledu
	}
	korzen, err := znajdzKorzen(*katalog)
	if err != nil {
		fmt.Fprintln(wyjscie, "MUTACJE: błąd:", err)
		return KodBledu
	}
	wpisy, err := wczytajListe(filepath.Join(korzen, filepath.FromSlash(PlikListy)))
	if err == nil && *id != "" {
		wpisy, err = wybierz(wpisy, *id)
	}
	if err != nil {
		fmt.Fprintln(wyjscie, "MUTACJE:", err)
		return KodBledu
	}
	if *lista {
		for _, w := range wpisy {
			fmt.Fprintf(wyjscie, "%s  %s  [%s]\n", w.ID, w.Opis, w.Pakiet)
		}
		return KodOK
	}
	return przebieg(korzen, wpisy, goTest(korzen), s, wyjscie)
}

func wybierz(wpisy []Wpis, id string) ([]Wpis, error) {
	for _, w := range wpisy {
		if w.ID == id {
			return []Wpis{w}, nil
		}
	}
	return nil, fmt.Errorf("brak wpisu o id %q", id)
}

// sciezkaGo: przenośne Go projektu, a bez niego go z PATH.
func sciezkaGo(korzen string) string {
	przenosne := filepath.Join(korzen, filepath.FromSlash(GoPrzenosne))
	if runtime.GOOS == "windows" {
		przenosne += ".exe"
	}
	if _, err := os.Stat(przenosne); err == nil {
		return przenosne
	}
	return "go"
}

// goTest uruchamia `go test -count=1 [-run wzorzec] pakiet` w korzeniu projektu.
func goTest(korzen string) uruchamiaczTestow {
	program := sciezkaGo(korzen)
	return func(pakiet, wzorzec string) (wynikTestow, error) {
		argumenty := []string{"test", "-count=1"}
		if wzorzec != "" {
			argumenty = append(argumenty, "-run", wzorzec)
		}
		cmd := exec.Command(program, append(argumenty, pakiet)...)
		cmd.Dir = korzen
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
		var bufor bytes.Buffer
		cmd.Stdout, cmd.Stderr = &bufor, &bufor
		err := cmd.Run()
		wynik := wynikTestow{Zielone: err == nil, Kompiluje: !nieBuduje(bufor.String()), Wyjscie: ogon(bufor.String())}
		var bladWyjscia *exec.ExitError
		if err != nil && !errors.As(err, &bladWyjscia) {
			return wynik, err
		}
		return wynik, nil
	}
}

func nieBuduje(wyjscie string) bool {
	return strings.Contains(wyjscie, "[build failed]") || strings.Contains(wyjscie, "[setup failed]")
}

func ogon(tekst string) string {
	linie := strings.Split(strings.TrimRight(tekst, "\r\n"), "\n")
	if len(linie) > OgonWyjscia {
		linie = linie[len(linie)-OgonWyjscia:]
	}
	return "    " + strings.Join(linie, "\n    ")
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
