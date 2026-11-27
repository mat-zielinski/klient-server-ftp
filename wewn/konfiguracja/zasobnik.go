package konfiguracja

import "time"

// Stałe obszaru „zasobnik”: ikona w zasobniku systemowym (Windows) albo StatusNotifier (Linux z pulpitem).

// Argument i warunki pokazania zasobnika.
const (
	// ArgumentBezZasobnika: zwykły serwer bez ikony w zasobniku.
	ArgumentBezZasobnika = "--bez-zasobnika"
	// ZmiennaSystemd: ustawiana przez systemd w procesach usług (wtedy bez zasobnika).
	ZmiennaSystemd = "INVOCATION_ID"
	// SystemWindows i SystemLinux: wartości runtime.GOOS, w których zasobnik działa.
	SystemWindows = "windows"
	SystemLinux   = "linux"
)

// Odświeżanie i limity.
const (
	// ZasobnikOkresOdswiezania: co ile sprawdzamy rejestr transferów (dymek, ikona, podmenu).
	ZasobnikOkresOdswiezania = time.Second
	// ZasobnikMaksTransferow: najwięcej pozycji w podmenu „Aktywne transfery”.
	ZasobnikMaksTransferow = 5
	// ZasobnikMaksDymka: najdłuższy dymek w znakach (Windows: 128 znaków UTF-16 z zerem końcowym).
	ZasobnikMaksDymka = 127
	// ZasobnikCzasWyjscia: ile czekamy na koniec pętli zasobnika po zatrzymaniu serwera, zanim proces
	// zakończy się sam (pętla, której okno się nie utworzyło, nie kończy się na Quit).
	ZasobnikCzasWyjscia = 5 * time.Second
	// ZasobnikOkresMotywu: co ile sprawdzamy, czy pasek zadań jest jasny (wybór ciemnej ikony).
	ZasobnikOkresMotywu = 30 * time.Second
	// ZasobnikOkresAdresu: co ile od nowa ustalamy adres dla klientów (lista interfejsów sieci).
	ZasobnikOkresAdresu = time.Minute
	// ZasobnikMaksNazwyPliku: dłuższa nazwa pliku w podmenu transferów jest skracana.
	ZasobnikMaksNazwyPliku = 40
	// ZasobnikKrokJednostek: podstawa jednostek prędkości (jak w interfejsie: KB = 1024 B).
	ZasobnikKrokJednostek = 1024
	// ZasobnikWielokropek: koniec skróconego tekstu.
	ZasobnikWielokropek = "…"
	// ZasobnikZnakPobrania i ZasobnikZnakWgrania: kierunek transferu w podmenu.
	ZasobnikZnakPobrania = "↓"
	ZasobnikZnakWgrania  = "↑"
)

// ZasobnikJednostki: jednostki prędkości (dopisywane „/s”).
var ZasobnikJednostki = []string{"B", "KB", "MB", "GB", "TB"}

// Formy liczebników w dymku: 1, 2–4 (bez 12–14), reszta.
var (
	ZasobnikFormyPobran = [3]string{"pobranie", "pobrania", "pobrań"}
	ZasobnikFormyWgran  = [3]string{"wgranie", "wgrania", "wgrań"}
)

// Teksty dymka i menu.
const (
	ZasobnikTekstDziala      = "działa"
	ZasobnikTekstStartuje    = "uruchamianie"
	ZasobnikTekstZamykanie   = "zamykanie"
	ZasobnikSeparatorNazwy   = " — "
	ZasobnikSeparatorPol     = " · "
	ZasobnikMenuOtworz       = "Otwórz panel"
	ZasobnikMenuKopiuj       = "Kopiuj adres dla klientów"
	ZasobnikMenuTransfery    = "Aktywne transfery"
	ZasobnikMenuBrakTransfer = "Brak aktywnych transferów"
	ZasobnikMenuAutostart    = "Uruchamiaj z Windows"
	ZasobnikMenuAutostartLin = "Uruchamiaj po zalogowaniu"
	ZasobnikMenuFolder       = "Folder danych i logów"
	ZasobnikMenuZakoncz      = "Zakończ"
	// ZasobnikKtoNieznany: transfer bez nazwy użytkownika (link publiczny).
	ZasobnikKtoNieznany = "gość"
)

// Autostart.
const (
	// KluczAutostartuWindows: HKCU, programy uruchamiane po zalogowaniu.
	KluczAutostartuWindows = `Software\Microsoft\Windows\CurrentVersion\Run`
	// KluczMotywuWindows i WartoscJasnegoPaska: HKCU, 1 = jasny pasek zadań (wtedy ciemne ikony).
	KluczMotywuWindows  = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	WartoscJasnegoPaska = "SystemUsesLightTheme"
	// NazwaWpisuAutostartu: nazwa wartości w kluczu Run.
	NazwaWpisuAutostartu = "Skrytka"
	// KatalogAutostartuLinux: podkatalog katalogu konfiguracji użytkownika (~/.config).
	KatalogAutostartuLinux = "autostart"
	// PlikAutostartuLinux: plik .desktop w KatalogAutostartuLinux.
	PlikAutostartuLinux = "skrytka.desktop"
	// UprawnieniaPlikuAutostartu: plik .desktop użytkownika.
	UprawnieniaPlikuAutostartu = 0o644
)

// Programy pomocnicze (schowek, menedżer plików).
const (
	// ProgramSchowkaWindows: tekst ze standardowego wejścia do schowka.
	ProgramSchowkaWindows = "clip"
	// ProgramEksploratora: otwiera folder w Eksploratorze Windows.
	ProgramEksploratora = "explorer"
	// ProgramSchowkaWayland i ProgramSchowkaX11 (z ArgumentySchowkaX11): schowek na Linuksie.
	ProgramSchowkaWayland = "wl-copy"
	ProgramSchowkaX11     = "xclip"
)

// ArgumentySchowkaX11: xclip do schowka „clipboard” (nie zaznaczenia).
var ArgumentySchowkaX11 = []string{"-selection", "clipboard"}
