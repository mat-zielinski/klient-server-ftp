# Obszar: udostępnienia (linki publiczne)

## Co robi

Pakiet `wewn/uslugi/udostepnienia` trzyma linki publiczne do pliku lub folderu w udziale, w pliku
`dane/udostepnienia.json` (zapis atomowy przez `konfiguracja.ZapiszAtomowo`, uprawnienia 0600).

- `Otworz(plik)`: wczytuje plik (brak pliku = pusta lista, zepsuty JSON = błąd) i od razu usuwa wygasłe linki.
- `Utworz(l, haslo)`: ID = 16 losowych bajtów (crypto/rand) w base64url; ścieżka przez `rdzen.OczyscSciezke`;
  `Wygasa` zero = domyślnie 7 dni, data w przeszłości = `ErrZlyLink`, dalej niż 365 dni = przycięte do 365 dni;
  pola `ID`, `Pobrano`, `Utworzono` z wejścia są ignorowane. Hasło: argon2id w formacie PHC
  (`$argon2id$v=19$m=19456,t=2,p=1$sol$klucz`), własna mała funkcja w `haslo.go`.
- `Lista(autor)`: bez `HashHasla`, z `MaHaslo`, bez wygasłych, od najnowszego.
- `Sprawdz(id, haslo)`: brak = `rdzen.ErrNieZnaleziono`, wygasły lub wyczerpany limit = `rdzen.ErrWygaslo`,
  złe hasło = `rdzen.ErrZleDane`, ponad 5 złych haseł w oknie 15 min = `rdzen.ErrKontoZablokowane`
  (wtedy nawet dobre hasło jest odrzucane do końca okna). Próba jest rezerwowana pod blokadą przed hashowaniem,
  więc równoległe zgadywanie nie przekroczy limitu; argon2id liczony poza blokadą. Dobre hasło zeruje licznik.
  Porównanie w stałym czasie (`subtle.ConstantTimeCompare`). Licznik prób jest tylko w pamięci.
- `ZaliczPobranie(id)`: pod blokadą sprawdza ważność i limit, zwiększa licznik i zapisuje plik; błąd zapisu
  cofa licznik.
- `Usun(id)`: błąd zapisu cofa usunięcie.
- Dodatki (poza kontraktem, bez zmiany sygnatur): `UstawZegar(func() time.Time)` (testy),
  `Sprzataj() (int, error)` (okresowe sprzątanie z warstwy aplikacji), `Pobierz(id)` (link bez sprawdzania hasła,
  np. kontrola autora przed `Usun`), błąd `ErrZlyLink`.

## Pliki

- `wewn/uslugi/udostepnienia/udostepnienia.go`: typy i logika.
- `wewn/uslugi/udostepnienia/haslo.go`: argon2id PHC, odczyt z kontrolą zakresu parametrów.
- `wewn/uslugi/udostepnienia/plik.go`: odczyt i atomowy zapis JSON.
- `wewn/konfiguracja/udostepnienia.go`: stałe.

## Stałe (`konfiguracja/udostepnienia.go`)

`DlugoscIDLinku = 16`, `MaksWaznoscLinku = 365 dni`, `DomyslnaWaznoscLinku = 7 dni`, `LimitProbHaslaLinku = 5`,
`OknoProbHaslaLinku = 15 min`, `MaksDlugoscHaslaLinku = 256`, `Argon2Link*` (m = 19 MiB, t = 2, p = 1, sól 16 B,
klucz 32 B).

## Testy

`udostepnienia_test.go`, `haslo_test.go`: walidacja tworzenia (w tym `/../` = `ErrPozaUdzialem`), domyślna
i przycięta ważność, wygasanie (zegar wstrzyknięty), sprzątanie przy otwarciu i w pliku, limit pobrań, hasło,
blokada prób (także równoległa), zerowanie licznika, trwałość w pliku (hash w pliku, brak jawnego hasła),
zepsuty i niespójny plik, równoległe `ZaliczPobranie` (80 gorutyn, limit 30 = dokładnie 30).
Wstrzyknięcia sprawdzone: zdjęcie limitu prób, limitu pobrań, wygasania, porównania hasła, czyszczenia ścieżki,
blokady w `ZaliczPobranie` i rezerwacji próby: za każdym razem testy czerwone.

## Czego nie zrobiono

- `-race` niedostępne (brak cgo w przenośnym Go).
- Licznik złych prób nie przetrwa restartu (w pamięci).
- Brak sprawdzania, czy udział istnieje i czy ścieżka istnieje na dysku: to warstwa HTTP (zna konfigurację i magazyn).
