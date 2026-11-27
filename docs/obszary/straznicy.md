# Obszar: strażnicy (narzedzia_dev, sprawdz, haki git)

## Co robi

Pilnuje zasad z `docs/zasady_agentow.md` i `PLAN.md` (sekcja C) automatycznie: warstwy, przenośność Windows / Linux,
limity jakości, testy, styl interfejsu i to, że testy blokad bezpieczeństwa naprawdę padają bez blokady.
Szczegóły reguł i użycie: `narzedzia_dev/README.md`.

## Pliki

| Plik | Rola |
|---|---|
| `narzedzia_dev/straznik/*.go` | Strażnik (package main): `projekt.go` (wczytanie i parsowanie), `architektura.go`, `system.go`, `jakosc.go`, `testy.go`, `styl.go`, `wyjatki.go`, `limity.go` (limity, warstwy), `main.go`. |
| `narzedzia_dev/straznik/*_test.go` | Testy tabelaryczne na przykładowych źródłach w `t.TempDir()`. |
| `narzedzia_dev/wyjatki.json` | Wyjątki strażnika (pusto). |
| `narzedzia_dev/mutacje/*.go` | Strażnik mutacji blokad + testy (udawany `go test`). |
| `narzedzia_dev/mutacje_blokad.json` | 11 mutacji: 8 blokad rdzenia (ścieżki: `..`, dysk, druga linia obrony, względny korzeń; bramka `CzyWolno`: konto wyłączone, admin a tylko do odczytu, zapis wymaga `PrawoZapis`, `ErrTylkoOdczyt`) i 3 reguły samego strażnika. |
| `sprawdz.bat`, `sprawdz.sh` | Komplet kontroli (`szybko` = vet + strażnik). |
| `.githooks/pre-commit`, `commit-msg`, `pre-push`, `wspolne.sh` | Haki git (sh). |
| `skrypty/wlacz_haki_git.bat`, `skrypty/wlacz_haki_git.sh` | `git config core.hooksPath .githooks`. |

## Warstwy (decyzja sesji głównej)

`konfiguracja` ← `rdzen` ← `magazyn` ← `uslugi/*` ← `siec` ← `http`, `ftp` ← `aplikacja` ← `cmd`.

- `siec` leży POD `http` i `ftp`: oba mogą importować `siec`; `siec` nie importuje `http` / `ftp`.
- `http` i `ftp` są na jednym poziomie: nie importują się nawzajem.
- `uslugi/*` (także `monitor`, `indeks`, `miniatury`, `sumy`) nie importują się nawzajem; mogą importować `magazyn`,
  `rdzen`, `konfiguracja`.
- `web` (osadzony interfejs) jest liściem na poziomie `konfiguracja`.
- `rdzen` bez `os`, `net`, `syscall`, `io/ioutil` i bez wywołania `time.Now()`.

Kolejność jest w `narzedzia_dev/straznik/limity.go` (`Warstwy`); nowa warstwa = wpis tam + test w `architektura_test.go`.

## Windows i Linux

- `go vet` w `sprawdz` idzie dla `GOOS=windows` i `GOOS=linux`.
- Reguła `architektura.system`: `golang.org/x/sys/windows` i elementy `syscall` tylko z Windows wyłącznie w
  `*_windows.go` albo pod `//go:build windows`; `golang.org/x/sys/unix` tylko w plikach nie budowanych na Windows.
- Haki git i `sprawdz.sh` to sh (Linux i Git Bash na Windows).

## Stałe

`narzedzia_dev/straznik/limity.go`: `PlikMaxLinii` 600, `FunkcjaMaxLinii` 80, `ZlozonoscMax` 15, `ArgumentyMax` 8,
`Warstwy`, `ZakazaneWRdzeniu`, `KatalogiPomijane`. `narzedzia_dev/straznik/system.go`: `PakietyWindows`,
`PakietyUnix`, `NazwySyscallWindows`, `PrefiksySyscallWindows`. Narzędzia nie należą do programu, więc ich stałe
nie są w `wewn/konfiguracja`.

## Testy

`go test ./narzedzia_dev/...`: architektura (warstwy, siec pod http/ftp, uslugi, rdzen), system (x/sys/windows,
x/sys/unix, syscall, nazwy plików, `//go:build`), jakość (granice limitów, wypisywanie, ciche błędy), testy (brak
testów, Sleep, Skip), wyjątki, styl web (udawany node), kody wyjścia; mutacje (przebieg czerwony / zielony / nie
buduje / błąd, przywrócenie po panice, z kopii po przerwaniu, brak nadpisania cudzej edycji, lista nieaktualna).
Sam strażnik ma wpisy w `mutacje_blokad.json` (test pada bez reguły).

## Czego nie zrobiono

- `_ = f()` jest wykrywane, ale nie wywołanie z pominiętym wynikiem błędu (`f()` jako instrukcja): to wymaga typów
  (`go/types`); do tego `errcheck` / `golangci-lint` (PLAN.md C3), którego nie ma w projekcie.
- Lista elementów `syscall` tylko z Windows jest ręczna; resztę łapie `go vet` dla obu systemów.
- `panic` tylko w `init` / budowie (zasada 6) nie jest sprawdzane.
- Haki nie były uruchomione przez git (brak repo); sprawdzono składnię (`sh -n`) i ich wzorce na przykładach.
- Mutacje działają na folderze projektu (brak repo, więc bez kopii roboczej `git worktree` jak w stacji).
