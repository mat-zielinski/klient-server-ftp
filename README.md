# Skrytka — serwer plików na Windows

Jeden plik **`skrytka.exe`** na Twoim komputerze udostępnia wskazane foldery (np. `D:\Udostepnione`) wybranym
osobom. Klient nic nie instaluje: otwiera przeglądarkę na telefonie albo laptopie, loguje się i pobiera pliki. Dla
programów typu FileZilla, WinSCP i Total Commander ten sam serwer mówi **FTPS** (FTP szyfrowane) na osobnym porcie.

| funkcja | jak |
|---|---|
| przeglądanie i pobieranie | przeglądarka (HTTPS), tabela albo kafelki, podgląd obrazów, PDF, wideo i tekstu, wznawianie pobrań (`Range`), folder jako ZIP |
| pobieranie programem | FTPS (FTP z jawnym TLS) na tych samych kontach i uprawnieniach |
| prawy przycisk na pliku | menu „Pobierz przez…”: przeglądarka, klient FTP, link bezpośredni, ZIP, link publiczny |
| transfery na żywo | klient: panel „Moje pobrania”; właściciel: Pulpit z prędkością i listą aktywnych pobrań |
| konta | użytkownicy z prawem odczytu albo zapisu do udziałów, hasła argon2id, blokada po złych hasłach |
| linki publiczne | z datą wygaśnięcia, hasłem i limitem pobrań |
| dziennik | kto, co i skąd pobrał, ile i jak długo; wykresy, eksport CSV |

Dwa tryby sieci (pole `tryb_sieci` w konfiguracji):

1. **VPN (WireGuard), zalecany.** Klient łączy się tunelem WireGuard z Twoją siecią i widzi komputer pod adresem LAN
   (np. `192.168.0.10`). Serwer słucha tylko na adresach LAN i tunelu, z internetu jest niewidoczny.
2. **Publiczny.** Port przekierowany na routerze na Twój komputer. Wtedy serwer wymusza HTTPS i FTPS, blokadę po złych
   hasłach i limit prób; zwykłe FTP bez szyfrowania jest wyłączone.

Serwer VPN zostaje tam, gdzie jest (router, UniFi z WireGuard). Skrytka go nie konfiguruje, tylko pokazuje adresy,
pod którymi jest osiągalna, i sprawdza porty.

Wygląd: jak stacja Programator_Czarek i interfejs UniFi (granatowe tło, karty, pasek boczny z ikonami), motyw ciemny
i jasny. Wersja jest w stopce interfejsu; źródło: `Wersja` w
[wewn/konfiguracja/wersja.go](wewn/konfiguracja/wersja.go), co wnosi każda wersja: [zmiany.md](zmiany.md).

> Zrzut ekranu: do dodania po sprincie 2 (strona Pliki w obu motywach).

## Uruchomienie

Instalacja krok po kroku: [INSTALACJA.txt](INSTALACJA.txt). W skrócie:

1. Skopiuj folder ze `skrytka.exe` w dowolne miejsce (pulpit, `C:\Skrytka`, inny dysk). Wszystkie ścieżki są względne
   do folderu programu.
2. Uruchom `skrytka.exe`. Przy pierwszym starcie przeglądarka pokazuje **kreator**: konto administratora, pierwszy
   udostępniany folder i tryb sieci.
3. Adresy serwera widać na Pulpicie (np. `https://192.168.0.10:8443`). Domyślne porty: HTTPS 8443, HTTP 8080
   (przekierowanie), FTPS 2121, porty pasywne FTPS 50000–50100.
4. Osobom pobierającym przekaż instrukcję [docs/dla_klienta.md](docs/dla_klienta.md).

## Budowanie

Wymagania: Go 1.27.4 przenośne w `_go-portable\` (nie jest w repo, rozpakuj oficjalne Go dla Windows amd64 do tego
folderu), Node i npm w systemie.

```bat
skrypty\buduj.bat
```

Skrypt buduje interfejs (`npm ci` i `npm run build` w `web\`, wynik w `web\dist\`) i serwer
(`_go-portable\bin\go.exe build` z `GOTOOLCHAIN=local`, interfejs wkompilowany przez `go:embed`), wynik:
`wyjscie\skrytka.exe`. Testy: `sprawdz.bat` (opis: [docs/testy.md](docs/testy.md)). Wydania i paczka:
[docs/wydania.md](docs/wydania.md).

## Katalogi

| katalog | zawartość | w repo |
|---|---|---|
| `cmd/skrytka/` | start programu (`main.go`) | tak |
| `wewn/konfiguracja/` | wszystkie stałe, ścieżki, wersja | tak |
| `wewn/rdzen/` | czysta logika: ścieżki, nazwy, uprawnienia, sortowanie, wyszukiwanie | tak |
| `wewn/magazyn/` | jedyny dostęp do dysku | tak |
| `wewn/uslugi/` | konta i sesje, dziennik, transfery na żywo, linki publiczne | tak |
| `wewn/http/`, `wewn/ftp/`, `wewn/siec/` | API HTTP, serwer FTPS, adresy, TLS, test portu, Zapora | tak |
| `wewn/aplikacja/` | składanie całości, kreator, usługa Windows | tak |
| `web/` | interfejs Svelte 5 + Vite; `web/dist/` = zbudowany (w repo tylko zaślepka `index.html`) | tak, bez `node_modules/` i `dist/` |
| `docs/` | dokumentacja, `docs/obszary/` opis każdego obszaru | tak |
| `skrypty/` | `buduj.bat` i inne skrypty | tak |
| `narzedzia_dev/` | strażnicy i narzędzia programisty | tak |
| `_go-portable/` | przenośne Go 1.27.4 | nie |
| `dane/` | `konfiguracja.json`, `konta.json`, `udostepnienia.json`, `certyfikaty/` | nie (dane tego komputera) |
| `logi/` | `dziennik_RRRR-MM.jsonl`, `bledy.log` | nie |
| `wyjscie/` | zbudowany `skrytka.exe` i paczka | nie |
| `kopie/` | kopie przed większą zmianą (`kopie\RRRR-MM-DD\`) | nie |

## Dokumentacja

| dokument | o czym |
|---|---|
| [INSTALACJA.txt](INSTALACJA.txt) | instalacja dla właściciela serwera |
| [docs/dla_klienta.md](docs/dla_klienta.md) | instrukcja dla osoby pobierającej |
| [PLAN.md](PLAN.md) | plan, technologia, sprinty |
| [docs/architektura.md](docs/architektura.md) | warstwy, pakiety, przepływy, bezpieczeństwo |
| [docs/api.md](docs/api.md), [docs/interfejs.md](docs/interfejs.md) | API HTTP; interfejs i komponenty |
| [docs/testy.md](docs/testy.md), [docs/wydania.md](docs/wydania.md) | testy i strażnicy; budowanie, wersje, paczka |
| [zmiany.md](zmiany.md), [do-zrobienia.md](do-zrobienia.md) | historia wersji; otwarte punkty |
| [CLAUDE.md](CLAUDE.md) | zasady pracy z kodem |

Repozytorium jest publiczne: https://github.com/mat-zielinski/klient-server-ftp (gałąź `main`). Dane tego komputera
(konta, hasła, certyfikaty, logi, konfiguracja) nigdy do niego nie trafiają.
