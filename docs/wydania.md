# Wydania, budowanie i paczka

Numer bieżący: `Wersja` w `wewn/konfiguracja/wersja.go` (stopka interfejsu, tytuł strony, `GET /api/stan`). Co wnosi
każda wersja: [zmiany.md](../zmiany.md).

## Wersje

1. Wersja podbijana przy **każdej** zmianie (0.1 → 0.2 …), w tej samej zmianie wpis w `zmiany.md`.
2. Pierwsza używalna wersja (logowanie, przeglądanie, pobieranie przez VPN) planowana po sprincie 2; pełna po
   sprincie 5 ([PLAN.md](../PLAN.md), sekcja E).

| wersja | data | stan |
|---|---|---|
| 0.1 | 2026-11-27 | sprint 0: szkielet warstw, kontrakty, konfiguracja, dokumenty; serwer jeszcze nie działa |

## Budowanie

Wymagania:

1. Go 1.27.4 przenośne w `_go-portable\` (nie jest w repo; rozpakuj tam oficjalne archiwum Go dla Windows amd64).
   Zawsze `GOTOOLCHAIN=local`, żeby Go nie pobierało innej wersji.
2. Node i npm w systemie (do budowy interfejsu).
3. Zależności Go z `go.sum` (`_go-portable\bin\go.exe mod download`), interfejsu z `web\package-lock.json`.

Polecenie: **`skrypty\buduj.bat`**. Kroki:

1. `npm ci` i `npm run build` w `web\` → `web\dist\` (wynik Vite).
2. `_go-portable\bin\go.exe build -trimpath -ldflags "-s -w" -o wyjscie\skrytka.exe .\cmd\skrytka`
   z `GOTOOLCHAIN=local`; interfejs trafia do `.exe` przez `go:embed` (`web/osadz.go`).
3. Wynik: `wyjscie\skrytka.exe` (jeden plik, bez instalacji).

`web\dist\` jest poza repo poza zaślepką `web\dist\index.html` (wyjątek w `.gitignore`): `go:embed` wymaga choć jednego
pliku, więc świeży klon buduje się w Go także bez `npm run build` (pokazuje wtedy stronę „Interfejs w budowie”).

## Paczka

Paczka dla innego komputera (planowany `skrypty\przygotuj_pakiet.bat`, sprint 5) zawiera:

1. `skrytka.exe`,
2. `INSTALACJA.txt` i `docs\dla_klienta.md`.

Paczka **nie zawiera** `dane\` (konta, hasła, linki, certyfikat, konfiguracja) ani `logi\`: na nowym komputerze kreator
zakłada je od nowa. Przenosiny tego samego serwera = kopia całego folderu programu razem z `dane\`.

## Git

1. Repozytorium: https://github.com/mat-zielinski/klient-server-ftp, gałąź `main`. **Publiczne.**
2. Commit i push robi sesja główna (agenci nie używają gita). Commit: `Skrytka X.Y: <co>`, po polsku, ze stopką
   współautora. Bez force-push i bez `--no-verify`.
3. Przed commitem: `git status`, przegląd listy plików (żadnych danych, haseł, certyfikatów, logów, konfiguracji
   WireGuard, `.exe`); co jest poza repo, ustala [.gitignore](../.gitignore).
4. Przed pushem pytanie oknem: testy częściowe czy pełne (`sprawdz.bat`).
