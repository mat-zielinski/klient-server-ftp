# Obszar: transfery (podgląd pobrań i wgrań na żywo)

## Co robi

Pakiet `wewn/uslugi/transfery` trzyma rejestr aktywnych przesyłów (HTTP/HTTPS i FTPS), liczy ich prędkość,
historię prędkości i pozostały czas oraz rozsyła zdarzenia `start` / `postep` / `koniec` słuchaczom (SSE
`GET /api/transfery/strumien`). Koniec transferu woła `naKoniec` (wpis w dzienniku) dokładnie raz.

- **Gorąca ścieżka** (`CzytajZ`, `PiszDo`, `Dodaj`): tylko `sync/atomic` (licznik bajtów, flaga przerwania),
  bez mutexa i bez alokacji (benchmark: ok. 6 ns na odczyt, 0 alokacji).
- **Jedna gorutyna próbkująca** co `TransferyOkresProbkowania` (1 s): dopisuje próbkę licznika, liczy prędkość
  jako średnią z okna `TransferyOknoPredkosciS` (3 s; na początku transferu od chwili startu), dopisuje do historii
  tempo z ostatniego okresu (najwyżej `TransferyProbkiHistorii` = 60 próbek), liczy `pozostalo_s`
  (w górę; -1 gdy rozmiar albo prędkość nieznane, 0 gdy wszystko przesłane) i wysyła `postep`,
  nie częściej niż co `TransferyOdstepPostepu` (500 ms) na transfer.
- `start` i `koniec` idą natychmiast (z `Rozpocznij` i `Zakoncz`).
- **Słuchacze**: kanał z buforem `TransferyBuforSluchacza`. Wysyłka nigdy nie blokuje. Przy pełnym kanale
  `postep` przepada, a `start`/`koniec` trafiają do kolejki zaległych tego słuchacza, którą opróżnia osobna
  gorutyna (powstaje tylko dla wolnego słuchacza), z zachowaniem kolejności. Wolny słuchacz nie blokuje więc
  rejestru ani innych, a `koniec` zawsze dociera. `Sluchaj("ala")` widzi tylko transfery `Kto == "ala"`,
  `Sluchaj("")` wszystkie. Funkcja wypisania jest idempotentna i zamyka kanał.
- `Aktywne(kto)` zwraca kopie (z kopią historii) posortowane wg startu; `PredkoscLaczna()` sumuje prędkości.
- ID transferu: 8 losowych bajtów (`crypto/rand`) w zapisie szesnastkowym.

## Rozszerzenia kontraktu (poza `kontrakt.go` z planu, sygnatury kontraktu bez zmian)

| element | opis |
|---|---|
| `(*Rejestr).Przerwij(id string) bool` | admin (`DELETE /api/transfery?id`): oznacza transfer; kolejny `Read`/`Write` z `CzytajZ`/`PiszDo` zwraca `ErrPrzerwany`. `false` = brak aktywnego transferu o tym ID. Wywołujący i tak woła `Zakoncz`; dla przerwanego wynik jest zawsze `przerwane` (niezależnie od podanego) |
| `ErrPrzerwany` | błąd odczytu/zapisu po przerwaniu („Transfer przerwany przez administratora.”) |
| `(*Transfer).Przerwany() bool` | dla kodu liczącego ręcznie przez `Dodaj` (FTPS): sprawdzaj i przerwij przesył |
| `(*Rejestr).Zamknij()` | zatrzymuje gorutynę próbkującą (wyłączanie aplikacji, testy); idempotentne |
| `NowyZOpcjami(naKoniec, ...Opcja)`, `ZZegarem(func() time.Time)`, `ZTykaczem(<-chan time.Time)` | wstrzyknięcie zegara i taktów (testy). `ZTykaczem(nil)` = bez gorutyny próbkującej |
| stałe `ZdarzenieStart/Postep/Koniec`, `WynikOk/Przerwane/Blad` | nazwy typów zdarzeń i wyników |

`Nowy(naKoniec)` = `NowyZOpcjami(naKoniec)` z `time.Now` i `time.Ticker`.

## Pliki

- `wewn/uslugi/transfery/kontrakt.go`: typy `Stan`, `Zdarzenie`, stałe.
- `wewn/uslugi/transfery/rejestr.go`: `Rejestr`, opcje, `Rozpocznij`, `Aktywne`, `PredkoscLaczna`, `Przerwij`, `Zamknij`.
- `wewn/uslugi/transfery/transfer.go`: `Transfer`, opakowania strumieni, `Dodaj`, `Zakoncz`, `ErrPrzerwany`.
- `wewn/uslugi/transfery/probkowanie.go`: gorutyna próbkująca, prędkość, historia, pozostały czas.
- `wewn/uslugi/transfery/sluchacze.go`: `Sluchaj`, rozsyłanie, kolejka zaległych.
- `wewn/uslugi/transfery/transfery_test.go`: testy i benchmarki.
- `wewn/konfiguracja/transfery.go`: stałe.

## Stałe (`wewn/konfiguracja/transfery.go`)

`TransferyOknoPredkosciS = 3`, `TransferyProbkiHistorii = 60`, `TransferyOkresProbkowania = 1 s`,
`TransferyOdstepPostepu = 500 ms`, `TransferyBuforSluchacza = 64`, `TransferyBajtyID = 8`.

## Testy

Sztuczny zegar i ręczne próbki (bez `time.Sleep`): prędkość z okna 3 s krok po kroku i pozostały czas,
historia ograniczona do 60 i oddawana jako kopia, tabele `pozostalo`/`tempo`, odstęp postępu 500 ms,
filtr użytkownika (`Sluchaj` i `Aktywne`), wolny słuchacz (rejestr się nie blokuje, postęp pomijany,
wszystkie `start`/`koniec` dochodzą w kolejności, szybki słuchacz dostaje wszystko), `Zakoncz` z 32 gorutyn
= jedno `naKoniec` i jedno `koniec`, `Przerwij` (odczyt, zapis, `io.Copy`, wynik `przerwane`, nieznane ID),
liczenie bajtów, równoległość (16 transferów, odczyt i `Dodaj` z wielu gorutyn przy ciągłym próbkowaniu),
ścieżka gorutyny próbkującej z wstrzykniętym tykaczem. Benchmarki `BenchmarkCzytajZ`
i `BenchmarkCzytajZRownolegle`.

## Czego nie zrobiono

- `-race` niedostępne (brak cgo w przenośnym Go); równoległość sprawdzona wielokrotnym uruchomieniem testów.
- Zdarzenie `suma` (co 1 s) wysyła warstwa HTTP z `PredkoscLaczna()` i `len(Aktywne(""))`.
- Wykres łączny z 5 minut trzyma interfejs (lub HTTP); rejestr nie przechowuje historii łącznej.
