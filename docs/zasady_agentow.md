# Zasady dla agentów (wspólne dla wszystkich)

Projekt: **Skrytka**, serwer plików na Windows (Go 1.27 + Svelte 5), wzór działania FileBrowser Quantum, wygląd jak
stacja Programator_Czarek + UniFi. Plan: `PLAN.md`. Kontrakty: `docs/api.md`, `docs/interfejs.md`, pliki
`kontrakt.go` w każdym pakiecie `wewn/`, `web/src/api/typy.ts`. Folder projektu: `C:\Users\mateu\Desktop\FTP_server`.

## Obowiązkowo

1. **Po polsku**: nazwy (bez ogonków w identyfikatorach: `sciezka`, `uzytkownik`), komentarze, komunikaty,
   dokumenty. Komunikaty: zdania wielką literą, bez długich myślników.
2. **Ruszasz TYLKO swoje pliki** (lista w poleceniu). Sygnatur z kontraktów nie zmieniasz. Brakuje czegoś
   w kontrakcie albo w cudzym pakiecie? Nie edytuj cudzego pliku: opisz to w raporcie (sekcja „Potrzeby”).
   Plik `kontrakt.go` w SWOIM pakiecie zastępujesz implementacją (możesz go rozbić na pliki), sygnatury zostają.
3. **Agenci nie używają gita** (repo prowadzi sesja główna). Bez `go get` / zmian `go.mod` (zależności dodała sesja główna; brakuje = raport).
4. **Warstwy, import tylko w dół**: `konfiguracja` ← `rdzen` ← `magazyn` ← `uslugi/*` ← `http`, `ftp`, `siec` ←
   `aplikacja` ← `cmd`. `rdzen` bez dysku, sieci i `time.Now` (czas jako argument).
5. **Wszystko globalnie**: każda liczba, limit, nazwa pliku, port → stała w `wewn/konfiguracja/<twoj_obszar>.go`
   (własny plik, np. `konta.go`, `ftp.go`); żadnych ścieżek na sztywno (`konfiguracja/sciezki.go`).
6. **Limity jakości**: plik ≤ 600 linii, funkcja ≤ 80 linii, złożoność ≤ 15, ≤ 8 argumentów; żadnego `fmt.Print*`
   (komunikaty przez dziennik), żadnego zignorowanego błędu bez `// celowo: <powód>`; `panic` tylko w `init`/budowie.
7. **Testy w tej samej pracy**: każda funkcja z logiką ma test tabelaryczny (`_test.go` obok), testy bezpieczeństwa
   (ścieżki, uprawnienia, hasła, sesje, limity) obowiązkowe; test musi paść bez sprawdzanej blokady (sprawdź
   wstrzyknięciem: tymczasowo zdejmij blokadę, test czerwony, przywróć). Dysk w testach tylko `t.TempDir()`.
   Bez `time.Sleep` w testach (zegar wstrzykiwany).
8. Narzędzia: Go = `C:\Users\mateu\Desktop\FTP_server\_go-portable\bin\go.exe` (np. `_go-portable/bin/go.exe test ./wewn/rdzen/...`),
   ustaw `GOTOOLCHAIN=local`. Node/npm są w systemie. Testy uruchamiaj TYLKO dla swojego pakietu, szeregowo.
9. Nie uruchamiaj serwera na portach publicznych, nie zmieniaj Zapory Windows, nie pobieraj nic z internetu poza tym,
   co już jest (`go mod download` z go.sum jest OK).
10. Dokument swojego obszaru: dopisz sekcję do `docs/obszary/<twoj_obszar>.md` (własny plik): co robi, pliki,
    stałe, testy, czego nie zrobiono.

## Raport końcowy (krótko, po polsku, A/B/C/D)

A. Co zrobione (pliki). B. Testy: które, wynik (`go test` / `npm run build`). C. Czego NIE zrobiono i dlaczego,
co nie było sprawdzone. D. Potrzeby: czego brakuje od innych pakietów / w kontraktach / w `go.mod`.
