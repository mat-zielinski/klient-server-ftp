package konfiguracja

import "time"

// Stałe obszaru „aplikacja”: składanie serwerów, zamykanie, jedna instancja, usługa Windows.

// Serwery HTTP i HTTPS (bez WriteTimeout: duże pobrania trwają dowolnie długo).
const (
	// CzasNaglowkaHTTP: limit na odczyt nagłówków żądania (ochrona przed powolnymi klientami).
	CzasNaglowkaHTTP = 15 * time.Second
	// CzasBezczynnosciHTTP: po tylu sekundach bezczynności połączenie keep-alive jest zamykane.
	CzasBezczynnosciHTTP = 2 * time.Minute
	// MaksBajtowNaglowkaHTTP: największy łączny rozmiar nagłówków żądania.
	MaksBajtowNaglowkaHTTP = 1 << 20
	// CzasZamykania: ile czekamy na łagodne zamknięcie serwerów, potem połączenia są zrywane.
	CzasZamykania = 15 * time.Second
	// SchematHTTPS w adresie przekierowania i otwieranej stronie.
	SchematHTTPS = "https"
	// AdresLokalnyHTTP: zwykłe HTTP słucha tylko na pętli zwrotnej.
	AdresLokalnyHTTP = "127.0.0.1"
	// ProtokolHTTP i ProtokolHTTPS: nazwy w dzienniku.
	ProtokolHTTP  = "http"
	ProtokolHTTPS = "https"
)

// OkresSprzataniaUdostepnien: co ile usuwane są wygasłe linki publiczne.
const OkresSprzataniaUdostepnien = time.Hour

// Jedna instancja programu.
const (
	// NazwaMuteksuInstancji: Windows, mutex nazwany wspólny dla wszystkich sesji użytkowników i usługi.
	NazwaMuteksuInstancji = `Global\Skrytka_jedna_instancja`
	// PlikBlokadyInstancji: Linux, plik w KatalogDanych z blokadą flock.
	PlikBlokadyInstancji = "skrytka.lock"
)

// Otwieranie przeglądarki przy pierwszym uruchomieniu (kreator).
const (
	// Windows.
	ProgramPrzegladarki  = "rundll32"
	ArgumentPrzegladarki = "url.dll,FileProtocolHandler"
	// Linux: brak programu albo brak ekranu (zmienne DISPLAY / WAYLAND_DISPLAY) = pomijamy.
	ProgramPrzegladarkiLinux = "xdg-open"
	ZmiennaEkranuX11         = "DISPLAY"
	ZmiennaEkranuWayland     = "WAYLAND_DISPLAY"
)

// Usługa systemd (Linux).
const (
	// PlikUslugiSystemd: plik jednostki zapisywany przez --usluga instaluj (tylko root).
	PlikUslugiSystemd = "/etc/systemd/system/skrytka.service"
	// JednostkaSystemd: nazwa jednostki dla systemctl.
	JednostkaSystemd = "skrytka.service"
	// ProgramSystemctl: sterowanie usługami systemd.
	ProgramSystemctl = "systemctl"
	// UzytkownikUslugiLinux: konto systemowe usługi (zakłada skrypty/instaluj-linux.sh); brak = root.
	UzytkownikUslugiLinux = "skrytka"
	// UprawnieniaPlikuUslugi: plik jednostki czytelny dla wszystkich, zapis tylko root.
	UprawnieniaPlikuUslugi = 0o644
)

// Usługa Windows.
const (
	NazwaUslugi = "Skrytka"
	OpisUslugi  = "Serwer plików Skrytka (WWW i FTPS)"
	// CzasOperacjiUslugi: ile czekamy na zatrzymanie usługi przy poleceniu --usluga stop.
	CzasOperacjiUslugi = 30 * time.Second
	// OdstepSprawdzaniaUslugi: co ile pytamy menedżera usług o stan przy zatrzymywaniu.
	OdstepSprawdzaniaUslugi = 300 * time.Millisecond
)

// Argumenty wiersza poleceń.
const (
	ArgumentUsluga    = "--usluga"
	ArgumentKonsola   = "--konsola"
	PolecenieInstaluj = "instaluj"
	PolecenieUsun     = "usun"
	PolecenieStart    = "start"
	PolecenieStop     = "stop"
)

// Kody wyjścia procesu.
const (
	KodOK             = 0
	KodBlad           = 1
	KodZleArgumenty   = 2
	KodDrugaInstancja = 3
)
