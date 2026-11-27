package konfiguracja

// Stałe obszaru „magazyn” (dostęp do dysku).

const (
	// MagazynRozmiarBufora: bufor kopiowania (zapis, ZIP) brany z sync.Pool.
	MagazynRozmiarBufora = 1 << 20
	// MagazynDomyslnyLimitSzukania: limit wyników, gdy wołający poda 0 albo mniej.
	MagazynDomyslnyLimitSzukania = 500
	// MagazynMaksLimitSzukania: górna granica limitu wyników wyszukiwania.
	MagazynMaksLimitSzukania = 5000
	// MagazynMaksSkokowDowiazan: ile dowiązań/junction można rozwinąć w jednej ścieżce (pętle).
	MagazynMaksSkokowDowiazan = 40
	// MagazynPrzyrostekCzesci: plik niedokończonego wgrywania w kawałkach.
	MagazynPrzyrostekCzesci = ".skrytka_czesc"
	// MagazynPrefiksTymczasowy: pliki tymczasowe zapisu atomowego (ukryte w listach).
	MagazynPrefiksTymczasowy = ".skrytka_tmp_"
	// MagazynFormatZnacznikaKosza: nazwa podfolderu kosza (czas usunięcia).
	MagazynFormatZnacznikaKosza = "2006-01-02_15-04-05"
	// MagazynMaksProbKosza: ile wariantów nazwy (_2, _3 …) próbować, gdy znacznik jest zajęty.
	MagazynMaksProbKosza = 1000
	// MagazynLiczbaBlokad: liczba blokad (paskowanie po ścieżce) dla zapisu i kawałków.
	MagazynLiczbaBlokad = 64
	// MagazynUprawnieniaFolderu: nowe foldery w udziale (Windows i tak dziedziczy ACL).
	MagazynUprawnieniaFolderu = 0o755
	// MagazynUprawnieniaCzesci: plik kawałków wgrywania.
	MagazynUprawnieniaCzesci = 0o644
)

// MagazynUkryteSystemowe: nazwy plików systemowych Windows i Linux ukrywane w listach (bez wielkości liter).
var MagazynUkryteSystemowe = []string{
	"desktop.ini", "thumbs.db", "$recycle.bin", "system volume information", // Windows
	"lost+found", // Linux
}

// MagazynUkrytePrefiksy: początki nazw ukrywanych (Linux: kosz pulpitu na nośniku ".Trash-1000").
var MagazynUkrytePrefiksy = []string{".trash-"}

// MagazynBezKompresji: rozszerzenia już skompresowane, w ZIP zapisywane metodą Store.
var MagazynBezKompresji = []string{
	".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic",
	".mp4", ".mkv", ".mov", ".avi", ".webm", ".mp3",
	".zip", ".7z", ".rar", ".gz",
}
