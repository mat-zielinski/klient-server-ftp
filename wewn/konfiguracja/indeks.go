package konfiguracja

import (
	"path/filepath"
	"time"
)

// Stałe obszaru „indeks” (indeks plików w tle, miniatury obrazów, sumy SHA-256).

// Indeks plików.
const (
	// IndeksOkresOdswiezania: co ile indeks przechodzi wszystkie udziały od nowa.
	IndeksOkresOdswiezania = 15 * time.Minute
	// IndeksOpoznienieOdswiezenia: zwłoka po Odswiez(udzial), żeby seria zapisów dała jedno przejście.
	IndeksOpoznienieOdswiezenia = 2 * time.Second
	// IndeksRownoleglosc: ile folderów jednego udziału czytamy naraz.
	IndeksRownoleglosc = 2
	// IndeksPrzerwaMiedzyFolderami: przerwa po każdym folderze (niski priorytet, dysk dla użytkowników).
	IndeksPrzerwaMiedzyFolderami = 2 * time.Millisecond
	// IndeksMaksGlebokosc: najgłębszy poziom folderów (ochrona przed pętlą dowiązań w udziale).
	IndeksMaksGlebokosc = 64
	// IndeksMaksPozycji: górna granica pozycji jednego udziału (pamięć; reszta nie trafia do indeksu).
	IndeksMaksPozycji = 5_000_000
)

// Miniatury obrazów.
const (
	// KatalogMiniaturNazwa: pamięć podręczna miniatur w folderze danych.
	KatalogMiniaturNazwa = "miniatury"
	// MiniaturaRozmiarS, MiniaturaRozmiarM: dłuższy bok miniatury w pikselach (kafelki / galeria).
	MiniaturaRozmiarS = 160
	MiniaturaRozmiarM = 480
	// MiniaturaJakoscJPEG: jakość zapisu miniatury.
	MiniaturaJakoscJPEG = 80
	// MiniaturaMaksBajtowZrodla: większych plików nie skalujemy.
	MiniaturaMaksBajtowZrodla = 50 << 20
	// MiniaturaMaksPikseliZrodla: ochrona przed bombą dekompresyjną (szerokość × wysokość z nagłówka).
	MiniaturaMaksPikseliZrodla = 50_000_000
	// MiniaturaMaksBajtowExif: ile początkowych bajtów JPEG przeglądamy w poszukiwaniu orientacji EXIF.
	MiniaturaMaksBajtowExif = 128 << 10
	// MiniaturaLimitCacheMB: powyżej tego rozmiaru pamięci podręcznej usuwamy najstarsze miniatury.
	MiniaturaLimitCacheMB = 512
	// MiniaturaCelSprzatania: po sprzątaniu cache zajmuje najwyżej tyle procent limitu.
	MiniaturaCelSprzatania = 90
	// MiniaturaSprzatanieCo: sprzątanie cache po tylu nowych miniaturach.
	MiniaturaSprzatanieCo = 200
	// MiniaturaUprawnieniaFolderu: folder pamięci podręcznej.
	MiniaturaUprawnieniaFolderu = 0o755
)

// MiniaturaRozszerzenia: formaty, które umiemy zdekodować (reszta obrazów bez miniatury).
var MiniaturaRozszerzenia = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

// Sumy SHA-256.
const (
	// SumyMaksWpisowCache: ile policzonych sum trzymamy w pamięci.
	SumyMaksWpisowCache = 10_000
)

// KatalogMiniatur zwraca dane/miniatury.
func KatalogMiniatur() string { return filepath.Join(KatalogDanych(), KatalogMiniaturNazwa) }
