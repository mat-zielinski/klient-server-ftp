# Obszar „rdzen” (`wewn/rdzen/`)

## Co robi

Czysta logika bez dysku, sieci i `time.Now`: piaskownica ścieżek, sprawdzanie nazw, rodzaje plików, sortowanie,
wyszukiwanie, bramka uprawnień. Sygnatury opisuje `wewn/rdzen/kontrakt.go`, typy i błędy `wewn/rdzen/typy.go`.

## Pliki

| plik | zawartość |
|---|---|
| `sciezki.go` | `OczyscSciezke`, `SciezkaNaDysku` (+ `polaczZKorzeniem`, `wewnatrzKorzenia`: druga linia obrony) |
| `nazwy.go` | `SprawdzNazwe` (+ `sprawdzCzlon`, wspólne z `OczyscSciezke`) |
| `rodzaje.go` | `RodzajPliku` (mapa rozszerzeń budowana z list w konfiguracji, duplikat = panic przy starcie) |
| `sortowanie.go` | `Sortuj`, porównanie naturalne z polską kolejnością liter |
| `szukanie.go` | `Pasuje`, zwijanie ogonków |
| `bramka.go` | `CzyWolno` |
| `*_test.go` | testy tabelaryczne |

Stałe: `wewn/konfiguracja/rdzen.go` (limity długości, nazwy urządzeń Windows, znaki zakazane, klucze sortowania,
listy rozszerzeń). Katalog kosza: `konfiguracja.KatalogKosza`.

## Zasady bezpieczeństwa ścieżek (Windows przede wszystkim)

1. Jedyny separator to `/`. Każdy `\` = `ErrPozaUdzialem` (Windows traktuje go jak separator).
2. `..` w dowolnym miejscu = `ErrPozaUdzialem` (nawet gdy nie wychodzi poza korzeń, np. `/a/../b`).
3. Początek `//` (UNC, `//?/`, `//./`) = `ErrPozaUdzialem`. Człon wyglądający na dysk (`C:`, `c:plik`) = `ErrPozaUdzialem`.
4. Każdy inny `:` (strumienie NTFS `plik:strumien`, `::$DATA`) = `ErrZlaNazwa`.
5. Każdy człon przechodzi `sprawdzCzlon`: niepusty, ≤ `MaksDlugoscNazwy` bajtów, poprawny UTF-8, bez `/\:*?"<>|`,
   bez znaków sterujących (C0, DEL, C1) i znaków sterowania kierunkiem tekstu (U+202A–202E, U+2066–2069: podszywanie
   się pod rozszerzenie), bez kropki ani spacji na końcu (Windows je obcina, więc `.. ` byłoby `..`), bez nazw
   urządzeń `CON PRN AUX NUL CONIN$ CONOUT$ CLOCK$ COM0-9 LPT0-9 COM¹²³ LPT¹²³`, także z rozszerzeniem (`nul.txt`)
   i ze spacją przed kropką (`NUL .txt`), bez względu na wielkość liter.
6. Limity: ścieżka ≤ `MaksDlugoscSciezki`, ≤ `MaksGlebokoscSciezki` członów.
7. `SciezkaNaDysku` wymaga bezwzględnego korzenia, a wynik po `filepath.Clean` musi być równy korzeniowi albo
   zaczynać się od „korzeń + separator” (korzeń `C:\a` nie przepuści `C:\ab`). Ta kontrola nie ufa
   `OczyscSciezke` i ma własny test z wejściem omijającym czyszczenie.
8. Ścieżka ma być już zdekodowana z URL dokładnie raz (to zadanie warstwy HTTP); `%2e%2e` jest dla rdzenia zwykłą nazwą.

## Czego rdzeń NIE robi (świadomie)

1. **Dowiązań symbolicznych i punktów połączenia NTFS (junction) nie rozwiązuje rdzeń, tylko magazyn.** Rdzeń nie
   dotyka dysku, więc nie wie, czy `/link` prowadzi poza udział. Magazyn po `SciezkaNaDysku` musi zrobić
   `filepath.EvalSymlinks` (na korzeniu i na wyniku) i ponownie sprawdzić, że wynik jest wewnątrz rozwiązanego
   korzenia, albo odmówić dowiązań wcale.
2. Krótkie nazwy 8.3 (`PROGRA~1`, `SKRYTK~1`) nie wyjdą poza udział, ale mogą ominąć porównanie nazw (np. zakaz
   kosza). Jeśli ma to znaczenie, magazyn powinien porównywać ścieżki po rozwiązaniu (np. `GetLongPathName`).
3. `OczyscSciezke` przepuszcza katalog kosza (`/.skrytka_kosz/...`), bo magazyn może go potrzebować; zakaz
   tworzenia takiej nazwy daje `SprawdzNazwe`. Ukrycie i blokadę wejścia do kosza z WWW/FTPS musi zrobić warstwa wyżej.
4. Bramka nie sprawdza, czy udział istnieje (to robi magazyn / usługi).

## Zachowanie pozostałych funkcji

1. `SprawdzNazwe` = `sprawdzCzlon` + zakaz `KatalogKosza` bez względu na wielkość liter; błąd zawsze `ErrZlaNazwa`.
2. `RodzajPliku`: folder = `folder`; inaczej wg ostatniego rozszerzenia bez względu na wielkość liter, nieznane = `inny`.
3. `Sortuj`: foldery zawsze pierwsze (także malejąco); klucze `nazwa|rozmiar|zmieniono`, nieznany = `nazwa`; remis
   rozstrzyga nazwa naturalnie, potem dokładne porównanie nazwy i ścieżki (kolejność zawsze ta sama). Porównanie
   naturalne: ciągi cyfr jako liczby dowolnej długości (`a2 < a10`, `007 = 7`), bez wielkości liter, kolejność
   polska (`a < ą < b`, `l < ł < m`, `z < ź < ż`), inne akcenty tuż po literze podstawowej (`e < é < ę`), cyfry i
   interpunkcja ASCII przed literami, inne pismo po alfabecie.
4. `Pasuje`: zawieranie bez wielkości liter i bez ogonków (polskie i częste łacińskie: `é`, `ü`, `č` …); puste
   zapytanie (po obcięciu spacji) pasuje do wszystkiego.
5. `CzyWolno`: wyłączony = `ErrBrakPrawa` (także admin); `OpAdministracja` tylko admin; admin wszystko poza zapisem
   w udziale tylko do odczytu (`ErrTylkoOdczyt`); odczyt/pobranie/udostępnienie wymagają `odczyt|zapis`;
   wgranie/zmiana/usunięcie wymagają `zapis` (inaczej `ErrBrakPrawa`) i udziału nie tylko do odczytu
   (`ErrTylkoOdczyt`); nieznana operacja lub nieznana wartość prawa = odmowa. Nazwa udziału porównywana dokładnie.

## Testy

`GOTOOLCHAIN=local _go-portable/bin/go.exe test ./wewn/rdzen/...` (zielone), `go vet` czysty.

1. `TestOczyscSciezke` (~80 złośliwych i poprawnych ścieżek), `TestOczyscSciezkeIdempotentna`, `TestSciezkaNaDysku`,
   `TestPolaczZKorzeniem`, `TestWewnatrzKorzenia` (`C:\a` vs `C:\ab`, UNC, inny dysk).
2. `TestSprawdzNazwe`, `TestRodzajPliku`, `TestRozszerzeniaBezDuplikatow`.
3. `TestPorownajNaturalnie`, `TestSortuj` (wszystkie klucze, obie kolejności), `TestSortujStalaKolejnosc`, `TestPasuje`.
4. `TestCzyWolno` (macierz: 8 użytkowników × wszystkie operacje × tylko do odczytu).
5. Test mutacyjny (2026-11-27, ręcznie skryptem): 28 wstrzyknięć zdejmujących pojedyncze blokady (`..`, `\`, UNC,
   dysk, człony, separator po korzeniu, kontrola korzenia, korzeń bezwzględny, limity, kropka/spacja, urządzenia,
   znaki zakazane/sterujące/C1/bidi, UTF-8, kosz, długość nazwy, 7 reguł bramki): każde dało `--- FAIL`.
