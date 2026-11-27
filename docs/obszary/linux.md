# Obszar: instalacja na Linuksie (systemd)

Skrypty powłoki POSIX (`#!/bin/sh`, `set -eu`, działają w dash i bash), które instalują, aktualizują i usuwają
Skrytkę jako usługę systemd. Kod Go nie jest tu zmieniany: jednostkę pisze sam program (`skrytka --usluga instaluj`,
`wewn/aplikacja/usluga_systemd.go`), skrypt dokłada nakładkę systemd i pliki pulpitu.

## Pliki

| plik | rola |
|---|---|
| `skrypty/instaluj-linux.sh` | instalacja i aktualizacja (root) |
| `skrypty/odinstaluj-linux.sh` | usunięcie (root); kopiowany też do `/opt/skrytka/` |
| `assets/linux/skrytka.png` | ikona 256×256 → `/usr/share/icons/hicolor/256x256/apps/skrytka.png` |
| `assets/linux/skrytka.desktop` | skrót w menu → `/usr/share/applications/skrytka.desktop` |
| `.github/workflows/wydanie.yml` | paczka tar.gz zawiera oba skrypty i `assets/linux/` |

## Układ po instalacji

| ścieżka | właściciel | zawartość |
|---|---|---|
| `/opt/skrytka/skrytka` | root, 0755 | program (usługa nie może go podmienić) |
| `/opt/skrytka/odinstaluj-linux.sh` | root, 0755 | skrypt usuwania |
| `/var/lib/skrytka/` | skrytka, 0750 | `dane/` (konfiguracja, konta, certyfikaty, blokada instancji), `logi/` |
| `/etc/systemd/system/skrytka.service` | root | pisze program (`User=skrytka`, `--konsola`, `CAP_NET_BIND_SERVICE`) |
| `/etc/systemd/system/skrytka.service.d/katalog.conf` | root | `Environment=SKRYTKA_KATALOG=/var/lib/skrytka` |

Katalog danych bez zmiany Go: `konfiguracja.KatalogAplikacji()` honoruje zmienną `SKRYTKA_KATALOG`, a jednostka
generowana przez program nie ma pola `Environment`, więc skrypt dokłada nakładkę (drop-in). `--usluga instaluj`
nadpisuje tylko `skrytka.service`, nakładka zostaje.

## instaluj-linux.sh

`sudo sh instaluj-linux.sh [--tak] [plik_programu]`

1. Sprawdza roota i `systemctl`, wykrywa architekturę (`x86_64` → amd64, `aarch64` → arm64, inna = błąd).
2. Program: argument, inaczej `./skrytka` obok skryptu (paczka), inaczej `wyjscie/skrytka-linux-<arch>` (repo).
3. Użytkownik systemowy `skrytka` (`useradd --system --user-group`, powłoka nologin, dom `/var/lib/skrytka`).
4. Zatrzymuje działającą usługę, kopiuje program przez plik tymczasowy i `mv` (atomowo).
5. `/var/lib/skrytka`: tworzy, przenosi `dane/` i `logi/` ze starego układu (`/opt/skrytka/dane`), gdy w nowym
   miejscu ich nie ma; `chown -R skrytka`.
6. Nakładka `katalog.conf`, `skrytka --usluga instaluj` (jednostka + `enable`), `daemon-reload`, `start`.
7. Ikona i `.desktop` (`install -D`), `gtk-update-icon-cache` i `update-desktop-database`, gdy są.
8. Zapora: wykrywa włączone ufw (`ufw status`) albo firewalld (`firewall-cmd --state`); pyta (`[t/N]`, domyślnie
   nie) o 8443/tcp, 2121/tcp, 50000-50100/tcp. Z `--tak` albo bez terminala nie otwiera nic i wypisuje komendę.
9. Wypisuje stan usługi, `https://127.0.0.1:8443`, adresy IPv4 (`ip -4 addr`, zapasowo `hostname -I`),
   `journalctl -u skrytka -f`.

Idempotentny: ponowne uruchomienie = aktualizacja programu, dane, konta i certyfikat zostają.

## odinstaluj-linux.sh

`sudo sh odinstaluj-linux.sh [--z-danymi] [--tak]`: `systemctl disable --now`, usunięcie jednostki i nakładki,
`daemon-reload`, usunięcie `/opt/skrytka`, ikony i `.desktop`. `--z-danymi` dodatkowo usuwa `/var/lib/skrytka`
i użytkownika `skrytka` po wpisaniu „tak” (bez terminala wymaga `--tak`); potwierdzenie pada przed jakąkolwiek
zmianą. Reguł zapory i udostępnianych folderów nie rusza.

## Skrót w menu: dlaczego przeglądarka

`Exec=xdg-open https://127.0.0.1:8443`. Na serwerze działa usługa; uruchomienie binarki z menu przez zwykłego
użytkownika (tryb z zasobnikiem) skończyłoby się błędem: bez `SKRYTKA_KATALOG` szukałaby danych w `/opt/skrytka/dane`
(brak praw zapisu), a z nią trafiłaby na blokadę instancji usługi (kod 3). Panel w przeglądarce działa zawsze, gdy
usługa nasłuchuje na pętli (domyślnie tak).

## Stałe

Porty w skrypcie to domyślne z `wewn/konfiguracja/konfiguracja.go` (`DomyslnyPortHTTPS` 8443, `DomyslnyPortFTPS`
2121, `DomyslnyPasywnyOd/Do` 50000-50100). Zmienione w Ustawieniach porty trzeba otworzyć w zaporze ręcznie,
a w skrócie `.desktop` poprawić adres.

## Sprawdzenie

`bash -n`, `sh -n` i `dash -n` obu skryptów: bez błędów. Przegląd ręczny (cytowanie zmiennych, `set -eu`,
brak bashyzmów); shellcheck niedostępny. Skryptów nie uruchomiono (praca na Windows).

## Czego nie zrobiono

- Brak testu na prawdziwym Linuksie (systemd, useradd, ufw/firewalld, gtk-update-icon-cache).
- Skrót `.desktop` ma port 8443 na sztywno; po zmianie portu HTTPS trzeba go poprawić.
- Ikona tylko 256×256 (inne rozmiary hicolor skaluje środowisko pulpitu).
- Tryb publiczny na porcie 443 lub innym: porty w zaporze ręcznie.
