package konfiguracja

// Stałe obszaru „ftp” (serwer FTPS, wewn/ftp). Właściciel: agent „ftp”.
const (
	// FTPMaksPolaczenNaIP: najwięcej jednoczesnych połączeń sterujących z jednego adresu IP
	// (FileZilla otwiera kilka naraz: lista + 2 transfery).
	FTPMaksPolaczenNaIP = 10
	// FTPBezczynnoscS: po tylu sekundach bez polecenia serwer rozłącza klienta.
	FTPBezczynnoscS = 300
	// FTPLimitPolaczeniaDanychS: tyle sekund czekamy na połączenie danych (tryb pasywny).
	FTPLimitPolaczeniaDanychS = 30
	// FTPPowitanie: odpowiedź 220 po połączeniu.
	FTPPowitanie = NazwaAplikacji + " FTPS " + Wersja + ". Wymagane AUTH TLS."
	// FTPBaner: odpowiedź na polecenie STAT.
	FTPBaner = NazwaAplikacji + " FTPS"
	// FTPProtokol: nazwa protokołu w transferach i dzienniku.
	FTPProtokol = "ftps"
	// FTPPrawaFolderu i FTPPrawaPliku: tryb pokazywany w liście (LIST); zapis i tak sprawdza bramka.
	FTPPrawaFolderu = 0o755
	FTPPrawaPliku   = 0o644
)
