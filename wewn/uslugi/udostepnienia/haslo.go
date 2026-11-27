package udostepnienia

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"skrytka/wewn/konfiguracja"
)

// errZlyHash: zapisany hash nie jest poprawnym ciągiem PHC argon2id.
var errZlyHash = errors.New("niepoprawny zapis hasha hasła linku")

// kodB64 to kodowanie soli i klucza w formacie PHC (base64 standardowe bez dopełnienia).
var kodB64 = base64.RawStdEncoding

// hashujHaslo zwraca hasło w formacie PHC: $argon2id$v=19$m=..,t=..,p=..$sol$klucz.
func hashujHaslo(haslo string) (string, error) {
	sol := make([]byte, konfiguracja.Argon2LinkDlugoscSoli)
	if _, err := rand.Read(sol); err != nil {
		return "", err
	}
	klucz := argon2.IDKey([]byte(haslo), sol, konfiguracja.Argon2LinkIteracje,
		konfiguracja.Argon2LinkPamiecKiB, konfiguracja.Argon2LinkWatki, konfiguracja.Argon2LinkDlugoscKlucza)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version,
		konfiguracja.Argon2LinkPamiecKiB, konfiguracja.Argon2LinkIteracje, konfiguracja.Argon2LinkWatki,
		kodB64.EncodeToString(sol), kodB64.EncodeToString(klucz)), nil
}

// parametryHasha: odczytane z zapisu PHC.
type parametryHasha struct {
	pamiec, iteracje uint32
	watki            uint8
	sol, klucz       []byte
}

// czytajHash rozbiera zapis PHC; parametry poza rozsądnym zakresem = błąd (ochrona przed złośliwym plikiem).
func czytajHash(zapis string) (parametryHasha, error) {
	var p parametryHasha
	czesci := strings.Split(zapis, "$")
	if len(czesci) != 6 || czesci[0] != "" || czesci[1] != "argon2id" {
		return p, errZlyHash
	}
	var wersja int
	if _, err := fmt.Sscanf(czesci[2], "v=%d", &wersja); err != nil || wersja != argon2.Version {
		return p, errZlyHash
	}
	if _, err := fmt.Sscanf(czesci[3], "m=%d,t=%d,p=%d", &p.pamiec, &p.iteracje, &p.watki); err != nil {
		return p, errZlyHash
	}
	if !poprawneParametry(p) {
		return p, errZlyHash
	}
	var err error
	if p.sol, err = kodB64.DecodeString(czesci[4]); err != nil || len(p.sol) == 0 {
		return p, errZlyHash
	}
	if p.klucz, err = kodB64.DecodeString(czesci[5]); err != nil || len(p.klucz) == 0 {
		return p, errZlyHash
	}
	return p, nil
}

// poprawneParametry odrzuca koszty spoza rozsądnego zakresu (ochrona przed zatrutym plikiem).
func poprawneParametry(p parametryHasha) bool {
	return p.pamiec > 0 && p.pamiec <= 4*konfiguracja.Argon2LinkPamiecKiB && p.iteracje > 0 &&
		p.iteracje <= 16*konfiguracja.Argon2LinkIteracje && p.watki > 0
}

// sprawdzHaslo porównuje hasło z zapisem PHC w stałym czasie.
func sprawdzHaslo(haslo, zapis string) bool {
	p, err := czytajHash(zapis)
	if err != nil {
		return false
	}
	klucz := argon2.IDKey([]byte(haslo), p.sol, p.iteracje, p.pamiec, p.watki, uint32(len(p.klucz)))
	return subtle.ConstantTimeCompare(klucz, p.klucz) == 1
}
