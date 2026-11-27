package konta

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Wektory RFC 6238, dodatek B (SHA1, sekret ASCII "12345678901234567890", 8 cyfr, krok 30 s).
func TestTOTPWektoryRFC6238(t *testing.T) {
	klucz := []byte("12345678901234567890")
	przypadki := []struct {
		unix int64
		kod  string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, p := range przypadki {
		krok := krokCzasu(time.Unix(p.unix, 0))
		if got := kodHOTP(klucz, uint64(krok), 8); got != p.kod {
			t.Errorf("T=%d: %s, chcę %s", p.unix, got, p.kod)
		}
		// 6 cyfr = ostatnie 6 cyfr wartości 8-cyfrowej.
		if got := kodHOTP(klucz, uint64(krok), 6); got != p.kod[2:] {
			t.Errorf("T=%d (6 cyfr): %s, chcę %s", p.unix, got, p.kod[2:])
		}
	}
}

// kodNa liczy poprawny kod dla sekretu base32 w chwili t.
func kodNa(t *testing.T, sekret string, chwila time.Time) string {
	t.Helper()
	klucz, err := dekodujSekret(sekret)
	if err != nil {
		t.Fatal(err)
	}
	return kodHOTP(klucz, uint64(krokCzasu(chwila)), konfiguracja.TOTPCyfry)
}

func TestSprawdzKodTOTPOkno(t *testing.T) {
	sekret, err := nowySekret()
	if err != nil {
		t.Fatal(err)
	}
	teraz := time.Date(2026, 11, 27, 12, 0, 15, 0, time.UTC)
	okres := konfiguracja.TOTPOkres
	przypadki := []struct {
		nazwa string
		kod   string
		chce  bool
	}{
		{"bieżący", kodNa(t, sekret, teraz), true},
		{"krok wstecz", kodNa(t, sekret, teraz.Add(-okres)), true},
		{"krok naprzód", kodNa(t, sekret, teraz.Add(okres)), true},
		{"dwa kroki wstecz", kodNa(t, sekret, teraz.Add(-2*okres)), false},
		{"dwa kroki naprzód", kodNa(t, sekret, teraz.Add(2*okres)), false},
		{"pusty", "", false},
		{"za krótki", "12345", false},
		{"litery", "12a456", false},
		{"za długi", kodNa(t, sekret, teraz) + "0", false},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			if _, ok := sprawdzKodTOTP(sekret, p.kod, teraz); ok != p.chce {
				t.Fatalf("ok=%v, chcę %v", ok, p.chce)
			}
		})
	}
	if _, ok := sprawdzKodTOTP("!!zly!!", "123456", teraz); ok {
		t.Fatal("zły sekret przyjęty")
	}
}

func TestSekretIAdresOtpauth(t *testing.T) {
	sekret, err := nowySekret()
	if err != nil {
		t.Fatal(err)
	}
	klucz, err := dekodujSekret(strings.ToLower(sekret))
	if err != nil || len(klucz) != konfiguracja.TOTPDlugoscSekretu || strings.Contains(sekret, "=") {
		t.Fatalf("sekret %q: %v", sekret, err)
	}
	adres, err := url.Parse(adresOtpauth("jan", sekret))
	if err != nil {
		t.Fatal(err)
	}
	q := adres.Query()
	if adres.Scheme != "otpauth" || adres.Host != "totp" || adres.Path != "/"+konfiguracja.TOTPWystawca+":jan" ||
		q.Get("secret") != sekret || q.Get("digits") != "6" || q.Get("period") != "30" ||
		q.Get("algorithm") != "SHA1" || q.Get("issuer") != konfiguracja.TOTPWystawca {
		t.Fatalf("zły adres otpauth: %s", adres)
	}
}
