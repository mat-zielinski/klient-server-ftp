package konta

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// kontaZ2FA: jan ma włączone 2FA; zwraca sekret. Zegar przesunięty o krok, by kod włączenia był już zużyty.
func kontaZ2FA(t *testing.T) (*Konta, *zegar, string, string) {
	t.Helper()
	k, z, plik := kontaZAdminemIJanem(t)
	sekret, _, err := k.Start2FA("jan")
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Wlacz2FA("jan", kodNa(t, sekret, z.Teraz())); err != nil {
		t.Fatal(err)
	}
	z.Przesun(konfiguracja.TOTPOkres)
	return k, z, sekret, plik
}

func TestWlaczanie2FA(t *testing.T) {
	k, z, _ := kontaZAdminemIJanem(t)
	if err := k.Wlacz2FA("jan", "123456"); !errors.Is(err, ErrBrakOczekujacego2FA) {
		t.Fatalf("bez Start2FA: %v", err)
	}
	if _, _, err := k.Start2FA("nikt"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Fatalf("Start2FA nieistniejącego: %v", err)
	}
	sekret, adres, err := k.Start2FA("jan")
	if err != nil || !strings.Contains(adres, sekret) {
		t.Fatalf("Start2FA: %q %v", adres, err)
	}
	if k.Ma2FA("jan") {
		t.Fatal("2FA włączone przed potwierdzeniem kodu")
	}
	zly := kodNa(t, sekret, z.Teraz().Add(5*konfiguracja.TOTPOkres))
	if err := k.Wlacz2FA("jan", zly); !errors.Is(err, ErrZlyKod) {
		t.Fatalf("zły kod przy włączaniu: %v", err)
	}
	if err := k.Wlacz2FA("jan", kodNa(t, sekret, z.Teraz())); err != nil {
		t.Fatal(err)
	}
	if !k.Ma2FA("jan") || k.Ma2FA("admin") {
		t.Fatal("Ma2FA nie zgadza się")
	}
	if _, _, err := k.Start2FA("jan"); !errors.Is(err, Err2FAJuzWlaczone) {
		t.Fatalf("ponowny start przy włączonym 2FA: %v", err)
	}
}

func TestLogowanieZ2FA(t *testing.T) {
	k, z, sekret, _ := kontaZ2FA(t)
	if _, err := k.Zaloguj("jan", hasloJana, "10.6.0.1"); !errors.Is(err, ErrWymaganyKod) {
		t.Fatalf("Zaloguj bez kodu: %v, chcę ErrWymaganyKod", err)
	}
	if _, err := k.ZalogujZKodem("jan", "zle-haslo-xyz", kodNa(t, sekret, z.Teraz()), "10.6.0.2"); !errors.Is(err, rdzen.ErrZleDane) {
		t.Fatalf("złe hasło z dobrym kodem: %v", err)
	}
	kod := kodNa(t, sekret, z.Teraz())
	u, err := k.ZalogujZKodem("jan", hasloJana, kod, "10.6.0.3")
	if err != nil || u.Login != "jan" {
		t.Fatalf("dobre hasło i kod: %+v %v", u, err)
	}
	if _, err := k.ZalogujZKodem("jan", hasloJana, kod, "10.6.0.4"); !errors.Is(err, ErrZlyKod) {
		t.Fatalf("powtórzony kod: %v, chcę ErrZlyKod", err)
	}
	// Kod z poprzedniego kroku (mieści się w oknie) też odrzucony po użyciu nowszego.
	if _, err := k.ZalogujZKodem("jan", hasloJana, kodNa(t, sekret, z.Teraz().Add(-konfiguracja.TOTPOkres)), "10.6.0.5"); !errors.Is(err, ErrZlyKod) {
		t.Fatalf("starszy kod po nowszym: %v", err)
	}
	z.Przesun(konfiguracja.TOTPOkres)
	if _, err := k.ZalogujZKodem("jan", hasloJana, kodNa(t, sekret, z.Teraz()), "10.6.0.6"); err != nil {
		t.Fatalf("kod z następnego kroku: %v", err)
	}
	// Konto bez 2FA: kod pomijany.
	if _, err := k.ZalogujZKodem("admin", hasloAdmina, "000000", "10.6.0.7"); err != nil {
		t.Fatalf("konto bez 2FA z kodem: %v", err)
	}
}

func TestBlokadaPoZlychKodach(t *testing.T) {
	k, z, sekret, _ := kontaZ2FA(t)
	zly := kodNa(t, sekret, z.Teraz().Add(10*konfiguracja.TOTPOkres))
	for i := 0; i < konfiguracja.MaksBlednychLogowan; i++ {
		if _, err := k.ZalogujZKodem("jan", hasloJana, zly, fmt.Sprintf("10.7.0.%d", i)); !errors.Is(err, ErrZlyKod) {
			t.Fatalf("próba %d: %v", i, err)
		}
	}
	if _, err := k.ZalogujZKodem("jan", hasloJana, kodNa(t, sekret, z.Teraz()), "10.7.1.1"); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("po złych kodach: %v, chcę blokadę", err)
	}
}

func TestBrakKoduNieLiczySieDoBlokady(t *testing.T) {
	k, z, sekret, _ := kontaZ2FA(t)
	for i := 0; i < konfiguracja.MaksBlednychLogowan+1; i++ {
		if _, err := k.Zaloguj("jan", hasloJana, "10.8.0.1"); !errors.Is(err, ErrWymaganyKod) {
			t.Fatalf("próba %d: %v", i, err)
		}
	}
	if _, err := k.ZalogujZKodem("jan", hasloJana, kodNa(t, sekret, z.Teraz()), "10.8.0.1"); err != nil {
		t.Fatalf("pierwszy krok logowania zablokował konto: %v", err)
	}
}

func TestWylacz2FA(t *testing.T) {
	k, _, _, _ := kontaZ2FA(t)
	if err := k.Wylacz2FA("admin", hasloAdmina); !errors.Is(err, Err2FANieWlaczone) {
		t.Fatalf("wyłączenie niewłączonego: %v", err)
	}
	if err := k.Wylacz2FA("jan", "zle-haslo-xyz"); !errors.Is(err, rdzen.ErrZleDane) || !k.Ma2FA("jan") {
		t.Fatalf("złe hasło wyłączyło 2FA: %v", err)
	}
	if err := k.Wylacz2FA("jan", hasloJana); err != nil || k.Ma2FA("jan") {
		t.Fatalf("wyłączenie: %v", err)
	}
	if _, err := k.Zaloguj("jan", hasloJana, "10.9.0.1"); err != nil {
		t.Fatalf("logowanie po wyłączeniu 2FA: %v", err)
	}
}

func TestWylacz2FABlokadaZlychHasel(t *testing.T) {
	k, _, _, _ := kontaZ2FA(t)
	for i := 0; i < konfiguracja.MaksBlednychLogowan; i++ {
		if err := k.Wylacz2FA("jan", "zle-haslo-xyz"); !errors.Is(err, rdzen.ErrZleDane) {
			t.Fatalf("próba %d: %v", i, err)
		}
	}
	if err := k.Wylacz2FA("jan", hasloJana); !errors.Is(err, rdzen.ErrKontoZablokowane) {
		t.Fatalf("po złych hasłach: %v, chcę blokadę", err)
	}
}

func TestPlik2FA(t *testing.T) {
	k, z, sekret, plik := kontaZ2FA(t)
	dane, err := os.ReadFile(plik)
	if err != nil || !strings.Contains(string(dane), `"dwa_fa"`) {
		t.Fatalf("brak 2FA w pliku: %v", err)
	}
	k2, err := Otworz(plik)
	if err != nil {
		t.Fatal(err)
	}
	k2.param, k2.teraz = tanieParametry, z.Teraz
	if !k2.Ma2FA("jan") {
		t.Fatal("2FA zgubione po ponownym otwarciu")
	}
	if _, err := k2.ZalogujZKodem("jan", hasloJana, kodNa(t, sekret, z.Teraz()), "10.10.0.1"); err != nil {
		t.Fatal(err)
	}
	for _, u := range k.Lista() {
		if strings.Contains(fmt.Sprintf("%+v", u), sekret) {
			t.Fatal("sekret 2FA widoczny w liście")
		}
	}
	// Usunięcie konta usuwa 2FA; nowe konto o tym loginie go nie dziedziczy.
	if err := k.Usun("jan"); err != nil {
		t.Fatal(err)
	}
	if err := k.Dodaj(rdzen.Uzytkownik{Login: "jan"}, hasloJana); err != nil {
		t.Fatal(err)
	}
	if k.Ma2FA("jan") {
		t.Fatal("nowe konto odziedziczyło 2FA")
	}
}

func TestOtworzZlySekret2FA(t *testing.T) {
	plik := t.TempDir() + "/konta.json"
	tresc := `{"konta":[{"login":"jan"}],"dwa_fa":{"jan":"!!!"}}`
	if err := os.WriteFile(plik, []byte(tresc), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Otworz(plik); err == nil {
		t.Fatal("zły sekret 2FA w pliku przyjęty")
	}
}
