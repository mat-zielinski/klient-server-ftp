---
name: skrytka
description: Praca z kodem serwera plików Skrytka (Go 1.27 + Svelte 5, HTTPS i FTPS, tryb WireGuard albo publiczny): warstwy, bramka uprawnień, tokeny wyglądu, testy, wersja, dokumenty, git. Użyj przy każdej zmianie w projekcie (nowy adres API, strona interfejsu, obszar serwera, poprawka bezpieczeństwa, wygląd) oraz przed każdym commitem i pushem: mówi, które testy puścić i które pliki .md i .txt zaktualizować.
---

# Skrytka — jak pracować z kodem

Język projektu: **polski** (nazwy bez ogonków w kodzie, komentarze, komunikaty, dokumenty, rozmowa). Odpowiedzi:
A/B/C/D, wewnątrz 1/2/3, zawsze z listą tego, czego **NIE** zrobiono, i tego, czego nie sprawdzono.

## Najpierw

1. Przeczytaj `CLAUDE.md` (zasady), `do-zrobienia.md` (otwarte punkty), `docs/architektura.md` (warstwy, pakiety,
   przepływy) i dokument obszaru `docs/obszary/<obszar>.md`.
2. Kontrakty: `docs/api.md` (HTTP), `docs/interfejs.md` (komponenty `web/`), `kontrakt.go` w pakietach `wewn/`,
   `web/src/api/typy.ts`. Zmiana kontraktu = świadoma decyzja sesji głównej, nie przy okazji.
3. **Coś nie działa?** Najpierw dziennik serwera `logi/dziennik_RRRR-MM.jsonl` (strona „Dziennik”), potem test, który
   odtwarza błąd.
4. **Wątpliwość w poleceniu (co, gdzie, jak wygląda)?** Zapytaj oknem `AskUserQuestion` z opcjami, nie zgaduj.

## Mapa dokumentacji

| dokument | o czym |
|---|---|
| `README.md`, `INSTALACJA.txt` | przegląd, uruchomienie, budowanie, katalogi; instalacja dla właściciela serwera |
| `PLAN.md` | technologia, sprinty, decyzje |
| `docs/architektura.md` | warstwy, pakiety, przepływ pobrania, transfery na żywo, bezpieczeństwo |
| `docs/api.md`, `docs/interfejs.md` | kontrakt HTTP; interfejs, tokeny, właściciele plików |
| `docs/dla_klienta.md` | instrukcja dla osoby pobierającej (WireGuard, przeglądarka, FTPS) |
| `docs/testy.md`, `docs/obszary/straznicy.md` | `sprawdz.bat`, strażnicy, mutacje blokad |
| `docs/wydania.md`, `zmiany.md`, `do-zrobienia.md` | wersje, budowanie, paczka, git; co wnosi wersja; otwarte punkty |
| `docs/obszary/` | po jednym dokumencie na obszar (pakiet) |

## Struktura (import tylko w dół)

`wewn/konfiguracja` ← `wewn/rdzen` ← `wewn/magazyn` ← `wewn/uslugi/*` ← `wewn/http`, `wewn/ftp`, `wewn/siec` ←
`wewn/aplikacja` ← `cmd/skrytka`. Interfejs `web/` rozmawia z serwerem tylko przez API (`docs/api.md`). Pilnuje strażnik
architektury (`docs/testy.md`).

## Zasady

1. **Wszystko globalnie:** nowa liczba, limit, port, czas, nazwa pliku → stała w `wewn/konfiguracja/<obszar>.go`
   (z komentarzem, co znaczy). Ścieżki tylko z `wewn/konfiguracja/sciezki.go` (względem folderu programu).
2. **`rdzen` czysty:** bez dysku, sieci i `time.Now` (czas jako argument). Dysk tylko w `magazyn`, każda ścieżka przez
   `rdzen.SciezkaNaDysku` (zakaz wyjścia poza udział: `..`, dyski, UNC, strumienie NTFS).
3. **Bramka uprawnień:** każda operacja na plikach (HTTP, FTPS, link publiczny) i każde logowanie przez
   `rdzen.CzyWolno` / `konta.Zaloguj`. Widok nie ma własnych reguł; nowa operacja = `Operacja` w rdzeniu + reguła +
   test tabelaryczny + wpis w mutacjach blokad.
4. **Widoki tylko tłumaczą:** `wewn/http` i `wewn/ftp` zamieniają protokół na wywołania usług i błędy rdzenia na kody
   (tabela w `docs/api.md`). Logika w usługach, z testami bez sieci.
5. **Wygląd tylko z tokenów** `web/src/styl/tokeny.css` (nazwy z motywu stacji) i ikon SVG z `web/src/ikony/ikony.ts`;
   zero `#rrggbb`, `rgb(`, `px` w komponentach, **bez emoji** w napisach. Oba motywy (ciemny, jasny) muszą działać.
   Responsywność bez przeskakiwania: pasek boczny zwęża się do ikon, tabela chowa kolumny wg progów z tokenów, nic nie
   znika bez zamiennika.
6. **Limity:** plik ≤ 600 linii, funkcja ≤ 80 linii, złożoność ≤ 15, ≤ 8 argumentów; bez `fmt.Print*`, bez
   zignorowanego błędu (wyjątek `// celowo: <powód>`). Za duże = dzielić, sufitów się nie dodaje.
7. **Dziennik:** nowa operacja widoczna dla właściciela (pobranie, wgranie, zmiana, logowanie, administracja) = jeden
   wpis `dziennik.Wpis` z wynikiem i czasem; nigdy hasło, token ani ciasteczko.
8. **Bezpieczeństwo w trybie publicznym:** tylko HTTPS i FTPS (bez zwykłego FTP i HTTP poza przekierowaniem), blokada po
   złych hasłach, CSRF przez nagłówek `X-Skrytka: 1`, ciasteczko `HttpOnly` + `SameSite=Strict` + `Secure`.

## Typowe zadania

- **Nowy adres API:** usługa + test w `wewn/uslugi/<obszar>` → obsługa w `wewn/http` + test (`httptest`) → wiersz
  w `docs/api.md` → typ w `web/src/api/typy.ts` → strona w `web/`.
- **Nowa strona interfejsu:** `web/src/strony/<Nazwa>.svelte` (tylko wspólne komponenty z `docs/interfejs.md`, tokeny,
  ikony) → klucz w nawigacji → opis w `docs/interfejs.md` → zrzut w obu motywach.
- **Nowa stała:** `wewn/konfiguracja/<obszar>.go` z komentarzem; liczba w kodzie obszaru = błąd strażnika.
- **Nowa ikona:** ścieżka SVG 24×24 w `web/src/ikony/ikony.ts`, ten sam styl linii co reszta.

## Sprawdzenie przed commitem

1. **Po KAŻDEJ zmianie kodu zapytaj oknem (`AskUserQuestion`)**: testy **częściowe** albo **pełne** (ew. bez); przed
   odpowiedzią żadnych testów. Nigdy obu po kolei i nigdy samemu dorzucać pełnych.
   - Częściowe = tylko zmienione pakiety:
     `set GOTOOLCHAIN=local` i `_go-portable\bin\go.exe test ./wewn/<pakiet>/...`; interfejs: `npm run build` i
     `npm run sprawdz` w `web\`.
   - Pełne = `sprawdz.bat` → na końcu `WYNIK: WSZYSTKO OK`. `sprawdz.bat szybko` = tylko strażnicy.
   - **Przed każdym pushem zapytaj oknem: częściowe czy pełne.** W raporcie napisz, które testy szły, a których nie.
2. Limity jakości spełnione (strażnik jakości, `docs/testy.md`).
3. Zmieniona strona interfejsu: zrzut w obu motywach.
4. Podbij `Wersja` w `wewn/konfiguracja/wersja.go`, dopisz wpis w `zmiany.md`, podaj nową wersję w raporcie.
5. Dokumenty z tabeli niżej poprawione w tej samej zmianie.

## Testy na poziomie profesjonalnym

1. **Zmiana logiki = testy w tej samej zmianie.** Stare testy dopasowane do nowego działania, nowe działanie ma test.
   Świadomy wyjątek tylko jawnie w opisie commitu: `[bez testów: <powód>]`.
2. **Test musi paść bez zmiany.** Sprawdź wstrzyknięciem: tymczasowo cofnij poprawkę albo zdejmij blokadę, test ma być
   czerwony, potem przywróć. Atrapa nie może omijać sprawdzanej ścieżki.
3. **Blokada bezpieczeństwa** (piaskownica ścieżek, bramka, hasła, blokada po złych hasłach, sesje, CSRF, limity
   linków, tylko odczyt) = wpis w liście mutacji blokad (fragment kodu, mutacja, testy); strażnik mutacji sprawdza, że
   jej testy padają bez niej. Opis: `docs/testy.md`, `docs/obszary/straznicy.md`.
4. **Testy tabelaryczne**, jeden test na zachowanie, bez tautologii. Dysk tylko `t.TempDir()`, zegar wstrzykiwany,
   bez `time.Sleep`, sieć tylko `httptest` i `127.0.0.1` na porcie losowym.
5. Znany błąd bez poprawki = test poprawnego działania z `t.Skip("do-zrobienia: …")` i punkt w `do-zrobienia.md`.

## Dokumenty przy każdej zmianie

Każda zmiana działania, stałych, plików albo przepływu pracy aktualizuje **w tej samej zmianie** pliki `.md` i `.txt`,
których dotyczy, bez osobnego przypominania:

| co się zmieniło | co zaktualizować |
|---|---|
| nowa wersja (każde podbicie `Wersja`) | `zmiany.md`: wpis na górze (data; **Nowe**, **Poprawione**, **Usunięte**, tylko potrzebne; co widzi użytkownik albo programista, bez listy plików) |
| zadanie zrobione albo nowe, punkt do sprawdzenia na żywo, decyzja | `do-zrobienia.md`: zrobione usuń, nowe dopisz do grupy |
| adres API, pole JSON, kod błędu | `docs/api.md`, `web/src/api/typy.ts` |
| strona, komponent, token, ikona | `docs/interfejs.md`, opis strony w `docs/obszary/` |
| pakiet, warstwa, przepływ, blokada bezpieczeństwa | `docs/architektura.md` i `docs/obszary/<obszar>.md` |
| klucz `dane/konfiguracja.json`, port, tryb sieci | `docs/architektura.md`, `INSTALACJA.txt`, przy portach i FTPS też `docs/dla_klienta.md` |
| to, co widzi osoba pobierająca (logowanie, menu „Pobierz przez…”, FTPS, certyfikat) | `docs/dla_klienta.md` |
| testy, strażnicy, `sprawdz.bat` | `docs/testy.md`, `docs/obszary/straznicy.md` |
| budowanie, paczka, git | `docs/wydania.md`, `README.md`, `INSTALACJA.txt` |
| nowy plik stanu albo katalog | `.gitignore` (czy nie trafi do publicznego repo), `README.md` (katalogi) |
| zasada pracy | `CLAUDE.md` i ten skill |

Zmiana działania = przeczytaj w dokumencie cały opis tego przebiegu i popraw zdania, które przestały być prawdą.
Usunięty lub przeniesiony plik: popraw wszystkie odwołania (grep po nazwie), mapy dokumentacji w `CLAUDE.md`, `README.md`
i tym pliku trzymaj zgodne. W raporcie napisz, które dokumenty zmieniono, a gdy żaden: dlaczego.

## Praca wieloagentowa

1. Każdy agent dostaje listę SWOICH plików (rozłączne z innymi); zasady wspólne: `docs/zasady_agentow.md`.
2. Agent poprawia dokumenty swojej zmiany (`docs/obszary/<obszar>.md`) albo zwraca dokładną listę zdań do zmiany; sesja
   główna przed commitem przechodzi tę listę i odhacza ją w raporcie. Dokument wspólny edytuje jeden wykonawca.
3. Agenci nie używają gita i nie zmieniają `go.mod`. **Commit i push robi sesja główna** po przeglądzie.
4. Agenci puszczają tylko testy częściowe swojego pakietu, szeregowo; pełne `sprawdz.bat` puszcza sesja główna.

## Git i bezpieczeństwo

- Repo **https://github.com/mat-zielinski/klient-server-ftp**, gałąź `main`. **Repo jest PUBLICZNE.**
- Nigdy do repo: haseł, hashy, tokenów, kluczy, certyfikatów (`dane/certyfikaty/`), `dane/konta.json`,
  `dane/udostepnienia.json`, `dane/konfiguracja.json`, logów (`logi/`), konfiguracji WireGuard z kluczami, kopii
  (`kopie/`), plików `.exe`, udostępnianych plików. Przed commitem `git status` i przegląd listy plików; nowy plik stanu
  = najpierw wpis w `.gitignore`. W dokumentach i testach tylko przykładowe adresy i dane.
- Commit: `Skrytka X.Y: <co>`, treść po polsku, stopka współautora. Bez force-push i `--no-verify`.
- Serwera nie uruchamiaj na porcie publicznym, Zapory Windows nie zmieniaj, niczego nie pobieraj z internetu bez
  polecenia użytkownika (`go mod download` z `go.sum` jest dozwolone).
