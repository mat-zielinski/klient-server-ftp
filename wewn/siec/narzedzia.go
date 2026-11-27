package siec

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"skrytka/wewn/konfiguracja"
)

// AdresPubliczny pyta zewnętrzną usługę o IP (z limitem czasu); błąd = "".
func AdresPubliczny() string {
	klient := &http.Client{Timeout: konfiguracja.LimitCzasuAdresuPublicznego}
	return adresPublicznyZ(klient, konfiguracja.UslugiAdresuPublicznego)
}

// adresPublicznyZ pyta usługi po kolei; pierwsza poprawna odpowiedź (samo IP) wygrywa.
func adresPublicznyZ(klient *http.Client, uslugi []string) string {
	for _, u := range uslugi {
		if ip := zapytajOIP(klient, u); ip != "" {
			return ip
		}
	}
	return ""
}

func zapytajOIP(klient *http.Client, url string) string {
	odp, err := klient.Get(url)
	if err != nil {
		return ""
	}
	defer odp.Body.Close() // celowo: błąd zamknięcia ciała odpowiedzi niczego nie zmienia
	if odp.StatusCode != http.StatusOK {
		return ""
	}
	dane, err := io.ReadAll(io.LimitReader(odp.Body, konfiguracja.MaksOdpowiedziAdresuPublicznego))
	if err != nil {
		return ""
	}
	ip := net.ParseIP(strings.TrimSpace(string(dane)))
	if ip == nil {
		return ""
	}
	return ip.String()
}

// SprawdzPort: czy port nasłuchuje lokalnie (true/false) z opisem.
func SprawdzPort(port int) (bool, string) {
	if port < 1 || port > 65535 {
		return false, fmt.Sprintf("Niepoprawny numer portu: %d.", port)
	}
	adres := net.JoinHostPort(konfiguracja.AdresTestuPortu, strconv.Itoa(port))
	if pol, err := net.DialTimeout("tcp", adres, konfiguracja.LimitCzasuTestuPortu); err == nil {
		pol.Close() // celowo: połączenie próbne, wynik zamknięcia bez znaczenia
		return true, fmt.Sprintf("Port %d nasłuchuje lokalnie.", port)
	}
	sluchacz, err := net.Listen("tcp", adres)
	if err != nil {
		return false, fmt.Sprintf("Port %d nie odpowiada, a nie da się go zająć (zajęty lub zablokowany).", port)
	}
	sluchacz.Close() // celowo: nasłuch próbny, zwalniamy od razu
	return false, fmt.Sprintf("Port %d jest wolny, nic na nim nie nasłuchuje.", port)
}

// wykonawcaPolecen uruchamia program i zwraca połączone wyjście (testy podmieniają atrapą).
var wykonawcaPolecen = func(program string, argumenty ...string) ([]byte, error) {
	return exec.Command(program, argumenty...).CombinedOutput()
}

var wzorNazwyReguly = regexp.MustCompile(`^[\p{L}\p{N}_.\-]+$`)

// RegulaZapory dodaje regułę Zapory Windows dla portów (wymaga uprawnień admina; zwraca opis/błąd).
// Reguła o tej samej nazwie jest najpierw usuwana, żeby nie mnożyć duplikatów.
func RegulaZapory(nazwa string, porty []int) error {
	argumenty, err := poleceniaZapory(nazwa, porty)
	if err != nil {
		return err
	}
	// celowo: usunięcie może się nie udać, gdy reguły jeszcze nie ma
	_, _ = wykonawcaPolecen(konfiguracja.ProgramZapory, argumenty[0]...)
	wyjscie, err := wykonawcaPolecen(konfiguracja.ProgramZapory, argumenty[1]...)
	if err != nil {
		return fmt.Errorf("Nie udało się dodać reguły Zapory (uruchom jako administrator): %s", dekodujWyjscie(wyjscie))
	}
	return nil
}

// poleceniaZapory zwraca argumenty netsh: [0] usunięcie starej reguły, [1] dodanie nowej.
func poleceniaZapory(nazwa string, porty []int) ([2][]string, error) {
	nazwa = strings.TrimSpace(nazwa)
	if nazwa == "" {
		nazwa = konfiguracja.DomyslnaNazwaRegulyZapory
	}
	if utf8.RuneCountInString(nazwa) > konfiguracja.MaksDlugoscNazwyRegulyZapory || !wzorNazwyReguly.MatchString(nazwa) {
		return [2][]string{}, errors.New("Niepoprawna nazwa reguły Zapory (dozwolone litery, cyfry, _ . -).")
	}
	lista, err := listaPortow(porty)
	if err != nil {
		return [2][]string{}, err
	}
	usun := []string{"advfirewall", "firewall", "delete", "rule", "name=" + nazwa}
	dodaj := []string{"advfirewall", "firewall", "add", "rule", "name=" + nazwa,
		"dir=in", "action=allow", "protocol=TCP", "localport=" + lista, "profile=any"}
	return [2][]string{usun, dodaj}, nil
}

// listaPortow: porty posortowane, bez powtórzeń, kolejne zwinięte w zakresy (50000-50100).
func listaPortow(porty []int) (string, error) {
	if len(porty) == 0 {
		return "", errors.New("Brak portów dla reguły Zapory.")
	}
	p := append([]int(nil), porty...)
	sort.Ints(p)
	for _, x := range p {
		if x < 1 || x > 65535 {
			return "", fmt.Errorf("Niepoprawny numer portu: %d.", x)
		}
	}
	czesci := []string{}
	for i := 0; i < len(p); {
		j := i
		for j+1 < len(p) && p[j+1] <= p[j]+1 {
			j++
		}
		if p[i] == p[j] {
			czesci = append(czesci, strconv.Itoa(p[i]))
		} else {
			czesci = append(czesci, strconv.Itoa(p[i])+"-"+strconv.Itoa(p[j]))
		}
		i = j + 1
	}
	return strings.Join(czesci, ","), nil
}

// dekodujWyjscie: netsh pisze w stronie OEM; poprawny UTF-8 zostawiamy, inaczej dekodujemy CP852.
func dekodujWyjscie(dane []byte) string {
	if utf8.Valid(dane) {
		return strings.TrimSpace(string(dane))
	}
	tekst, err := charmap.CodePage852.NewDecoder().Bytes(dane)
	if err != nil {
		return strings.TrimSpace(string(dane))
	}
	return strings.TrimSpace(string(tekst))
}
