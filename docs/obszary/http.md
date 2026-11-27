# Obszar „http” (wewn/http)

## Co robi

Widok HTTP: tłumaczy żądania z `docs/api.md` na wywołania usług. `Nowy(z Zaleznosci)` buduje `net/http.ServeMux`
(wzorce `"GET /api/lista"`, bez zewnętrznych bibliotek) i owija go pośrednikami (od zewnątrz): odzyskiwanie po panic
(wpis `serwer` w dzienniku, 500), nagłówki bezpieczeństwa (CSP `default-src 'self'` z `blob:`/`data:` dla obrazów,
`frame-ancestors 'self'`, `nosniff`, `Referrer-Policy: no-referrer`), sesja z ciasteczka `skrytka_sesja` do kontekstu,
CSRF (`X-Skrytka: 1` dla POST/PUT/DELETE/PATCH).

Zasady:
1. Każda operacja na plikach: `rdzen.OczyscSciezke` i `rdzen.CzyWolno` PRZED magazynem (`dostepDoUdzialu`, `celZadania`).
   Nieznany udział dla kogoś bez prawa = 403 (nie zdradzamy listy udziałów).
2. Odpowiedzi nigdy nie zawierają `rdzen.Uzytkownik` wprost (hash), tylko `widokUzytkownika`.
3. IP klienta tylko z `r.RemoteAddr` (X-Forwarded-For ignorowany), protokół z `r.TLS`.
4. Pobieranie: `http.ServeContent` (Range, ETag, If-None-Match, If-Modified-Since); transfer rejestrowany dopiero przy
   odpowiedzi 200/206 na GET (304/416/HEAD nie są pobraniem); bajty przez `Transfer.PiszDo`; wpis dziennika pobrania
   robi WYŁĄCZNIE callback `naKoniec` rejestru w aplikacji (jedno źródło dla HTTP i FTPS); HTTP tylko kończy transfer
   z wynikiem ok/przerwane/blad. Wgranie: jeden transfer na cały plik (kawałki przez `Transfer.CzytajZ`), wpis dziennika „wgranie” też WYŁĄCZNIE z `naKoniec` (HTTP nie pisze własnego). `Content-Disposition` z nazwą ASCII i `filename*=UTF-8''…` (RFC 5987).
5. Pliki użytkownika: bez `podglad` zawsze `application/octet-stream`; przy podglądzie JS jako `text/plain`, HTML jako
   `text/plain` + piaskownica CSP, SVG/XML z piaskownicą CSP (ochrona przed XSS z wgranych plików).
6. Linki publiczne: ścieżka podrzędna po `OczyscSciezke` i sprawdzeniu, że leży w ścieżce linku; autor linku musi nadal
   mieć prawo pobierania; ścieżki w odpowiedzi względne do linku; `ZaliczPobranie` przed wysłaniem (wznowienie Range
   od środka nie liczy się drugi raz); folder = ZIP (także `/api/publiczny/<id>/zip`, podfolder przez `sciezka`). Link z
   hasłem: po poprawnym haśle ciasteczko `skrytka_link` (HMAC-SHA256 z id i czasu wygaśnięcia, klucz losowy w pamięci,
   30 min, `Path=/api/publiczny/<id>`); `pobierz`/`zip`/info przyjmują hasło ALBO ważne ciasteczko; brak hasła nie liczy
   się jako zła próba.
7. SSE `/api/transfery/strumien`: subskrypcja przed nagłówkami, `no-cache`, `X-Accel-Buffering: no`, zdarzenie
   `suma` co 1 s, `: ping` co 15 s; admin wszystkie transfery, inni własne.
8. SPA: nieznana ścieżka = `index.html` (`no-cache`), pliki z hashem w nazwie (`nazwa-HASH.js`) cache rok `immutable`,
   gzip js/css/svg/html/json z pamięci, ETag z SHA-256. Nieznany adres `/api/…` = 404 JSON.

## Pliki

| plik | zawartość |
|---|---|
| `serwer.go` | `Zaleznosci`, `Nowy`, interfejsy usług (atrapy w testach), adapter rejestru transferów (`Przerwij`) |
| `posrednie.go` | pośrednicy, sesja w kontekście, `zalogowany` / `tylkoAdmin`, `czytajJSON` z limitem, IP, protokół |
| `bledy.go` | mapowanie błędów rdzenia i usług na kody (docs/api.md), komunikaty po polsku, 500 do dziennika |
| `sesja.go` | `/api/stan`, `/api/zdrowie`, `/api/kreator` (tylko gdy `CzyPusto`, pod blokadą), logowanie, wylogowanie |
| `pliki.go` | udziały, lista, szukaj, folder, zmień (pełne ścieżki), usuń do kosza, bramka `dostepDoUdzialu` |
| `pobieranie.go` | pobierz, ZIP (GET z powtarzanym `sciezka`, POST), `pisarzTransferu`, Content-Disposition |
| `wgrywanie.go` | wgrywanie w kawałkach, wznowienie (`RozmiarKawalkow`), foldery nadrzędne, nagłówek `X-Rozmiar-Calkowity` |
| `wgrania.go` | aktywne wgrania: jeden transfer na plik, sprzątanie porzuconych (60 s; przy żądaniu i tykaczem co 10 s, gorutyna tylko gdy są aktywne) |
| `ciasteczko_linku.go` | klucz HMAC, ciasteczko linku publicznego (wydanie, sprawdzenie), `otworzLink` (hasło albo ciasteczko) |
| `protokoly.go` | menu „Pobierz przez…” (host z nagłówka `Host`, FTPS z konfiguracji i flagi konta, odcisk) |
| `transfery.go` | lista, SSE, przerwanie (admin) |
| `dziennik.go` | wpisy (nie-admin: wymuszone `Kto`=login i `Rodzaj`=pobranie), statystyki (admin), eksport CSV |
| `udostepnienia.go` | linki: lista, tworzenie, usuwanie (nie-admin tylko własne), strona `/s/<id>`, API publiczne (info, pobierz, zip) |
| `admin.go` | konta, ustawienia (walidacja, „Wymaga restartu serwera”), sieć (adres publiczny pamiętany 10 min), zapora, pulpit |
| `statyczne.go` | pliki interfejsu SPA (`sw.js` no-cache, `.webmanifest` = `application/manifest+json`) |
| `dodatki.go` | interfejsy usług fali 2 (monitor, indeks, miniatury, sumy), nil = 503, `odswiezIndeks` |
| `monitor.go` | trasy fali 2, `/api/monitor`, SSE `/api/monitor/strumien` (`event: probka`) |
| `indeks.go` | `/api/indeks/szukaj` (gotowy indeks albo magazyn), `/api/indeks/statystyki`, `/api/miniatura` (ServeFile, bez transferu), `/api/suma` |
| `konto.go` | `/api/konta/urzadzenia` (bieżąca przez `konta.IDSesji`) |

Stałe: `wewn/konfiguracja/http.go` (ciasteczko, CSRF, CSP, limity JSON/kawałka/szukania/dziennika, okresy SSE, cache,
wzór nazwy z hashem, rozszerzenia gzip, prefiksy adresów, kroki statystyk).

## Testy (`go test ./wewn/http/...`, prawdziwe usługi w `t.TempDir()`, atrapy dziennika, sieci i tykacza)

CSRF (z dowodem braku zmiany na dysku), 401 bez sesji (16 adresów + zły token), 403 bez prawa (16 przypadków, w tym
udział tylko do odczytu i nieznany udział), 400 ucieczka ścieżką (`..`, `\`, dysk, UNC, strumień NTFS, zmiana nazwy
poza udział, usunięcie korzenia), Range 206 / 416 / 304 + transfery zakończone przez callback (HTTP nie pisze
pobrań do dziennika), nazwa UTF-8 (RFC 5987), typy treści (XSS),
ZIP wielu, wgrywanie w kawałkach (wznowienie, 409, foldery nadrzędne), protokoły (Host, FTPS, zły Host), SSE (start,
suma, koniec, ping, filtr własnych), przerwanie transferu, dziennik nie-admina, linki publiczne (hasło, ucieczka,
limit, ukrycie ścieżki, autor bez prawa, usuwanie cudzego), konta, ustawienia, SPA (index, cache, gzip, ETag),
mapowanie błędów, panic, limit treści JSON, logowanie (IP bez X-Forwarded-For, blokada 429), kreator jednokrotny.
Testy mutacyjne (zdjęcie blokady = test czerwony): CSRF, bramka w `dostepDoUdzialu`, filtr dziennika, autor linku,
kreator, typ JS, usuwanie cudzego linku.

Fala 2: logowanie przez `Konta.Zaloguj` (login i hasło), sesje przez
`UtworzZOpisem(login, IP, User-Agent)`; `/api/stan` z `WersjaBudowy`; `/api/dziennik/mapa` liczona z
`Czytaj` (pobrania, nie-admin własne, czas lokalny, pon=0); po każdej zmianie plików `Indeks.Odswiez(udzial)`;
`/api/pulpit` z `monitor.ZajetoscDysku` dla folderów z konfiguracji. Testy fali 2:
urządzenia, monitor (503, 403, SSE), indeks (magazyn, prawdziwy indeks, atrapa), odświeżanie indeksu, miniatury
(PNG, 404/400/403, brak transferu), sumy, mapa, zajętość dysku, wersja budowy, sw.js/webmanifest, nil = wyłączone.
Mutacje fali 2: filtr mapy, nil usług, odświeżanie indeksu, nagłówek cache miniatur.

## Czego nie zrobiono

1. Mapa dziennika czyta najwyżej `DziennikLimitMaks` (50 000) wpisów z zakresu.
2. Interfejs jeszcze nie wysyła `X-Rozmiar-Calkowity` (bez niego wgranie w transferach ma rozmiar 0 = nieznany).
   Aktywne wgrania są tylko w pamięci procesu (restart serwera je gubi; wznowienie zaczyna nowy transfer).
3. Kreator nie sprawdza, czy folder udziału istnieje (dysk = magazyn/aplikacja przy przeładowaniu konfiguracji).

## Poprawki: wgrania w transferach, ciasteczko linku, ZIP linku

- `PUT /api/wgraj`: jeden transfer `wgranie` na cały plik (klucz login+udział+ścieżka). Start: `przesuniecie=0`
  (poprzedni tego pliku = `przerwane`) albo brak aktywnego; rozmiar = `X-Rozmiar-Calkowity` minus przesunięcie.
  Kawałki liczone `CzytajZ`; `koniec=1` = `ok`; błąd kawałka/zatwierdzenia = `blad`; przerwanie admina albo
  rozłączenie = `przerwane` (`ErrPrzerwany` = 409); bez kawałka `CzasBezczynnosciWgrania` (60 s) = `przerwane`
  (sprawdzane przy każdym `PUT /api/wgraj` i tykaczem `OkresSprzataniaWgran`). Wpis dziennika tylko z `naKoniec`.
- Ciasteczko linku: `NazwaCiasteczkaLinku`, `WaznoscCiasteczkaLinku` (30 min), `DlugoscKluczaLinkow` (32 B),
  `PrefiksAPILinku`. Wartość `<unix>.<base64url(HMAC-SHA256(klucz, id|unix))>`, porównanie `hmac.Equal`, czas
  dalej niż 30 min w przód odrzucany. Ciasteczko nie omija wygaśnięcia ani limitu pobrań linku.
- `GET /api/publiczny/<id>/zip[?sciezka]`: tylko folder (plik = 400), `ZaliczPobranie` raz, przez `wyslijZip`.
- Testy: `wgrania_test.go` (tabela: dwa kawałki, bez nagłówka, porzucone przy żądaniu i tykaczem, start od zera,
  wznowienie, zły kawałek, zły nagłówek; przerwanie admina; brak własnego wpisu „wgranie”), `ciasteczko_linku_test.go`
  (atrybuty, dobre/złe/wygasłe/podrobione/innego linku, limit linku, podfolder poza linkiem = 400, ZIP). Mutacje
  (test czerwony): podpis HMAC, id w podpisie, wygasanie ciasteczka, limit linku przy ciasteczku, próg 60 s, sprzątanie.
