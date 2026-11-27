package siec

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"skrytka/wewn/konfiguracja"
)

// zegar: czas teraźniejszy (testy podmieniają).
var zegar = time.Now

// KonfiguracjaTLS: plik z konfiguracji, Let's Encrypt (autocert, tryb publiczny z domeną) albo
// certyfikat własny generowany raz do dane/certyfikaty (ECDSA P-256, SAN: wszystkie adresy lokalne).
func KonfiguracjaTLS(k konfiguracja.Konfiguracja) (*tls.Config, error) {
	if k.Certyfikat != "" || k.Klucz != "" {
		if k.Certyfikat == "" || k.Klucz == "" {
			return nil, errors.New("Podaj w konfiguracji jednocześnie plik certyfikatu i plik klucza.")
		}
		para, err := tls.LoadX509KeyPair(k.Certyfikat, k.Klucz)
		if err != nil {
			return nil, fmt.Errorf("Nie udało się wczytać certyfikatu z pliku: %w", err)
		}
		return bazowaKonfiguracjaTLS(para), nil
	}
	katalog := konfiguracja.KatalogCertyfikatowPelny()
	domena := strings.TrimSpace(k.DomenaLetsEncrypt)
	if k.TrybSieci == konfiguracja.TrybPubliczny && domena != "" {
		return konfiguracjaACME(katalog, domena)
	}
	para, err := certyfikatWlasny(katalog, nazwyCertyfikatu(), zegar())
	if err != nil {
		return nil, err
	}
	return bazowaKonfiguracjaTLS(para), nil
}

// bazowaKonfiguracjaTLS: TLS 1.2+, tylko AEAD z ECDHE (TLS 1.3 ma swoje szyfry, wszystkie nowoczesne).
func bazowaKonfiguracjaTLS(para tls.Certificate) *tls.Config {
	c := nowoczesnyTLS()
	c.Certificates = []tls.Certificate{para}
	return c
}

func nowoczesnyTLS() *tls.Config {
	return &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
}

// konfiguracjaACME: Let's Encrypt przez autocert (wyzwanie TLS-ALPN-01, czyli port 443 z internetu).
func konfiguracjaACME(katalog, domena string) (*tls.Config, error) {
	cache := filepath.Join(katalog, konfiguracja.KatalogACME)
	if err := os.MkdirAll(cache, konfiguracja.UprawnieniaKatalogu); err != nil {
		return nil, err
	}
	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domena),
		Cache:      autocert.DirCache(cache),
	}
	c := nowoczesnyTLS()
	c.GetCertificate = m.GetCertificate
	c.NextProtos = m.TLSConfig().NextProtos // h2, http/1.1 i acme-tls/1
	return c, nil
}

// nazwyCertyfikatu: wszystkie adresy lokalne + localhost.
func nazwyCertyfikatu() []string {
	nazwy := []string{konfiguracja.NazwaHostaLokalnego}
	for _, a := range AdresyLokalne() {
		nazwy = append(nazwy, a.IP)
	}
	return nazwy
}

// certyfikatWlasny wczytuje zapisany certyfikat; gdy brak, wygasa albo nie obejmuje którejś nazwy,
// generuje nowy (ECDSA P-256) i zapisuje (klucz 0600).
func certyfikatWlasny(katalog string, nazwy []string, teraz time.Time) (tls.Certificate, error) {
	plikCrt := filepath.Join(katalog, konfiguracja.PlikCertyfikatuWlasnego)
	plikKey := filepath.Join(katalog, konfiguracja.PlikKluczaWlasnego)
	if para, err := tls.LoadX509KeyPair(plikCrt, plikKey); err == nil && certyfikatAktualny(para, nazwy, teraz) {
		return para, nil
	}
	crtPEM, keyPEM, err := generujCertyfikat(nazwy, teraz)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := os.MkdirAll(katalog, konfiguracja.UprawnieniaKatalogu); err != nil {
		return tls.Certificate{}, err
	}
	if err := konfiguracja.ZapiszAtomowo(plikKey, keyPEM); err != nil {
		return tls.Certificate{}, err
	}
	if err := konfiguracja.ZapiszAtomowo(plikCrt, crtPEM); err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(crtPEM, keyPEM)
}

// certyfikatAktualny: ważny jeszcze dłużej niż zapas odnowienia i obejmuje wszystkie nazwy.
func certyfikatAktualny(para tls.Certificate, nazwy []string, teraz time.Time) bool {
	if len(para.Certificate) == 0 {
		return false
	}
	c, err := x509.ParseCertificate(para.Certificate[0])
	if err != nil {
		return false
	}
	if teraz.Before(c.NotBefore) || teraz.Add(konfiguracja.ZapasOdnowieniaCertyfikatu).After(c.NotAfter) {
		return false
	}
	for _, n := range nazwy {
		if c.VerifyHostname(n) != nil {
			return false
		}
	}
	return true
}

// generujCertyfikat tworzy samopodpisany certyfikat serwera; zwraca PEM certyfikatu i klucza.
func generujCertyfikat(nazwy []string, teraz time.Time) ([]byte, []byte, error) {
	klucz, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	numer, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), konfiguracja.BityNumeruSeryjnego))
	if err != nil {
		return nil, nil, err
	}
	szablon := &x509.Certificate{
		SerialNumber:          numer,
		Subject:               pkix.Name{CommonName: konfiguracja.NazwaAplikacji, Organization: []string{konfiguracja.NazwaAplikacji}},
		NotBefore:             teraz.Add(-konfiguracja.CofniecieWaznosciCertyfikatu),
		NotAfter:              teraz.Add(konfiguracja.WaznoscCertyfikatuWlasnego),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, n := range nazwy {
		if ip := net.ParseIP(n); ip != nil {
			szablon.IPAddresses = append(szablon.IPAddresses, ip)
		} else {
			szablon.DNSNames = append(szablon.DNSNames, n)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, szablon, szablon, &klucz.PublicKey, klucz)
	if err != nil {
		return nil, nil, err
	}
	kluczDER, err := x509.MarshalPKCS8PrivateKey(klucz)
	if err != nil {
		return nil, nil, err
	}
	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kluczDER})
	return crtPEM, keyPEM, nil
}

// OdciskCertyfikatu SHA-256 (pokazywany klientowi do sprawdzenia przy pierwszym wejściu).
// Format AB:CD:…; brak certyfikatu stałego (np. Let's Encrypt wydawany w locie) = "".
func OdciskCertyfikatu(c *tls.Config) string {
	if c == nil || len(c.Certificates) == 0 || len(c.Certificates[0].Certificate) == 0 {
		return ""
	}
	suma := sha256.Sum256(c.Certificates[0].Certificate[0])
	czesci := make([]string, len(suma))
	for i, b := range suma {
		czesci[i] = strings.ToUpper(hex.EncodeToString([]byte{b}))
	}
	return strings.Join(czesci, ":")
}
