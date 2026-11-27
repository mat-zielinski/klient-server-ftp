package magazyn

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// opisKosza: zawartość pliku <znacznik>.json obok folderu znacznika (oryginalna ścieżka elementu).
type opisKosza struct {
	Sciezka string `json:"sciezka"`
}

// DoKosza przenosi do <udział>/.skrytka_kosz/<znacznik czasu>/<ścieżka>. Każde usunięcie dostaje własny
// folder znacznika ("<znacznik>_2", "_3" …, gdy zajęty) i plik opisu z oryginalną ścieżką.
// Dowiązanie trafia do kosza samo (bez celu).
func (m *Magazyn) DoKosza(udzial, sciezka string) error {
	c, err := m.pelnaBezKonca(udzial, sciezka)
	if err != nil {
		return err
	}
	if c.czysta == "/" {
		return rdzen.ErrZlaNazwa
	}
	if _, err := os.Lstat(c.dysk); err != nil {
		return mapuj(err)
	}
	kosz, err := katalogKosza(c.u.korzen, true)
	if err != nil {
		return err
	}
	blok := m.blokada(kosz)
	blok.Lock()
	defer blok.Unlock()
	return przeniesDoKosza(c.dysk, kosz, m.zegar().Format(konfiguracja.MagazynFormatZnacznikaKosza), c.czysta)
}

// katalogKosza: <korzeń>/.skrytka_kosz; utworz=true zakłada go, gdy brak. Kosz będący dowiązaniem
// albo plikiem = ErrPozaUdzialem (nikt nie może przekierować kosza poza udział).
func katalogKosza(korzen string, utworz bool) (string, error) {
	kosz := filepath.Join(korzen, konfiguracja.KatalogKosza)
	fi, err := os.Lstat(kosz)
	if errors.Is(err, fs.ErrNotExist) && utworz {
		if err := os.Mkdir(kosz, konfiguracja.MagazynUprawnieniaFolderu); err != nil && !errors.Is(err, fs.ErrExist) {
			return "", mapuj(err)
		}
		fi, err = os.Lstat(kosz)
	}
	if err != nil {
		return "", mapuj(err)
	}
	if czyDowiazanie(fi.Mode()) || !fi.IsDir() {
		return "", rdzen.ErrPozaUdzialem
	}
	return kosz, nil
}

// przeniesDoKosza zakłada nowy folder kosz/<znacznik[_n]> (zajęty = kolejny wariant), zapisuje opis
// i przenosi źródło do kosz/<znacznik[_n]>/<czysta>. Wołający trzyma blokadę kosza.
func przeniesDoKosza(zrodlo, kosz, znacznik, czysta string) error {
	for i := 1; i <= konfiguracja.MagazynMaksProbKosza; i++ {
		nazwa := znacznik
		if i > 1 {
			nazwa += "_" + strconv.Itoa(i)
		}
		folder := filepath.Join(kosz, nazwa)
		if _, err := os.Lstat(folder + konfiguracja.MagazynPrzyrostekOpisuKosza); err == nil {
			continue
		}
		if err := os.Mkdir(folder, konfiguracja.MagazynUprawnieniaFolderu); err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return mapuj(err)
		}
		return przeniesDoFolderu(zrodlo, folder, czysta)
	}
	return rdzen.ErrJuzIstnieje
}

// przeniesDoFolderu: opis, foldery pośrednie i przeniesienie; błąd = sprzątnięcie folderu znacznika.
func przeniesDoFolderu(zrodlo, folder, czysta string) error {
	opis, err := json.Marshal(opisKosza{Sciezka: czysta})
	if err == nil {
		err = os.WriteFile(folder+konfiguracja.MagazynPrzyrostekOpisuKosza, opis, konfiguracja.MagazynUprawnieniaCzesci)
	}
	docelowy := filepath.Join(folder, filepath.FromSlash(czysta[1:]))
	if err == nil {
		err = os.MkdirAll(filepath.Dir(docelowy), konfiguracja.MagazynUprawnieniaFolderu)
	}
	if err == nil {
		err = os.Rename(zrodlo, docelowy)
	}
	if err != nil {
		_ = os.RemoveAll(folder)                                         // celowo: sprzątanie po nieudanym przeniesieniu
		_ = os.Remove(folder + konfiguracja.MagazynPrzyrostekOpisuKosza) // celowo: jw.
		return mapuj(err)
	}
	return nil
}

// Przywroc przenosi wpis kosza na jego oryginalne miejsce (brakujące foldery nadrzędne są tworzone).
// Zajęte miejsce: bez nadpisz = rdzen.ErrJuzIstnieje; z nadpisz to, co tam leży, idzie najpierw do kosza.
func (m *Magazyn) Przywroc(udzial, id string, nadpisz bool) error {
	znacznik, sciezka, err := rozbierzID(id)
	if err != nil {
		return err
	}
	c, err := m.pelnaBezKonca(udzial, sciezka)
	if err != nil {
		return err
	}
	kosz, err := katalogKosza(c.u.korzen, false)
	if err != nil {
		return err
	}
	odblokuj := m.zablokujDwie(kosz, c.dysk)
	defer odblokuj()
	zrodlo, err := elementWKoszu(kosz, znacznik, sciezka)
	if err != nil {
		return err
	}
	if err := m.zwolnijMiejsce(c, kosz, nadpisz); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.dysk), konfiguracja.MagazynUprawnieniaFolderu); err != nil {
		return rdzen.ErrJuzIstnieje // celowo: na drodze leży plik zamiast folderu
	}
	if err := os.Rename(zrodlo, c.dysk); err != nil {
		return mapuj(err)
	}
	sprzatnijZnacznik(filepath.Join(kosz, znacznik), filepath.Dir(zrodlo))
	return nil
}

// zwolnijMiejsce: wolne miejsce = nic; zajęte bez nadpisz = ErrJuzIstnieje; z nadpisz = obecny element do kosza.
func (m *Magazyn) zwolnijMiejsce(c cel, kosz string, nadpisz bool) error {
	if _, err := os.Lstat(c.dysk); err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil // celowo: plik zamiast folderu nadrzędnego zgłosi MkdirAll jako ErrJuzIstnieje
		}
		return mapuj(err)
	}
	if !nadpisz {
		return rdzen.ErrJuzIstnieje
	}
	return przeniesDoKosza(c.dysk, kosz, m.zegar().Format(konfiguracja.MagazynFormatZnacznikaKosza), c.czysta)
}

// UsunTrwale usuwa wpis kosza bez możliwości przywrócenia. Dowiązania wewnątrz są usuwane same (bez celu).
func (m *Magazyn) UsunTrwale(udzial, id string) error {
	znacznik, sciezka, err := rozbierzID(id)
	if err != nil {
		return err
	}
	kosz, err := m.koszDoZmiany(udzial)
	if err != nil {
		return err
	}
	blok := m.blokada(kosz)
	blok.Lock()
	defer blok.Unlock()
	return usunElement(kosz, znacznik, sciezka)
}

// Oproznij usuwa trwale wpisy usunięte dawniej niż starszeNiz (0 albo mniej = wszystkie). Zwraca ich liczbę.
func (m *Magazyn) Oproznij(udzial string, starszeNiz time.Duration) (int, error) {
	kosz, err := m.koszDoZmiany(udzial)
	if errors.Is(err, rdzen.ErrNieZnaleziono) {
		if _, jest := m.udzialy[udzial]; jest {
			return 0, nil // celowo: brak kosza = nic do opróżnienia
		}
	}
	if err != nil {
		return 0, err
	}
	blok := m.blokada(kosz)
	blok.Lock()
	defer blok.Unlock()
	wpisy, err := m.wpisyKosza(kosz)
	if err != nil {
		return 0, err
	}
	granica := m.zegar().Add(-starszeNiz)
	usuniete := 0
	for _, w := range wpisy {
		if starszeNiz > 0 && !w.Usunieto.Before(granica) {
			continue
		}
		znacznik, sciezka, err := rozbierzID(w.ID)
		if err == nil {
			err = usunElement(kosz, znacznik, sciezka)
		}
		if err != nil {
			return usuniete, err
		}
		usuniete++
	}
	return usuniete, nil
}

// koszDoZmiany: kosz udziału do zapisu (udział tylko do odczytu = ErrTylkoOdczyt, brak kosza = ErrNieZnaleziono).
func (m *Magazyn) koszDoZmiany(udzial string) (string, error) {
	u, ok := m.udzialy[udzial]
	if !ok {
		return "", rdzen.ErrNieZnaleziono
	}
	if u.TylkoOdczyt {
		return "", rdzen.ErrTylkoOdczyt
	}
	return katalogKosza(u.korzen, false)
}

// usunElement: trwałe usunięcie elementu kosza i pustych folderów po nim. Wołający trzyma blokadę kosza.
func usunElement(kosz, znacznik, sciezka string) error {
	dysk, err := elementWKoszu(kosz, znacznik, sciezka)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dysk); err != nil {
		return mapuj(err)
	}
	sprzatnijZnacznik(filepath.Join(kosz, znacznik), filepath.Dir(dysk))
	return nil
}

// sprzatnijZnacznik usuwa puste foldery od "od" w górę aż do folderu znacznika; gdy zniknie i on, także jego opis.
func sprzatnijZnacznik(folder, od string) {
	for p := od; ; p = filepath.Dir(p) {
		if _, ok := wKorzeniu(folder, p); !ok {
			return
		}
		if os.Remove(p) != nil {
			return // celowo: folder niepusty (albo już go nie ma), wyżej nie idziemy
		}
		if p == folder {
			_ = os.Remove(folder + konfiguracja.MagazynPrzyrostekOpisuKosza) // celowo: opis bez folderu jest zbędny
			return
		}
	}
}

// zablokujDwie bierze blokady dwóch ścieżek w stałej kolejności (bez zakleszczeń); zwraca zwolnienie.
func (m *Magazyn) zablokujDwie(a, b string) func() {
	pa, pb := m.blokada(a), m.blokada(b)
	if pa == pb {
		pa.Lock()
		return pa.Unlock
	}
	if m.indeksBlokady(pa) > m.indeksBlokady(pb) {
		pa, pb = pb, pa
	}
	pa.Lock()
	pb.Lock()
	return func() {
		pb.Unlock()
		pa.Unlock()
	}
}

func (m *Magazyn) indeksBlokady(b *sync.Mutex) int {
	for i := range m.blokady {
		if &m.blokady[i] == b {
			return i
		}
	}
	return -1
}
