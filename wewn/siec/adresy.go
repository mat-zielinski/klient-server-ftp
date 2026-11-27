// Pakiet siec: adresy, tryby sieci, TLS (certyfikat własny / Let's Encrypt / z pliku), test portu,
// reguła Zapory Windows.
package siec

import (
	"net"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// Adres interfejsu do pokazania na Pulpicie i w menu „Kopiuj link”.
type Adres struct {
	Interfejs string `json:"interfejs"`
	IP        string `json:"ip"`
	Rodzaj    string `json:"rodzaj"` // "lan" | "wireguard" | "publiczny" | "petla"
}

// interfejsSieci: to, czego potrzebujemy z net.Interface (wstrzykiwane w testach).
type interfejsSieci struct {
	Nazwa  string
	Opis   string
	WGorze bool
	Petla  bool
	IP     []net.IP
}

// zrodloInterfejsow zwraca interfejsy systemu; testy podmieniają.
var zrodloInterfejsow = interfejsySystemu

// AdresyLokalne: wszystkie IPv4 interfejsów w górze; WireGuard rozpoznany po nazwie/opisie interfejsu.
func AdresyLokalne() []Adres {
	interfejsy, err := zrodloInterfejsow()
	if err != nil {
		return []Adres{}
	}
	return adresyZInterfejsow(interfejsy)
}

// adresyZInterfejsow: czysta logika wyboru i klasyfikacji adresów.
func adresyZInterfejsow(interfejsy []interfejsSieci) []Adres {
	wynik := []Adres{}
	for _, inf := range interfejsy {
		if !inf.WGorze {
			continue
		}
		for _, ip := range inf.IP {
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() || ip4.IsUnspecified() {
				continue
			}
			wynik = append(wynik, Adres{Interfejs: inf.Nazwa, IP: ip4.String(), Rodzaj: rodzajAdresu(inf, ip4)})
		}
	}
	return wynik
}

// rodzajAdresu: pętla, WireGuard (tylko po nazwie/opisie), prywatny = lan, reszta = publiczny.
func rodzajAdresu(inf interfejsSieci, ip net.IP) string {
	switch {
	case ip.IsLoopback() || inf.Petla:
		return konfiguracja.RodzajPetla
	case czyWireGuard(inf.Nazwa, inf.Opis):
		return konfiguracja.RodzajWireGuard
	case ip.IsPrivate():
		return konfiguracja.RodzajLAN
	default:
		return konfiguracja.RodzajPubliczny
	}
}

// czyWireGuard sprawdza nazwę i opis interfejsu ze wzorcami z konfiguracji.
func czyWireGuard(nazwa, opis string) bool {
	n, o := strings.ToLower(nazwa), strings.ToLower(opis)
	for _, w := range konfiguracja.WzorceWireGuardZawiera {
		if strings.Contains(n, w) || strings.Contains(o, w) {
			return true
		}
	}
	for _, w := range konfiguracja.WzorceWireGuardPrzedrostek {
		if strings.HasPrefix(n, w) {
			return true
		}
	}
	return false
}

// interfejsySystemu czyta net.Interfaces i uzupełnia opisy (na Windows z GetAdaptersAddresses).
func interfejsySystemu() ([]interfejsSieci, error) {
	lista, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	opisy := opisyInterfejsow()
	wynik := make([]interfejsSieci, 0, len(lista))
	for _, inf := range lista {
		adresy, err := inf.Addrs()
		if err != nil {
			continue // celowo: interfejs bez czytelnych adresów pomijamy, reszta dalej działa
		}
		is := interfejsSieci{
			Nazwa:  inf.Name,
			Opis:   opisy[inf.Index],
			WGorze: inf.Flags&net.FlagUp != 0,
			Petla:  inf.Flags&net.FlagLoopback != 0,
		}
		for _, a := range adresy {
			if ipn, ok := a.(*net.IPNet); ok {
				is.IP = append(is.IP, ipn.IP)
			}
		}
		wynik = append(wynik, is)
	}
	return wynik, nil
}

// AdresyNasluchu wg trybu: vpn = LAN + WireGuard + 127.0.0.1; publiczny = 0.0.0.0;
// AdresNasluchu w konfiguracji ma pierwszeństwo. Zwraca same IP (bez portu).
func AdresyNasluchu(k konfiguracja.Konfiguracja) []string {
	if a := strings.TrimSpace(k.AdresNasluchu); a != "" {
		return []string{a}
	}
	if k.TrybSieci == konfiguracja.TrybPubliczny {
		return []string{konfiguracja.AdresWszystkie}
	}
	wynik := []string{}
	widziane := map[string]bool{}
	dodaj := func(ip string) {
		if !widziane[ip] {
			widziane[ip] = true
			wynik = append(wynik, ip)
		}
	}
	for _, a := range AdresyLokalne() {
		if a.Rodzaj == konfiguracja.RodzajLAN || a.Rodzaj == konfiguracja.RodzajWireGuard {
			dodaj(a.IP)
		}
	}
	dodaj(konfiguracja.AdresPetli)
	return wynik
}
