package konfiguracja

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// TrybSieci: jak klienci dochodzą do serwera.
type TrybSieci string

const (
	// TrybVPN: słucha tylko na adresach LAN / tunelu WireGuard, z internetu niewidoczny.
	TrybVPN TrybSieci = "vpn"
	// TrybPubliczny: port otwarty na routerze, wymuszone HTTPS i FTPS.
	TrybPubliczny TrybSieci = "publiczny"
)

// Udzial to folder na dysku udostępniony pod nazwą.
type Udzial struct {
	Nazwa       string `json:"nazwa"`
	Sciezka     string `json:"sciezka"`
	TylkoOdczyt bool   `json:"tylko_odczyt"`
}

// Konfiguracja całej aplikacji (dane/konfiguracja.json).
type Konfiguracja struct {
	TrybSieci TrybSieci `json:"tryb_sieci"`
	// AdresNasluchu: pusty = wybór automatyczny wg trybu (vpn: adresy LAN i WireGuard).
	AdresNasluchu string   `json:"adres_nasluchu"`
	PortHTTP      int      `json:"port_http"`
	PortHTTPS     int      `json:"port_https"`
	PortFTPS      int      `json:"port_ftps"`
	PortyPasywne  [2]int   `json:"porty_pasywne_ftp"`
	FTPSWlaczone  bool     `json:"ftps_wlaczone"`
	Udzialy       []Udzial `json:"udzialy"`
	// Certyfikat i Klucz: puste = certyfikat własny generowany w dane/certyfikaty.
	Certyfikat string `json:"certyfikat"`
	Klucz      string `json:"klucz"`
	// DomenaLetsEncrypt: w trybie publicznym, puste = certyfikat własny.
	DomenaLetsEncrypt string `json:"domena_lets_encrypt"`
	Motyw             string `json:"motyw"`
}

// Domyślne porty i motyw.
const (
	DomyslnyPortHTTP       = 8080
	DomyslnyPortHTTPS      = 8443
	DomyslnyPortFTPS       = 2121
	DomyslnyPasywnyOd      = 50000
	DomyslnyPasywnyDo      = 50100
	DomyslnyMotyw          = "ciemny"
	UprawnieniaPlikuDanych = 0o600
	UprawnieniaKatalogu    = 0o700
)

// Domyslna zwraca konfigurację pierwszego uruchomienia (tryb VPN, bez udziałów).
func Domyslna() Konfiguracja {
	return Konfiguracja{
		TrybSieci:    TrybVPN,
		PortHTTP:     DomyslnyPortHTTP,
		PortHTTPS:    DomyslnyPortHTTPS,
		PortFTPS:     DomyslnyPortFTPS,
		PortyPasywne: [2]int{DomyslnyPasywnyOd, DomyslnyPasywnyDo},
		FTPSWlaczone: true,
		Udzialy:      []Udzial{},
		Motyw:        DomyslnyMotyw,
	}
}

// Wczytaj czyta plik konfiguracji; brak pliku = konfiguracja domyślna bez błędu.
func Wczytaj(sciezka string) (Konfiguracja, error) {
	k := Domyslna()
	dane, err := os.ReadFile(sciezka)
	if errors.Is(err, os.ErrNotExist) {
		return k, nil
	}
	if err != nil {
		return k, err
	}
	if err := json.Unmarshal(dane, &k); err != nil {
		return Domyslna(), err
	}
	return k, nil
}

// Zapisz zapisuje konfigurację atomowo (plik tymczasowy + zamiana).
func Zapisz(sciezka string, k Konfiguracja) error {
	dane, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return err
	}
	return ZapiszAtomowo(sciezka, dane)
}

// ZapiszAtomowo: wspólna funkcja zapisu plików stanu (konta, udostępnienia, konfiguracja).
func ZapiszAtomowo(sciezka string, dane []byte) error {
	if err := os.MkdirAll(filepath.Dir(sciezka), UprawnieniaKatalogu); err != nil {
		return err
	}
	tmp := sciezka + ".tmp"
	if err := os.WriteFile(tmp, dane, UprawnieniaPlikuDanych); err != nil {
		return err
	}
	return os.Rename(tmp, sciezka)
}
