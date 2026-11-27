package konfiguracja

import "time"

// Stałe obszaru „http” (widok HTTP, docs/api.md).

// Sesja, CSRF i nagłówki.
const (
	// NazwaCiasteczkaSesji: ciasteczko z tokenem sesji (HttpOnly, SameSite=Strict, Secure przy HTTPS).
	NazwaCiasteczkaSesji = "skrytka_sesja"
	// NaglowekCSRF i WartoscCSRF: wymagane w każdym żądaniu zmieniającym stan (POST/PUT/DELETE/PATCH).
	NaglowekCSRF = "X-Skrytka"
	WartoscCSRF  = "1"
	// NaglowekHaslaLinku: hasło linku publicznego (zamiast parametru w adresie).
	NaglowekHaslaLinku = "X-Haslo"
	// PolitykaCSP: tylko własne źródła; obrazy i media także z blob: i data: (podgląd w interfejsie).
	PolitykaCSP = "default-src 'self'; img-src 'self' blob: data:; media-src 'self' blob:; " +
		"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'"
	// PolitykaCSPPlikuAktywnego: dodatkowa piaskownica dla plików użytkownika typu HTML/SVG/XML.
	PolitykaCSPPlikuAktywnego = "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'"
	// PolitykaReferrer: adresy plików nie wyciekają do innych stron.
	PolitykaReferrer = "no-referrer"
)

// Limity żądań.
const (
	// MaksCialoJSON: największa treść żądania JSON (bajty).
	MaksCialoJSON = 1 << 20
	// MaksKawalekWgrania: największy kawałek PUT /api/wgraj (bajty).
	MaksKawalekWgrania = 8 << 20
	// LimitWynikowSzukania: najwięcej wyników GET /api/szukaj.
	LimitWynikowSzukania = 500
	// DomyslnyLimitDziennika i MaksLimitDziennika: liczba wpisów GET /api/dziennik.
	DomyslnyLimitDziennika = 500
	MaksLimitDziennika     = 10000
	// LimitEksportuDziennika: najwięcej wierszy CSV w eksporcie.
	LimitEksportuDziennika = 100000
	// MaksDlugoscKtoWDzienniku: login z nieudanego logowania przycinany do tylu bajtów.
	MaksDlugoscKtoWDzienniku = 64
	// DomyslnyZakresDziennika: zakres statystyk/eksportu, gdy nie podano „od”.
	DomyslnyZakresDziennika = 7 * 24 * time.Hour
	// MinPort i MaksPort: dozwolony zakres portów w ustawieniach.
	MinPort  = 1
	MaksPort = 65535
)

// Kroki wykresu statystyk dziennika (parametr krok=godzina|dzien).
const (
	KrokStatystykGodzina = "godzina"
	KrokStatystykDzien   = "dzien"
)

// Transfery na żywo (SSE).
const (
	// OkresPingSSE: komentarz podtrzymujący połączenie.
	OkresPingSSE = 15 * time.Second
	// OkresSumySSE: zdarzenie „suma” (łączna prędkość i liczba aktywnych).
	OkresSumySSE = time.Second
)

// Pliki interfejsu (SPA).
const (
	// CacheZasobowZHashem: pliki z hashem w nazwie (Vite: nazwa-HASH.js) można trzymać rok.
	CacheZasobowZHashem = "public, max-age=31536000, immutable"
	// CacheBez: index.html i pliki bez hasha zawsze sprawdzane.
	CacheBez = "no-cache"
	// WzorNazwyZHashem: wyrażenie rozpoznające nazwę z hashem (co najmniej 8 znaków przed rozszerzeniem).
	WzorNazwyZHashem = `[.-][A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`
	// PlikIndeksu: strona startowa SPA.
	PlikIndeksu = "index.html"
)

// RozszerzeniaGzip: pliki interfejsu kompresowane przy wysyłce (gdy klient przyjmuje gzip).
var RozszerzeniaGzip = []string{".js", ".mjs", ".css", ".svg", ".html", ".json"}

// Adresy i nazwy.
const (
	// PrefiksAPI: wszystko pod nim to JSON (nieznany adres = 404 JSON, nie index.html).
	PrefiksAPI = "/api/"
	// PrefiksLinkuPublicznego: strona linku publicznego /s/<id>.
	PrefiksLinkuPublicznego = "/s/"
	// PrzyrostekZip: rozszerzenie pobieranego archiwum.
	PrzyrostekZip = ".zip"
	// PrefiksEksportuDziennika i FormatDatyEksportu: nazwa pliku CSV dziennik_<od>_<do>.csv.
	PrefiksEksportuDziennika = "dziennik_"
	FormatDatyEksportu       = "2006-01-02"
	// KrokGodzina i KrokDzien: przedziały wykresu statystyk.
	KrokGodzina = time.Hour
	KrokDzien   = 24 * time.Hour
	// PrefiksKtoLinku: pole „kto” w dzienniku i transferach dla pobrań przez link publiczny.
	PrefiksKtoLinku = "link:"
	// CzasPamieciAdresuPublicznego: jak długo pamiętać adres zewnętrzny (zapytanie do usługi w sieci).
	CzasPamieciAdresuPublicznego = 10 * time.Minute
)

// Fala 2.
const (
	// CacheMiniatur: miniatura zależy od wersji pliku, więc dzień w pamięci przeglądarki wystarcza.
	CacheMiniatur = "private, max-age=86400"
	// PlikServiceWorkera: zawsze no-cache (nowa wersja interfejsu musi go podmienić).
	PlikServiceWorkera = "sw.js"
	// MapaDni i MapaGodzin: wymiary mapy ciepła pobrań (pon..niedz × godzina).
	MapaDni    = 7
	MapaGodzin = 24
)

// TypyStatycznych: typy treści, których system może nie znać (rozszerzenie → typ).
var TypyStatycznych = map[string]string{
	".webmanifest": "application/manifest+json",
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
}

// Wgrywanie przez HTTP widoczne w transferach na żywo (jeden transfer na cały plik).
const (
	// NaglowekRozmiaruCalkowitego: rozmiar całego wgrywanego pliku (pasek postępu); brak = 0 (nieznany).
	NaglowekRozmiaruCalkowitego = "X-Rozmiar-Calkowity"
	// CzasBezczynnosciWgrania: brak kolejnego kawałka przez tyle czasu = transfer przerwany.
	CzasBezczynnosciWgrania = 60 * time.Second
	// OkresSprzataniaWgran: jak często tykacz serwera HTTP szuka porzuconych wgrań.
	OkresSprzataniaWgran = 10 * time.Second
)

// Ciasteczko linku publicznego z hasłem (pobieranie zwykłym odnośnikiem, bez blob:).
const (
	// NazwaCiasteczkaLinku: HttpOnly, SameSite=Strict, Path=/api/publiczny/<id>.
	NazwaCiasteczkaLinku = "skrytka_link"
	// WaznoscCiasteczkaLinku: ważność ciasteczka po poprawnym haśle (nie dłużej niż sam link).
	WaznoscCiasteczkaLinku = 30 * time.Minute
	// DlugoscKluczaLinkow: losowy klucz HMAC-SHA256 w pamięci procesu (restart = nowe hasło).
	DlugoscKluczaLinkow = 32
	// PrefiksAPILinku: adres API linku publicznego (ścieżka ciasteczka = prefiks + id).
	PrefiksAPILinku = "/api/publiczny/"
)
