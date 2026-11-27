package siec

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

var czasTestu = time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)

// przygotujKatalog: SKRYTKA_KATALOG na t.TempDir, stały zegar i atrapa interfejsów.
func przygotujKatalog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(konfiguracja.ZmiennaKataloguAplikacji, dir)
	stary := zegar
	zegar = func() time.Time { return czasTestu }
	t.Cleanup(func() { zegar = stary })
	podmienInterfejsy(t, atrapaInterfejsow)
	return dir
}

func certyfikatZ(t *testing.T, c *tls.Config) *x509.Certificate {
	t.Helper()
	x, err := x509.ParseCertificate(c.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestCertyfikatWlasnyGenerowanyIUzywanyPonownie(t *testing.T) {
	przygotujKatalog(t)
	k := konfiguracja.Domyslna()
	c1, err := KonfiguracjaTLS(k)
	if err != nil {
		t.Fatal(err)
	}
	if c1.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %x, chcę TLS 1.2", c1.MinVersion)
	}
	x := certyfikatZ(t, c1)
	for _, n := range []string{"localhost", "192.168.0.15", "10.10.10.6", "127.0.0.1"} {
		if err := x.VerifyHostname(n); err != nil {
			t.Errorf("SAN nie obejmuje %s: %v", n, err)
		}
	}
	if c1.Certificates[0].PrivateKey == nil {
		t.Fatal("brak klucza prywatnego")
	}
	if !x.NotAfter.After(czasTestu.Add(9 * 365 * 24 * time.Hour)) {
		t.Fatalf("certyfikat ważny tylko do %v", x.NotAfter)
	}
	plikKey := filepath.Join(konfiguracja.KatalogCertyfikatowPelny(), konfiguracja.PlikKluczaWlasnego)
	if _, err := os.Stat(plikKey); err != nil {
		t.Fatalf("brak pliku klucza: %v", err)
	}
	c2, err := KonfiguracjaTLS(k)
	if err != nil {
		t.Fatal(err)
	}
	if OdciskCertyfikatu(c1) != OdciskCertyfikatu(c2) {
		t.Fatal("drugie uruchomienie wygenerowało nowy certyfikat zamiast użyć zapisanego")
	}
}

func TestCertyfikatWlasnyOdnawianyPrzyNowymAdresie(t *testing.T) {
	przygotujKatalog(t)
	c1, err := KonfiguracjaTLS(konfiguracja.Domyslna())
	if err != nil {
		t.Fatal(err)
	}
	podmienInterfejsy(t, func() ([]interfejsSieci, error) {
		l, _ := atrapaInterfejsow()
		return append(l, interfejsSieci{Nazwa: "Nowy", WGorze: true, IP: []net.IP{net.ParseIP("192.168.0.99")}}), nil
	})
	c2, err := KonfiguracjaTLS(konfiguracja.Domyslna())
	if err != nil {
		t.Fatal(err)
	}
	if OdciskCertyfikatu(c1) == OdciskCertyfikatu(c2) {
		t.Fatal("nowy adres, a certyfikat nie został odnowiony")
	}
	if err := certyfikatZ(t, c2).VerifyHostname("192.168.0.99"); err != nil {
		t.Fatal(err)
	}
}

func TestCertyfikatAktualny(t *testing.T) {
	katalog := t.TempDir()
	nazwy := []string{"localhost", "192.168.0.15"}
	para, err := certyfikatWlasny(katalog, nazwy, czasTestu)
	if err != nil {
		t.Fatal(err)
	}
	przypadki := []struct {
		nazwa string
		nazwy []string
		teraz time.Time
		chce  bool
	}{
		{"świeży", nazwy, czasTestu, true},
		{"podzbiór nazw", []string{"localhost"}, czasTestu, true},
		{"brak adresu", append(nazwy, "10.0.0.1"), czasTestu, false},
		{"tuż przed końcem", nazwy, czasTestu.Add(konfiguracja.WaznoscCertyfikatuWlasnego - time.Hour), false},
		{"przed ważnością", nazwy, czasTestu.Add(-2 * konfiguracja.CofniecieWaznosciCertyfikatu), false},
	}
	for _, p := range przypadki {
		if got := certyfikatAktualny(para, p.nazwy, p.teraz); got != p.chce {
			t.Errorf("%s: certyfikatAktualny = %v, chcę %v", p.nazwa, got, p.chce)
		}
	}
	if certyfikatAktualny(tls.Certificate{}, nazwy, czasTestu) {
		t.Error("pusty certyfikat uznany za aktualny")
	}
}

func TestCertyfikatZPliku(t *testing.T) {
	przygotujKatalog(t)
	crt, key, err := generujCertyfikat([]string{"serwer.example"}, czasTestu)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	plikCrt, plikKey := filepath.Join(dir, "a.crt"), filepath.Join(dir, "a.key")
	if err := os.WriteFile(plikCrt, crt, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plikKey, key, 0o600); err != nil {
		t.Fatal(err)
	}
	k := konfiguracja.Domyslna()
	k.Certyfikat, k.Klucz = plikCrt, plikKey
	c, err := KonfiguracjaTLS(k)
	if err != nil {
		t.Fatal(err)
	}
	if err := certyfikatZ(t, c).VerifyHostname("serwer.example"); err != nil {
		t.Fatal("wczytano nie ten certyfikat")
	}
	k.Klucz = ""
	if _, err := KonfiguracjaTLS(k); err == nil {
		t.Fatal("sam certyfikat bez klucza powinien dać błąd")
	}
	k.Klucz = filepath.Join(dir, "brak.key")
	if _, err := KonfiguracjaTLS(k); err == nil {
		t.Fatal("brakujący plik klucza powinien dać błąd")
	}
}

func TestKonfiguracjaLetsEncrypt(t *testing.T) {
	przygotujKatalog(t)
	k := konfiguracja.Domyslna()
	k.TrybSieci, k.DomenaLetsEncrypt = konfiguracja.TrybPubliczny, "pliki.example.com"
	c, err := KonfiguracjaTLS(k)
	if err != nil {
		t.Fatal(err)
	}
	if c.GetCertificate == nil || len(c.Certificates) != 0 {
		t.Fatal("tryb publiczny z domeną powinien używać autocert")
	}
	if c.MinVersion != tls.VersionTLS12 {
		t.Fatal("autocert bez minimum TLS 1.2")
	}
	if _, err := os.Stat(filepath.Join(konfiguracja.KatalogCertyfikatowPelny(), konfiguracja.KatalogACME)); err != nil {
		t.Fatalf("brak katalogu pamięci ACME: %v", err)
	}
	// Domena w trybie VPN jest pomijana: certyfikat własny.
	k.TrybSieci = konfiguracja.TrybVPN
	c, err = KonfiguracjaTLS(k)
	if err != nil || len(c.Certificates) != 1 {
		t.Fatalf("tryb VPN z domeną: chcę certyfikat własny, err=%v", err)
	}
}

func TestOdciskCertyfikatu(t *testing.T) {
	katalog := t.TempDir()
	para, err := certyfikatWlasny(katalog, []string{"localhost"}, czasTestu)
	if err != nil {
		t.Fatal(err)
	}
	c := &tls.Config{Certificates: []tls.Certificate{para}}
	odcisk := OdciskCertyfikatu(c)
	if !regexp.MustCompile(`^([0-9A-F]{2}:){31}[0-9A-F]{2}$`).MatchString(odcisk) {
		t.Fatalf("zły format odcisku: %q", odcisk)
	}
	suma := sha256.Sum256(para.Certificate[0])
	if odcisk[:5] != hexBajt(suma[0])+":"+hexBajt(suma[1]) {
		t.Fatalf("odcisk %q nie pasuje do SHA-256", odcisk)
	}
	for _, pusty := range []*tls.Config{nil, {}, {Certificates: []tls.Certificate{{}}}} {
		if OdciskCertyfikatu(pusty) != "" {
			t.Error("brak certyfikatu powinien dać pusty odcisk")
		}
	}
}

func hexBajt(b byte) string {
	const cyfry = "0123456789ABCDEF"
	return string([]byte{cyfry[b>>4], cyfry[b&0xf]})
}
