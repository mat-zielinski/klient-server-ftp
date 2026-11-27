package indeks

import (
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	"skrytka/wewn/rdzen"
)

var errNieznanyUdzial = errors.New("nieznany udział")

// rodzaje: numer rodzaju w pozycji (uint8) -> nazwa z rdzenia.
var rodzaje = []string{
	rdzen.RodzajFolder, rdzen.RodzajObraz, rdzen.RodzajWideo, rdzen.RodzajDzwiek,
	rdzen.RodzajPDF, rdzen.RodzajTekst, rdzen.RodzajArchiwum, rdzen.RodzajInny,
}

// numerRodzaju: odwrotność rodzaje; nieznany = RodzajInny.
func numerRodzaju(r string) uint8 {
	for i, n := range rodzaje {
		if n == r {
			return uint8(i)
		}
	}
	return uint8(len(rodzaje) - 1)
}

// pozycja: zwarty opis pliku albo folderu (ścieżka = foldery[rodzic] + nazwa).
type pozycja struct {
	nazwa    string
	zwinieta string // małe litery bez ogonków; ta sama pamięć co nazwa, gdy identyczne
	rozmiar  int64
	czas     int64 // UnixNano
	rodzic   int32
	rodzaj   uint8
	folder   bool
}

// zbior: niezmienny po zbudowaniu indeks jednego udziału.
type zbior struct {
	foldery    []string // ścieżki folderów ("/", "/a", "/a/b"); indeks = pozycja.rodzic
	pozycje    []pozycja
	statystyka StatystykaUdzialu
}

// wpis odtwarza rdzen.Wpis z pozycji.
func (z *zbior) wpis(p *pozycja) rdzen.Wpis {
	return rdzen.Wpis{
		Nazwa:     p.nazwa,
		Sciezka:   path.Join(z.foldery[p.rodzic], p.nazwa),
		Folder:    p.folder,
		Rozmiar:   p.rozmiar,
		Zmieniono: time.Unix(0, p.czas),
		Rodzaj:    rodzaje[p.rodzaj],
	}
}

// nowaPozycja z wpisu magazynu.
func nowaPozycja(w rdzen.Wpis, rodzic int32) pozycja {
	return pozycja{
		nazwa:    w.Nazwa,
		zwinieta: zwin(w.Nazwa), // ta sama pamięć co nazwa, gdy nic do zwinięcia
		rozmiar:  w.Rozmiar,
		czas:     w.Zmieniono.UnixNano(),
		rodzic:   rodzic,
		rodzaj:   numerRodzaju(w.Rodzaj),
		folder:   w.Folder,
	}
}

// uporzadkuj sortuje pozycje wg ścieżki (wyniki powtarzalne mimo równoległego przejścia).
func (z *zbior) uporzadkuj() {
	sort.Slice(z.pozycje, func(i, j int) bool {
		a, b := &z.pozycje[i], &z.pozycje[j]
		if a.rodzic != b.rodzic {
			fa, fb := z.foldery[a.rodzic], z.foldery[b.rodzic]
			if fa != fb {
				return fa < fb
			}
		}
		return a.nazwa < b.nazwa
	})
}

// podFolderami zwraca znaczniki folderów leżących w start (start włącznie); start "/" = nil (wszystkie).
func (z *zbior) podFolderami(start string) []bool {
	if start == "/" {
		return nil
	}
	znaczniki := make([]bool, len(z.foldery))
	przedrostek := start + "/"
	for i, f := range z.foldery {
		znaczniki[i] = f == start || strings.HasPrefix(f, przedrostek)
	}
	return znaczniki
}
