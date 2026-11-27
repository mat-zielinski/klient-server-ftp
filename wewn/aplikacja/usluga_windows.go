//go:build windows

package aplikacja

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"skrytka/wewn/konfiguracja"
)

// czyUsluga: proces uruchomiony przez menedżera usług Windows.
func czyUsluga() bool {
	jest, err := svc.IsWindowsService()
	return err == nil && jest
}

// uruchomJakoUsluge: serwer pod kontrolą menedżera usług (Stop / Shutdown = łagodne zamknięcie).
func uruchomJakoUsluge() int {
	if err := svc.Run(konfiguracja.NazwaUslugi, obslugaUslugi{}); err != nil {
		return konfiguracja.KodBlad
	}
	return konfiguracja.KodOK
}

// obslugaUslugi: pętla poleceń menedżera usług.
type obslugaUslugi struct{}

// Execute: start serwera, odpowiedzi na zapytania, zatrzymanie na Stop / Shutdown.
func (obslugaUslugi) Execute(_ []string, zadania <-chan svc.ChangeRequest, stany chan<- svc.Status) (bool, uint32) {
	stany <- svc.Status{State: svc.StartPending}
	zwolnij, err := jednaInstancja(identyfikatorInstancji())
	if err != nil {
		return false, uint32(konfiguracja.KodDrugaInstancja)
	}
	defer zwolnij()
	ctx, anuluj := context.WithCancel(context.Background())
	defer anuluj()
	koniec := make(chan error, 1)
	bezPrzegladarki := func(string) error { return nil } // usługa nie ma pulpitu
	go func() { koniec <- uruchomSerwer(ctx, Opcje{OtworzPrzegladarke: bezPrzegladarki}) }()
	stany <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case err := <-koniec:
			stany <- svc.Status{State: svc.StopPending}
			return false, kodBledu(err)
		case z := <-zadania:
			switch z.Cmd {
			case svc.Interrogate:
				stany <- z.CurrentStatus
			case svc.Stop, svc.Shutdown:
				stany <- svc.Status{State: svc.StopPending}
				anuluj()
				return false, kodBledu(<-koniec)
			}
		}
	}
}

// kodBledu: kod wyjścia usługi z błędu serwera.
func kodBledu(err error) uint32 {
	if err != nil {
		return uint32(konfiguracja.KodBlad)
	}
	return uint32(konfiguracja.KodOK)
}

// sterujUsluga: instaluj | usun | start | stop (wymaga uprawnień administratora). Zwraca opis wyniku.
func sterujUsluga(polecenie string) (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("Brak dostępu do menedżera usług (uruchom jako administrator): %w", err)
	}
	defer func() {
		_ = m.Disconnect() // celowo: rozłączenie po wykonanym poleceniu, błąd bez skutków
	}()
	if polecenie == konfiguracja.PolecenieInstaluj {
		return instalujUsluge(m)
	}
	s, err := m.OpenService(konfiguracja.NazwaUslugi)
	if err != nil {
		return "", fmt.Errorf("Usługa %s nie jest zainstalowana: %w", konfiguracja.NazwaUslugi, err)
	}
	defer func() {
		_ = s.Close() // celowo: zamknięcie uchwytu, błąd bez skutków
	}()
	switch polecenie {
	case konfiguracja.PolecenieUsun:
		if err := s.Delete(); err != nil {
			return "", fmt.Errorf("Nie można usunąć usługi: %w", err)
		}
		return "Usługa usunięta (zniknie po zatrzymaniu, jeśli jeszcze działa).", nil
	case konfiguracja.PolecenieStart:
		if err := s.Start(); err != nil {
			return "", fmt.Errorf("Nie można uruchomić usługi: %w", err)
		}
		return "Usługa uruchomiona.", nil
	default:
		return zatrzymajUsluge(s)
	}
}

// instalujUsluge: start automatyczny z systemem, ten sam plik .exe.
func instalujUsluge(m *mgr.Mgr) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if s, err := m.OpenService(konfiguracja.NazwaUslugi); err == nil {
		_ = s.Close() // celowo: tylko sprawdzaliśmy, czy istnieje
		return "", errors.New("Usługa " + konfiguracja.NazwaUslugi + " jest już zainstalowana.")
	}
	s, err := m.CreateService(konfiguracja.NazwaUslugi, exe, mgr.Config{
		DisplayName: konfiguracja.NazwaAplikacji,
		Description: konfiguracja.OpisUslugi,
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return "", fmt.Errorf("Nie można zainstalować usługi: %w", err)
	}
	_ = s.Close() // celowo: usługa założona, uchwyt niepotrzebny
	return "Usługa zainstalowana (start z systemem). Uruchom: skrytka.exe --usluga start", nil
}

// zatrzymajUsluge wysyła Stop i czeka na zatrzymanie najwyżej CzasOperacjiUslugi.
func zatrzymajUsluge(s *mgr.Service) (string, error) {
	stan, err := s.Control(svc.Stop)
	if err != nil {
		return "", fmt.Errorf("Nie można zatrzymać usługi: %w", err)
	}
	termin := time.Now().Add(konfiguracja.CzasOperacjiUslugi)
	zegar := time.NewTicker(konfiguracja.OdstepSprawdzaniaUslugi)
	defer zegar.Stop()
	for stan.State != svc.Stopped {
		if time.Now().After(termin) {
			return "", errors.New("Usługa nie zatrzymała się w wyznaczonym czasie.")
		}
		<-zegar.C
		if stan, err = s.Query(); err != nil {
			return "", fmt.Errorf("Nie można odczytać stanu usługi: %w", err)
		}
	}
	return "Usługa zatrzymana.", nil
}
