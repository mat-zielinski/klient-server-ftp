package konta

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// sesja w pamięci; sam token nie jest przechowywany, tylko jego SHA-256.
type sesja struct {
	login        string
	ip           string
	przegladarka string
	utworzono    time.Time
	ostatnio     time.Time
}

func (s sesja) wygasa() time.Time { return s.ostatnio.Add(konfiguracja.CzasSesji) }

// Urzadzenie: opis sesji do listy „zalogowane urządzenia” (bez tokenu).
type Urzadzenie struct {
	// ID: krótki skrót tokenu (nie da się z niego odtworzyć tokenu).
	ID           string    `json:"id"`
	IP           string    `json:"ip"`
	Przegladarka string    `json:"przegladarka"`
	Utworzono    time.Time `json:"utworzono"`
	Ostatnio     time.Time `json:"ostatnio"`
}

type kluczSesji = [sha256.Size]byte

// Sesje: token losowy 32 B (base64url), ważność CzasSesji, przedłużana przy użyciu.
type Sesje struct {
	mu    sync.Mutex
	k     *Konta
	wpisy map[kluczSesji]sesja
}

// NoweSesje tworzy magazyn sesji powiązany z kontami.
func NoweSesje(k *Konta) *Sesje {
	s := &Sesje{k: k, wpisy: map[kluczSesji]sesja{}}
	k.zarejestrujSesje(s)
	return s
}

// Utworz sesję dla loginu (konto musi istnieć i być włączone).
func (s *Sesje) Utworz(login string) (string, error) {
	return s.UtworzZOpisem(login, "", "")
}

// UtworzZOpisem: sesja z adresem IP i nazwą przeglądarki (lista urządzeń). Ponad MaksSesjiNaKonto
// wypada sesja tego konta używana najdawniej.
func (s *Sesje) UtworzZOpisem(login, ip, przegladarka string) (string, error) {
	u, err := s.k.Uzytkownik(login)
	if err != nil {
		return "", err
	}
	if u.Wylaczony {
		return "", rdzen.ErrZleDane
	}
	token, err := losowyToken()
	if err != nil {
		return "", err
	}
	teraz := s.k.teraz()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sprzataj(teraz)
	s.ograniczLiczbe(login, konfiguracja.MaksSesjiNaKonto-1)
	s.wpisy[sha256.Sum256([]byte(token))] = sesja{
		login: login, ip: obetnij(ip), przegladarka: obetnij(przegladarka), utworzono: teraz, ostatnio: teraz,
	}
	return token, nil
}

// Sprawdz token: aktualny użytkownik (wyłączony / usunięty = false). Przedłuża sesję.
func (s *Sesje) Sprawdz(token string) (rdzen.Uzytkownik, bool) {
	if len(token) != dlugoscTokenu() {
		return rdzen.Uzytkownik{}, false
	}
	klucz := sha256.Sum256([]byte(token))
	teraz := s.k.teraz()
	s.mu.Lock()
	ses, jest := s.wpisy[klucz]
	if jest && !teraz.Before(ses.wygasa()) {
		delete(s.wpisy, klucz)
		jest = false
	}
	s.mu.Unlock()
	if !jest {
		return rdzen.Uzytkownik{}, false
	}
	u, err := s.k.Uzytkownik(ses.login)
	if err != nil || u.Wylaczony {
		s.Zakoncz(token)
		return rdzen.Uzytkownik{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	biezaca, nadal := s.wpisy[klucz]
	if !nadal || biezaca.login != ses.login {
		// Sesja zakończona w międzyczasie (np. zmiana hasła).
		return rdzen.Uzytkownik{}, false
	}
	biezaca.ostatnio = teraz
	s.wpisy[klucz] = biezaca
	return u, true
}

// Zakoncz jedną sesję.
func (s *Sesje) Zakoncz(token string) {
	klucz := sha256.Sum256([]byte(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.wpisy, klucz)
}

// ZakonczWszystkie sesje loginu.
func (s *Sesje) ZakonczWszystkie(login string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ograniczLiczbe(login, 0)
}

// Urzadzenia: ważne sesje loginu, od ostatnio używanej.
func (s *Sesje) Urzadzenia(login string) []Urzadzenie {
	teraz := s.k.teraz()
	s.mu.Lock()
	defer s.mu.Unlock()
	wynik := []Urzadzenie{}
	for klucz, ses := range s.wpisy {
		if ses.login != login || !teraz.Before(ses.wygasa()) {
			continue
		}
		wynik = append(wynik, Urzadzenie{
			ID: idZKlucza(klucz), IP: ses.ip, Przegladarka: ses.przegladarka,
			Utworzono: ses.utworzono, Ostatnio: ses.ostatnio,
		})
	}
	sort.Slice(wynik, func(i, j int) bool {
		if !wynik[i].Ostatnio.Equal(wynik[j].Ostatnio) {
			return wynik[i].Ostatnio.After(wynik[j].Ostatnio)
		}
		return wynik[i].ID < wynik[j].ID
	})
	return wynik
}

// ZakonczUrzadzenie kończy sesję o danym ID, tylko jeśli należy do loginu (inaczej ErrNieZnaleziono).
func (s *Sesje) ZakonczUrzadzenie(login, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for klucz, ses := range s.wpisy {
		if ses.login == login && idZKlucza(klucz) == id {
			delete(s.wpisy, klucz)
			return nil
		}
	}
	return rdzen.ErrNieZnaleziono
}

// IDSesji: identyfikator urządzenia dla tokenu (do oznaczenia bieżącej sesji na liście).
func IDSesji(token string) string { return idZKlucza(sha256.Sum256([]byte(token))) }

func idZKlucza(klucz kluczSesji) string {
	return hex.EncodeToString(klucz[:konfiguracja.DlugoscIDSesji])
}

// ograniczLiczbe zostawia najwyżej zostaw sesji loginu (usuwa najdawniej używane). Wywołujący trzyma s.mu.
func (s *Sesje) ograniczLiczbe(login string, zostaw int) {
	var klucze []kluczSesji
	for klucz, ses := range s.wpisy {
		if ses.login == login {
			klucze = append(klucze, klucz)
		}
	}
	if len(klucze) <= zostaw {
		return
	}
	sort.Slice(klucze, func(i, j int) bool {
		return s.wpisy[klucze[i]].ostatnio.Before(s.wpisy[klucze[j]].ostatnio)
	})
	for _, klucz := range klucze[:len(klucze)-zostaw] {
		delete(s.wpisy, klucz)
	}
}

// sprzataj usuwa wygasłe sesje. Wywołujący trzyma s.mu.
func (s *Sesje) sprzataj(teraz time.Time) {
	for klucz, ses := range s.wpisy {
		if !teraz.Before(ses.wygasa()) {
			delete(s.wpisy, klucz)
		}
	}
}

// obetnij skraca opis do MaksDlugoscOpisuUrzadzenia znaków (z poprawnym UTF-8).
func obetnij(tekst string) string {
	if !utf8.ValidString(tekst) {
		tekst = string([]rune(tekst)) // zamienia złe bajty na U+FFFD
	}
	if r := []rune(tekst); len(r) > konfiguracja.MaksDlugoscOpisuUrzadzenia {
		return string(r[:konfiguracja.MaksDlugoscOpisuUrzadzenia])
	}
	return tekst
}

// losowyToken: DlugoscTokenuSesji bajtów z crypto/rand jako base64url bez dopełnienia.
func losowyToken() (string, error) {
	b := make([]byte, konfiguracja.DlugoscTokenuSesji)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func dlugoscTokenu() int { return base64.RawURLEncoding.EncodedLen(konfiguracja.DlugoscTokenuSesji) }
