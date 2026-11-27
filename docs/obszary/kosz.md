# Obszar: kosz aplikacji (magazyn + HTTP + strona „Kosz”)

Usunięte pliki i foldery (HTTP `DELETE /api/plik`, FTPS `DELE`/`RMD`) nie znikają od razu: trafiają do kosza udziału.
Strona „Kosz” pozwala je przejrzeć, przywrócić, usunąć trwale i (admin) opróżnić cały kosz.

## Układ na dysku

```
<udział>/.skrytka_kosz/
  2026-11-27_10-30-00/f/a.txt     element z oryginalną ścieżką /f/a.txt
  2026-11-27_10-30-00.json        opis: {"sciezka":"/f/a.txt"}
  2026-11-27_10-30-00_2/...       drugie usunięcie w tej samej sekundzie
```

1. Każde `DoKosza` zakłada **własny** folder znacznika (`os.Mkdir`, zajęty = `_2`, `_3` …) i plik opisu obok niego
   (nie w środku, więc nie zderzy się z treścią użytkownika). Znacznik czasu z wstrzykiwanego zegara magazynu.
2. Stary układ (sprzed opisów, kilka usunięć w jednym znaczniku) jest czytany zgadywaniem: folder z dokładnie jednym
   elementem = folder pośredni (schodzimy niżej); plik, pusty folder albo folder z kilkoma elementami = wpis.
   Skutek: usunięty folder z jednym plikiem pokazuje się jako ten plik (przywrócenie i tak odtwarza to samo drzewo).
3. Foldery w koszu o nazwie innej niż znacznik są pomijane; czas usunięcia = nazwa znacznika w strefie zegara magazynu.

## Identyfikator wpisu

`id` = base64url bez dopełnienia z `<znacznik>/<ścieżka względna>`. `rozbierzID` sprawdza bez dysku: długość
(`MagazynMaksDlugoscIDKosza`), base64, UTF-8, poprawny znacznik (`_n` tylko n ≥ 2, bez zer wiodących), ścieżka
kanoniczna przez `rdzen.OczyscSciezke` (bez `..`, `\`, dysków, UNC, strumieni, podwójnych ukośników), bez ukrytych członów.
Na dysku (`elementWKoszu`): kosz i folder znacznika nie mogą być dowiązaniem / junction, dowiązania po drodze
rozwijane przez `rozwiazWUdziale` z korzeniem = folder znacznika (wyjście = `ErrPozaUdzialem`), sam element może
być dowiązaniem (działa się na nim, nie na celu); gdy znacznik ma opis, ścieżka musi się z nim zgadzać.

## Operacje magazynu (`wewn/magazyn/kosz.go`, `kosz_lista.go`)

| funkcja | działanie |
|---|---|
| `ListaKosza(udzial)` | `[]WpisKosza{ID, Nazwa, Sciezka, Usunieto, Rozmiar, Folder, Rodzaj}` od najnowszych; brak kosza = pusta lista; kosz-dowiązanie = `ErrPozaUdzialem` |
| `Przywroc(udzial, id, nadpisz)` | na oryginalne miejsce (`pelnaBezKonca`, więc bramka tylko-odczytu i dowiązań jak przy zmianie nazwy); brak rodzica = `MkdirAll`; plik zamiast folderu nadrzędnego = `ErrJuzIstnieje`; zajęte bez `nadpisz` = `ErrJuzIstnieje`, z `nadpisz` obecny element najpierw idzie do kosza (nic nie ginie). Blokady kosza i celu w stałej kolejności (`zablokujDwie`) |
| `UsunTrwale(udzial, id)` | `os.RemoveAll` elementu (dowiązania w środku usuwane same, bez celu), sprzątanie pustych folderów i opisu |
| `Oproznij(udzial, starszeNiz)` | trwałe usunięcie wpisów z `Usunieto` przed `zegar() - starszeNiz` (0 = wszystkie), zwraca liczbę |
| `SciezkaWpisuKosza(id)` | oryginalna ścieżka z identyfikatora (do dziennika), bez dysku |

Udział tylko do odczytu = `rdzen.ErrTylkoOdczyt` dla `Przywroc`, `UsunTrwale`, `Oproznij` przed dotknięciem dysku.
Zły identyfikator = `magazyn.ErrZlyIDKosza` (HTTP 400).

## HTTP (`wewn/http/kosz.go`)

`GET /api/kosz`, `POST /api/kosz/przywroc`, `DELETE /api/kosz`, `DELETE /api/kosz/wszystko` (tabela w `docs/api.md`).
Bramka: `dostepDoUdzialu(u, OpUsuniecie, udzial)` (prawo zapisu, nieznany udział dla kogoś bez prawa = 403);
opróżnianie dodatkowo `tylkoAdmin`. CSRF jak reszta (pośrednik). Każda zmiana = wpis dziennika `RodzajZmiana`
i `Indeks.Odswiez` (przez `zapiszZmiane`). Usługa kosza przez asercję typu na `uslugaMagazynu` (brak = 503),
`serwer.go` zmieniony tylko o `s.trasyKosza(m)`.

## Interfejs

`web/src/strony/Kosz.svelte` (pozycja „Kosz” w rejestrze `STRONY` po Udostępnieniach, ikona `kosz`, dla wszystkich),
`web/src/komponenty/kosz/kosz.ts` (typ `WpisKosza`, wywołania API, sortowanie, udziały z koszem),
`web/src/komponenty/kosz/TabelaKosza.svelte` (nazwa z ikoną rodzaju, oryginalne miejsce, usunięto, rozmiar, akcje;
na wąskim oknie miejsce i data schodzą pod nazwę). Wybór udziału (`Wybor`, tylko gdy więcej niż jeden udział z prawem
zapisu), Przywróć (409 = okno „Miejsce jest zajęte” z „Przywróć i nadpisz”), Usuń trwale (potwierdzenie,
nieodwracalne), Opróżnij kosz (admin, potwierdzenie), stany: ładowanie (szkielet), błąd z Ponów, brak udziałów
z koszem, pusty kosz.

## Stałe (`wewn/konfiguracja/magazyn.go`)

`MagazynPrzyrostekOpisuKosza` (`.json`), `MagazynMaksDlugoscIDKosza` (4096), `KoszOproznijPoDniach` (30),
`KoszWiekAutomatycznegoOprozniania` (30 × 24 h), `KoszOkresOprozniania` (24 h). W interfejsie
`DNI_AUTOMATYCZNEGO_OPROZNIANIA` = 30 (`komponenty/kosz/kosz.ts`).

## Testy

- `wewn/magazyn/kosz_lista_test.go`: lista (kolejność, rozmiar folderu, ID bez znaków ścieżki), stary układ,
  przywrócenie z tworzeniem folderów i sprzątaniem znacznika, kolizja (bez/z nadpisaniem, folder, plik zamiast
  folderu nadrzędnego), trwałe usunięcie, junction w usuniętym folderze (opróżnienie nie rusza celu), ucieczka przez
  identyfikator (15 przypadków), junction w folderze znacznika i kosz jako junction, opróżnianie wg wieku (zegar
  wstrzykiwany), tylko do odczytu, `czasZnacznika`.
- `wewn/http/kosz_test.go`: pełny przebieg (lista, 409, nadpisanie, trwałe usunięcie, opróżnienie, dziennik,
  odświeżanie indeksu), prawa (401, 403 dla odczytu / braku prawa / nieznanego / tylko do odczytu / nie-admina,
  400 dla złego i uciekającego id), CSRF z dowodem braku zmiany.
- Mutacje sprawdzone ręcznie (test czerwony): rozwijanie dowiązań w `elementWKoszu`, sprawdzenie kosza-dowiązania
  w `katalogKosza`, tylko-odczyt w `koszDoZmiany`, operacja bramki w `kosz()` (`OpOdczyt` zamiast `OpUsuniecie`),
  `tylkoAdmin` przy opróżnianiu.

## Czego nie zrobiono

1. Automatyczne opróżnianie raz na dobę wymaga wpięcia w `wewn/aplikacja` (inny obszar): stałe są gotowe.
2. FTPS nie ma dostępu do kosza (tylko usuwanie do niego).
3. Brak limitu rozmiaru kosza; rozmiar folderów liczony przy każdej liście (duży kosz = wolniejsza lista).
4. Mutacje nie są jeszcze wpisane do listy mutacji strażnika (`docs/obszary/straznicy.md`).
5. Strona nie była sprawdzona na żywo z serwerem ani zrzutami w obu motywach.
