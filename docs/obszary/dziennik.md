# Obszar: dziennik (`wewn/uslugi/dziennik`)

## Co robi

Dziennik serwera: każdy wpis (logowanie, pobranie, wgranie, zmiana, udostępnienie, administracja, serwer) to jedna
linia JSON w `logi/dziennik_RRRR-MM.jsonl` (miesiąc wg czasu wpisu, strefa lokalna). Dostarcza odczyt z filtrem,
statystyki pobrań do strony „Dziennik pobrań” i eksport CSV.

- **Zapis** (`Zapisz`): nigdy nie blokuje. Wpis trafia do kanału buforowanego (`DziennikBufor`), jedna gorutyna
  zapisuje go do pliku przez `bufio.Writer`. Pełny kanał = wpis przepada, licznik rośnie, a gorutyna przy najbliższej
  okazji dopisuje wpis `serwer`/`blad` „Przepełnienie bufora dziennika: utracono N wpisów.”. Pusty `Czas` = zegar,
  `Opis` przycinany do `DziennikOpisMax` znaków (UTF-8 bez uszkodzeń). Błędy dysku są liczone (`BledyZapisu()`),
  serwera nie zatrzymują.
- **fsync** co `DziennikOkresSynchronizacji` (1 s, tylko gdy coś zapisano) i w `Zamknij`. `Zamknij` jest
  idempotentne; `Zapisz` po zamknięciu nic nie robi.
- **Odczyt** (`Czytaj`): najpierw bariera w kanale (wszystkie wcześniejsze `Zapisz` są widoczne), potem pliki
  miesięcy przecinających zakres, od najnowszego; wpisy w miesiącu sortowane malejąco po czasie. `Od` włącznie,
  `Do` wyłącznie, `Szukaj` bez wielkości liter w ścieżce i opisie, `Limit` 0 = `DziennikLimitMaks`.
  Uszkodzone linie (np. urwany zapis) są pomijane.
- **Pamięć podręczna**: zamknięte miesiące (starsze niż bieżący wg zegara) trzymane w pamięci, najwyżej
  `DziennikPamiecMiesiecy`; pozycja ważna, póki plik ma ten sam rozmiar i czas zmiany (dopisek z „starym” czasem ją
  unieważnia).
- **Statystyki**: tylko `RodzajPobranie` (wszystkie wyniki: bajty przerwanych też obciążyły łącze). Wykres
  w przedziałach `krok` od `od` (puste przedziały = zera), `SredniaBps` = bajty/czas pobrań, które mają `CzasMs`.
  Top `DziennikTop` plików (`udzial/sciezka`, wg liczby) i klientów (`Kto`, gdy pusty `IP`, wg bajtów), bajty wg
  protokołu (pusty = `DziennikProtokolNieznany`), sumy. Błędy: `ErrZlyZakres` (od puste albo od ≥ do),
  `ErrZlyKrok`, `ErrZaDuzoPunktow` (> `DziennikMaksPunktowWykresu`). `do` puste = teraz.
- **CSV** (`EksportCSV(w, Filtr)`): separator `;`, nagłówek jak pola JSON, czas `RRRR-MM-DD GG:MM:SS` lokalnie;
  pola zaczynające się od `= + - @ TAB CR` dostają apostrof (ochrona przed formułami w arkuszu).

## Pliki

- `kontrakt.go`: typy i stałe kontraktu (bez zmian sygnatur).
- `zapis.go`: `Dziennik`, `Otworz`, `OtworzZOpcjami(katalog, Opcje{Teraz, Bufor})`, `Zapisz`, `Zamknij`,
  `BledyZapisu`, gorutyna zapisująca.
- `odczyt.go`: `Czytaj`, wybór plików miesięcy, pamięć podręczna.
- `statystyki.go`: `Statystyki`, rankingi.
- `csv.go`: `EksportCSV`.
- `wewn/konfiguracja/dziennik.go`: wszystkie stałe obszaru.

## Testy (`go test ./wewn/uslugi/dziennik/...`)

Zapis/odczyt i ponowne otwarcie, pusty czas = zegar, przycinanie opisu; przełom miesiąca (trzy pliki, kolejność,
zakres jednego miesiąca); filtr tabelarycznie; uszkodzona linia; pamięć zamkniętego miesiąca i jej unieważnienie,
wyrzucanie najstarszej pozycji; 20 gorutyn × 100 `Zapisz`; przepełnienie bufora (gorutyna wstrzymana haczykiem,
bez `time.Sleep`); statystyki z pustym przedziałem, granice zakresu, pominięcie innych rodzajów, top 10, złe
argumenty; CSV (średnik w polu, neutralizacja formuły); `przytnij`, `kluczZNazwy`, `bezpiecznePole`.
Wstrzyknięcie sprawdzone: bez przycinania opisu i bez neutralizacji formuł testy są czerwone.

## Czego nie zrobiono

- Brak usuwania starych miesięcy (retencji), w odróżnieniu od wzoru z Programatora: nie było w zadaniu.
- Brak maskowania haseł/tokenów w treści: wołający nie może ich wpisywać (zasada kontraktu).
- `go test -race` niedostępny (go-portable bez cgo).
- Ograniczenie „zwykły użytkownik widzi tylko własne pobrania” realizuje warstwa HTTP przez `Filtr.Kto`.
