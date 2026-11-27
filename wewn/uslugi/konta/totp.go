package konta

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238: HMAC-SHA1 to standard aplikacji Authenticator
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"skrytka/wewn/konfiguracja"
)

var (
	base32BezDopelnienia = base32.StdEncoding.WithPadding(base32.NoPadding)
	errZlySekret         = errors.New("nieprawidłowy sekret 2FA")
)

// nowySekret: TOTPDlugoscSekretu losowych bajtów w base32 bez dopełnienia.
func nowySekret() (string, error) {
	b := make([]byte, konfiguracja.TOTPDlugoscSekretu)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32BezDopelnienia.EncodeToString(b), nil
}

// dekodujSekret czyta sekret base32 (wielkość liter i dopełnienie bez znaczenia).
func dekodujSekret(sekret string) ([]byte, error) {
	s := strings.TrimRight(strings.ToUpper(sekret), "=")
	b, err := base32BezDopelnienia.DecodeString(s)
	if err != nil || len(b) == 0 {
		return nil, errZlySekret
	}
	return b, nil
}

// kodHOTP: RFC 4226 (HMAC-SHA1, dynamiczne obcięcie), kod o podanej liczbie cyfr.
func kodHOTP(klucz []byte, licznik uint64, cyfry int) string {
	var bufor [8]byte
	binary.BigEndian.PutUint64(bufor[:], licznik)
	m := hmac.New(sha1.New, klucz)
	m.Write(bufor[:]) // celowo: hash.Hash.Write nigdy nie zwraca błędu
	suma := m.Sum(nil)
	przes := suma[len(suma)-1] & 0x0f
	wartosc := binary.BigEndian.Uint32(suma[przes:przes+4]) & 0x7fffffff
	modulo := uint32(1)
	for i := 0; i < cyfry; i++ {
		modulo *= 10
	}
	return fmt.Sprintf("%0*d", cyfry, wartosc%modulo)
}

// krokCzasu: numer kroku TOTP (czas Unix / TOTPOkres).
func krokCzasu(t time.Time) int64 {
	return t.Unix() / int64(konfiguracja.TOTPOkres/time.Second)
}

// sprawdzKodTOTP szuka kodu w oknie ±TOTPOkno kroków; zwraca dopasowany krok.
// Porównanie w stałym czasie; kod musi mieć dokładnie TOTPCyfry cyfr.
func sprawdzKodTOTP(sekret, kod string, teraz time.Time) (int64, bool) {
	if len(kod) != konfiguracja.TOTPCyfry || strings.Trim(kod, "0123456789") != "" {
		return 0, false
	}
	klucz, err := dekodujSekret(sekret)
	if err != nil {
		return 0, false
	}
	biezacy := krokCzasu(teraz)
	znaleziony, ok := int64(0), false
	for d := int64(-konfiguracja.TOTPOkno); d <= konfiguracja.TOTPOkno; d++ {
		krok := biezacy + d
		if krok < 0 {
			continue
		}
		wzor := kodHOTP(klucz, uint64(krok), konfiguracja.TOTPCyfry)
		if subtle.ConstantTimeCompare([]byte(wzor), []byte(kod)) == 1 && !ok {
			znaleziony, ok = krok, true
		}
	}
	return znaleziony, ok
}

// adresOtpauth: adres do kodu QR (otpauth://totp/Wystawca:login?secret=…).
func adresOtpauth(login, sekret string) string {
	etykieta := url.PathEscape(konfiguracja.TOTPWystawca + ":" + login)
	q := url.Values{}
	q.Set("secret", sekret)
	q.Set("issuer", konfiguracja.TOTPWystawca)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(konfiguracja.TOTPCyfry))
	q.Set("period", fmt.Sprint(int64(konfiguracja.TOTPOkres/time.Second)))
	return "otpauth://totp/" + etykieta + "?" + q.Encode()
}
