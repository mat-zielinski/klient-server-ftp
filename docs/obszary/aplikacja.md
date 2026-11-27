# Obszar: aplikacja (składanie, start, zamykanie, usługa)

Pakiet `wewn/aplikacja` składa wszystkie warstwy w działający serwer. `cmd/skrytka/main.go` jest cienki: woła tylko
`aplikacja.Uruchom(os.Args[1:])` i kończy proces zwróconym kodem. Działa na **Windows** i **Linuksie** (kod zależny od
systemu w plikach `*_windows.go` / `*_linux.go`; inne systemy kompilują się z zaślepkami `system_inne.go`).

## Co robi

1. **Argumenty** (`argumenty.go`): brak = serwer, przy pierwszym uruchomieniu (brak kont) otwiera przeglądarkę na
   `https://127.0.0.1:<port>` (kreator); `--konsola` = serwer bez przeglądarki, wpisy serwera także na stderr;
   `--usluga instaluj|usun|start|stop` = sterowanie usługą. Zły argument = komunikat użycia, kod 2.
2. **Składanie** (`aplikacja.go`): `konfiguracja.Wczytaj`, `dziennik.Otworz(logi)`, `konta.Otworz`, `konta.NoweSesje`,
   `udostepnienia.Otworz`, `transfery.Nowy(naKoniec)`, `siec.KonfiguracjaTLS` + odcisk, `web.Pliki()`, `magazyn.Nowy`.
   Brak folderu udziału nie blokuje startu (wpis błędu, pusty magazyn). `naKoniec` transferu zapisuje wpis
   `RodzajPobranie` / `RodzajWgranie` (kto, IP, protokół, udział, ścieżka, bajty = przesłane, czas od startu, wynik).
3. **Serwery** (`serwery.go`):
   1. HTTPS na `PortHTTPS` dla każdego adresu z `siec.AdresyNasluchu` (jeden `http.Server`, wiele nasłuchów; kopia
      `tls.Config`, żeby net/http nie zmieniał wspólnej konfiguracji używanej przez FTPS). Limity:
      `ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes`; **bez** `WriteTimeout` (duże pobrania). Żaden adres się
      nie otworzył = błąd startu.
   2. HTTP na `PortHTTP` tylko `127.0.0.1`; w trybie publicznym wyłącznie przekierowanie 308 na HTTPS (host z nagłówka
      tylko gdy to IP albo `localhost`, inaczej `127.0.0.1`: nagłówek nie może skierować w obce miejsce).
   3. FTPS (`ftps.go`) gdy `FTPSWlaczone`: `ftp.Uruchom` na każdym adresie nasłuchu i `PortFTPS`. W trybie publicznym
      w tle raz pytamy `siec.AdresPubliczny` i restartujemy FTPS z polem `AdresPubliczny` (PASV za routerem).
   4. Co `OkresSprzataniaUdostepnien` (godzina) `Udostepnienia.Sprzataj()`.
   5. Fala 2 (`uslugi_magazynu.go`): `monitor.Nowy(monitor.ZLicznikami(...))` + `Uruchom(ctx)` przy starcie; liczniki
      = otwarte połączenia HTTP/HTTPS (hak `http.Server.ConnState`: nowe +1, zamknięte / przejęte −1) + sterujące FTPS (`ftp.Polaczenia()`) i
      `len(Rejestr.Aktywne(""))`. Na każdy magazyn: `indeks.Nowy(m)` + `Uruchom` (kontekst potomny serwera),
      `miniatury.Nowe(m, "")`, `sumy.Nowe(m)`; przy podmianie magazynu wszystkie trzy odtwarzane, indeksowanie
      starego anulowane. `Miniatury.Sprzataj()` raz w tle przy starcie.
   6. Wpisy pobrań i wgrań (HTTP i FTPS) robi wyłącznie `naKoniec` rejestru transferów (test: jedno pobranie HTTPS =
      jeden wpis).
4. **Konfiguracja w pamięci** (`konfig.go`): `Konfiguracja()` zwraca kopię pod mutexem; `ZapiszKonfiguracje`
   (szeregowana) sprawdza udziały (`magazyn.Nowy`), zapisuje plik atomowo, podmienia stan. Zmiana udziałów działa od
   razu: nowy magazyn, nowa obsługa HTTP podmieniana atomowo (sesje zostają), restart FTPS. Porty, tryb, adres, FTPS
   i certyfikat działają po restarcie (wpis w dzienniku).
5. **Zamykanie**: Ctrl+C / SIGTERM / Stop usługi → anulowanie kontekstu → `Shutdown` z limitem `CzasZamykania`
   (potem `Close`), zatrzymanie FTPS, `Rejestr.Zamknij()`, wpis „Serwer zatrzymany.”, `Dziennik.Zamknij()`.
6. **Dziennik serwera** (`RodzajSerwer`): start (`WersjaBudowy` albo `Wersja`, tryb, adresy HTTPS/HTTP/FTPS), stop, błędy (nasłuch,
   udziały, TLS, FTPS, sprzątanie linków, przeglądarka).
7. **Jedna instancja**: Windows mutex `Global\Skrytka_jedna_instancja` (brak dostępu do mutexu usługi też = działa);
   Linux blokada `flock` na `dane/skrytka.lock`. Druga instancja: komunikat i kod 3.
8. **Komunikaty bez konsoli**: na stderr; w `skrytka.exe` (bez konsoli) okienko `MessageBox`.
9. **Usługa**:
   1. Windows (`usluga_windows.go`, `x/sys/windows/svc` i `svc/mgr`): `instaluj` = start automatyczny, ten sam .exe;
      program sam rozpoznaje, że działa jako usługa (`svc.IsWindowsService`), wtedy bez przeglądarki. Wymaga admina.
   2. Linux (`usluga_linux.go`, `usluga_systemd.go`): tylko root; `instaluj` zapisuje
      `/etc/systemd/system/skrytka.service` (`ExecStart=<exe> --konsola`, `User=skrytka` gdy konto istnieje, wtedy
      też `AmbientCapabilities=CAP_NET_BIND_SERVICE`), `systemctl daemon-reload` i `enable`; `usun` = `disable --now`,
      usunięcie pliku, `daemon-reload`; `start` / `stop` przez `systemctl`. Bez roota: komunikat z instrukcją.
10. **Przeglądarka**: Windows `rundll32 url.dll,FileProtocolHandler`; Linux `xdg-open` tylko gdy jest `DISPLAY` albo
    `WAYLAND_DISPLAY` i program istnieje, inaczej pomijamy.

## Pliki

`wewn/aplikacja/`: `aplikacja.go`, `argumenty.go`, `konfig.go`, `serwery.go`, `ftps.go`, `uruchom.go`,
`system_windows.go`, `system_linux.go`, `system_inne.go`, `usluga_windows.go`, `usluga_linux.go`,
`usluga_systemd.go`, `uslugi_magazynu.go`; testy `pomocnicze_test.go`, `serwer_test.go`. Stałe: `wewn/konfiguracja/aplikacja.go`.
Skrypty: `skrypty/buduj.bat`, `skrypty/buduj.sh`, `skrypty/instaluj-linux.sh`, `START.bat`.

## Budowanie

- `skrypty\buduj.bat` (Windows) i `skrypty/buduj.sh` (Linux): `npm run build` w `web/`, potem `CGO_ENABLED=0`,
  `-trimpath -ldflags "-s -w"`: `wyjscie\skrytka.exe` (`-H windowsgui`, bez konsoli), `wyjscie\skrytka_konsola.exe`
  (ten sam program z konsolą, do diagnozy; `--konsola` działa w obu), `wyjscie\skrytka-linux-amd64`, `wyjscie\skrytka-linux-arm64` (Raspberry Pi 64-bit).
- `START.bat`: buduje, gdy brak `wyjscie\skrytka.exe`, i uruchamia go (dane w `wyjscie\dane`, logi w `wyjscie\logi`).
- `skrypty/instaluj-linux.sh` i `odinstaluj-linux.sh` (root): program w `/opt/skrytka`, dane i logi w `/var/lib/skrytka`
  (nakładka systemd z `SKRYTKA_KATALOG`), użytkownik `skrytka`, ikona i skrót w menu. Opis: [linux.md](linux.md).

## Stałe (`wewn/konfiguracja/aplikacja.go`)

Limity HTTP (`CzasNaglowkaHTTP`, `CzasBezczynnosciHTTP`, `MaksBajtowNaglowkaHTTP`), `CzasZamykania`,
`OkresSprzataniaUdostepnien`, `NazwaMuteksuInstancji`, `PlikBlokadyInstancji`, programy przeglądarki, nazwy i pliki
usługi (Windows i systemd), argumenty, kody wyjścia (0 ok, 1 błąd, 2 złe argumenty, 3 druga instancja).

## Testy

`go test ./wewn/aplikacja/...`: tabelaryczne (argumenty, wpis transferu, przekierowanie z obcym hostem, adres
kreatora, port, restart wymagany, kopia konfiguracji, plik jednostki systemd, wersja, licznik połączeń) i składania (katalog `t.TempDir()`
przez `SKRYTKA_KATALOG`, porty 0, tylko 127.0.0.1): `/api/stan` odpowiada i po zamknięciu serwer milknie, kreator
otwiera przeglądarkę raz, HTTP w trybie publicznym tylko przekierowuje, zapis udziałów działa od razu i zły folder
niczego nie zmienia, a indeks/miniatury/sumy są odtwarzane, transfer trafia do dziennika, kreator + pobranie
HTTPS = dokładnie jeden wpis „pobranie”, FTPS startuje i zamyka się, jedna instancja. Mutacje sprawdzone ręcznie:
zdjęcie filtra hosta przekierowania i odłączenie `naKoniec` dają czerwone testy.

## Czego nie zrobiono

1. Ikona w zasobniku jest (`fyne.io/systray`, opis: [zasobnik.md](zasobnik.md)); brak powiadomień systemowych przy starcie pobrania.
2. Usługa Windows i systemd nie były uruchamiane naprawdę (wymagają admina / roota); testowana tylko treść jednostki
   i argumenty. Kompilacja Linux sprawdzona (`GOOS=linux go vet`), działanie na Linuksie nie.
3. FTPS w teście tylko startuje i się zamyka, bez połączenia klienta.
4. Let's Encrypt: wyzwanie TLS-ALPN-01 wymaga portu 443 (`PortHTTPS=443`); wyzwanie HTTP-01 na porcie 80 nieobsłużone.
