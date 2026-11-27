package konta

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"skrytka/wewn/konfiguracja"
)

// parametryArgon: koszt haszowania (w testach obniżany, w produkcji z konfiguracji).
type parametryArgon struct {
	czas   uint32
	pamiec uint32 // KiB
	watki  uint8
	sol    int
	klucz  uint32
}

func domyslneParametry() parametryArgon {
	return parametryArgon{
		czas:   konfiguracja.ArgonCzas,
		pamiec: konfiguracja.ArgonPamiecKiB,
		watki:  konfiguracja.ArgonWatki,
		sol:    konfiguracja.ArgonDlugoscSoli,
		klucz:  konfiguracja.ArgonDlugoscKlucza,
	}
}

const prefiksArgon = "$argon2id$"

var errZlyHash = errors.New("nieprawidłowy format hasha hasła")

// sprawdzDlugoscHasla: hasło musi mieć od DlugoscHasla do MaksDlugoscHasla znaków.
func sprawdzDlugoscHasla(haslo string) error {
	n := utf8.RuneCountInString(haslo)
	if n < konfiguracja.DlugoscHasla {
		return ErrSlabeHaslo
	}
	if n > konfiguracja.MaksDlugoscHasla || !utf8.ValidString(haslo) {
		return ErrZbytDlugieHaslo
	}
	return nil
}

// zahaszuj zwraca hash w formacie PHC: $argon2id$v=19$m=…,t=…,p=…$sól$hash (base64 bez dopełnienia).
func zahaszuj(haslo string, p parametryArgon) (string, error) {
	sol := make([]byte, p.sol)
	if _, err := rand.Read(sol); err != nil {
		return "", err
	}
	klucz := argon2.IDKey([]byte(haslo), sol, p.czas, p.pamiec, p.watki, p.klucz)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("%sv=%d$m=%d,t=%d,p=%d$%s$%s", prefiksArgon, argon2.Version,
		p.pamiec, p.czas, p.watki, b64.EncodeToString(sol), b64.EncodeToString(klucz)), nil
}

// rozbierzHash czyta parametry, sól i klucz z zapisu PHC (z limitami kosztu).
func rozbierzHash(hash string) (parametryArgon, []byte, []byte, error) {
	var p parametryArgon
	if !strings.HasPrefix(hash, prefiksArgon) {
		return p, nil, nil, errZlyHash
	}
	czesci := strings.Split(hash[len(prefiksArgon):], "$")
	if len(czesci) != 4 {
		return p, nil, nil, errZlyHash
	}
	var wersja int
	if _, err := fmt.Sscanf(czesci[0], "v=%d", &wersja); err != nil || wersja != argon2.Version {
		return p, nil, nil, errZlyHash
	}
	if _, err := fmt.Sscanf(czesci[1], "m=%d,t=%d,p=%d", &p.pamiec, &p.czas, &p.watki); err != nil {
		return p, nil, nil, errZlyHash
	}
	if p.pamiec == 0 || p.pamiec > konfiguracja.ArgonMaksPamiecKiB || p.czas == 0 ||
		p.czas > konfiguracja.ArgonMaksCzas || p.watki == 0 {
		return p, nil, nil, errZlyHash
	}
	sol, err1 := base64.RawStdEncoding.DecodeString(czesci[2])
	klucz, err2 := base64.RawStdEncoding.DecodeString(czesci[3])
	if err1 != nil || err2 != nil || len(sol) == 0 || len(klucz) == 0 {
		return p, nil, nil, errZlyHash
	}
	p.sol, p.klucz = len(sol), uint32(len(klucz))
	return p, sol, klucz, nil
}

// sprawdzHaslo porównuje hasło z hashem w stałym czasie; zły format = false.
func sprawdzHaslo(hash, haslo string) bool {
	p, sol, klucz, err := rozbierzHash(hash)
	if err != nil {
		return false
	}
	wyliczony := argon2.IDKey([]byte(haslo), sol, p.czas, p.pamiec, p.watki, p.klucz)
	return subtle.ConstantTimeCompare(wyliczony, klucz) == 1
}
