# Obszar: zasobnik (ikona obok zegarka Windows, StatusNotifier na Linuksie)

Część pakietu `wewn/aplikacja`. Biblioteka `fyne.io/systray` (Windows bez cgo, Linux przez D-Bus / StatusNotifierItem).

## Kiedy jest ikona

`czyPokazacZasobnik` (czysta funkcja, testowana): zwykłe uruchomienie bez `--konsola` i bez `--bez-zasobnika`,
na Windows zawsze, na Linuksie tylko z pulpitem (`DISPLAY` albo `WAYLAND_DISPLAY`) i nie pod systemd
(`INVOCATION_ID`). Usługa Windows (`svc.IsWindowsService`) w ogóle nie dochodzi do tej decyzji; jednostka systemd
uruchamia `--konsola`. `--bez-zasobnika` nie łączy się z `--usluga`.

## Jak działa

1. `uruchomZZasobnikiem` (`zasobnik.go`): serwer (`uruchomSerwer`) w gorutynie, `systray.Run` na wątku głównym
   (biblioteka blokuje go w `init`). `Opcje.PoStarcie` przekazuje gotową aplikację do zasobnika.
2. „Zakończ” anuluje kontekst serwera: łagodne zamknięcie jak przy Ctrl+C, potem `systray.Quit` i wyjście.
   Ctrl+C / SIGTERM / błąd serwera też kończą pętlę zasobnika. Pętla, która się nie uruchomiła (okno nie powstało),
   nie wraca po `Quit`: po `ZasobnikCzasWyjscia` od zatrzymania serwera proces kończy się sam (`os.Exit`).
3. Panika biblioteki (np. brak D-Bus przy zamykaniu na Linuksie) jest łapana: wpis w dzienniku, serwer działa dalej.
   Błędy wewnętrzne biblioteki (np. brak D-Bus przy starcie) biblioteka tylko wypisuje na stderr, serwer działa.
4. Co `ZasobnikOkresOdswiezania` (1 s) jedna gorutyna czyta `Rejestr.Aktywne("")` i zmienia tylko to, co się
   zmieniło: dymek, ikonę (`ZasobnikAktywny` przy transferach), podmenu. Adres dla klientów liczony co
   `ZasobnikOkresAdresu` (lista interfejsów) i przy „Kopiuj adres”; jasność paska co `ZasobnikOkresMotywu`.

## Dymek i ikona

- „Skrytka — działa · 192.168.1.5:8443”, w trakcie transferów „Skrytka — 2 pobierania · 12,3 MB/s”
  („1 pobranie, 1 wgranie · …”); przy starcie „uruchamianie”, przy zamykaniu „zamykanie”. Najwyżej 127 znaków.
- Prędkość jak w interfejsie (`web/src/stan/format.ts`): podstawa 1024, przecinek.
- Adres (`adresDlaKlientow`): `adres_nasluchu` z konfiguracji (gdy nie 0.0.0.0), w trybie publicznym znany adres
  publiczny, potem pierwszy LAN, pierwszy WireGuard, na końcu 127.0.0.1; port z faktycznego nasłuchu HTTPS.
- Ikony z `wewn/aplikacja/ikony`. Windows: ICO; na jasnym pasku zadań (`HKCU\...\Themes\Personalize`,
  `SystemUsesLightTheme = 1`) warianty `ZasobnikCiemny` / `ZasobnikCiemnyAktywny`. Linux: StatusNotifier czyta
  PNG, więc `pngZIco` wyjmuje największy PNG z ICO (zawsze jasne ikony); brak PNG = `ikony.Aplikacja`.

## Menu

Nagłówek „Skrytka <wersja>” (wyszarzony); „Otwórz panel” (też zwykłe kliknięcie ikony; `https://127.0.0.1:<port>/`
przez `Opcje.OtworzPrzegladarke`); „Kopiuj adres dla klientów” (`https://IP:port`; Windows `clip` bez okna konsoli,
Linux `wl-copy` albo `xclip -selection clipboard`); podmenu „Aktywne transfery” (do 5 pozycji „↓ kto · plik ·
prędkość”, `↑` = wgranie, „gość” = link publiczny, albo „Brak aktywnych transferów”); separator; „Uruchamiaj
z Windows” (zaznaczalne; `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, wartość `Skrytka` = ścieżka exe
w cudzysłowie; zaznaczone tylko, gdy wpis wskazuje ten exe) / na Linuksie „Uruchamiaj po zalogowaniu”
(`~/.config/autostart/skrytka.desktop`); „Folder danych i logów” (folder aplikacji: Eksplorator / `xdg-open`);
separator; „Zakończ”.

## Pliki

`wewn/aplikacja/zasobnik.go` (logika i składanie), `zasobnik_menu.go` (`//go:build windows || linux`, systray),
`zasobnik_windows.go` (rejestr, motyw, schowek, Eksplorator), `zasobnik_linux.go` (.desktop, schowek, xdg-open),
`zasobnik_inne.go` (zaślepki), `zasobnik_test.go`; argument w `argumenty.go`, wybór w `uruchom.go`.
Stałe: `wewn/konfiguracja/zasobnik.go`.

## Testy

`go test ./wewn/aplikacja/...` (tabelaryczne, czysta logika): argument `--bez-zasobnika`, decyzja o pokazaniu
(system, ekran, systemd, konsola, usługa), tekst dymka, liczebniki, prędkość, pozycje podmenu (limit, skracanie
nazwy, gość), adres dla klientów, PNG z ICO (także uszkodzone ICO i osadzone ikony), wybór ikony (jasny / ciemny
pasek, brak ciemnych), wpis autostartu Windows, plik .desktop (znaki ucieczki). Mutacja sprawdzona ręcznie:
zdjęcie warunków `Konsola` i systemd w `czyPokazacZasobnik` daje czerwony test.
Ręcznie: zbudowany `skrytka.exe` (`-H windowsgui`) w izolowanym `SKRYTKA_KATALOG` (127.0.0.1, porty
28080/28443/22121) żyje, `/api/zdrowie` odpowiada, ikona ustawiona (plik `systray_temp_icon_*` biblioteki).

## Czego nie zrobiono

1. **Powiadomienie systemowe przy starcie pobrania** (przez kogoś innego niż lokalny admin): `fyne.io/systray`
   nie ma publicznego API powiadomień (ma tylko ikonę, dymek i menu). Do zrobienia: Windows przez
   `Shell_NotifyIcon` z `NIF_INFO` (wymaga uchwytu ikony z biblioteki, którego nie udostępnia) albo osobną
   bibliotekę (np. `github.com/go-toast/toast`), Linux przez `org.freedesktop.Notifications` w D-Bus
   (`godbus/dbus` już jest w `go.sum`). Nowa biblioteka wymaga zgody.
2. Klikanie pozycji menu nie było sprawdzane automatycznie (wymaga pulpitu); Linux tylko kompilacja.
3. Wylogowanie z Windows przy działającym programie: zamyka proces system (bez łagodnego zatrzymania serwera).
