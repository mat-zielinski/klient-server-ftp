package siec

import (
	"errors"
	"net"
	"reflect"
	"testing"

	"skrytka/wewn/konfiguracja"
)

// atrapaInterfejsow: LAN, WireGuard (po nazwie i po opisie), pętla, interfejs w dole, IPv6, link-local.
func atrapaInterfejsow() ([]interfejsSieci, error) {
	return []interfejsSieci{
		{Nazwa: "Ethernet", WGorze: true, IP: []net.IP{net.ParseIP("192.168.0.15"), net.ParseIP("fe80::1")}},
		{Nazwa: "wg0", WGorze: true, IP: []net.IP{net.ParseIP("10.10.10.6")}},
		{Nazwa: "Dom", Opis: "WireGuard Tunnel", WGorze: true, IP: []net.IP{net.ParseIP("10.10.20.2")}},
		{Nazwa: "Loopback Pseudo-Interface 1", WGorze: true, Petla: true, IP: []net.IP{net.ParseIP("127.0.0.1")}},
		{Nazwa: "Wi-Fi", WGorze: false, IP: []net.IP{net.ParseIP("192.168.1.20")}},
		{Nazwa: "APIPA", WGorze: true, IP: []net.IP{net.ParseIP("169.254.3.4")}},
		{Nazwa: "Modem", WGorze: true, IP: []net.IP{net.ParseIP("83.1.2.3")}},
		// 10.10.10.x bez nazwy WireGuard to zwykły LAN (rozpoznanie tylko po nazwie/opisie).
		{Nazwa: "Ethernet 2", WGorze: true, IP: []net.IP{net.ParseIP("10.10.10.7")}},
	}, nil
}

func podmienInterfejsy(t *testing.T, f func() ([]interfejsSieci, error)) {
	t.Helper()
	stare := zrodloInterfejsow
	zrodloInterfejsow = f
	t.Cleanup(func() { zrodloInterfejsow = stare })
}

func TestAdresyLokalne(t *testing.T) {
	podmienInterfejsy(t, atrapaInterfejsow)
	chce := []Adres{
		{"Ethernet", "192.168.0.15", konfiguracja.RodzajLAN},
		{"wg0", "10.10.10.6", konfiguracja.RodzajWireGuard},
		{"Dom", "10.10.20.2", konfiguracja.RodzajWireGuard},
		{"Loopback Pseudo-Interface 1", "127.0.0.1", konfiguracja.RodzajPetla},
		{"Modem", "83.1.2.3", konfiguracja.RodzajPubliczny},
		{"Ethernet 2", "10.10.10.7", konfiguracja.RodzajLAN},
	}
	if got := AdresyLokalne(); !reflect.DeepEqual(got, chce) {
		t.Fatalf("AdresyLokalne = %+v, chcę %+v", got, chce)
	}
}

func TestAdresyLokalneBladSystemu(t *testing.T) {
	podmienInterfejsy(t, func() ([]interfejsSieci, error) { return nil, errors.New("awaria") })
	if got := AdresyLokalne(); got == nil || len(got) != 0 {
		t.Fatalf("przy błędzie chcę pustą listę (nie nil), jest %+v", got)
	}
}

func TestCzyWireGuard(t *testing.T) {
	przypadki := []struct {
		nazwa, opis string
		chce        bool
	}{
		{"wg0", "", true},
		{"WG-dom", "", true},
		{"MojTunelWireGuard", "", true},
		{"Dom", "WireGuard Tunnel", true},
		{"Ethernet", "Intel(R) Ethernet", false},
		{"Wi-Fi", "Intel Wireless", false},
		{"Ethernet wg", "", false}, // „wg” tylko jako przedrostek nazwy
	}
	for _, p := range przypadki {
		if got := czyWireGuard(p.nazwa, p.opis); got != p.chce {
			t.Errorf("czyWireGuard(%q,%q) = %v, chcę %v", p.nazwa, p.opis, got, p.chce)
		}
	}
}

func TestAdresyNasluchu(t *testing.T) {
	podmienInterfejsy(t, atrapaInterfejsow)
	przypadki := []struct {
		nazwa string
		k     konfiguracja.Konfiguracja
		chce  []string
	}{
		{"vpn", konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybVPN},
			[]string{"192.168.0.15", "10.10.10.6", "10.10.20.2", "10.10.10.7", "127.0.0.1"}},
		{"publiczny", konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybPubliczny}, []string{"0.0.0.0"}},
		{"adres z konfiguracji wygrywa", konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybPubliczny, AdresNasluchu: " 192.168.0.15 "},
			[]string{"192.168.0.15"}},
		{"pusty tryb jak vpn", konfiguracja.Konfiguracja{},
			[]string{"192.168.0.15", "10.10.10.6", "10.10.20.2", "10.10.10.7", "127.0.0.1"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if got := AdresyNasluchu(p.k); !reflect.DeepEqual(got, p.chce) {
				t.Fatalf("AdresyNasluchu = %v, chcę %v", got, p.chce)
			}
		})
	}
}

// Bezpieczeństwo: tryb VPN nigdy nie słucha na adresie publicznym ani na 0.0.0.0.
func TestAdresyNasluchuVPNBezPublicznych(t *testing.T) {
	podmienInterfejsy(t, atrapaInterfejsow)
	for _, a := range AdresyNasluchu(konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybVPN}) {
		if a == "83.1.2.3" || a == konfiguracja.AdresWszystkie {
			t.Fatalf("tryb VPN słucha na %s", a)
		}
	}
}

// Prawdziwe interfejsy: tylko sprawdzenie, że nie wybucha i są to IPv4.
func TestAdresyLokalneSystem(t *testing.T) {
	for _, a := range AdresyLokalne() {
		if ip := net.ParseIP(a.IP); ip == nil || ip.To4() == nil {
			t.Errorf("adres %+v nie jest IPv4", a)
		}
	}
}
