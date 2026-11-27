package konta

import (
	"errors"
	"time"

	"skrytka/wewn/rdzen"
)

// Błędy 2FA.
var (
	ErrWymaganyKod         = errors.New("wymagany kod z aplikacji uwierzytelniającej")
	ErrZlyKod              = errors.New("nieprawidłowy albo już użyty kod")
	Err2FAJuzWlaczone      = errors.New("weryfikacja dwuetapowa jest już włączona")
	ErrBrakOczekujacego2FA = errors.New("najpierw rozpocznij włączanie weryfikacji dwuetapowej")
	Err2FANieWlaczone      = errors.New("weryfikacja dwuetapowa nie jest włączona")
)

// Start2FA tworzy sekret oczekujący na potwierdzenie (Wlacz2FA); zwraca sekret base32 i adres otpauth.
// Kolejne wywołanie zastępuje poprzedni oczekujący sekret.
func (k *Konta) Start2FA(login string) (string, string, error) {
	sekret, err := nowySekret()
	if err != nil {
		return "", "", err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, jest := k.konta[login]; !jest {
		return "", "", rdzen.ErrNieZnaleziono
	}
	if _, jest := k.dwaFA[login]; jest {
		return "", "", Err2FAJuzWlaczone
	}
	k.oczekujace[login] = sekret
	return sekret, adresOtpauth(login, sekret), nil
}

// Wlacz2FA potwierdza oczekujący sekret poprawnym kodem i zapisuje go w pliku kont.
func (k *Konta) Wlacz2FA(login, kod string) error {
	teraz := k.teraz()
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, jest := k.konta[login]; !jest {
		return rdzen.ErrNieZnaleziono
	}
	sekret, jest := k.oczekujace[login]
	if !jest {
		return ErrBrakOczekujacego2FA
	}
	krok, ok := sprawdzKodTOTP(sekret, kod, teraz)
	if !ok {
		return ErrZlyKod
	}
	k.dwaFA[login] = sekret
	if err := k.zapisz(); err != nil {
		delete(k.dwaFA, login)
		return err
	}
	delete(k.oczekujace, login)
	k.ostatniKrok[login] = krok
	return nil
}

// Wylacz2FA wyłącza 2FA po podaniu hasła (złe hasło liczy się do blokady loginu).
func (k *Konta) Wylacz2FA(login, haslo string) error {
	teraz := k.teraz()
	kLogin := kluczLoginu(login)
	if k.blokady.czyZablokowane(teraz, kLogin) {
		return rdzen.ErrKontoZablokowane
	}
	k.mu.RLock()
	u, jest := k.konta[login]
	k.mu.RUnlock()
	if !jest {
		return rdzen.ErrNieZnaleziono
	}
	if !k.porownaj(u.HashHasla, haslo) {
		k.blokady.zarejestrujBlad(teraz, kLogin)
		return rdzen.ErrZleDane
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	sekret, wlaczone := k.dwaFA[login]
	if !wlaczone {
		return Err2FANieWlaczone
	}
	delete(k.dwaFA, login)
	if err := k.zapisz(); err != nil {
		k.dwaFA[login] = sekret
		return err
	}
	delete(k.ostatniKrok, login)
	return nil
}

// Ma2FA: czy konto ma włączone 2FA.
func (k *Konta) Ma2FA(login string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	_, jest := k.dwaFA[login]
	return jest
}

// sprawdzDrugiSkladnik: nil bez 2FA; ErrWymaganyKod gdy brak kodu; ErrZlyKod gdy kod zły
// albo z kroku czasu nie późniejszego niż ostatnio użyty (ochrona przed powtórzeniem).
func (k *Konta) sprawdzDrugiSkladnik(login, kod string, teraz time.Time) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	sekret, wlaczone := k.dwaFA[login]
	if !wlaczone {
		return nil
	}
	if kod == "" {
		return ErrWymaganyKod
	}
	krok, ok := sprawdzKodTOTP(sekret, kod, teraz)
	if ostatni, byl := k.ostatniKrok[login]; !ok || (byl && krok <= ostatni) {
		return ErrZlyKod
	}
	k.ostatniKrok[login] = krok
	return nil
}
