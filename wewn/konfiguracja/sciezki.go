package konfiguracja

import (
	"os"
	"path/filepath"
)

// Jedyne miejsce, które zna położenie programu. Wszystkie ścieżki względem folderu aplikacji.

// Nazwy plików i katalogów stanu.
const (
	KatalogDanychNazwa    = "dane"
	KatalogLogowNazwa     = "logi"
	PlikKonfiguracjiNazwa = "konfiguracja.json"
	PlikKontNazwa         = "konta.json"
	PlikUdostepnienNazwa  = "udostepnienia.json"
	KatalogCertyfikatow   = "certyfikaty"
	// KatalogKosza: kosz aplikacji w korzeniu każdego udziału (ukryty w listach).
	KatalogKosza = ".skrytka_kosz"
	// ZmiennaKataloguAplikacji pozwala testom i deweloperom wskazać inny folder.
	ZmiennaKataloguAplikacji = "SKRYTKA_KATALOG"
)

// KatalogAplikacji zwraca folder z plikiem .exe (albo SKRYTKA_KATALOG).
func KatalogAplikacji() string {
	if k := os.Getenv(ZmiennaKataloguAplikacji); k != "" {
		return k
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// KatalogDanych: konfiguracja, konta, udostępnienia, certyfikaty.
func KatalogDanych() string { return filepath.Join(KatalogAplikacji(), KatalogDanychNazwa) }

// KatalogLogow: dziennik_RRRR-MM.jsonl, bledy.log.
func KatalogLogow() string { return filepath.Join(KatalogAplikacji(), KatalogLogowNazwa) }

// PlikKonfiguracji zwraca pełną ścieżkę dane/konfiguracja.json.
func PlikKonfiguracji() string { return filepath.Join(KatalogDanych(), PlikKonfiguracjiNazwa) }

// PlikKont zwraca pełną ścieżkę dane/konta.json.
func PlikKont() string { return filepath.Join(KatalogDanych(), PlikKontNazwa) }

// PlikUdostepnien zwraca pełną ścieżkę dane/udostepnienia.json.
func PlikUdostepnien() string { return filepath.Join(KatalogDanych(), PlikUdostepnienNazwa) }

// KatalogCertyfikatowPelny zwraca dane/certyfikaty.
func KatalogCertyfikatowPelny() string { return filepath.Join(KatalogDanych(), KatalogCertyfikatow) }
