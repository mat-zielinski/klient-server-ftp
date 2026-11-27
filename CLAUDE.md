# CLAUDE.md — Skrytka (serwer plików na Windows)

Zasady pracy z kodem dla Claude Code i innych narzędzi AI. Szczegóły są w `docs/` (mapa niżej). Zasady przeniesione ze
stacji `Programator_Czarek` (jej `CLAUDE.md` i skill `programator-totalny`) i przełożone na Go + Svelte.

**Na start sesji:** [do-zrobienia.md](do-zrobienia.md) (otwarte punkty: do poprawy, do sprawdzenia na żywo, decyzje),
potem [PLAN.md](PLAN.md) (sprinty) i dokument obszaru, którego dotyczy zadanie (`docs/obszary/<obszar>.md`).
Skill projektu: `.claude/skills/skrytka/SKILL.md` (jak pracować, testy, dokumenty, git).

## Język i forma odpowiedzi

- Projekt jest **po polsku**: kod (nazwy bez ogonków: `sciezka`, `uzytkownik`; komentarze z ogonkami), komunikaty
  programu, dokumentacja i rozmowa. Komunikaty: zdania wielką literą, bez długich myślników.
- Raport dla użytkownika: góra **A/B/C/D**, wewnątrz **1/2/3**, jedna myśl = jeden punkt. Zawsze wypisz, czego
  **NIE** zrobiono i dlaczego, oraz co nie było sprawdzone (np. „tylko testy, bez uruchomienia przez WireGuard”).
- **Gdy polecenie zostawia wątpliwość (co, gdzie, jak wygląda), najpierw zapytaj** oknem `AskUserQuestion`
  z opcjami, zamiast zgadywać i zamiast pytać w samym tekście.

## Czym jest projekt

Jeden plik `skrytka.exe` na komputerze właściciela udostępnia wskazane foldery: przez przeglądarkę (HTTPS, interfejs
w stylu stacji i UniFi) i przez **FTPS** (FileZilla, WinSCP, Total Commander). Klient nic nie instaluje poza ewentualnie
WireGuardem. Dwa tryby sieci: **VPN** (WireGuard, serwer widoczny tylko pod adresem LAN / tunelu, zalecany) i
**publiczny** (port otwarty na routerze, wymuszone HTTPS i FTPS). Wzór działania: FileBrowser Quantum. Plan i sprinty:
[PLAN.md](PLAN.md).

## Architektura (skrót, pełny opis: [docs/architektura.md](docs/architektura.md))

| warstwa | katalog | zasada |
|---|---|---|
| konfiguracja | `wewn/konfiguracja/` | WSZYSTKIE stałe, ścieżki (`sciezki.go`), wersja (`wersja.go`); każdy obszar ma własny plik (`konta.go`, `siec.go` …) |
| rdzen | `wewn/rdzen/` | czysta logika: piaskownica ścieżek, nazwy, rodzaje plików, sortowanie, wyszukiwanie, bramka `CzyWolno`; bez dysku, sieci i `time.Now` |
| magazyn | `wewn/magazyn/` | JEDYNY dostęp do dysku, każda ścieżka przez `rdzen.SciezkaNaDysku` |
| uslugi | `wewn/uslugi/{konta,dziennik,transfery,udostepnienia}` | przebiegi bez HTTP i FTP |
| widoki | `wewn/http/`, `wewn/ftp/`, `wewn/siec/` | tylko tłumaczą protokół na wywołania usług |
| aplikacja | `wewn/aplikacja/` | składa całość, kreator, usługa Windows |
| start | `cmd/skrytka/main.go` | woła tylko `aplikacja.Uruchom` |
| interfejs | `web/` | Svelte 5 + Vite, wkompilowany w `.exe` (`web/osadz.go`, `go:embed`) |

1. **Import tylko w dół**: `konfiguracja` ← `rdzen` ← `magazyn` ← `uslugi/*` ← `http`, `ftp`, `siec` ← `aplikacja` ←
   `cmd`. Pilnuje strażnik architektury (opis: [docs/testy.md](docs/testy.md)).
2. **Wszystko globalnie** w `wewn/konfiguracja`: każda liczba, limit, port, nazwa pliku, czas. Żadnych ścieżek na
   sztywno: wszystko względem folderu programu (`konfiguracja.KatalogAplikacji`, zmienna `SKRYTKA_KATALOG` w testach).
3. **Każda operacja na plikach i każde logowanie przez bramkę** `rdzen.CzyWolno`; widoki HTTP i FTPS nie mają własnych
   reguł uprawnień.
4. **Wygląd tylko z tokenów** `web/src/styl/tokeny.css` (nazwy i wartości z motywu stacji) i **ikon SVG** z
   `web/src/ikony/ikony.ts`; zero `#rrggbb`, `rgb(` i `px` poza tokenami, **bez emoji**. Dwa motywy (ciemny, jasny),
   każda strona działa w obu. Responsywność bez przeskakiwania: elementy się zwężają, nic nie znika bez zamiennika.
   Szczegóły: [docs/interfejs.md](docs/interfejs.md).
5. **Limity jakości**: plik ≤ 600 linii, funkcja ≤ 80 linii, złożoność ≤ 15, ≤ 8 argumentów; bez `fmt.Print*`
   (komunikaty przez dziennik), bez zignorowanego błędu (wyjątek `// celowo: <powód>`), `panic` tylko przy budowie.
6. **Zmiana logiki = testy w tej samej zmianie**: test nowego działania, który **pada bez zmiany** (sprawdzone
   wstrzyknięciem: tymczasowo cofnij poprawkę, test czerwony, przywróć). Nowa blokada bezpieczeństwa (ścieżki,
   uprawnienia, hasła, sesje, limity, CSRF) = wpis w liście **mutacji blokad** (strażnik sprawdza, że jej testy padają
   bez niej). Testy tabelaryczne, dysk tylko `t.TempDir()`, zegar wstrzykiwany, bez `time.Sleep`.
7. **Dziennik serwera** `logi/dziennik_RRRR-MM.jsonl` (kto, co, skąd, ile, czas, wynik) i strona „Dziennik”; nigdy
   hasło, token ani treść ciasteczka. Przy zgłoszeniu „coś nie działa” zacznij od dziennika.

## Uruchomienie i sprawdzanie

- Narzędzia: Go 1.27.4 przenośne w `_go-portable\` (`_go-portable\bin\go.exe`, zawsze `GOTOOLCHAIN=local`), Node i npm
  w systemie. Go nie jest zainstalowany w systemie.
- Budowanie: `skrypty\buduj.bat` (interfejs `npm run build` w `web\`, potem `go build` do `wyjscie\skrytka.exe`).
  Opis: [docs/wydania.md](docs/wydania.md).
- **Po każdej zmianie kodu ZAWSZE zapytaj oknem (`AskUserQuestion`), jakie testy teraz: częściowe albo pełne.**
  Częściowe = tylko testy zmienionych pakietów (`_go-portable\bin\go.exe test ./wewn/<pakiet>/...`, przy interfejsie
  `npm run build` i `npm run sprawdz` w `web\`), ok. 1 min; pełne = `sprawdz.bat`. Nigdy obu po kolei i nigdy samemu
  dorzucać pełnych. **Przed każdym pushem też zapytaj oknem: częściowe czy pełne.**
- **Testy całościowe: `sprawdz.bat`** (strażnicy + `go vet` + `go test ./...` + mutacje blokad + budowa interfejsu);
  `sprawdz.bat szybko` = tylko strażnicy, bez testów. Opis: [docs/testy.md](docs/testy.md).
- Serwera nie uruchamiaj na porcie publicznym i nie zmieniaj Zapory Windows bez polecenia użytkownika.

## Wersja

`Wersja` w `wewn/konfiguracja/wersja.go` (widać ją w stopce interfejsu, w tytule strony i w `GET /api/stan`).
**Podbijaj przy KAŻDEJ zmianie** (0.1 → 0.2 …) i w raporcie podaj, jaka wersja ma być teraz widoczna. Każda nowa wersja
= wpis w [zmiany.md](zmiany.md) w tej samej zmianie (co wnosi i co poprawia).

## Dokumenty w tej samej zmianie

Każda zmiana działania, stałych, plików albo przepływu pracy aktualizuje **w tej samej zmianie** pliki `.md` i `.txt`,
których dotyczy (tabela „co się zmieniło → co zaktualizować” w skillu `skrytka`). Zmiana działania = przeczytaj cały
opis tego przebiegu w dokumencie i popraw zdania, które przestały być prawdą; nie dopisuj akapitu obok starego opisu.

## Praca wieloagentowa

1. Każdy agent ma **rozłączne pliki** (lista w poleceniu; właściciele interfejsu: [docs/interfejs.md](docs/interfejs.md),
   zasady wspólne: [docs/zasady_agentow.md](docs/zasady_agentow.md)). Sygnatur z kontraktów (`kontrakt.go`,
   [docs/api.md](docs/api.md), `web/src/api/typy.ts`) agent nie zmienia; brak = sekcja „Potrzeby” w raporcie.
2. Agent poprawia dokumenty SWOJEJ zmiany (`docs/obszary/<obszar>.md`) albo zwraca dokładną listę zdań do zmiany.
3. **Agenci nie używają gita** (żadnego `git add/rm/mv/stash/checkout/reset/commit/push`) i nie zmieniają `go.mod`.
   **Commit i push robi sesja główna** po przeglądzie raportów.
4. Testy agentów tylko częściowe, dla własnego pakietu, szeregowo.

## Git

- Repozytorium: **https://github.com/mat-zielinski/klient-server-ftp**, gałąź **`main`**.
- **Repo jest PUBLICZNE.** Nigdy do repo: danych użytkownika i udostępnianych plików, haseł, hashy haseł, tokenów,
  kluczy i certyfikatów (`dane/certyfikaty/`), plików `dane/konta.json`, `dane/udostepnienia.json`,
  `dane/konfiguracja.json`, logów (`logi/`), konfiguracji WireGuard (`*.conf` z kluczami), kopii (`kopie/`), plików
  `.exe`. Pilnuje [.gitignore](.gitignore); przed każdym commitem sprawdź `git status` i listę plików w zmianie.
  W dokumentach i testach tylko przykładowe adresy (`192.168.0.10`, `example.com`) i wymyślone dane.
- Commit po polsku: `Skrytka X.Y: <co>` (X.Y = nowa wersja z `wersja.go`), treść po polsku, na końcu stopka
  współautora. Bez force-push i bez `--no-verify`.
- Kopia przed większą zmianą bez commitu: `kopie\RRRR-MM-DD\` (poza repo).

## Mapa dokumentacji

| dokument | o czym |
|---|---|
| [README.md](README.md) | co to jest, uruchomienie, budowanie, katalogi |
| [INSTALACJA.txt](INSTALACJA.txt) | instalacja krok po kroku dla właściciela serwera |
| [PLAN.md](PLAN.md) | plan, technologia, sprinty, decyzje |
| [do-zrobienia.md](do-zrobienia.md) | otwarte punkty: do poprawy, do sprawdzenia na żywo, do decyzji |
| [zmiany.md](zmiany.md) | co wnosi każda wersja |
| [docs/architektura.md](docs/architektura.md) | warstwy, pakiety, przepływ pobrania i transferów na żywo, bezpieczeństwo |
| [docs/api.md](docs/api.md) | kontrakt HTTP serwer ↔ interfejs |
| [docs/interfejs.md](docs/interfejs.md) | interfejs `web/`: zasady wyglądu, właściciele plików, wspólne komponenty |
| [docs/dla_klienta.md](docs/dla_klienta.md) | instrukcja dla osoby pobierającej (WireGuard, przeglądarka, FTPS) |
| [docs/testy.md](docs/testy.md) | `sprawdz.bat`, strażnicy, testy częściowe i pełne |
| [docs/wydania.md](docs/wydania.md) | wersje, budowanie, paczka, git |
| [docs/zasady_agentow.md](docs/zasady_agentow.md) | zasady i raport dla agentów |
| `docs/obszary/` | opis każdego obszaru (pakietu) od jego agenta, m.in. `straznicy.md` |
