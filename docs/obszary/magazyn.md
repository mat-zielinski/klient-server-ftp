# Obszar: magazyn (`wewn/magazyn/`)

Jedyny dostęp do dysku. Wszystkie warstwy wyżej (usługi, HTTP, FTPS) czytają i zapisują pliki wyłącznie przez
`*magazyn.Magazyn`.

## Co robi

- **Ścieżki**: każda ścieżka użytkownika idzie przez `rdzen.OczyscSciezke` i `rdzen.SciezkaNaDysku`, a potem przez
  `rozwiazWUdziale`, które człon po członie rozwija dowiązania symboliczne i junction NTFS (`os.Lstat` +
  `os.Readlink`). Od Go 1.23 `filepath.EvalSymlinks` nie rozwija junction na Windows, dlatego rozwijanie jest
  własne. Cel poza korzeniem udziału, pętla
  dowiązań (ponad `MagazynMaksSkokowDowiazan`) = `rdzen.ErrPozaUdzialem`. Operacje działają na ścieżce już
  rozwiniętej, więc system nie przechodzi przez żadne niesprawdzone dowiązanie. Druga linia obrony: `filepath.EvalSymlinks`
  istniejącego wyniku też musi leżeć w korzeniu (korzeń rozwinięty przez `EvalSymlinks` w `Nowy`).
- **Windows i Linux**: ten sam kod; porównanie ścieżek przez `filepath.Rel` (na Windows bez wielkości liter).
  Sprawdzane `GOOS=linux go vet ./wewn/magazyn/...`; testy junction na Linuksie są pomijane.
- **Zmiana nazwy i kosz** działają na samym dowiązaniu (rozwinięty jest tylko folder nadrzędny), nie na jego celu.
- **Ukrywanie**: `.skrytka_kosz`, `*.skrytka_czesc`, `.skrytka_tmp_*`, `desktop.ini`, `Thumbs.db`, `$RECYCLE.BIN`,
  `System Volume Information` (Windows), `lost+found`, `.Trash-*` (Linux), bez wielkości liter, nie pojawiają się w `Lista`, `Szukaj`, `ZipDo`,
  `RozmiarFolderu`, a każda ścieżka zawierająca taki człon (także po rozwinięciu dowiązania) = `ErrNieZnaleziono`.
  W szczególności do kosza nie da się wejść z zewnątrz: `/.skrytka_kosz/...` (także wielkimi literami) =
  `ErrNieZnaleziono` dla każdej operacji. Nowe nazwy przechodzą przez `rdzen.SprawdzNazwe`.
- **Lista**: `File.ReadDir(-1)` (bez sortowania, informacje o plikach z wyliczenia katalogu); dowiązania wychodzące
  poza udział albo zerwane są pomijane.
- **Zapis atomowy**: plik tymczasowy `.skrytka_tmp_*` w tym samym folderze, na końcu zamiana. Bez nadpisania
  zamiana idzie przez `os.Link` (nie nadpisze pliku, który pojawił się w międzyczasie); na FAT/exFAT sprawdzenie
  i `os.Rename` pod blokadą. Przerwany strumień nie zostawia ani celu, ani pliku tymczasowego.
- **Kawałki**: `DopiszKawalek` pisze do `<plik>.skrytka_czesc` od przesunięcia; przesunięcie mniejsze niż część
  obcina ją (ponowienie kawałka), większe = `magazyn.ErrZlePrzesuniecie`. `RozmiarKawalkow` mówi, od którego
  bajtu wznowić. `Zatwierdz` zamienia część w plik (te same zasady nadpisania co `Zapisz`).
- **Kosz**: `DoKosza` przenosi do `<udział>/.skrytka_kosz/<RRRR-MM-DD_GG-MM-SS>/<ścieżka>`; zajęte miejsce =
  znacznik `_2`, `_3` … Zegar wstrzykiwany (pole `zegar`).
- **Tylko do odczytu**: każda operacja zapisu w takim udziale = `rdzen.ErrTylkoOdczyt` przed dotknięciem dysku.
- **Szukaj**: `filepath.WalkDir`, `rdzen.Pasuje`, limit (0 = domyślny, górna granica z konfiguracji), bez wchodzenia
  w dowiązania, nieczytelne podfoldery pomijane.
- **ZIP**: strumieniowo `archive/zip` prosto do `io.Writer`; Store dla `MagazynBezKompresji` (jpg, png, mp4, mkv,
  zip, 7z, rar …), Deflate dla reszty; puste foldery zachowane; korzeń udziału pod nazwą udziału; powtórzone
  nazwy najwyższego poziomu dostają „ (2)”. ZIP64 automatycznie.
- **Wydajność i współbieżność**: bufory 1 MB z `sync.Pool`; po `Nowy` stan tylko do odczytu; zapis/kawałki/kosz
  chronione paskowanymi blokadami (`MagazynLiczbaBlokad`, klucz = ścieżka bez wielkości liter).
- Błędy systemu mapowane na `rdzen.ErrNieZnaleziono` / `ErrJuzIstnieje` / `ErrBrakPrawa` (bez ścieżek dysku).

## Pliki

| plik | zawartość |
|---|---|
| `magazyn.go` | `Magazyn`, `Nowy`, `Udzialy`, `Udzial`, rozwiązywanie ścieżek i dowiązań, ukrywanie, mapowanie błędów, pula buforów, blokady |
| `lista.go` | `Lista`, `Info`, `Otworz` |
| `zapis.go` | `Zapisz`, `DopiszKawalek`, `RozmiarKawalkow`, `Zatwierdz`, `NowyFolder`, `ZmienNazwe` |
| `kosz.go` | `DoKosza` |
| `szukanie.go` | `Szukaj`, `RozmiarFolderu` |
| `zip.go` | `ZipDo` |
| `wewn/konfiguracja/magazyn.go` | stałe `Magazyn*` (bufor, limity, przyrostki, format znacznika, listy ukrytych i bez kompresji) |

## Testy (`go test ./wewn/magazyn/...`)

- `bezpieczenstwo_test.go`: ucieczka przez junction (`mklink /J`, wszystkie operacje, zagnieżdżony, junction do
  wnętrza działa, lista/ZIP/szukanie nie pokazują sekretu), ucieczka przez symlink (bezwzględny, względny, do
  pliku, pętla, kosz na dowiązaniu; bez uprawnień = `t.Skip` z powodem), `rozwiazWUdziale`, `bezwzglednyCel`,
  tylko do odczytu (każda operacja zapisu), ukrywanie (lista, info, szukanie, ZIP, rozmiar).
- `dzialanie_test.go`: `Nowy`, `Udzialy`, lista/info/otwórz, zapis (tabela + przerwany strumień), zapis
  współbieżny (16 gorutyn na ten sam plik bez nadpisania = dokładnie 1 sukces), kawałki z wznawianiem
  i ponowieniem, nowy folder i zmiana nazwy (w tym tylko wielkość liter, folder do wnętrza), kosz z `_2`/`_3`,
  szukanie z limitem, ZIP (rozpakowanie i porównanie treści, metody Store/Deflate, pusty folder, polskie nazwy).
- Blokady sprawdzone wstrzyknięciem (wyłączenie rozwijania dowiązań, tylko-odczytu i ukrywania = testy czerwone).

## Czego nie zrobiono

- Wyścig TOCTOU: ktoś z dostępem do dysku może podmienić folder na junction między sprawdzeniem a otwarciem.
  Pełna ochrona wymagałaby otwierania przez uchwyty (`GetFinalPathNameByHandle`, `x/sys/windows`).
- Dowiązanie, którego cel zapisano przez inną nazwę tej samej lokalizacji (np. przez junction w ścieżce korzenia
  albo nazwę 8.3), jest traktowane jak wyjście poza udział (odrzucane, czyli bezpiecznie).
- Test symlinków na tej maszynie jest pomijany (brak trybu dewelopera); junction testowane w pełni.
- Kosz nie ma czyszczenia ani przywracania (brak w kontrakcie).
- `-race` niedostępny (brak cgo w przenośnym Go).
