# Obszar: monitor (stan serwera na żywo)

## Co robi

Pakiet `wewn/uslugi/monitor` co `MonitorOkresProbkowania` (1 s) zbiera próbkę: CPU %, RAM użyty i razem,
odczyt i zapis dysków (B/s), sieć na interfejs (B/s, rodzaj `lan` / `wireguard` / `petla`) i sumy bez pętli,
liczbę połączeń i transferów (funkcja wstrzykiwana), gorutyny, pamięć procesu (`runtime.MemStats.Sys`).
Struktura `Probka` i jej JSON są dokładnie jak w `docs/api.md` (sekcja „Fala 2”).

- **Prędkości z różnic liczników** między kolejnymi odczytami podzielonych przez czas z zegara. Pierwsza próbka
  ma prędkości i CPU równe 0. Spadek licznika: gdy poprzednia wartość była w górnej połowie zakresu 32 bitów,
  liczymy przekręcenie licznika 32-bitowego, w przeciwnym razie reset (prędkość 0, liczenie od nowa).
  Interfejs nowy (bez poprzedniego odczytu) ma 0. CPU: (Δrazem − Δbezczynny) / Δrazem, zaokrąglone do 0,1.
- **Historia**: bufor kołowy 600 próbek (10 min co 1 s) i 1440 średnich minutowych (24 h). Średnia minuty
  powstaje, gdy przyjdzie pierwsza próbka następnej minuty (czas = początek minuty); interfejs nieobecny
  w części próbek liczy się w nich jako 0. Bieżąca, niepełna minuta nie jest w `Historia1m()`.
- **Słuchacze** (`Sluchaj`): kanał z buforem `MonitorBuforSluchacza` (4); wysyłka nigdy nie blokuje, przy pełnym
  kanale próbka dla tego słuchacza przepada. Funkcja wypisania jest idempotentna i zamyka kanał.
- **Błąd odczytu systemu**: próbka i tak powstaje (czas, połączenia, transfery, gorutyny, pamięć procesu),
  bez pól systemowych; błąd w `OstatniBlad()`.
- **Rodzaj interfejsu**: powtórzona mała funkcja klasyfikacji (pakietu `siec` nie importujemy), te same wzorce
  `WzorceWireGuardZawiera` / `WzorceWireGuardPrzedrostek` i stałe `Rodzaj*` z `konfiguracja/siec.go`.
  Pętla = flaga loopback albo nazwa `lo`. Brak rodzaju `publiczny` (monitor nie patrzy na adresy IP).

## API pakietu

| element | opis |
|---|---|
| `Nowy(opcje ...Opcja) *Monitor` | domyślnie `time.Now`, źródło systemowe, liczniki zerowe |
| `ZZegarem(func() time.Time)`, `ZZrodlem(Zrodlo)`, `ZLicznikami(func() (polaczenia, transfery int))`, `ZTykaczem(<-chan time.Time)` | opcje (testy, wstrzyknięcie liczników z `http`/`ftp` i `transfery`) |
| `(*Monitor).Uruchom(ctx)` | pierwsza próbka od razu, potem gorutyna co 1 s do końca `ctx`. **Nie blokuje**; drugie wywołanie nic nie robi |
| `Teraz() Probka`, `Historia1s() []Probka`, `Historia1m() []Probka` | kopie list, od najstarszej; nigdy `nil` |
| `Sluchaj() (<-chan Probka, func())` | dla SSE `GET /api/monitor/strumien` |
| `System() System` | `os`, `arch`, `cpu_rdzenie`, `ram_razem`, `host`, `czas_pracy_s` (od utworzenia monitora = czas pracy serwera) |
| `OstatniBlad() error` | rozszerzenie: błąd ostatniego odczytu systemu |
| `System.DyskIODostepny` (`dysk_io_dostepny`) | rozszerzenie JSON: `false` = brak liczników dysku, prędkości dysku = 0 |
| `ZajetoscDysku(sciezka) (Zajetosc, error)` | rozszerzenie: `{"razem","wolne"}` w bajtach (zajętość udziałów; Windows `GetDiskFreeSpaceEx`, Linux `Statfs`) |
| `Zrodlo`, `Odczyt`, `LicznikSieci`, `NoweZrodloSystemowe()` | źródło surowych liczników (atrapa w testach) |

## Odczyty systemowe (bez cgo)

- **Windows** (`zrodlo_windows.go`): CPU `GetSystemTimes` i RAM `GlobalMemoryStatusEx` przez
  `windows.NewLazySystemDLL("kernel32.dll")` (brak opakowań w `x/sys`); dysk: `IOCTL_DISK_PERFORMANCE` na
  `\\.\PhysicalDrive0..15` otwartych bez praw odczytu (działa bez administratora, sprawdzone); sieć:
  `net.Interfaces` (tylko w górze) + `GetIfEntry2Ex` (liczniki 64-bitowe, opis karty do rozpoznania WireGuard).
- **Linux** (`zrodlo_linux.go`): `/proc/stat` (user..steal, bezczynny = idle + iowait), `/proc/meminfo`
  (użyty = MemTotal − MemAvailable), `/proc/diskstats` (sektory po 512 B, tylko dyski z `/sys/block/<n>/device`,
  bez partycji, loop, ram, dm), `/proc/net/dev` (tylko interfejsy w górze wg `net.Interfaces`), `syscall.Statfs`.
  Parsery są w `proc.go` bez znacznika systemu (testowane na każdym systemie).
- Inne systemy (`zrodlo_inne.go`): źródło zawsze zwraca błąd; kompiluje się.

## Pliki

- `wewn/konfiguracja/monitor.go`: stałe (okres, rozmiary historii, bufor słuchacza, granice liczników, ścieżki
  `/proc` i `/sys`, `\\.\PhysicalDrive`, kod IOCTL).
- `wewn/uslugi/monitor/typy.go` (typy), `monitor.go` (Monitor, opcje, próbkowanie), `liczenie.go` (różnice
  liczników, CPU, sieć, rodzaj), `bufor.go` (bufor kołowy, sumator minutowy), `sluchacze.go`, `proc.go`,
  `zrodlo_windows.go`, `zrodlo_linux.go`, `zrodlo_inne.go`.
- Testy: `monitor_test.go` (atrapa źródła i zegar: CPU, prędkości, reset i przekręcenie liczników, dysk
  niedostępny, błąd źródła, bufory kołowe, uśrednianie minutowe, nieblokujący słuchacz, `Uruchom` z tykaczem,
  JSON), `proc_test.go` (parsery /proc), `dym_test.go` (prawdziwy system: jedna próbka, granice, zajętość).
  Test słuchacza sprawdzony wstrzyknięciem (wysyłka blokująca = test czerwony).

## Czego nie zrobiono / niesprawdzone

- Odczyty Linuksa sprawdzone tylko `go vet` z `GOOS=linux` i testami parserów; test „dym” nie był uruchomiony na
  Linuksie (brak maszyny).
- Na Windows liczniki dysku zależą od włączonych liczników wydajności (domyślnie włączone); przy braku
  `dysk_io_dostepny` = false.
- `pamiec_procesu` = pamięć pobrana od systemu przez środowisko Go (`MemStats.Sys`), nie zestaw roboczy procesu.
- Brak testu wyścigów (`-race` wymaga cgo).
