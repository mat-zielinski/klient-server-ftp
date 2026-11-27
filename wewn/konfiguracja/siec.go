package konfiguracja

import "time"

// Stałe obszaru „siec”: adresy, TLS, test portu, Zapora Windows, IP publiczne.

// Usługi zwracające adres IP publiczny jako zwykły tekst (pytane po kolei do pierwszej udanej).
var UslugiAdresuPublicznego = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://checkip.amazonaws.com",
}

const (
	// LimitCzasuAdresuPublicznego: łączny limit jednego zapytania o IP publiczne.
	LimitCzasuAdresuPublicznego = 3 * time.Second
	// MaksOdpowiedziAdresuPublicznego: tyle bajtów odpowiedzi czytamy najwyżej.
	MaksOdpowiedziAdresuPublicznego = 64

	// LimitCzasuTestuPortu: ile czekamy na połączenie z 127.0.0.1 przy teście portu.
	LimitCzasuTestuPortu = 1 * time.Second
	// AdresTestuPortu: test portu idzie tylko na pętlę zwrotną.
	AdresTestuPortu = "127.0.0.1"

	// DomyslnaNazwaRegulyZapory: domyślna nazwa reguły Zapory Windows (bez spacji).
	DomyslnaNazwaRegulyZapory = "Skrytka"
	// MaksDlugoscNazwyRegulyZapory: dłuższa nazwa jest odrzucana.
	MaksDlugoscNazwyRegulyZapory = 64
	// ProgramZapory: polecenie Windows do zarządzania Zaporą.
	ProgramZapory = "netsh"

	// WaznoscCertyfikatuWlasnego: certyfikat własny ważny 10 lat.
	WaznoscCertyfikatuWlasnego = 10 * 365 * 24 * time.Hour
	// ZapasOdnowieniaCertyfikatu: odnawiamy, gdy do końca ważności zostało mniej niż tyle.
	ZapasOdnowieniaCertyfikatu = 30 * 24 * time.Hour
	// CofniecieWaznosciCertyfikatu: „ważny od” cofnięte o tyle (rozjechane zegary klientów).
	CofniecieWaznosciCertyfikatu = time.Hour
	// PlikCertyfikatuWlasnego i PlikKluczaWlasnego w dane/certyfikaty.
	PlikCertyfikatuWlasnego = "wlasny.crt"
	PlikKluczaWlasnego      = "wlasny.key"
	// KatalogACME: pamięć podręczna Let's Encrypt (autocert) w dane/certyfikaty.
	KatalogACME = "acme"
	// NazwaHostaLokalnego dopisywana zawsze do SAN certyfikatu własnego.
	NazwaHostaLokalnego = "localhost"
	// BityNumeruSeryjnego certyfikatu własnego.
	BityNumeruSeryjnego = 128
	// AdresPetli dopisywany do nasłuchu w trybie VPN.
	AdresPetli = "127.0.0.1"
	// AdresWszystkie: nasłuch na wszystkich interfejsach (tryb publiczny).
	AdresWszystkie = "0.0.0.0"
)

// Rodzaje adresów (pole Adres.Rodzaj).
const (
	RodzajLAN       = "lan"
	RodzajWireGuard = "wireguard"
	RodzajPubliczny = "publiczny"
	RodzajPetla     = "petla"
)

// WzorceWireGuardZawiera: nazwa lub opis interfejsu zawierający którykolwiek (bez wielkości liter) = WireGuard.
var WzorceWireGuardZawiera = []string{"wireguard"}

// WzorceWireGuardPrzedrostek: nazwa interfejsu zaczynająca się od któregokolwiek = WireGuard (np. wg0).
var WzorceWireGuardPrzedrostek = []string{"wg"}

const (
	// RozmiarBuforaAdapterow: początkowy bufor GetAdaptersAddresses (Windows, opisy interfejsów).
	RozmiarBuforaAdapterow = 15000
	// ProbyBuforaAdapterow: ile razy powiększamy bufor, gdy system zgłosi przepełnienie.
	ProbyBuforaAdapterow = 3
)
