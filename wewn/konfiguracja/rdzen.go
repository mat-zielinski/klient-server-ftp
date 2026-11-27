package konfiguracja

// Stałe obszaru „rdzen”: limity ścieżek i nazw, rozszerzenia rodzajów plików, klucze sortowania.

// Limity długości (w bajtach UTF-8) sprawdzane przez rdzen.OczyscSciezke i rdzen.SprawdzNazwe.
const (
	// MaksDlugoscNazwy: najdłuższa nazwa pliku/folderu (NTFS: 255 jednostek UTF-16).
	MaksDlugoscNazwy = 255
	// MaksDlugoscSciezki: najdłuższa ścieżka względna w udziale.
	MaksDlugoscSciezki = 4096
	// MaksGlebokoscSciezki: najwięcej członów ścieżki względnej.
	MaksGlebokoscSciezki = 128
)

// Klucze sortowania (rdzen.Sortuj); nieznany klucz = KluczSortNazwa.
const (
	KluczSortNazwa     = "nazwa"
	KluczSortRozmiar   = "rozmiar"
	KluczSortZmieniono = "zmieniono"
)

// ZnakiZakazaneWNazwie: znaki niedozwolone w nazwie pliku na Windows (poza znakami sterującymi).
const ZnakiZakazaneWNazwie = `/\:*?"<>|`

// NazwyZarezerwowaneWindows: nazwy urządzeń; zakazane także z rozszerzeniem ("nul.txt") i bez względu na wielkość liter.
var NazwyZarezerwowaneWindows = []string{
	"CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$", "CLOCK$",
	"COM0", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "COM¹", "COM²", "COM³",
	"LPT0", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "LPT¹", "LPT²", "LPT³",
}

// Rozszerzenia (małymi literami, z kropką) przypisane do rodzajów plików (rdzen.RodzajPliku).
var (
	RozszerzeniaObrazow = []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif", ".heic", ".heif", ".tif", ".tiff",
	}
	RozszerzeniaWideo = []string{
		".mp4", ".mkv", ".webm", ".mov", ".avi", ".wmv", ".m4v", ".mpg", ".mpeg", ".3gp", ".flv",
	}
	RozszerzeniaDzwiekow = []string{
		".mp3", ".wav", ".flac", ".ogg", ".oga", ".opus", ".m4a", ".aac", ".wma", ".mid", ".midi",
	}
	RozszerzeniaPDF     = []string{".pdf"}
	RozszerzeniaTekstow = []string{
		".txt", ".md", ".log", ".csv", ".tsv", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf",
		".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".svelte", ".html", ".htm", ".css", ".c", ".h", ".cpp", ".hpp",
		".cs", ".java", ".rs", ".sh", ".bat", ".cmd", ".ps1", ".sql", ".srt", ".vtt",
	}
	RozszerzeniaArchiwow = []string{
		".zip", ".7z", ".rar", ".tar", ".gz", ".tgz", ".bz2", ".xz", ".zst", ".iso",
	}
)
