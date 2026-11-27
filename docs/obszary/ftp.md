# Obszar „ftp”: serwer FTPS

## Co robi

`ftp.Uruchom(ctx, Zaleznosci)` uruchamia serwer FTPS (jawny TLS, `AUTH TLS`) na bibliotece
`github.com/fclairamb/ftpserverlib`, na tym samym magazynie, kontach, bramce, transferach i dzienniku co WWW.

1. **Szyfrowanie wymagane**: `MandatoryEncryption`, czyli `USER`/`PASS` bez TLS = 421 i rozłączenie, kanał danych
   bez `PROT P` = odmowa transferu. TLS minimum 1.2 (kopia `Zaleznosci.TLS`). Brak TLS = `Uruchom` zwraca błąd.
2. **Tylko tryb pasywny** (`PORT`/`EPRT` wyłączone, ochrona przed FTP bounce), porty z `Konfiguracja.PortyPasywne`
   (zły zakres albo 0 = port losowy systemu), połączenie danych tylko z tego samego IP co sterowanie.
3. **Logowanie**: `konta.Zaloguj(login, haslo, ip)` (blokada po złych hasłach jest w kontach), dopuszczone tylko konta
   `FTPS=true` i niewyłączone. Klient dostaje zawsze „błędne dane logowania” (albo „konto chwilowo zablokowane”);
   powód trafia do dziennika (`logowanie_blad`, login, IP, protokół `ftps`, bez hasła). Udane = `logowanie`.
4. **Drzewo**: `/` = wirtualny folder z udziałami, do których użytkownik ma prawo odczytu (bramka `rdzen.CzyWolno`);
   `/<udzial>/<sciezka>` = magazyn. Udział bez prawa odczytu jest niewidoczny (550 „nie znaleziono”). Ścieżka przechodzi
   przez `path.Clean` i `rdzen.OczyscSciezke`; nazwy nowych plików/folderów przez `rdzen.SprawdzNazwe`.
5. **Każda operacja przez bramkę**, z aktualnym stanem konta pobieranym przy każdym poleceniu (wyłączenie konta,
   zdjęcie flagi FTPS albo odebranie prawa działa od razu w otwartej sesji):
   lista/Stat = `OpOdczyt`, `RETR` = `OpPobranie`, `STOR`/`MKD` = `OpWgranie`, `RNFR`/`RNTO` = `OpZmiana` (tylko w obrębie
   udziału), `DELE`/`RMD` = `OpUsuniecie` (do kosza aplikacji `magazyn.DoKosza`; `DELE` tylko pliki, `RMD` tylko puste
   foldery). Zapis w `/` i na samym udziale (usunięcie, zmiana nazwy) zabroniony.
6. **Transfery**: pobranie i wgranie rejestrowane w `transfery.Rejestr` (protokół `ftps`, login, IP, udział, ścieżka,
   rozmiar), liczone przez `CzytajZ`/`PiszDo`, więc przerwanie przez administratora działa (`ErrPrzerwany`). Wynik:
   `ok`, `przerwane` (ABOR / zerwane połączenie danych) albo `blad`. **Wpis dziennika na koniec transferu robi rejestr
   przez `naKoniec`** (zgodnie z kontraktem transferów), więc `ftp` go nie dubluje. Zmiany (folder, nazwa, kosz) `ftp`
   zapisuje sam jako `zmiana`.
7. **REST** (wznawianie) przy pobieraniu: `Seek` na pliku. Wgrywanie przez `magazyn.Zapisz` strumieniem (plik
   tymczasowy, zamiana na końcu); przerwane wgrywanie nie zostawia pliku. `REST`+`STOR` i `APPE` są odrzucane.
8. **Limit połączeń na IP** (`FTPMaksPolaczenNaIP`), rozłączenie po bezczynności (`FTPBezczynnoscS`).
9. **Zamknięcie**: anulowanie `ctx` zamyka nasłuch, nasłuchy pasywne i wszystkie połączenia; `Uruchom` zwraca `nil`.

## Pliki

| plik | zawartość |
|---|---|
| `wewn/ftp/serwer.go` | `Zaleznosci`, `Uruchom`, interfejsy zależności, `sterownik` (MainDriver: ustawienia, TLS, limit IP, logowanie) |
| `wewn/ftp/klient.go` | `klient` (ClientDriver / afero.Fs): rozkład ścieżki, bramka, Stat, ReadDir, MKD, DELE, RMD, rename |
| `wewn/ftp/pliki.go` | `GetHandle`, pobieranie z REST, wgrywanie przez rurę, `os.FileInfo` |
| `wewn/konfiguracja/ftp.go` | stałe obszaru |

## Stałe (`wewn/konfiguracja/ftp.go`)

`FTPMaksPolaczenNaIP` (10), `FTPBezczynnoscS` (300), `FTPLimitPolaczeniaDanychS` (30), `FTPPowitanie`, `FTPBaner`,
`FTPProtokol` (`ftps`), `FTPPrawaFolderu`, `FTPPrawaPliku`.

## Testy

`GOTOOLCHAIN=local _go-portable/bin/go.exe test ./wewn/ftp/...` (plus `GOOS=linux go vet ./wewn/ftp/...`).

1. Jednostkowe na zaślepkach (`zaslepki_test.go`): rozkład ścieżek (`..`, `C:`, strumienie NTFS, `\`), mapowanie
   błędów, korzeń tylko z widocznymi udziałami, ukryty udział = nie istnieje, bramka zapisu (korzeń, udział, tylko do
   odczytu, zła nazwa), DELE/RMD, rename między udziałami, odebranie praw w trakcie sesji, REST przy pobieraniu,
   przerwane pobranie i wgranie, błąd magazynu, odrzucenie `APPE`/`REST`+`STOR`, limit połączeń na IP, logowanie
   (konto bez FTPS, blokada, brak hasła w dzienniku), ustawienia TLS.
2. Integracyjne (`integracja_test.go`): serwer na `127.0.0.1:0`, certyfikat generowany w teście, prawdziwe magazyn,
   konta, transfery, bramka; własny minimalny klient FTPS (`klient_ftps_test.go`, `net/textproto` + `crypto/tls`, bo
   `jlaffaye/ftp` nie ma w module cache). Sprawdzają: USER/PASS bez TLS, dane bez TLS, złe logowania, listę korzenia,
   RETR, REST+RETR, STOR, STOR w udziale tylko do odczytu i w korzeniu, MKD, RNFR/RNTO, DELE do kosza, RMD, wyniki
   transferów, zamknięcie przez kontekst.
3. Testy mutacyjne wykonane ręcznie (blokada zdjęta = czerwono): flaga FTPS przy logowaniu, limit IP,
   `MandatoryEncryption`, zakaz zapisu w korzeniu, sprawdzanie konta w trakcie sesji.

## Czego nie zrobiono

1. Wznawianie wgrywania (`REST`+`STOR`, `APPE`): odrzucane; wymagałoby `DopiszKawalek` z semantyką pliku `.skrytka_czesc`.
2. Tryb aktywny wyłączony; `SITE`, `MFMT`, `CHMOD`, rekurencyjne usuwanie: nieobsługiwane.
3. Tryb publiczny za NAT: `Zaleznosci.AdresPubliczny` (IPv4) idzie do `Settings.PublicHost` w odpowiedzi `PASV`;
   puste = adres lokalny połączenia (VPN/LAN). Zły adres (nie IPv4) = błąd startu `Uruchom`.
4. FTPS niejawny (port 990) nie jest obsługiwany, tylko jawny `AUTH TLS`.
5. Nie sprawdzone z prawdziwym FileZillą / Total Commanderem.
