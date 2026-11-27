# Architektura Skrytki

Skrytka = serwer plików na Windows: jeden `skrytka.exe` (Go 1.27) z wkompilowanym interfejsem (Svelte 5 + Vite,
`go:embed`), HTTPS i FTPS na wspólnym magazynie, kontach, bramce uprawnień, rejestrze transferów i dzienniku. Start:
`cmd/skrytka/main.go` → `aplikacja.Uruchom`. Wersja: `Wersja` w `wewn/konfiguracja/wersja.go`. Plan i sprinty:
[PLAN.md](../PLAN.md); kontrakt HTTP: [api.md](api.md); interfejs: [interfejs.md](interfejs.md). Szczegóły każdego
obszaru: `docs/obszary/<obszar>.md`.

## Warstwy

```
konfiguracja ← rdzen ← magazyn ← uslugi/* ← http, ftp, siec ← aplikacja ← cmd/skrytka
```

Import tylko „w dół” albo w obrębie własnej warstwy. Pilnuje strażnik architektury ([testy.md](testy.md)).

| warstwa | katalog | co zawiera | czego NIE wolno |
|---|---|---|---|
| konfiguracja | `wewn/konfiguracja/` | WSZYSTKIE stałe (każdy obszar we własnym pliku: `konta.go`, `siec.go`, `dziennik.go`, `transfery.go`, `udostepnienia.go`, `rdzen.go`), ścieżki (`sciezki.go`), wersja (`wersja.go`), typ `Konfiguracja` i jej zapis atomowy (`konfiguracja.go`) | importować inne pakiety projektu |
| rdzen | `wewn/rdzen/` | czysta logika: piaskownica ścieżek (`OczyscSciezke`, `SciezkaNaDysku`), nazwy (`SprawdzNazwe`), rodzaje plików, sortowanie naturalne, dopasowanie bez ogonków, bramka `CzyWolno`, błędy wspólne (`ErrNieZnaleziono`, `ErrBrakPrawa` …), typy `Wpis`, `Uzytkownik`, `Operacja` | dysk, sieć, `time.Now` (czas jako argument) |
| magazyn | `wewn/magazyn/` | JEDYNY dostęp do dysku: lista, info, otwarcie, zapis atomowy, wgrywanie w kawałkach, folder, zmiana nazwy, kosz `.skrytka_kosz`, szukanie, ZIP strumieniem | HTTP, FTP, konta |
| uslugi | `wewn/uslugi/konta`, `dziennik`, `transfery`, `udostepnienia` | konta (argon2id, blokada po złych hasłach, sesje), dziennik JSONL i statystyki, transfery na żywo, linki publiczne | HTTP, FTP |
| widoki | `wewn/http/`, `wewn/ftp/`, `wewn/siec/` | API HTTP i pliki interfejsu; serwer FTPS (`ftpserverlib`); adresy, tryby sieci, TLS, test portu, Zapora Windows | logika uprawnień i plików (tylko wywołania usług) |
| aplikacja | `wewn/aplikacja/` | składa całość, kreator pierwszego uruchomienia, usługa Windows, zamykanie | — |
| start | `cmd/skrytka/main.go` | `os.Exit(aplikacja.Uruchom(os.Args[1:]))` | cokolwiek więcej |
| interfejs | `web/` | Svelte 5, wygląd z tokenów `web/src/styl/tokeny.css`, ikony `web/src/ikony/ikony.ts`; osadzenie: `web/osadz.go` | rozmawiać z serwerem inaczej niż przez `/api/` |

## Pakiety i kontrakty

Każdy pakiet `wewn/` ma (albo miał w sprincie 0) plik `kontrakt.go` z wiążącymi sygnaturami; właściciel pakietu
zastępuje zaślepki implementacją, sygnatury zostają.

| pakiet | najważniejsze sygnatury |
|---|---|
| `rdzen` | `OczyscSciezke`, `SciezkaNaDysku`, `SprawdzNazwe`, `RodzajPliku`, `Sortuj`, `Pasuje`, `CzyWolno(u, op, udzial, tylkoOdczyt)` |
| `magazyn` | `Nowy(udzialy)`, `Lista`, `Info`, `Otworz`, `Zapisz`, `DopiszKawalek` + `Zatwierdz`, `NowyFolder`, `ZmienNazwe`, `DoKosza`, `Szukaj`, `ZipDo`, `RozmiarFolderu` |
| `uslugi/konta` | `Otworz(plik)`, `Zaloguj(login, haslo, ip)`, `Dodaj`, `Zmien`, `UstawHaslo`, `Usun`; `Sesje`: `Utworz`, `Sprawdz`, `Zakoncz`, `ZakonczWszystkie` |
| `uslugi/dziennik` | typy `Wpis`, `Filtr`, `Statystyki`; rodzaje (`pobranie`, `logowanie_blad` …) i wyniki (`ok`, `przerwane`, `blad`) |
| `uslugi/transfery` | `Nowy(naKoniec)`, `Rozpocznij(Stan) *Transfer`, `CzytajZ` / `PiszDo` / `Dodaj`, `Zakoncz(wynik)`, `Aktywne(kto)`, `Sluchaj(kto)`, `PredkoscLaczna` |
| `uslugi/udostepnienia` | `Otworz`, `Utworz(link, haslo)`, `Lista(autor)`, `Usun`, `Sprawdz(id, haslo)`, `ZaliczPobranie` |
| `siec` | `AdresyLokalne`, `AdresyNasluchu(k)`, `KonfiguracjaTLS(k)`, `OdciskCertyfikatu`, `AdresPubliczny`, `SprawdzPort`, `RegulaZapory` |
| `http` | `Nowy(Zaleznosci) http.Handler` |
| `ftp` | `Uruchom(ctx, Zaleznosci) error` |
| `aplikacja` | `Uruchom(argumenty) int` |

Pliki stanu (ścieżki tylko z `konfiguracja/sciezki.go`, względem folderu programu albo `SKRYTKA_KATALOG`):
`dane/konfiguracja.json`, `dane/konta.json`, `dane/udostepnienia.json`, `dane/certyfikaty/`,
`logi/dziennik_RRRR-MM.jsonl`. Zapis zawsze atomowy (`konfiguracja.ZapiszAtomowo`: plik tymczasowy + zamiana),
uprawnienia plików 0600, katalogów 0700.

## Przepływ pobrania (HTTPS)

`GET /api/pobierz?udzial=Filmy&sciezka=/a.mkv`:

1. `http` czyta ciasteczko `skrytka_sesja` → `Sesje.Sprawdz(token)` → `rdzen.Uzytkownik` (wyłączony albo usunięty =
   brak sesji, 401).
2. `magazyn.Udzial(udzial)` → opis udziału (nazwa, folder, `TylkoOdczyt`).
3. `rdzen.CzyWolno(u, OpPobranie, udzial, tylkoOdczyt)` → brak prawa = 403.
4. `magazyn.Otworz(udzial, sciezka)`: ścieżka przez `rdzen.OczyscSciezke` i `rdzen.SciezkaNaDysku` (wyjście poza
   udział = 400), folder = 404.
5. `transfery.Rozpocznij(Stan{Kto, IP, Protokol: "https", Kierunek: "pobranie", Udzial, Sciezka, Rozmiar})` →
   uchwyt `Transfer`.
6. Strumień pliku przez `Transfer.PiszDo(w)` (liczenie bajtów) z obsługą `Range`, `ETag`, `Last-Modified`,
   `Content-Disposition: attachment` (nazwa UTF-8, RFC 5987). Plik nie jest buforowany w pamięci.
7. Koniec: `Transfer.Zakoncz(ok | przerwane | blad)` → funkcja `naKoniec` rejestru zapisuje `dziennik.Wpis`
   (`RodzajPobranie`, bajty, czas, wynik).

FTPS (`wewn/ftp`) idzie tą samą drogą: logowanie `Konta.Zaloguj` (tylko konta z flagą FTPS), korzeń = udziały
użytkownika jako foldery, każda operacja przez `rdzen.CzyWolno`, transfer z `Protokol: "ftps"` i `Transfer.Dodaj(n)`.
ZIP (`/api/zip`) to transfer z `Rozmiar: 0` (nieznany), strumień `magazyn.ZipDo`. Link publiczny (`/s/<id>`) przechodzi
przez `udostepnienia.Sprawdz` (wygasły albo limit = 410, złe hasło = 401) i jest w dzienniku jako `Kto: "link:<id>"`.

Menu „Pobierz przez…” (`GET /api/protokoly`) buduje adresy z hosta, którym klient właśnie wszedł (nagłówek `Host`),
więc link jest poprawny i przez WireGuard, i z LAN.

## Transfery na żywo

1. `transfery.Rejestr` trzyma aktywne transfery (bezpieczny dla wielu gorutyn). Jedna gorutyna próbkuje co
   `TransferyOkresProbkowania` (1 s): prędkość = średnia z `TransferyOknoPredkosciS` sekund, historia 60 próbek do
   mini wykresu, pozostały czas (`-1` = nieznany).
2. `Sluchaj(kto)` daje kanał zdarzeń `start`, `postep` (najwyżej co 500 ms na transfer), `koniec` (z wynikiem);
   `kto == ""` = wszystkie (admin), inaczej tylko własne. Kanał ma bufor `TransferyBuforSluchacza`; wolny słuchacz
   nie blokuje transferów.
3. `GET /api/transfery/strumien` (SSE) przekłada kanał na `event: start|postep|koniec` i co 1 s `event: suma`
   (łączna prędkość, liczba aktywnych).
4. Interfejs: `web/src/api/transfery.ts` (`sluchajTransferow`, EventSource z samoczynnym wznawianiem) → panel
   „Moje pobrania” (strona Pliki) i Pulpit administratora (wykres łącznej prędkości z 5 minut, tabela z przyciskiem
   „Przerwij” = `DELETE /api/transfery?id`).

## Bezpieczeństwo

| zagrożenie | blokada | gdzie |
|---|---|---|
| wyjście poza udział (`..`, `C:`, UNC, strumienie NTFS `:`) | `OczyscSciezke` + `SciezkaNaDysku`, wynik zawsze w korzeniu udziału | `rdzen`, jedyne wejście `magazyn` |
| złe nazwy plików (CON, NUL, znaki zakazane, kropka na końcu, kosz) | `SprawdzNazwe` | `rdzen` |
| brak prawa, zapis do udziału tylko do odczytu, konto wyłączone | `CzyWolno` (tabela operacja → prawo) | `rdzen`, wołane przez HTTP, FTPS i linki |
| zgadywanie haseł | `MaksBlednychLogowan` w `OknoBlednychLogowan` na IP i na login → `ErrKontoZablokowane` (429); porównanie w stałym czasie także dla nieistniejącego loginu | `uslugi/konta` |
| kradzież hasła z pliku | argon2id z solą, parametry w `konfiguracja/konta.go`, górne granice przy odczycie | `uslugi/konta` |
| przejęcie sesji | token losowy 32 B, ciasteczko `HttpOnly`, `SameSite=Strict`, `Secure` przy HTTPS; zmiana hasła kończy sesje | `uslugi/konta`, `http` |
| CSRF | nagłówek `X-Skrytka: 1` przy każdym POST, PUT, DELETE | `http` |
| podsłuch | tryb publiczny: tylko HTTPS i FTPS (AUTH TLS wymagane), bez zwykłego FTP; certyfikat własny (ECDSA P-256, odcisk SHA-256 do sprawdzenia przez klienta) albo Let's Encrypt | `siec`, `ftp` |
| linki publiczne | ID 16 B losowe, wygasanie (najwyżej `MaksWaznoscLinku`), hasło argon2id, limit pobrań, blokada po złych hasłach | `uslugi/udostepnienia` |
| wyciek w dzienniku | dziennik nigdy nie zapisuje hasła, tokenu ani ciasteczka | `uslugi/dziennik` |
| widoczność z internetu w trybie VPN | nasłuch tylko na adresach LAN, WireGuard i pętli (`AdresyNasluchu`) | `siec` |
| dane w publicznym repo | `.gitignore` dla `dane/`, `logi/`, certyfikatów, `.exe` | repo |

Każda blokada z tej tabeli ma test, który pada bez niej, i wpis w liście mutacji blokad ([testy.md](testy.md)).

## Jak dodać …

- **adres API:** usługa + test → `wewn/http` + test `httptest` → [api.md](api.md) → `web/src/api/typy.ts`.
- **operację na plikach:** `Operacja` i reguła w `rdzen` (test tabelaryczny, mutacja) → metoda `magazyn` → wywołanie
  w `http` i `ftp` → wpis dziennika.
- **stałą:** `wewn/konfiguracja/<obszar>.go` z komentarzem.
- **stronę interfejsu:** [interfejs.md](interfejs.md).
