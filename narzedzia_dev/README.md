# narzedzia_dev: narzędzia deweloperskie Skrytki

Programy pilnujące jakości kodu (nie wchodzą do działania serwera). Napisane w Go, tylko biblioteka standardowa
(`go/ast`, `go/parser`, `go/build/constraint`). Wzór: `Programator_Czarek/narzedzia_dev` (strażnicy w Pythonie).

## Jedno polecenie: `sprawdz.bat` (Windows) / `sprawdz.sh` (Linux, też Git Bash)

```bat
sprawdz.bat           :: go vet (Windows i Linux) + strażnik + go test ./... + npm run build (web) + mutacje blokad
sprawdz.bat szybko    :: tylko go vet (Windows i Linux) + strażnik (kroki 1-2)
```

Na końcu `WYNIK: WSZYSTKO OK` albo `WYNIK: NIEUDANE kroki N: <lista>`. Kod wyjścia 0 = OK, 1 = któryś krok nieudany,
9009 (bat) / 127 (sh) = brak Go. Go: `_go-portable\bin\go.exe`, a bez niego `go` z PATH; zawsze `GOTOOLCHAIN=local`.
`go vet` idzie dwa razy (`GOOS=windows` i `GOOS=linux`), bo program działa na obu systemach.

## Zawartość

| Plik | Do czego |
|---|---|
| `straznik/` | Strażnik architektury, systemu, jakości, testów i stylu web (`go run ./narzedzia_dev/straznik`). |
| `wyjatki.json` | Wyjątki strażnika; każdy z uzasadnieniem. Na start pusto; wyjątek nieużywany = błąd. |
| `mutacje/` | Strażnik mutacji blokad (`go run ./narzedzia_dev/mutacje`). |
| `mutacje_blokad.json` | Lista blokad bezpieczeństwa i ich mutacji. |

## Strażnik (`straznik/`)

```
go run ./narzedzia_dev/straznik [-katalog KATALOG] [-tylko architektura,jakosc,testy,styl]
```

Kod wyjścia: 0 = OK, 1 = naruszenia, 2 = błąd uruchomienia (np. błąd składni). Wyjście: `plik:linia: [reguła] opis`.
Pomija katalogi zaczynające się od `.` i `_` oraz `node_modules`, `testdata`, `vendor`, `dist`. Limity i listy:
`straznik/limity.go`, `straznik/system.go`.

| Reguła | Co sprawdza |
|---|---|
| `architektura.warstwa` | Import tylko w dół: `konfiguracja`, `web` (0) ← `rdzen` (1) ← `magazyn` (2) ← `uslugi/*` (3) ← `siec` (4) ← `http`, `ftp` (5) ← `aplikacja` (6) ← `cmd/*` (7). Ta sama warstwa (np. `http` → `ftp`) też zabroniona; podpakiety jednego komponentu (`http/api` → `http`) wolno. Import `narzedzia_dev` z programu zabroniony. |
| `architektura.uslugi` | Pakiety `uslugi/*` nie importują się nawzajem (składanie w `aplikacja`). |
| `architektura.nieznany_pakiet` | Pakiet w `wewn/` spoza listy warstw (dopisz do `Warstwy` w `limity.go`). |
| `architektura.rdzen_import` | `rdzen` (poza testami) bez `os`, `os/*`, `net`, `net/*`, `io/ioutil`, `syscall`. |
| `architektura.rdzen_czas` | `rdzen` (poza testami) nie woła `time.Now()`; czas jako argument. |
| `architektura.system` | `golang.org/x/sys/windows` i elementy `syscall` tylko z Windows (np. `UTF16PtrFromString`, `FILE_ATTRIBUTE_*`) tylko w plikach `*_windows.go` albo z `//go:build windows`; `golang.org/x/sys/unix` tylko w plikach nie budowanych na Windows. Resztę różnic łapie `go vet` z `GOOS=linux` / `GOOS=windows`. |
| `jakosc.rozmiar_pliku` | Plik `.go` (także test) ≤ 600 linii. |
| `jakosc.dlugosc_funkcji` | Funkcja poza testami ≤ 80 linii (od `func` do `}`). |
| `jakosc.zlozonosc` | Złożoność cyklomatyczna ≤ 15 (jak gocyclo: `if`, `for`, `range`, `case`, `&&`, `||`; funkcje anonimowe wliczone). |
| `jakosc.argumenty` | ≤ 8 parametrów (odbiorca metody się nie liczy). |
| `jakosc.wypisywanie` | Poza `cmd/` zakaz `fmt.Print*`, `print`, `println`, `log.Print*`, `log.Fatal*`, `log.Panic*` (komunikaty przez dziennik). `fmt.Fprint*` do podanego `io.Writer` wolno. |
| `jakosc.zignorowany_blad` | `_ = f()` (i `_, _ = f()`) bez komentarza `// celowo: <powód>` w tej samej albo poprzedniej linii. |
| `jakosc.pusty_blok_bledu` | Pusty `if x != nil {}` bez komentarza `// celowo: <powód>` w środku. |
| `testy.brak_testow` | Pakiet w `wewn/**` z logiką (funkcja z ciałem poza `kontrakt.go`) ma pliki `_test.go`. |
| `testy.sleep` | W testach zakaz `time.Sleep` (zegar wstrzykiwany albo czekanie na warunek). |
| `testy.skip_bez_powodu` | `t.Skip` / `t.Skipf` z niepustym powodem; `t.SkipNow()` zabronione. |
| `styl.web` | Uruchamia `node web/sprawdz_styl.mjs`, jeśli plik istnieje; kod różny od 0 = naruszenie (z wyjściem skryptu). |
| `wyjatki.zly_wpis`, `wyjatki.nieuzywany` | Wpis w `wyjatki.json` bez reguły, pliku albo uzasadnienia; wyjątek, który już niczego nie obejmuje. |

Wyjątek (`wyjatki.json`): `{"regula": "...", "plik": "wewn/x/y.go", "nazwa": "Typ.Metoda" (opcjonalnie), "uzasadnienie": "..."}`.
Bez `nazwa` obejmuje wszystkie naruszenia tej reguły w pliku. Wyjątki są ostatecznością: najpierw popraw kod.

## Mutacje blokad (`mutacje/`)

```
go run ./narzedzia_dev/mutacje [-katalog KATALOG] [-id ID] [-lista]
```

Dla każdego wpisu `mutacje_blokad.json` (`id`, `opis`, `plik`, `szukaj`, `zamien`, `pakiet`, opcjonalnie `testy`
= wzorzec `-run`): sprawdza aktualność (fragment `szukaj` dokładnie raz, pakiet istnieje), uruchamia testy BEZ mutacji
(muszą przejść), potem podmienia fragment, uruchamia `go test -count=1 [-run testy] <pakiet>` i wymaga czerwonego
wyniku. Kod, który po mutacji się nie buduje, to błąd wpisu (nie liczy się jako czerwony).

Plik wraca zawsze: po zwykłym końcu, po błędzie i panice (`defer`), po Ctrl+C (obsługa sygnału), a po twardym
przerwaniu przy następnym uruchomieniu z kopii `*.mutacja_kopia`. Jeśli ktoś zmienił plik w trakcie mutacji, strażnik
go nie nadpisuje: oryginał zostaje w `*.mutacja_konflikt`, a przebieg kończy się błędem.

Kod wyjścia: 0 = każda mutacja wywraca swoje testy, 1 = fałszywie zielone (test nie pilnuje blokady), 2 = błąd
uruchomienia albo lista nieaktualna. Wpis z polem `do_uzupelnienia` jest pomijany z ostrzeżeniem.

Zasada: nowa blokada bezpieczeństwa = wpis w `mutacje_blokad.json` + test, który bez niej pada.

## Haki git (`.githooks/`, skrypty sh: Linux i Git Bash na Windows)

Włączenie raz na komputer: `skrypty\wlacz_haki_git.bat` (Windows) albo `skrypty/wlacz_haki_git.sh` (Linux).

| Hak | Co robi |
|---|---|
| `pre-commit` | Blokuje dane tego komputera w commicie (`dane/konta.json`, `dane/udostepnienia.json`, `dane/konfiguracja.json`, `dane/certyfikaty/*`, `logi/*`, `kopie/*`, `*.conf`, kopie strażnika mutacji); potem `sprawdz.sh szybko`. |
| `commit-msg` | Zmiana `*.go` (poza `*_test.go` i `wewn/konfiguracja/wersja.go`) bez zmiany żadnego `*_test.go` = blokada, chyba że treść ma `[bez testów: <powód>]`. |
| `pre-push` | Te same dane komputera w indeksie i w wysyłanych commitach = blokada; niezacommitowane `*.go` = blokada; `sprawdz.sh szybko`; testy zmienionych pakietów (nowa gałąź albo zmiana `go.mod` = `go test ./...`). |

Wspólne funkcje haków: `.githooks/wspolne.sh`. Obejście `--no-verify` jest zakazane.
