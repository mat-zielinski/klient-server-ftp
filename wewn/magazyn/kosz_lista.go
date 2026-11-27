package magazyn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// ErrZlyIDKosza: identyfikator wpisu kosza nie da się odczytać (zły base64, zły znacznik, pusta ścieżka).
var ErrZlyIDKosza = errors.New("nieprawidłowy identyfikator wpisu kosza")

// WpisKosza: jeden usunięty element. ID = base64url("<znacznik>/<ścieżka względna>"), bez ścieżek dysku.
type WpisKosza struct {
	ID       string
	Nazwa    string
	Sciezka  string // oryginalne miejsce w udziale ("/a/b.txt")
	Usunieto time.Time
	Rozmiar  int64 // plik: rozmiar; folder: suma plików
	Folder   bool
	Rodzaj   string // rdzen.RodzajPliku (ikona w interfejsie)
}

// ListaKosza: wpisy kosza udziału, od najnowszych. Brak kosza = pusta lista.
func (m *Magazyn) ListaKosza(udzial string) ([]WpisKosza, error) {
	u, ok := m.udzialy[udzial]
	if !ok {
		return nil, rdzen.ErrNieZnaleziono
	}
	kosz, err := katalogKosza(u.korzen, false)
	if errors.Is(err, rdzen.ErrNieZnaleziono) {
		return []WpisKosza{}, nil
	}
	if err != nil {
		return nil, err
	}
	return m.wpisyKosza(kosz)
}

// wpisyKosza czyta foldery znaczników (dowiązania i obce nazwy pomijane).
func (m *Magazyn) wpisyKosza(kosz string) ([]WpisKosza, error) {
	foldery, err := os.ReadDir(kosz)
	if err != nil {
		return nil, mapuj(err)
	}
	loc := m.zegar().Location()
	wynik := []WpisKosza{}
	for _, d := range foldery {
		if !d.IsDir() || czyDowiazanie(d.Type()) {
			continue
		}
		czas, ok := czasZnacznika(d.Name(), loc)
		if !ok {
			continue
		}
		wynik = append(wynik, wpisyZnacznika(kosz, d.Name(), czas)...)
	}
	sort.SliceStable(wynik, func(i, j int) bool {
		if !wynik[i].Usunieto.Equal(wynik[j].Usunieto) {
			return wynik[i].Usunieto.After(wynik[j].Usunieto)
		}
		return wynik[i].Sciezka < wynik[j].Sciezka
	})
	return wynik, nil
}

// czasZnacznika: "RRRR-MM-DD_GG-MM-SS" z opcjonalnym "_n" (n ≥ 2) → czas w strefie zegara magazynu.
func czasZnacznika(nazwa string, loc *time.Location) (time.Time, bool) {
	dl := len(konfiguracja.MagazynFormatZnacznikaKosza)
	if len(nazwa) < dl {
		return time.Time{}, false
	}
	if reszta := nazwa[dl:]; reszta != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(reszta, "_"))
		if !strings.HasPrefix(reszta, "_") || err != nil || n < 2 || strconv.Itoa(n) != reszta[1:] {
			return time.Time{}, false
		}
	}
	t, err := time.ParseInLocation(konfiguracja.MagazynFormatZnacznikaKosza, nazwa[:dl], loc)
	return t, err == nil
}

// wpisyZnacznika: jeden wpis z pliku opisu; bez opisu (kosz sprzed opisów) zgadywanie z drzewa folderów.
func wpisyZnacznika(kosz, znacznik string, czas time.Time) []WpisKosza {
	folder := filepath.Join(kosz, znacznik)
	if sciezka, ok := czytajOpis(folder); ok {
		if w, ok := wpisKosza(folder, znacznik, sciezka, czas); ok {
			return []WpisKosza{w}
		}
		return nil
	}
	var wynik []WpisKosza
	zbierzStare(folder, znacznik, "", czas, &wynik)
	return wynik
}

// czytajOpis: oryginalna ścieżka z <folder>.json; zła albo brak = ok=false.
func czytajOpis(folder string) (string, bool) {
	b, err := os.ReadFile(folder + konfiguracja.MagazynPrzyrostekOpisuKosza)
	if err != nil {
		return "", false
	}
	var o opisKosza
	if json.Unmarshal(b, &o) != nil {
		return "", false
	}
	czysta, err := rdzen.OczyscSciezke(o.Sciezka)
	if err != nil || czysta != o.Sciezka || czysta == "/" || czyUkrytaSciezka(czysta) {
		return "", false
	}
	return czysta, true
}

// zbierzStare: stary układ (kilka usunięć w jednym znaczniku). Folder z dokładnie jednym elementem uznajemy
// za folder pośredni i schodzimy niżej; plik, pusty folder albo folder z kilkoma elementami = wpis.
func zbierzStare(folder, znacznik, rel string, czas time.Time, wynik *[]WpisKosza) {
	dysk := filepath.Join(folder, filepath.FromSlash(rel))
	fi, err := os.Lstat(dysk)
	if err != nil {
		return
	}
	if rel != "" && (!fi.IsDir() || czyDowiazanie(fi.Mode())) {
		dodajWpis(folder, znacznik, rel, czas, wynik)
		return
	}
	dzieci, err := os.ReadDir(dysk)
	if err != nil {
		return
	}
	if rel != "" && len(dzieci) != 1 {
		dodajWpis(folder, znacznik, rel, czas, wynik)
		return
	}
	for _, d := range dzieci {
		if !czyUkryta(d.Name()) {
			zbierzStare(folder, znacznik, path.Join(rel, d.Name()), czas, wynik)
		}
	}
}

func dodajWpis(folder, znacznik, rel string, czas time.Time, wynik *[]WpisKosza) {
	if w, ok := wpisKosza(folder, znacznik, "/"+rel, czas); ok {
		*wynik = append(*wynik, w)
	}
}

// wpisKosza buduje wpis dla elementu folder/<sciezka>; element musi istnieć.
func wpisKosza(folder, znacznik, sciezka string, czas time.Time) (WpisKosza, bool) {
	dysk := filepath.Join(folder, filepath.FromSlash(sciezka[1:]))
	fi, err := os.Lstat(dysk)
	if err != nil {
		return WpisKosza{}, false
	}
	w := WpisKosza{
		ID: kodujID(znacznik, sciezka), Nazwa: path.Base(sciezka), Sciezka: sciezka, Usunieto: czas,
		Folder: fi.IsDir() && !czyDowiazanie(fi.Mode()),
	}
	w.Rodzaj = rdzen.RodzajPliku(w.Nazwa, w.Folder)
	if w.Folder {
		w.Rozmiar, _ = rozmiarDrzewa(dysk) // celowo: nieczytelny podfolder = rozmiar częściowy
	} else if !czyDowiazanie(fi.Mode()) {
		w.Rozmiar = fi.Size()
	}
	return w, true
}

// kodujID: base64url bez dopełnienia z "<znacznik><ścieżka>" (ścieżka zaczyna się od "/").
func kodujID(znacznik, sciezka string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(znacznik + sciezka))
}

// rozbierzID odwraca kodujID i sprawdza wszystko, co da się sprawdzić bez dysku: kanoniczną ścieżkę
// (bez "..", "\", dysków, ukrytych członów) i poprawny znacznik. Zwraca znacznik i ścieżkę "/a/b".
func rozbierzID(id string) (string, string, error) {
	if id == "" || len(id) > konfiguracja.MagazynMaksDlugoscIDKosza {
		return "", "", ErrZlyIDKosza
	}
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil || !utf8.Valid(b) {
		return "", "", ErrZlyIDKosza
	}
	znacznik, reszta, ok := strings.Cut(string(b), "/")
	if !ok || reszta == "" {
		return "", "", ErrZlyIDKosza
	}
	if _, ok := czasZnacznika(znacznik, time.UTC); !ok {
		return "", "", ErrZlyIDKosza
	}
	sciezka := "/" + reszta
	czysta, err := rdzen.OczyscSciezke(sciezka)
	if err != nil {
		return "", "", err
	}
	if czysta != sciezka || czysta == "/" {
		return "", "", ErrZlyIDKosza
	}
	if czyUkrytaSciezka(czysta) {
		return "", "", rdzen.ErrNieZnaleziono
	}
	return znacznik, czysta, nil
}

// elementWKoszu: ścieżka dysku elementu kosz/<znacznik>/<sciezka>. Folder znacznika nie może być dowiązaniem,
// dowiązania po drodze muszą zostać w folderze znacznika (sam element może być dowiązaniem: działa się na nim).
// Gdy znacznik ma opis, ścieżka musi się z nim zgadzać.
func elementWKoszu(kosz, znacznik, sciezka string) (string, error) {
	folder := filepath.Join(kosz, znacznik)
	fi, err := os.Lstat(folder)
	if err != nil {
		return "", mapuj(err)
	}
	if czyDowiazanie(fi.Mode()) || !fi.IsDir() {
		return "", rdzen.ErrPozaUdzialem
	}
	if opis, ok := czytajOpis(folder); ok && opis != sciezka {
		return "", rdzen.ErrNieZnaleziono
	}
	lex, err := rdzen.SciezkaNaDysku(folder, sciezka)
	if err != nil {
		return "", err
	}
	rodzic, err := rozwiazWUdziale(folder, filepath.Dir(lex))
	if err != nil {
		return "", err
	}
	dysk := filepath.Join(rodzic, filepath.Base(lex))
	if _, ok := wKorzeniu(folder, dysk); !ok {
		return "", rdzen.ErrPozaUdzialem
	}
	if _, err := os.Lstat(dysk); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", rdzen.ErrNieZnaleziono
		}
		return "", mapuj(err)
	}
	return dysk, nil
}

// SciezkaWpisuKosza: oryginalna ścieżka zapisana w identyfikatorze (do dziennika), bez dotykania dysku.
func SciezkaWpisuKosza(id string) (string, error) {
	_, sciezka, err := rozbierzID(id)
	return sciezka, err
}
