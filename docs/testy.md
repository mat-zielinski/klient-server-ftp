# Testy i strażnicy

Zasady: [CLAUDE.md](../CLAUDE.md) i skill `skrytka` (sekcja „Testy na poziomie profesjonalnym”). Szczegóły każdego
strażnika (reguły, wyjątki, kody wyjścia) opisuje jego autor w [obszary/straznicy.md](obszary/straznicy.md); ten
dokument to przegląd.

## Kiedy jakie testy

Po każdej zmianie kodu pytanie oknem (`AskUserQuestion`): **częściowe** czy **pełne**. Nigdy obu po kolei i nigdy
pełnych bez zgody. Przed każdym pushem to samo pytanie.

| rodzaj | polecenie (z folderu projektu) | czas |
|---|---|---|
| częściowe, Go | `set GOTOOLCHAIN=local` i `_go-portable\bin\go.exe test ./wewn/<pakiet>/...` (tylko zmienione pakiety) | ok. 1 min |
| częściowe, interfejs | `npm run build` i `npm run sprawdz` w `web\` | ok. 1 min |
| pełne | `sprawdz.bat` | do zmierzenia |
| tylko strażnicy | `sprawdz.bat szybko` | kilka sekund |

Na końcu `sprawdz.bat` wypisuje `WYNIK: WSZYSTKO OK` albo listę kroków z błędem (kod wyjścia różny od 0).

## Co robi `sprawdz.bat`

Według [PLAN.md](../PLAN.md) (sekcja C, punkty 2–5 i 8):

1. **Strażnik architektury** (odpowiednik `sprawdz_architekture.py` stacji, w Go): import tylko w dół warstw
   `konfiguracja ← rdzen ← magazyn ← uslugi ← http, ftp, siec ← aplikacja ← cmd`; `rdzen` bez `os`, `net`
   i `time.Now`; dysk tylko w `magazyn` (i plikach stanu przez `konfiguracja.ZapiszAtomowo`).
2. **Strażnik jakości** (`golangci-lint` + własny): plik ≤ 600 linii, funkcja ≤ 80 linii, złożoność ≤ 15,
   ≤ 8 argumentów; bez `fmt.Print*`; bez zignorowanego błędu (wyjątek komentarzem `// celowo: <powód>`); `panic` tylko
   przy budowie; liczby i ścieżki na sztywno poza `wewn/konfiguracja` zgłaszane.
3. **Strażnik interfejsu**: w `web/src` poza `styl/tokeny.css` zakazane `#rrggbb`, `rgb(`, `px` (wyjątek `0`), emoji
   w napisach i ikony spoza `ikony/ikony.ts`.
4. **Strażnik testów**: test coś sprawdza, bez `time.Sleep`, bez duplikatów, `t.Skip` tylko z powodem
   `do-zrobienia: …`; każdy pakiet z logiką ma pliki `_test.go`.
5. `go vet ./...` i **`go test ./...`** (testy tabelaryczne, dysk `t.TempDir()`, sieć `httptest` i `127.0.0.1`).
6. **Mutacje blokad**: dla każdej blokady bezpieczeństwa z listy (fragment kodu, mutacja, testy) strażnik zdejmuje ją
   na kopii, puszcza wskazane testy i wymaga, żeby padły. Lista blokad i jej plik: [obszary/straznicy.md](obszary/straznicy.md).
7. **Budowa interfejsu**: `npm run build` i `npm run sprawdz` w `web\`.

`sprawdz.bat szybko` = punkty 1–4 bez testów.

## Pisanie testów

1. Zmiana logiki = testy w tej samej zmianie; test **pada bez zmiany** (sprawdzone wstrzyknięciem: cofnij poprawkę,
   test czerwony, przywróć).
2. Testy tabelaryczne (`[]struct{nazwa …}` + `t.Run`), jeden test na zachowanie, bez tautologii.
3. Dysk tylko `t.TempDir()`, katalog programu przez zmienną `SKRYTKA_KATALOG`; zegar wstrzykiwany, bez `time.Sleep`;
   sieć tylko `httptest` albo `127.0.0.1` na porcie losowym (`:0`).
4. Obowiązkowe testy bezpieczeństwa: ścieżki (`..`, `C:`, UNC, `:`), nazwy zarezerwowane, bramka (każda operacja ×
   każde prawo × tylko odczyt), hasła i blokada po złych próbach, sesje (wygasanie, wyłączone konto), CSRF, linki
   (wygasanie, limit, hasło).
5. Znany błąd bez poprawki: test poprawnego działania z `t.Skip("do-zrobienia: …")` i punkt w
   [do-zrobienia.md](../do-zrobienia.md).

## Stan

Wersja 0.1: `sprawdz.bat` i strażnicy powstają w sprincie 0 (agent `straznicy`); do czasu ich powstania testy
częściowe idą ręcznie poleceniami z tabeli wyżej. Czas pełnego przebiegu do zmierzenia po sprincie 1.
