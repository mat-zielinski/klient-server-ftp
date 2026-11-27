// Pakiet magazyn: JEDYNY dostęp do dysku. Każda ścieżka przez rdzen.SciezkaNaDysku, potem rozwinięcie
// dowiązań i junction z kontrolą, że wynik leży w korzeniu udziału.
package magazyn

import (
	"errors"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// ErrZlePrzesuniecie: kawałek zaczyna się za końcem już zapisanej części (luka w danych).
var ErrZlePrzesuniecie = errors.New("złe przesunięcie kawałka")

// Magazyn obsługuje udziały z konfiguracji. Po Nowy tylko do odczytu, więc bezpieczny dla wielu gorutyn.
type Magazyn struct {
	udzialy   map[string]udzialWew
	kolejnosc []konfiguracja.Udzial
	// zegar: czas do znaczników kosza (testy podstawiają własny).
	zegar   func() time.Time
	blokady [konfiguracja.MagazynLiczbaBlokad]sync.Mutex
}

// udzialWew: udział z rozwiązanym korzeniem.
type udzialWew struct {
	konfiguracja.Udzial
	korzen string
}

// cel: wynik rozwiązania ścieżki użytkownika.
type cel struct {
	u      udzialWew
	czysta string // "/a/b" (rdzen.OczyscSciezke)
	dysk   string // prawdziwa ścieżka na dysku, dowiązania rozwinięte
}

var pulaBuforow = sync.Pool{New: func() any {
	b := make([]byte, konfiguracja.MagazynRozmiarBufora)
	return &b
}}

// kopiuj kopiuje z buforem z puli.
func kopiuj(w io.Writer, r io.Reader) (int64, error) {
	b := pulaBuforow.Get().(*[]byte)
	defer pulaBuforow.Put(b)
	return io.CopyBuffer(w, r, *b)
}

// Nowy tworzy magazyn; nieistniejący folder udziału, plik zamiast folderu albo powtórzona nazwa = błąd.
func Nowy(udzialy []konfiguracja.Udzial) (*Magazyn, error) {
	m := &Magazyn{udzialy: map[string]udzialWew{}, zegar: time.Now}
	for _, u := range udzialy {
		if _, jest := m.udzialy[u.Nazwa]; jest || u.Nazwa == "" {
			return nil, errors.New("powtórzona albo pusta nazwa udziału: " + u.Nazwa)
		}
		korzen, err := rozwiazKorzen(u.Sciezka)
		if err != nil {
			return nil, errors.New("folder udziału " + u.Nazwa + " niedostępny: " + err.Error())
		}
		m.udzialy[u.Nazwa] = udzialWew{Udzial: u, korzen: korzen}
		m.kolejnosc = append(m.kolejnosc, u)
	}
	return m, nil
}

// rozwiazKorzen: bezwzględna, rozwinięta ścieżka folderu udziału.
func rozwiazKorzen(sciezka string) (string, error) {
	abs, err := filepath.Abs(sciezka)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.New("to nie jest folder")
	}
	return filepath.Clean(real), nil
}

// Udzialy w kolejności z konfiguracji (kopia).
func (m *Magazyn) Udzialy() []konfiguracja.Udzial {
	return append([]konfiguracja.Udzial(nil), m.kolejnosc...)
}

// Udzial zwraca opis udziału albo rdzen.ErrNieZnaleziono.
func (m *Magazyn) Udzial(nazwa string) (konfiguracja.Udzial, error) {
	u, ok := m.udzialy[nazwa]
	if !ok {
		return konfiguracja.Udzial{}, rdzen.ErrNieZnaleziono
	}
	return u.Udzial, nil
}

// pelna rozwiązuje ścieżkę użytkownika; zapis=true w udziale tylko do odczytu = rdzen.ErrTylkoOdczyt.
func (m *Magazyn) pelna(udzial, sciezka string, zapis bool) (cel, error) {
	c, lex, err := m.lex(udzial, sciezka, zapis)
	if err != nil {
		return c, err
	}
	real, err := rozwiazWUdziale(c.u.korzen, lex)
	if err != nil {
		return c, err
	}
	if err := sprawdzUkryteNaDysku(c.u.korzen, real); err != nil {
		return c, err
	}
	if err := drugaLiniaObrony(c.u.korzen, real); err != nil {
		return c, err
	}
	c.dysk = real
	return c, nil
}

// drugaLiniaObrony: filepath.EvalSymlinks istniejącego wyniku też musi leżeć w korzeniu (nie rozwija
// junction na Windows, to robi rozwiazWUdziale; tu chroni przed błędem w tamtej logice).
func drugaLiniaObrony(korzen, real string) error {
	ev, err := filepath.EvalSymlinks(real)
	if errors.Is(err, fs.ErrNotExist) {
		return nil // celowo: nieistniejąca końcówka (nowy plik) sprawdzona już przez rozwiazWUdziale
	}
	if err != nil {
		return rdzen.ErrPozaUdzialem
	}
	if _, ok := wKorzeniu(korzen, ev); !ok {
		return rdzen.ErrPozaUdzialem
	}
	return sprawdzUkryteNaDysku(korzen, ev)
}

// pelnaBezKonca jak pelna, ale ostatni człon zostaje nierozwinięty: zmiana nazwy i kosz działają
// na samym dowiązaniu, nie na jego celu.
func (m *Magazyn) pelnaBezKonca(udzial, sciezka string) (cel, error) {
	c, lex, err := m.lex(udzial, sciezka, true)
	if err != nil || c.czysta == "/" {
		c.dysk = c.u.korzen
		return c, err
	}
	rodzic, err := rozwiazWUdziale(c.u.korzen, filepath.Dir(lex))
	if err != nil {
		return c, err
	}
	if err := sprawdzUkryteNaDysku(c.u.korzen, rodzic); err != nil {
		return c, err
	}
	c.dysk = filepath.Join(rodzic, filepath.Base(lex))
	return c, nil
}

// lex: udział, prawo zapisu, czyszczenie ścieżki (rdzen) i ukryte nazwy.
func (m *Magazyn) lex(udzial, sciezka string, zapis bool) (cel, string, error) {
	u, ok := m.udzialy[udzial]
	if !ok {
		return cel{}, "", rdzen.ErrNieZnaleziono
	}
	c := cel{u: u}
	if zapis && u.TylkoOdczyt {
		return c, "", rdzen.ErrTylkoOdczyt
	}
	czysta, err := rdzen.OczyscSciezke(sciezka)
	if err != nil {
		return c, "", err
	}
	if czyUkrytaSciezka(czysta) {
		return c, "", rdzen.ErrNieZnaleziono
	}
	lex, err := rdzen.SciezkaNaDysku(u.korzen, czysta)
	if err != nil {
		return c, "", err
	}
	c.czysta = czysta
	return c, lex, nil
}

// sprawdzUkryteNaDysku: dowiązanie mogło zaprowadzić np. do kosza; ukryte = nie znaleziono.
func sprawdzUkryteNaDysku(korzen, real string) error {
	rel, err := filepath.Rel(korzen, real)
	if err != nil {
		return rdzen.ErrPozaUdzialem
	}
	if czyUkrytaSciezka(filepath.ToSlash(rel)) {
		return rdzen.ErrNieZnaleziono
	}
	return nil
}

// wKorzeniu zwraca ścieżkę względną celu wobec korzenia albo ok=false, gdy cel jest poza nim.
// filepath.Rel na Windows porównuje bez wielkości liter.
func wKorzeniu(korzen, sciezka string) (string, bool) {
	rel, err := filepath.Rel(korzen, sciezka)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return rel, true
}

// czyDowiazanie: dowiązanie symboliczne albo punkt ponownej analizy (junction ma ModeIrregular).
func czyDowiazanie(tryb fs.FileMode) bool {
	return tryb&(fs.ModeSymlink|fs.ModeIrregular) != 0
}

// rozwiazWUdziale rozwija człon po członie dowiązania i junction poniżej korzenia (filepath.EvalSymlinks
// od Go 1.23 nie rozwija junction na Windows, więc robimy to sami). Cel poza korzeniem = ErrPozaUdzialem.
// Nieistniejąca końcówka ścieżki zostaje bez zmian (nowy plik, nowy folder).
func rozwiazWUdziale(korzen, dysk string) (string, error) {
	rel, ok := wKorzeniu(korzen, dysk)
	if !ok {
		return "", rdzen.ErrPozaUdzialem
	}
	if rel == "." {
		return korzen, nil
	}
	reszta := strings.Split(rel, string(filepath.Separator))
	biezacy, skoki := korzen, 0
	for len(reszta) > 0 {
		nast := filepath.Join(biezacy, reszta[0])
		reszta = reszta[1:]
		fi, err := os.Lstat(nast)
		if errors.Is(err, fs.ErrNotExist) {
			return filepath.Join(append([]string{nast}, reszta...)...), nil
		}
		if err != nil {
			return "", mapuj(err)
		}
		if !czyDowiazanie(fi.Mode()) {
			biezacy = nast
			continue
		}
		cel, err := os.Readlink(nast)
		if err != nil {
			biezacy = nast // celowo: punkt ponownej analizy bez celu (np. deduplikacja) to zwykły plik
			continue
		}
		if skoki++; skoki > konfiguracja.MagazynMaksSkokowDowiazan {
			return "", rdzen.ErrPozaUdzialem
		}
		relCelu, ok := wKorzeniu(korzen, dlugaNazwa(bezwzglednyCel(biezacy, cel)))
		if !ok {
			return "", rdzen.ErrPozaUdzialem
		}
		biezacy = korzen
		if relCelu != "." {
			reszta = append(strings.Split(relCelu, string(filepath.Separator)), reszta...)
		}
	}
	return biezacy, nil
}

// bezwzglednyCel zamienia cel dowiązania na ścieżkę bezwzględną (względny, "\x" od dysku, pełny).
// biezacy jest już rozwinięty, więc leksykalne ".." jest poprawne.
func bezwzglednyCel(biezacy, cel string) string {
	switch {
	case filepath.IsAbs(cel):
		return filepath.Clean(cel)
	case filepath.VolumeName(cel) == "" && (strings.HasPrefix(cel, string(filepath.Separator)) || strings.HasPrefix(cel, "/")):
		return filepath.Clean(filepath.VolumeName(biezacy) + cel)
	case filepath.VolumeName(cel) != "":
		return filepath.Clean(cel) // celowo: "C:x" (względny do dysku) trafi poza korzeń i zostanie odrzucony
	}
	return filepath.Join(biezacy, cel)
}

// czyUkryta: nazwa niewidoczna dla użytkownika (kosz, części, pliki tymczasowe, pliki systemowe).
func czyUkryta(nazwa string) bool {
	n := strings.ToLower(nazwa)
	if n == strings.ToLower(konfiguracja.KatalogKosza) ||
		strings.HasSuffix(n, konfiguracja.MagazynPrzyrostekCzesci) ||
		strings.HasPrefix(n, konfiguracja.MagazynPrefiksTymczasowy) {
		return true
	}
	for _, s := range konfiguracja.MagazynUkryteSystemowe {
		if n == s {
			return true
		}
	}
	for _, p := range konfiguracja.MagazynUkrytePrefiksy {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

// czyUkrytaSciezka: którykolwiek człon ścieżki ("/" jako separator) jest ukryty.
func czyUkrytaSciezka(sciezka string) bool {
	for _, c := range strings.Split(sciezka, "/") {
		if c != "" && c != "." && czyUkryta(c) {
			return true
		}
	}
	return false
}

// mapuj zamienia błędy systemu na błędy wspólne (bez ujawniania ścieżek dysku).
func mapuj(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		return rdzen.ErrNieZnaleziono
	case errors.Is(err, fs.ErrExist):
		return rdzen.ErrJuzIstnieje
	case errors.Is(err, fs.ErrPermission):
		return rdzen.ErrBrakPrawa
	}
	return err
}

// blokada dla ścieżki dysku (paskowanie, bez wielkości liter jak NTFS).
func (m *Magazyn) blokada(dysk string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(dysk))) // celowo: zapis do skrótu nie zwraca błędu
	return &m.blokady[h.Sum32()%konfiguracja.MagazynLiczbaBlokad]
}
