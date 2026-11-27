# Obszar „ci” (`.github/`)

## Co robi

Automatyczne sprawdzanie i wydania w GitHub Actions (repozytorium publiczne
`github.com/mat-zielinski/klient-server-ftp`, gałąź `main`) oraz cotygodniowe aktualizacje zależności (Dependabot).
Część „Jakość i wydania” z [PLAN.md](../../PLAN.md), sekcja H.

## Pliki

| plik | zawartość |
|---|---|
| `.github/workflows/ci.yml` | budowa i testy przy pushu i pull requeście do `main`, na `windows-latest` i `ubuntu-latest` |
| `.github/workflows/wydanie.yml` | wydanie przy pushu tagu `v*`: binarki, paczki, sumy, GitHub Releases |
| `.github/dependabot.yml` | aktualizacje Go (`go.mod`), npm (`web/`) i akcji GitHub co tydzień |

## CI (`ci.yml`)

Kroki na obu systemach (macierz bez `fail-fast`, czyli jeden system nie przerywa drugiego):

1. `actions/setup-go` z wersją z `go.mod` i pamięcią podręczną (`go.sum`); `GOTOOLCHAIN=local`, `CGO_ENABLED=0`.
2. `actions/setup-node` (Node LTS) z pamięcią podręczną npm (`web/package-lock.json`).
3. W `web/`: `npm ci`, `npm run build`, `npm run straznik` (strażnik stylu). Budowa idzie przed Go, bo `go:embed`
   bierze `web/dist`.
4. `go vet ./...` na bieżącym systemie oraz z `GOOS=linux` i `GOOS=windows` (pliki `*_linux.go` i `*_windows.go`
   sprawdzane na obu runnerach).
5. `go test ./...`.
6. `go run ./narzedzia_dev/straznik` (architektura, jakość, testy, styl; potrzebuje `node` w PATH).
7. Budowa binarki na swoim systemie: `wyjscie/skrytka.exe` (Windows) albo `wyjscie/skrytka` (Linux).

Uprawnienia: tylko `contents: read`. Nowy push na tę samą gałąź anuluje stary przebieg (`concurrency`).

## Wydanie (`wydanie.yml`)

Uruchamia je push tagu `v*`, np. `git tag v0.2 && git push origin v0.2` (robi sesja główna). Tag z myślnikiem
(`v0.2-rc1`) daje wydanie oznaczone jako wstępne.

1. Zadanie `interfejs`: `npm ci`, `npm run build`, `npm run straznik`, wynik `web/dist` przekazany dalej (raz).
2. Zadanie `binarki` (macierz, wszystko na Linux, kompilacja krzyżowa, `CGO_ENABLED=0`):
   `-trimpath -ldflags "-s -w -X skrytka/wewn/konfiguracja.WersjaBudowy=<tag>"`.

| cel | pliki w paczce | paczka |
|---|---|---|
| `windows-amd64` | `skrytka.exe` (`-H windowsgui`, bez okna konsoli), `skrytka_konsola.exe` (z konsolą) | `skrytka-<tag>-windows-amd64.zip` |
| `linux-amd64` | `skrytka`, `instaluj-linux.sh` | `skrytka-<tag>-linux-amd64.tar.gz` |
| `linux-arm64` (Raspberry Pi) | `skrytka`, `instaluj-linux.sh` | `skrytka-<tag>-linux-arm64.tar.gz` |

   Każda paczka ma też `README.md`, `INSTALACJA.txt` i `dla_klienta.md`, w katalogu `skrytka-<tag>-<cel>/`.
   Brak `skrypty/instaluj-linux.sh` przerywa wydanie (paczka Linux byłaby niepełna).
3. Zadanie `wydanie`: `SHA256SUMS.txt` (wszystkie paczki), `gh release create` z `GITHUB_TOKEN` i notatkami
   wygenerowanymi z commitów. Tylko to zadanie ma `contents: write`.

## Dependabot

Raz w tygodniu: `gomod` (korzeń), `npm` (`/web`), `github-actions`. Aktualizacje drobne (minor, patch) zebrane
w jeden pull request na ekosystem; akcje GitHub w jednej grupie. Do 5 otwartych pull requestów na ekosystem.

## Zasady

1. Akcje przypięte do wersji głównych (`actions/checkout@v5`, `actions/setup-go@v6`, `actions/setup-node@v6`,
   `actions/upload-artifact@v5`, `actions/download-artifact@v6`); podbija je Dependabot. Bez akcji firm trzecich
   (wydanie przez `gh`, które jest na runnerze).
2. Nazwa tagu idzie do skryptów przez `env`, nie wprost przez `${{ }}` w treści polecenia.
3. Żadnych sekretów poza `GITHUB_TOKEN`.

## Czego nie zrobiono

1. Mutacje blokad (`narzedzia_dev/mutacje`) i `npm run sprawdz` (svelte-check) nie są w CI (nie było w zadaniu;
   do dodania, gdy będą stabilne i zmierzymy czas).
2. Wydanie nie powtarza testów: zakłada, że tag stawiamy na commicie z zielonym CI.
3. Pliki nie były uruchomione w GitHub Actions ani sprawdzone parserem YAML (brak parsera na tym komputerze,
   przegląd ręczny). Pierwszy push pokaże wynik.
4. `-X ...WersjaBudowy` nic nie robi, dopóki w `wewn/konfiguracja` nie ma `var WersjaBudowy string` (linker Go
   pomija nieistniejącą zmienną bez błędu).
