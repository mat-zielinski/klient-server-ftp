# Obszar: indeks, miniatury, sumy (`wewn/uslugi/indeks`, `wewn/uslugi/miniatury`, `wewn/uslugi/sumy`)

Trzy usługi fali 2 (PLAN.md, sekcja H; docs/api.md, „Fala 2”). Dysk udziałów czytają wyłącznie przez
`*magazyn.Magazyn`; kod bez plików systemowych (Windows i Linux). Stałe: `wewn/konfiguracja/indeks.go`.

## Indeks (`indeks`)

- `Nowy(m, opcje...) *Indeks`, opcje: `ZOkresem`, `ZOpoznieniem`, `ZPrzerwa`, `ZRownolegloscia`, `ZZegarem`.
- `Uruchom(ctx)` **wraca od razu**: na każdy udział gorutyna, pierwsze przejście, potem co `IndeksOkresOdswiezania`
  i po `Odswiez(udzial)` (prośby łączone w jedną, zwłoka `IndeksOpoznienieOdswiezenia`). Koniec z `ctx`.
- `Odswiez(udzial)` nie blokuje (HTTP woła po zapisie, zmianie nazwy, koszu). `Przeindeksuj(ctx, udzial) error`
  robi przejście synchronicznie (testy, ewentualnie przycisk admina).
- Przejście: `magazyn.Lista` folder po folderze (bez ukrytych, bez kosza), `IndeksRownoleglosc` pracowników,
  przerwa `IndeksPrzerwaMiedzyFolderami` po każdym folderze (niski priorytet). Nowy zbiór podmieniany atomowo;
  przerwane przejście albo błąd korzenia = zostaje poprzedni. Nieczytelny podfolder jest pomijany.
  Ochrona przed pętlą dowiązań: `IndeksMaksGlebokosc`, `IndeksMaksPozycji`.
- Struktura w pamięci: lista ścieżek folderów + zwarte pozycje (nazwa, nazwa zwinięta bez ogonków i wielkości
  liter, współdzieli pamięć z nazwą, gdy identyczna; rozmiar, czas UnixNano, numer folderu, numer rodzaju).
  Posortowane wg ścieżki, więc wyniki są powtarzalne.
- `Szukaj(udzial, sciezkaStartowa, zapytanie, limit) []rdzen.Wpis`: słowa zapytania (spacje) muszą wszystkie być
  w nazwie (jak `rdzen.Pasuje`), od folderu startowego (bez niego samego), limit jak `magazyn.Szukaj`
  (0 = `MagazynDomyslnyLimitSzukania`, najwyżej `MagazynMaksLimitSzukania`). Niegotowy / nieznany udział
  albo zła ścieżka = `nil` (HTTP wtedy `magazyn.Szukaj`). Udany wynik bez trafień = pusta, nie-nil lista.
- `Gotowy(udzial) bool`, `Statystyki() []StatystykaUdzialu` (JSON jak w api.md: `nazwa, pliki, foldery, bajty,
  wg_rodzaju{rodzaj:{pliki,bajty}}, zindeksowano`; niezindeksowany = zera i zerowy czas).
- Wydajność (Ryzen 5 3600, 200 tys. pozycji): pełny przegląd bez trafień ok. 7 ms, częste słowo z limitem 0,2 ms,
  podfolder 0,6 ms (`BenchmarkSzukaj`, `TestCzasSzukania200Tys` pilnuje < 50 ms).

## Miniatury (`miniatury`)

- `Nowe(m, katalog, opcje...)` (`katalog == ""` = `konfiguracja.KatalogMiniatur()` = `dane/miniatury`),
  opcje `ZLimitemCache`, `ZLimitemZrodla`.
- `Miniatura(ctx, udzial, sciezka, "s"|"m") (plik string, err)`: s = 160 px, m = 480 px dłuższy bok, bez
  powiększania. Plik `<sha256(udział, ścieżka, rozmiar, data, wymiar)>.jpg`, JPEG jakość 80, zapis przez plik
  tymczasowy i zamianę. Trafienie odświeża czas pliku (sprzątanie usuwa najdawniej używane).
- Formaty: jpg/jpeg, png, gif, webp (`golang.org/x/image/webp`), skalowanie `draw.CatmullRom` na białym tle.
- Ochrona: rozmiar pliku ≤ `MiniaturaMaksBajtowZrodla`, `image.DecodeConfig` przed dekodowaniem i
  szerokość × wysokość ≤ `MiniaturaMaksPikseliZrodla` (bomba dekompresyjna), dekodowanie przez `io.LimitReader`,
  semafor dekodowań = liczba rdzeni / 2, jedno tworzenie danej miniatury naraz.
- Orientacja EXIF (JPEG): własny odczyt tagu 0x0112 z APP1 (II/MM), orientacje 2..8 stosowane po skalowaniu.
- Błędy: `ErrNieObraz`, `ErrNieObslugiwany` (np. bmp, heic, svg), `ErrZaDuzy`, `ErrZlyRozmiar`, `ErrUszkodzony`,
  oraz błędy magazynu (`rdzen.ErrNieZnaleziono`, `ErrPozaUdzialem` …). Proponowane mapowanie HTTP: nie obraz /
  nieobsługiwany / za duży / uszkodzony = 404, zły rozmiar = 400.
- `Sprzataj()`: gdy cache > `MiniaturaLimitCacheMB`, usuwa najstarsze (wg czasu pliku) do
  `MiniaturaCelSprzatania` % limitu; wołane w tle co `MiniaturaSprzatanieCo` nowych miniatur (aplikacja może
  dodatkowo wołać przy starcie).

## Sumy (`sumy`)

- `Nowe(m)`, `SHA256(ctx, udzial, sciezka) (string, error)`: strumieniowo (`magazyn.Otworz`), szesnastkowo małymi
  literami. Pamięć wg (udział, ścieżka) z rozmiarem i datą zmiany; zapamiętywane tylko, gdy plik nie zmienił się
  w trakcie czytania. Limit `SumyMaksWpisowCache`. Jedno liczenie danej wersji pliku naraz (czekający dostają
  ten sam wynik). Przerwanie `ctx` przerywa czytanie. Folder = `rdzen.ErrNieZnaleziono`.

## Pliki

| plik | zawartość |
|---|---|
| `indeks/indeks.go` | typ, opcje, `Uruchom`, `Odswiez`, `Przeindeksuj`, `Gotowy` |
| `indeks/przejscie.go` | równoległe przejście udziału przez magazyn |
| `indeks/zbior.go` | zwarta struktura, sortowanie, filtr folderu startowego |
| `indeks/szukanie.go` | zwijanie ogonków, słowa zapytania, `Szukaj` |
| `indeks/statystyki.go` | `StatystykaUdzialu`, `Statystyki` |
| `miniatury/miniatury.go` | typ, opcje, `Miniatura`, klucz cache, jedno tworzenie naraz |
| `miniatury/tworzenie.go` | dekodowanie z kontrolą nagłówka, skalowanie, zapis JPEG |
| `miniatury/exif.go` | odczyt orientacji EXIF, obrót i odbicia |
| `miniatury/cache.go` | `Sprzataj` |
| `sumy/sumy.go` | `SHA256`, pamięć, jedno liczenie naraz |
| `konfiguracja/indeks.go` | stałe wszystkich trzech usług, `KatalogMiniatur()` |

## Testy

- indeks: szukanie wielu słów, ogonki i wielkość liter, folder startowy (także „przedrostek nazwy to nie
  podfolder”), ukryte i kosz, złe ścieżki startowe, limit, pola `Wpis`, odświeżenie po zmianie na dysku,
  przerwane przejście zostawia stary zbiór, `Uruchom` + `Odswiez` (hak `poPrzejsciu`, bez `time.Sleep`),
  statystyki (także kopia mapy), zgodność zwijania z `rdzen.Pasuje`, benchmark 200 tys.
- miniatury: png/jpg/gif generowane w teście (wymiary i kolory), błędy (nie obraz, bmp, uszkodzony, folder,
  brak, poza udziałem, zły rozmiar), bomba (nagłówek PNG 20000 × 20000 z poprawnym CRC; wstrzyknięciem
  sprawdzone, że bez kontroli DecodeConfig test pada), limity bajtów i pikseli, trafienie cache (podmieniony plik
  zwracany bez skalowania), nowy klucz po zmianie źródła, równoczesne tworzenie, sprzątanie, EXIF II/MM,
  uszkodzone segmenty, wszystkie orientacje, zdjęcie z orientacją 6.
- sumy: zgodność z `crypto/sha256` (pusty, polskie nazwy, 3 MB losowych), wektor „abc”, błędy, trafienie
  pamięci (ta sama długość i data), zmiana daty / rozmiaru, limit pamięci, równoczesne liczenie, przerwanie ctx.

## Czego nie zrobiono

- Brak testu WebP z prawdziwym plikiem (Go nie ma kodera WebP; dekoder zarejestrowany, ścieżka wspólna z resztą).
- Indeks przechodzi cały udział przy każdym `Odswiez` (bez przyrostowej aktualizacji jednego folderu).
- Dowiązania do folderów wewnątrz udziału są indeksowane jak zwykłe foldery (duplikaty); pętle ogranicza
  głębokość i limit pozycji.
- Sumy nie mają osobnego limitu równoległości (czytanie dysku ogranicza magazyn i system).
