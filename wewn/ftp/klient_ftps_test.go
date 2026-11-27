package ftp

// Minimalny klient FTPS (jawny TLS) do testów integracyjnych: net/textproto + crypto/tls.
// github.com/jlaffaye/ftp nie ma w module cache, a zasady zabraniają pobierania.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"
)

// certyfikatTestowy: samopodpisany certyfikat dla 127.0.0.1 i pula zaufania klienta.
// time.Now tylko do ważności certyfikatu (nie steruje logiką testu).
func certyfikatTestowy(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	klucz, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	szablon := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "skrytka-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, szablon, szablon, &klucz.PublicKey, klucz)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pula := x509.NewCertPool()
	pula.AddCert(cert)
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: klucz}}}, pula
}

type klientFTPS struct {
	t      *testing.T
	surowe net.Conn
	c      *textproto.Conn
	cfg    *tls.Config
}

func polacz(t *testing.T, adres string, pula *x509.CertPool) *klientFTPS {
	t.Helper()
	surowe, err := net.Dial("tcp", adres)
	if err != nil {
		t.Fatal(err)
	}
	k := &klientFTPS{t: t, surowe: surowe, c: textproto.NewConn(surowe),
		cfg: &tls.Config{RootCAs: pula, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}}
	t.Cleanup(func() { _ = k.c.Close() }) // celowo: połączenie mogło już zostać zamknięte
	k.oczekuj(220)
	return k
}

// polecenie wysyła linię i zwraca kod i treść odpowiedzi (kod 0 = połączenie zamknięte).
func (k *klientFTPS) polecenie(format string, arg ...any) (int, string) {
	k.t.Helper()
	if err := k.c.PrintfLine(format, arg...); err != nil {
		return 0, err.Error()
	}
	return k.odpowiedz()
}

func (k *klientFTPS) odpowiedz() (int, string) {
	kod, tresc, err := k.c.ReadResponse(0)
	if err != nil && kod == 0 {
		return 0, err.Error()
	}
	return kod, tresc
}

func (k *klientFTPS) oczekuj(kod int) string {
	k.t.Helper()
	got, tresc := k.odpowiedz()
	if got != kod {
		k.t.Fatalf("oczekiwano %d, jest %d %s", kod, got, tresc)
	}
	return tresc
}

func (k *klientFTPS) wymagaj(kod int, format string, arg ...any) string {
	k.t.Helper()
	got, tresc := k.polecenie(format, arg...)
	if got != kod {
		k.t.Fatalf("%s: oczekiwano %d, jest %d %s", fmt.Sprintf(format, arg...), kod, got, tresc)
	}
	return tresc
}

// tls przełącza sterowanie na TLS (AUTH TLS).
func (k *klientFTPS) tls() {
	k.t.Helper()
	k.wymagaj(234, "AUTH TLS")
	tc := tls.Client(k.surowe, k.cfg)
	if err := tc.Handshake(); err != nil {
		k.t.Fatal(err)
	}
	k.c = textproto.NewConn(tc)
}

// zaloguj: AUTH TLS, USER, PASS, PBSZ, PROT P, TYPE I.
func (k *klientFTPS) zaloguj(login, haslo string) {
	k.t.Helper()
	k.tls()
	k.wymagaj(331, "USER %s", login)
	k.wymagaj(230, "PASS %s", haslo)
	k.wymagaj(200, "PBSZ 0")
	k.wymagaj(200, "PROT P")
	k.wymagaj(200, "TYPE I")
}

// dane otwiera połączenie danych (EPSV); szyfrowane = TLS na kanale danych.
func (k *klientFTPS) dane(szyfrowane bool) net.Conn {
	k.t.Helper()
	tresc := k.wymagaj(229, "EPSV")
	i, j := strings.Index(tresc, "(|||"), strings.LastIndex(tresc, "|)")
	if i < 0 || j < i {
		k.t.Fatalf("EPSV: %s", tresc)
	}
	port, err := strconv.Atoi(tresc[i+4 : j])
	if err != nil {
		k.t.Fatal(err)
	}
	c, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		k.t.Fatal(err)
	}
	if !szyfrowane {
		return c
	}
	return tls.Client(c, k.cfg)
}

// pobierz: RETR (po REST, gdy przesuniecie > 0), zwraca kod końcowy i dane.
func (k *klientFTPS) pobierz(sciezka string, przesuniecie int64) (int, []byte) {
	k.t.Helper()
	if przesuniecie > 0 {
		k.wymagaj(350, "REST %d", przesuniecie)
	}
	d := k.dane(true)
	defer d.Close()
	kod, tresc := k.polecenie("RETR %s", sciezka)
	if kod != 150 {
		return kod, []byte(tresc)
	}
	b, err := io.ReadAll(d)
	if err != nil {
		k.t.Fatal(err)
	}
	kod, _ = k.odpowiedz()
	return kod, b
}

// lista: NLST, nazwy posortowane jak przyszły.
func (k *klientFTPS) lista(sciezka string) (int, []string) {
	k.t.Helper()
	d := k.dane(true)
	defer d.Close()
	kod, tresc := k.polecenie("NLST %s", sciezka)
	if kod != 150 {
		return kod, []string{tresc}
	}
	b, err := io.ReadAll(d)
	if err != nil {
		k.t.Fatal(err)
	}
	kod, _ = k.odpowiedz()
	return kod, strings.Fields(string(b))
}

// wyslij: STOR, zwraca kod końcowy.
func (k *klientFTPS) wyslij(sciezka string, tresc []byte) int {
	k.t.Helper()
	d := k.dane(true)
	kod, _ := k.polecenie("STOR %s", sciezka)
	if kod != 150 {
		_ = d.Close() // celowo: odmowa, połączenie danych zbędne
		return kod
	}
	if _, err := d.Write(tresc); err != nil {
		k.t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		k.t.Fatal(err)
	}
	kod, _ = k.odpowiedz()
	return kod
}

// surowePodTLS: surowe połączenie danych pod TLS klienta; zapamiętuje, czy TCP doszło do EOF.
// Gdy serwer wyśle close_notify, crypto/tls kończy odczyt na alercie i nie czyta już z TCP,
// więc eof == false oznacza poprawne zakończenie TLS (EOF z samego TCP = brak close_notify).
type surowePodTLS struct {
	net.Conn
	eof bool
}

func (s *surowePodTLS) Read(b []byte) (int, error) {
	n, err := s.Conn.Read(b)
	if err == io.EOF {
		s.eof = true
	}
	return n, err
}

// wynikRETR: kod końcowy, dane, błąd odczytu i czy przyszedł close_notify.
type wynikRETR struct {
	kod         int
	dane        []byte
	blad        error
	closeNotify bool
}

// pobierzJakSchannel: RETR jak curl ze Schannel: klient dosyła rekord TLS na kanał danych (serwer go nie
// czyta), a dane odbiera dopiero po 226. Serwer zamykający gniazdo z nieprzeczytanymi danymi wysyła RST,
// które wyrzuca nieodebrane dane klienta.
func (k *klientFTPS) pobierzJakSchannel(sciezka string) wynikRETR {
	k.t.Helper()
	tresc := k.wymagaj(229, "EPSV")
	i, j := strings.Index(tresc, "(|||"), strings.LastIndex(tresc, "|)")
	port, err := strconv.Atoi(tresc[i+4 : j])
	if err != nil {
		k.t.Fatalf("EPSV: %s", tresc)
	}
	c, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		k.t.Fatal(err)
	}
	sur := &surowePodTLS{Conn: c}
	d := tls.Client(sur, k.cfg)
	defer d.Close()
	k.wymagaj(150, "RETR %s", sciezka)
	if _, err := d.Write([]byte("resztka")); err != nil {
		k.t.Fatal(err)
	}
	kod, _ := k.odpowiedz()
	b, err := io.ReadAll(d)
	return wynikRETR{kod: kod, dane: b, blad: err, closeNotify: !sur.eof}
}
