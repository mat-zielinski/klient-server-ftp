package aplikacja

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"skrytka/wewn/konfiguracja"
)

// errDrugaInstancja: program już działa (inny proces albo usługa).
var errDrugaInstancja = errors.New("Skrytka już działa na tym komputerze. Otwórz stronę w przeglądarce " +
	"albo zamknij działający program przed ponownym uruchomieniem.")

// errFTPSNieZamknal: serwer FTPS nie zakończył się w CzasZamykania.
var errFTPSNieZamknal = errors.New("Serwer FTPS nie zakończył pracy w wyznaczonym czasie.")

// Uruchom zwraca kod wyjścia procesu. Argumenty: brak = serwer (+ przeglądarka z kreatorem przy pierwszym
// uruchomieniu); "--konsola" = serwer bez przeglądarki, wpisy serwera na ekranie;
// "--usluga instaluj|usun|start|stop" = sterowanie usługą Windows.
func Uruchom(argumenty []string) int {
	if czyUsluga() {
		return uruchomJakoUsluge()
	}
	p, err := czytajArgumenty(argumenty)
	if err != nil {
		pokazKomunikat(err.Error())
		return konfiguracja.KodZleArgumenty
	}
	if p.Usluga != "" {
		opis, err := sterujUsluga(p.Usluga)
		if err != nil {
			pokazKomunikat(err.Error())
			return konfiguracja.KodBlad
		}
		pokazKomunikat(opis)
		return konfiguracja.KodOK
	}
	return uruchomZwykle(p)
}

// uruchomZwykle: serwer w bieżącym procesie do Ctrl+C / zamknięcia.
func uruchomZwykle(p polecenie) int {
	zwolnij, err := jednaInstancja(identyfikatorInstancji())
	if err != nil {
		pokazKomunikat(err.Error())
		if errors.Is(err, errDrugaInstancja) {
			return konfiguracja.KodDrugaInstancja
		}
		return konfiguracja.KodBlad
	}
	defer zwolnij()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := uruchomSerwer(ctx, Opcje{Konsola: p.Konsola}); err != nil {
		pokazKomunikat("Serwer nie działa: " + err.Error())
		return konfiguracja.KodBlad
	}
	return konfiguracja.KodOK
}
