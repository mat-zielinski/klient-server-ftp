# API HTTP (kontrakt serwer ↔ interfejs)

Wszystko pod `/api/`, JSON w UTF-8, czasy w RFC 3339. Uwierzytelnienie: ciasteczko `skrytka_sesja`
(`HttpOnly`, `SameSite=Strict`, `Secure` przy HTTPS). Każde żądanie zmieniające stan (POST/PUT/DELETE) wymaga nagłówka
`X-Skrytka: 1` (ochrona CSRF). Błąd: kod HTTP + `{"blad": "Komunikat dla człowieka."}`.

Mapowanie błędów rdzenia: `ErrNieZnaleziono` 404, `ErrBrakPrawa` / `ErrTylkoOdczyt` 403, `ErrPozaUdzialem` /
`ErrZlaNazwa` / `ErrZleDane`(dane wejściowe) 400, `ErrZleDane` przy logowaniu 401, `ErrKontoZablokowane` 429,
`ErrJuzIstnieje` 409, `ErrWygaslo` 410.

Parametry ścieżki pliku zawsze w zapytaniu: `?udzial=<nazwa>&sciezka=/a/b.txt` (zakodowane URL).

## Sesja

| metoda i adres | wejście | wyjście |
|---|---|---|
| `GET /api/stan` | — | `{"wersja","nazwa","kreator":bool,"zalogowany":bool,"uzytkownik":{login,admin,prawa,ftps}\|null,"motyw"}` |
| `POST /api/kreator` | `{"login","haslo","udzial":{"nazwa","sciezka","tylko_odczyt"},"tryb_sieci"}` | tylko gdy brak kont; zakłada admina i loguje |
| `POST /api/logowanie` | `{"login","haslo"}` | ustawia ciasteczko, zwraca jak `uzytkownik` |
| `POST /api/wylogowanie` | — | `{}` |

## Pliki

| metoda i adres | wejście | wyjście |
|---|---|---|
| `GET /api/udzialy` | — | `[{"nazwa","tylko_odczyt","prawo"}]` (tylko z prawem ≥ odczyt) |
| `GET /api/lista?udzial&sciezka&sort=nazwa\|rozmiar\|zmieniono&malejaco=0\|1` | — | `{"sciezka","wpisy":[Wpis]}` |
| `GET /api/szukaj?udzial&sciezka&q` | — | `[Wpis]` (najwyżej 500) |
| `GET /api/pobierz?udzial&sciezka[&podglad=1]` | — | plik: `Range`, `ETag`, `Last-Modified`; bez `podglad` = `Content-Disposition: attachment` (nazwa UTF-8, RFC 5987). Rejestrowany w transferach i dzienniku |
| `POST /api/zip?udzial` | `{"sciezki":["/a","/b.txt"]}` lub `GET /api/zip?udzial&sciezka=/a[&sciezka=/b.txt…]` (parametr powtarzany; zwykły link = strumień w przeglądarce) | strumień ZIP; jeden element = `<nazwa>.zip`, wiele albo korzeń = `<udział>.zip` |
| `GET /api/protokoly?udzial&sciezka` | — | menu „Pobierz przez…”, patrz niżej |
| `POST /api/folder?udzial&sciezka` | — | `{}` |
| `POST /api/zmien?udzial` | `{"z","na"}`: PEŁNE ścieżki w udziale (zmiana nazwy albo przeniesienie) | `{}` |
| `DELETE /api/plik?udzial&sciezka` | — | do kosza, `{}` |
| `PUT /api/wgraj?udzial&sciezka&przesuniecie=N&koniec=0\|1&nadpisz=0\|1` | treść = kawałek (do 8 MB) | `{"przeslano":N}`; `koniec=1` zatwierdza; przy `przesuniecie=0` serwer tworzy brakujące foldery nadrzędne (każdy przez bramkę `OpWgranie`, istniejący = OK); istniejący plik przy `nadpisz=0` = 409; złe przesunięcie = 409 |
| `GET /api/wgraj?udzial&sciezka` | — | `{"przeslano":N}` (do wznowienia) |

### Menu „Pobierz przez…” (prawy przycisk na pliku)

`GET /api/protokoly` zwraca opcje dostępne dla tego pliku i tego użytkownika:

```json
{"opcje": [
  {"id": "https",  "nazwa": "Przeglądarka (HTTPS)", "opis": "Pobranie z wznawianiem", "adres": "https://192.168.0.10:8443/api/pobierz?...", "dostepne": true},
  {"id": "ftps",   "nazwa": "Klient FTP (FTPS)", "opis": "FileZilla, Total Commander, WinSCP", "adres": "ftps://jan@192.168.0.10:2121/Filmy/a.mkv", "dostepne": true, "port": 2121},
  {"id": "link",   "nazwa": "Kopiuj link bezpośredni", "opis": "Wymaga zalogowania", "adres": "https://…", "dostepne": true},
  {"id": "zip",    "nazwa": "Jako ZIP", "opis": "Dla folderów i wielu plików", "adres": "…", "dostepne": true},
  {"id": "publiczny", "nazwa": "Link publiczny…", "opis": "Bez logowania, z datą ważności", "dostepne": true}
], "adresy": [Adres], "odcisk_certyfikatu": "AB:CD:…"}
```

`dostepne:false` + `"powod"` gdy np. FTPS wyłączony albo konto bez prawa FTPS. Adres hosta = host, którym klient
właśnie wszedł (nagłówek `Host`), żeby przez WireGuard i z LAN link był poprawny.

## Transfery na żywo

| metoda i adres | wyjście |
|---|---|
| `GET /api/transfery` | `[Stan]` (admin: wszystkie, inni: własne) |
| `GET /api/transfery/strumien` | SSE: `event: start\|postep\|koniec`, `data: Zdarzenie` (JSON); co 1 s także `event: suma`, `data: {"predkosc_bps":N,"aktywne":N}` | Admin dostaje wszystkie transfery (UI filtruje), inni tylko własne; komentarz `: ping` co 15 s.
| `DELETE /api/transfery?id` | admin: przerwanie transferu |

Klient widzi swój panel „Moje pobrania” (prędkość, postęp, pozostały czas, mini wykres), admin na Pulpicie wszystkie
transfery i łączny wykres prędkości z ostatnich 5 minut.

## Dziennik (admin; zwykły użytkownik: tylko własne pobrania)

| metoda i adres | wyjście |
|---|---|
| `GET /api/dziennik?od&do&rodzaj&kto&szukaj&limit` | `[Wpis]` |
| `GET /api/dziennik/statystyki?od&do&krok=godzina\|dzien` | `Statystyki` (wykres bajtów i liczby pobrań, top pliki, top klienci, podział na protokoły) | Tylko admin (nie-admin 403).
| `GET /api/dziennik/eksport?od&do` | CSV (`text/csv`, `dziennik_<od>_<do>.csv`; nie-admin: własne pobrania) |

## Udostępnienia

| metoda i adres | wejście | wyjście |
|---|---|---|
| `GET /api/udostepnienia` | — | `[Link]` (admin wszystkie, inni swoje) |
| `POST /api/udostepnienia` | `{"udzial","sciezka","wygasa","haslo","limit_pobran"}`; `wygasa` "" = 7 dni (maks. 365) | `Link` + `"adres"` |
| `DELETE /api/udostepnienia?id` | — | `{}` |
| `GET /s/<id>` | — | strona publiczna (ta sama aplikacja, widok „link”) |
| `GET /api/publiczny/<id>[?haslo=…]` (hasło w nagłówku `X-Haslo`) | — | info / lista folderu |
| `GET /api/publiczny/<id>/pobierz[?sciezka]` | — | plik |

## Administracja (admin)

| metoda i adres | wejście | wyjście |
|---|---|---|
| `GET/POST /api/konta`, `PUT /api/konta?login`, `DELETE /api/konta?login`, `POST /api/konta/haslo?login` | `Uzytkownik` + `haslo` | lista / `{}` |
| `GET /api/ustawienia` / `PUT /api/ustawienia` | `Konfiguracja` | zapis; zmiana portów/trybu = komunikat „Wymaga restartu serwera” |
| `GET /api/siec` | — | `{"adresy":[Adres],"publiczny","tryb","porty":{http,https,ftps},"porty_otwarte":{…},"odcisk_certyfikatu"}` |
| `POST /api/siec/zapora` | — | dodaje regułę Zapory Windows |
| `GET /api/pulpit` | — | `{"aktywne":N,"predkosc_bps":N,"dzis":{"bajty","pobrania"},"udzialy":[{"nazwa","wolne","razem"}],"czas_pracy_s"}` |

## Fala 2 (standard profesjonalny)

| metoda i adres | wyjście |
|---|---|
| `GET /api/zdrowie` | `{"ok":true,"wersja"}` bez logowania |
| `GET /api/monitor` (admin) | `{"teraz": Probka, "historia_1s": [Probka] (10 min), "historia_1m": [Probka] (24 h), "system": {"os","arch","cpu_rdzenie","ram_razem","host","czas_pracy_s"}}` |
| `GET /api/monitor/strumien` (admin) | SSE `event: probka`, `data: Probka` co 1 s |
| `GET /api/indeks/szukaj?udzial&sciezka&q&limit` | `[Wpis]` z indeksu (szybkie), gdy udział niezindeksowany: przejście magazynu; prawo odczytu; `limit` ≤ 500 |
| `GET /api/indeks/statystyki` (admin) | `{"udzialy":[{"nazwa","pliki","foldery","bajty","wg_rodzaju":{"obraz":{"pliki","bajty"},…},"zindeksowano"}]}` |
| `GET /api/dziennik/mapa?od&do` | `{"komorki":[[bajty × 24] × 7]}` (pon..niedz × godzina, tylko pobrania; nie-admin = własne; czas lokalny serwera, poniedziałek = 0; najwyżej 50 000 wpisów) |
| `GET /api/miniatura?udzial&sciezka&rozmiar=s\|m` | JPEG, `Cache-Control: private, max-age=86400`; nie obraz / format bez miniatury / za duży / uszkodzony = 404, zły rozmiar = 400; bez transferu i dziennika |
| `GET /api/suma?udzial&sciezka` | `{"sha256"}` (liczone strumieniowo, pamięć podręczna wg rozmiaru i daty); prawo pobierania; folder = 404 |
| `GET /api/konta/urzadzenia` / `DELETE /api/konta/urzadzenia?id` | sesje bieżącego konta `{"id","ip","przegladarka","utworzono","ostatnio","biezaca"}`; usunięcie cudzej albo nieznanej = 404 |
| `POST /api/konta/2fa/start` → `{"sekret","adres_otpauth"}`; `POST /api/konta/2fa/wlacz {"kod"}`; `POST /api/konta/2fa/wylacz {"haslo"}`; logowanie: `{"login","haslo","kod"?}` → 401 `{"blad","wymaga_kodu":true}`, zły kod = 401 bez `wymaga_kodu`; `wlacz` zły kod = 400; `wylacz` złe hasło = 403; ponowny `start` / `wylacz` bez 2FA = 409 |

Usługa wyłączona (pole `Monitor`/`Indeks`/`Miniatury`/`Sumy` w `Zaleznosci` = nil) = 503 `{"blad"}`. `GET /api/stan`: `wersja` = wersja wydania (`WersjaBudowy`), dodatkowe pole `ma_2fa`. `GET /api/pulpit`: `udzialy[].wolne/razem` z dysku pod folderem udziału.

`Probka`: `{"czas","cpu_proc","ram_uzyte","ram_razem","dysk_odczyt_bps","dysk_zapis_bps","siec":[{"interfejs","rodzaj","wysylanie_bps","odbior_bps"}],"wysylanie_bps","odbior_bps","polaczenia","transfery","gorutyny","pamiec_procesu"}`.
