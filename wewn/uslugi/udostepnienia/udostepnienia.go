// Pakiet udostepnienia: linki publiczne do pliku/folderu (dane/udostepnienia.json).
package udostepnienia

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// ErrZlyLink: niepoprawne dane nowego linku (brak udziału, ujemny limit, data w przeszłości, za długie hasło).
var ErrZlyLink = errors.New("Niepoprawne dane linku.")

// Link publiczny.
type Link struct {
	ID          string    `json:"id"` // losowy, 16 B base64url
	Udzial      string    `json:"udzial"`
	Sciezka     string    `json:"sciezka"`
	Autor       string    `json:"autor"`
	Utworzono   time.Time `json:"utworzono"`
	Wygasa      time.Time `json:"wygasa"` // zero = bez wygasania (tylko stare wpisy; nowe zawsze mają datę)
	HashHasla   string    `json:"hash_hasla,omitempty"`
	MaHaslo     bool      `json:"ma_haslo"`
	LimitPobran int       `json:"limit_pobran"` // 0 = bez limitu
	Pobrano     int       `json:"pobrano"`
}

// proby: złe hasła podane do jednego linku w bieżącym oknie.
type proby struct {
	licznik int
	od      time.Time
}

// Udostepnienia: bezpieczne dla wielu gorutyn.
type Udostepnienia struct {
	mu    sync.Mutex
	plik  string
	linki map[string]Link
	proby map[string]*proby
	teraz func() time.Time
}

// Otworz plik linków (brak pliku = pusta lista). Wygasłe linki są od razu usuwane.
func Otworz(plik string) (*Udostepnienia, error) {
	u := &Udostepnienia{plik: plik, proby: map[string]*proby{}, teraz: time.Now}
	linki, err := wczytaj(plik)
	if err != nil {
		return nil, err
	}
	u.linki = linki
	if _, err := u.Sprzataj(); err != nil {
		return nil, err
	}
	return u, nil
}

// UstawZegar podmienia źródło czasu (testy).
func (u *Udostepnienia) UstawZegar(teraz func() time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.teraz = teraz
}

// Sprzataj usuwa wygasłe linki i zapisuje plik, gdy coś usunięto. Zwraca liczbę usuniętych.
func (u *Udostepnienia) Sprzataj() (int, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.sprzatajBezBlokady()
}

func (u *Udostepnienia) sprzatajBezBlokady() (int, error) {
	teraz := u.teraz()
	usuniete := map[string]Link{}
	for id, l := range u.linki {
		if wygaslWCzasie(l, teraz) {
			usuniete[id] = l
			delete(u.linki, id)
			delete(u.proby, id)
		}
	}
	if len(usuniete) == 0 {
		return 0, nil
	}
	if err := zapisz(u.plik, u.linki); err != nil {
		for id, l := range usuniete {
			u.linki[id] = l
		}
		return 0, err
	}
	return len(usuniete), nil
}

// Utworz link (haslo puste = bez hasła). Wygasa zero = domyślna ważność, dalej niż maksimum = przycięte.
func (u *Udostepnienia) Utworz(l Link, haslo string) (Link, error) {
	sciezka, err := rdzen.OczyscSciezke(l.Sciezka)
	if err != nil {
		return Link{}, err
	}
	if strings.TrimSpace(l.Udzial) == "" || l.LimitPobran < 0 || len(haslo) > konfiguracja.MaksDlugoscHaslaLinku {
		return Link{}, ErrZlyLink
	}
	nowy := Link{Udzial: l.Udzial, Sciezka: sciezka, Autor: l.Autor, LimitPobran: l.LimitPobran}
	if haslo != "" {
		if nowy.HashHasla, err = hashujHaslo(haslo); err != nil {
			return Link{}, err
		}
		nowy.MaHaslo = true
	}
	if nowy.ID, err = noweID(); err != nil {
		return Link{}, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	teraz := u.teraz()
	nowy.Utworzono = teraz
	if nowy.Wygasa, err = ustalWygasniecie(l.Wygasa, teraz); err != nil {
		return Link{}, err
	}
	if _, zajete := u.linki[nowy.ID]; zajete {
		return Link{}, rdzen.ErrJuzIstnieje
	}
	if _, err := u.sprzatajBezBlokady(); err != nil {
		return Link{}, err
	}
	u.linki[nowy.ID] = nowy
	if err := zapisz(u.plik, u.linki); err != nil {
		delete(u.linki, nowy.ID)
		return Link{}, err
	}
	return bezHasha(nowy), nil
}

// ustalWygasniecie: zero = domyślna ważność, przeszłość = błąd, ponad maksimum = maksimum.
func ustalWygasniecie(wygasa, teraz time.Time) (time.Time, error) {
	if wygasa.IsZero() {
		return teraz.Add(konfiguracja.DomyslnaWaznoscLinku), nil
	}
	if !wygasa.After(teraz) {
		return time.Time{}, ErrZlyLink
	}
	if maks := teraz.Add(konfiguracja.MaksWaznoscLinku); wygasa.After(maks) {
		return maks, nil
	}
	return wygasa, nil
}

// Lista linków (autor "" = wszystkie), bez HashHasla, od najnowszego. Wygasłe pomija.
func (u *Udostepnienia) Lista(autor string) []Link {
	u.mu.Lock()
	defer u.mu.Unlock()
	teraz := u.teraz()
	wynik := []Link{}
	for _, l := range u.linki {
		if (autor == "" || l.Autor == autor) && !wygaslWCzasie(l, teraz) {
			wynik = append(wynik, bezHasha(l))
		}
	}
	sort.Slice(wynik, func(i, j int) bool {
		if !wynik[i].Utworzono.Equal(wynik[j].Utworzono) {
			return wynik[i].Utworzono.After(wynik[j].Utworzono)
		}
		return wynik[i].ID < wynik[j].ID
	})
	return wynik
}

// Pobierz zwraca link bez sprawdzania hasła i ważności (np. by sprawdzić autora przed usunięciem).
func (u *Udostepnienia) Pobierz(id string) (Link, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	l, ok := u.linki[id]
	if !ok {
		return Link{}, rdzen.ErrNieZnaleziono
	}
	return bezHasha(l), nil
}

// Usun link.
func (u *Udostepnienia) Usun(id string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	l, ok := u.linki[id]
	if !ok {
		return rdzen.ErrNieZnaleziono
	}
	delete(u.linki, id)
	if err := zapisz(u.plik, u.linki); err != nil {
		u.linki[id] = l
		return err
	}
	delete(u.proby, id)
	return nil
}

// Sprawdz link przy wejściu: brak = rdzen.ErrNieZnaleziono, wygasły / limit = rdzen.ErrWygaslo,
// złe hasło = rdzen.ErrZleDane, za dużo złych haseł w oknie = rdzen.ErrKontoZablokowane.
func (u *Udostepnienia) Sprawdz(id, haslo string) (Link, error) {
	u.mu.Lock()
	l, ok := u.linki[id]
	if !ok {
		u.mu.Unlock()
		return Link{}, rdzen.ErrNieZnaleziono
	}
	teraz := u.teraz()
	if wyczerpany(l, teraz) {
		u.mu.Unlock()
		return Link{}, rdzen.ErrWygaslo
	}
	if !l.MaHaslo && l.HashHasla == "" {
		u.mu.Unlock()
		return bezHasha(l), nil
	}
	// Próba jest liczona z góry (rezerwacja), by równoległe zgadywanie nie przekroczyło limitu.
	if !u.zarezerwujProbe(id, teraz) {
		u.mu.Unlock()
		return Link{}, rdzen.ErrKontoZablokowane
	}
	u.mu.Unlock()

	// Hashowanie poza blokadą: argon2id jest kosztowny, nie wstrzymuje innych linków.
	dobre := len(haslo) <= konfiguracja.MaksDlugoscHaslaLinku && sprawdzHaslo(haslo, l.HashHasla)
	if !dobre {
		return Link{}, rdzen.ErrZleDane
	}
	u.mu.Lock()
	delete(u.proby, id)
	u.mu.Unlock()
	return bezHasha(l), nil
}

// zarezerwujProbe dolicza próbę; false = link zablokowany (wywołujący trzyma u.mu).
func (u *Udostepnienia) zarezerwujProbe(id string, teraz time.Time) bool {
	p := u.proby[id]
	if p == nil || teraz.Sub(p.od) >= konfiguracja.OknoProbHaslaLinku {
		p = &proby{od: teraz}
		u.proby[id] = p
	}
	if p.licznik >= konfiguracja.LimitProbHaslaLinku {
		return false
	}
	p.licznik++
	return true
}

// ZaliczPobranie zwiększa licznik (atomowo z zapisem); wyczerpany lub wygasły = rdzen.ErrWygaslo.
func (u *Udostepnienia) ZaliczPobranie(id string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	l, ok := u.linki[id]
	if !ok {
		return rdzen.ErrNieZnaleziono
	}
	if wyczerpany(l, u.teraz()) {
		return rdzen.ErrWygaslo
	}
	przed := l
	l.Pobrano++
	u.linki[id] = l
	if err := zapisz(u.plik, u.linki); err != nil {
		u.linki[id] = przed
		return err
	}
	return nil
}

// wygaslWCzasie: minęła data wygaśnięcia (zero = nigdy).
func wygaslWCzasie(l Link, teraz time.Time) bool {
	return !l.Wygasa.IsZero() && !teraz.Before(l.Wygasa)
}

// wyczerpany: wygasł w czasie albo osiągnął limit pobrań.
func wyczerpany(l Link, teraz time.Time) bool {
	return wygaslWCzasie(l, teraz) || (l.LimitPobran > 0 && l.Pobrano >= l.LimitPobran)
}

// bezHasha: kopia do oddania na zewnątrz.
func bezHasha(l Link) Link {
	l.MaHaslo = l.MaHaslo || l.HashHasla != ""
	l.HashHasla = ""
	return l
}

// noweID: losowe bajty w base64url bez dopełnienia.
func noweID() (string, error) {
	b := make([]byte, konfiguracja.DlugoscIDLinku)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
