// Pakiet konta: użytkownicy (dane/konta.json, hasła argon2id), logowanie z blokadą po złych
// hasłach, sesje w pamięci.
package konta

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// Błędy pakietu (uzupełniają rdzen.Err*).
var (
	ErrSlabeHaslo      = errors.New("hasło jest za krótkie")
	ErrZbytDlugieHaslo = errors.New("hasło jest za długie albo ma niedozwolone znaki")
	ErrOstatniAdmin    = errors.New("nie można odebrać uprawnień ani usunąć ostatniego aktywnego administratora")
)

// plikKont: zawartość dane/konta.json.
type plikKont struct {
	Konta []rdzen.Uzytkownik `json:"konta"`
}

// Konta: magazyn użytkowników, bezpieczny dla wielu gorutyn.
type Konta struct {
	mu    sync.RWMutex
	plik  string
	konta map[string]rdzen.Uzytkownik

	blokady *blokady
	param   parametryArgon
	teraz   func() time.Time
	// porownaj: sprawdzanie hasła (podmieniane w testach, by policzyć wywołania).
	porownaj func(hash, haslo string) bool

	atrapaRaz sync.Once
	atrapa    string // hash losowego hasła, liczony dla nieistniejących loginów

	sesjeMu sync.Mutex
	sesje   []*Sesje
}

// Otworz plik kont (brak pliku = pusta lista).
func Otworz(plik string) (*Konta, error) {
	k := &Konta{
		plik:     plik,
		konta:    map[string]rdzen.Uzytkownik{},
		blokady:  noweBlokady(),
		param:    domyslneParametry(),
		teraz:    time.Now,
		porownaj: sprawdzHaslo,
	}
	dane, err := os.ReadFile(plik)
	if errors.Is(err, os.ErrNotExist) {
		return k, nil
	}
	if err != nil {
		return nil, err
	}
	var p plikKont
	if err := json.Unmarshal(dane, &p); err != nil {
		return nil, err
	}
	for _, u := range p.Konta {
		if err := sprawdzLogin(u.Login); err != nil {
			return nil, err
		}
		if _, jest := k.konta[u.Login]; jest {
			return nil, rdzen.ErrJuzIstnieje
		}
		k.konta[u.Login] = kopia(u)
	}
	return k, nil
}

// CzyPusto: true przy pierwszym uruchomieniu (kreator zakłada admina).
func (k *Konta) CzyPusto() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.konta) == 0
}

// Zaloguj sprawdza hasło; po MaksBlednychLogowan złych próbach z jednego IP albo na jeden login
// w oknie czasu = rdzen.ErrKontoZablokowane. Porównanie w stałym czasie, także dla nieistniejącego loginu.
func (k *Konta) Zaloguj(login, haslo, ip string) (rdzen.Uzytkownik, error) {
	teraz := k.teraz()
	kIP, kLogin := kluczIP(ip), kluczLoginu(login)
	if k.blokady.czyZablokowane(teraz, kIP, kLogin) {
		return rdzen.Uzytkownik{}, rdzen.ErrKontoZablokowane
	}
	if len(haslo) > konfiguracja.MaksDlugoscHasla*utf8.UTFMax {
		// Bez haszowania: ochrona przed kosztownymi próbami, nic nie mówi o istnieniu loginu.
		k.blokady.zarejestrujBlad(teraz, kIP, kLogin)
		return rdzen.Uzytkownik{}, rdzen.ErrZleDane
	}
	k.mu.RLock()
	u, jest := k.konta[login]
	k.mu.RUnlock()
	hash := u.HashHasla
	if !jest {
		hash = k.hashAtrapy()
	}
	dobre := k.porownaj(hash, haslo)
	if !jest || !dobre {
		k.blokady.zarejestrujBlad(teraz, kIP, kLogin)
		return rdzen.Uzytkownik{}, rdzen.ErrZleDane
	}
	if u.Wylaczony {
		return rdzen.Uzytkownik{}, rdzen.ErrZleDane
	}
	k.blokady.wyczysc(kLogin)
	return bezHasha(u), nil
}

// hashAtrapy: hash losowego hasła, by nieistniejący login kosztował tyle samo co istniejący.
func (k *Konta) hashAtrapy() string {
	k.atrapaRaz.Do(func() {
		haslo, err := losowyToken()
		if err != nil {
			return
		}
		if h, err := zahaszuj(haslo, k.param); err == nil {
			k.atrapa = h
		}
	})
	if k.atrapa == "" {
		// Awaryjnie (brak losowości): poprawny format, porównanie i tak się wykona.
		return "$argon2id$v=19$m=8,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}
	return k.atrapa
}

// Uzytkownik po loginie (bez hasha hasła).
func (k *Konta) Uzytkownik(login string) (rdzen.Uzytkownik, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	u, jest := k.konta[login]
	if !jest {
		return rdzen.Uzytkownik{}, rdzen.ErrNieZnaleziono
	}
	return bezHasha(u), nil
}

// Lista bez pola HashHasla (wyczyszczone), posortowana po loginie.
func (k *Konta) Lista() []rdzen.Uzytkownik {
	k.mu.RLock()
	defer k.mu.RUnlock()
	wynik := make([]rdzen.Uzytkownik, 0, len(k.konta))
	for _, u := range k.konta {
		wynik = append(wynik, bezHasha(u))
	}
	sort.Slice(wynik, func(i, j int) bool { return wynik[i].Login < wynik[j].Login })
	return wynik
}

// Dodaj konto z hasłem (min. DlugoscHasla znaków).
func (k *Konta) Dodaj(u rdzen.Uzytkownik, haslo string) error {
	if err := sprawdzLogin(u.Login); err != nil {
		return err
	}
	if err := sprawdzDlugoscHasla(haslo); err != nil {
		return err
	}
	if _, err := k.Uzytkownik(u.Login); err == nil {
		return rdzen.ErrJuzIstnieje
	}
	hash, err := zahaszuj(haslo, k.param)
	if err != nil {
		return err
	}
	nowy := kopia(u)
	nowy.HashHasla = hash
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, jest := k.konta[u.Login]; jest {
		return rdzen.ErrJuzIstnieje
	}
	return k.zmienIZapisz(u.Login, &nowy)
}

// Zmien uprawnienia/flagi (hasło bez zmian); nie pozwala odebrać admina ostatniemu adminowi.
// Wyłączenie konta kończy jego sesje.
func (k *Konta) Zmien(u rdzen.Uzytkownik) error {
	k.mu.Lock()
	stary, jest := k.konta[u.Login]
	if !jest {
		k.mu.Unlock()
		return rdzen.ErrNieZnaleziono
	}
	if aktywnyAdmin(stary) && !aktywnyAdmin(u) && k.ileAktywnychAdminow() <= 1 {
		k.mu.Unlock()
		return ErrOstatniAdmin
	}
	nowy := kopia(u)
	nowy.HashHasla = stary.HashHasla
	err := k.zmienIZapisz(u.Login, &nowy)
	k.mu.Unlock()
	if err == nil && nowy.Wylaczony {
		k.zakonczSesje(u.Login)
	}
	return err
}

// UstawHaslo zmienia hasło i kończy wszystkie sesje tego konta.
func (k *Konta) UstawHaslo(login, haslo string) error {
	if err := sprawdzDlugoscHasla(haslo); err != nil {
		return err
	}
	if _, err := k.Uzytkownik(login); err != nil {
		return err
	}
	hash, err := zahaszuj(haslo, k.param)
	if err != nil {
		return err
	}
	k.mu.Lock()
	u, jest := k.konta[login]
	if !jest {
		k.mu.Unlock()
		return rdzen.ErrNieZnaleziono
	}
	u = kopia(u)
	u.HashHasla = hash
	err = k.zmienIZapisz(login, &u)
	k.mu.Unlock()
	if err == nil {
		k.zakonczSesje(login)
	}
	return err
}

// Usun konto (nie ostatniego admina); kończy jego sesje.
func (k *Konta) Usun(login string) error {
	k.mu.Lock()
	u, jest := k.konta[login]
	if !jest {
		k.mu.Unlock()
		return rdzen.ErrNieZnaleziono
	}
	if aktywnyAdmin(u) && k.ileAktywnychAdminow() <= 1 {
		k.mu.Unlock()
		return ErrOstatniAdmin
	}
	err := k.zmienIZapisz(login, nil)
	k.mu.Unlock()
	if err == nil {
		k.zakonczSesje(login)
	}
	return err
}

// zmienIZapisz ustawia (nowy != nil) albo usuwa konto i zapisuje plik; przy błędzie zapisu
// przywraca stan w pamięci. Wywołujący trzyma k.mu do zapisu.
func (k *Konta) zmienIZapisz(login string, nowy *rdzen.Uzytkownik) error {
	stary, byl := k.konta[login]
	if nowy == nil {
		delete(k.konta, login)
	} else {
		k.konta[login] = *nowy
	}
	if err := k.zapisz(); err != nil {
		if byl {
			k.konta[login] = stary
		} else {
			delete(k.konta, login)
		}
		return err
	}
	return nil
}

// zapisz zapisuje dane/konta.json atomowo (0600). Wywołujący trzyma k.mu.
func (k *Konta) zapisz() error {
	p := plikKont{Konta: make([]rdzen.Uzytkownik, 0, len(k.konta))}
	for _, u := range k.konta {
		p.Konta = append(p.Konta, u)
	}
	sort.Slice(p.Konta, func(i, j int) bool { return p.Konta[i].Login < p.Konta[j].Login })
	dane, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return konfiguracja.ZapiszAtomowo(k.plik, dane)
}

// ileAktywnychAdminow liczy włączone konta z prawem admina. Wywołujący trzyma k.mu.
func (k *Konta) ileAktywnychAdminow() int {
	n := 0
	for _, u := range k.konta {
		if aktywnyAdmin(u) {
			n++
		}
	}
	return n
}

// zarejestrujSesje: magazyn sesji, który ma być czyszczony przy zmianie hasła / usunięciu konta.
func (k *Konta) zarejestrujSesje(s *Sesje) {
	k.sesjeMu.Lock()
	defer k.sesjeMu.Unlock()
	k.sesje = append(k.sesje, s)
}

func (k *Konta) zakonczSesje(login string) {
	k.sesjeMu.Lock()
	lista := append([]*Sesje(nil), k.sesje...)
	k.sesjeMu.Unlock()
	for _, s := range lista {
		s.ZakonczWszystkie(login)
	}
}

func aktywnyAdmin(u rdzen.Uzytkownik) bool { return u.Admin && !u.Wylaczony }

// sprawdzLogin: 1..MaksDlugoscLoginu znaków z [a-zA-Z0-9._-], bez kropki na początku.
func sprawdzLogin(login string) error {
	if login == "" || len(login) > konfiguracja.MaksDlugoscLoginu || login[0] == '.' {
		return rdzen.ErrZlaNazwa
	}
	for _, z := range login {
		dozwolony := (z >= 'a' && z <= 'z') || (z >= 'A' && z <= 'Z') || (z >= '0' && z <= '9') ||
			z == '.' || z == '_' || z == '-'
		if !dozwolony {
			return rdzen.ErrZlaNazwa
		}
	}
	return nil
}

// kopia: głęboka kopia (osobna mapa praw), by wywołujący nie mógł zmienić stanu magazynu.
func kopia(u rdzen.Uzytkownik) rdzen.Uzytkownik {
	prawa := make(map[string]rdzen.Prawo, len(u.Prawa))
	for nazwa, p := range u.Prawa {
		prawa[nazwa] = p
	}
	u.Prawa = prawa
	return u
}

func bezHasha(u rdzen.Uzytkownik) rdzen.Uzytkownik {
	u = kopia(u)
	u.HashHasla = ""
	return u
}
