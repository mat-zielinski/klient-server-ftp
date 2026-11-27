# Obszar: konta (sprint 1, agent „konta”)

## Co robi

1. Magazyn użytkowników `dane/konta.json` (`{"konta":[…]}`), zapis przez `konfiguracja.ZapiszAtomowo` (plik
   tymczasowy + zamiana, 0600). Błąd zapisu cofa zmianę w pamięci.
2. Hasła argon2id (`golang.org/x/crypto/argon2`), zapis PHC `$argon2id$v=19$m=65536,t=2,p=2$sól$hash`
   (base64 bez dopełnienia), sól 16 B z `crypto/rand`, porównanie `subtle.ConstantTimeCompare`. Parametry czytane
   z hasha, więc zmiana stałych nie psuje starych haseł; przy odczycie limity `ArgonMaksPamiecKiB` i `ArgonMaksCzas`.
3. Logowanie `Zaloguj(login, haslo, ip)`: dla nieistniejącego loginu liczony jest hash atrapy (ten sam koszt);
   wyłączone konto = `ErrZleDane` (nie zdradza, że hasło dobre); hasło dłuższe niż limit odrzucane bez haszowania.
4. Blokada po złych hasłach osobno na IP i na login: `MaksBlednychLogowan` prób w `OknoBlednychLogowan` zakłada
   blokadę na `CzasBlokady` (`ErrKontoZablokowane`, nawet przy dobrym haśle). Próby w czasie blokady jej nie
   przedłużają. Dobre logowanie czyści licznik loginu (licznik IP zostaje). Sprzątanie liczników przy
   `ProgSprzataniaBlokad` wpisach.
5. Sesje w pamięci: token 32 B z `crypto/rand`, base64url; w mapie trzymany tylko SHA-256 tokenu. Ważność
   `CzasSesji` od ostatniego użycia (przedłużana w `Sprawdz`). `Sprawdz` za każdym razem czyta aktualne konto
   (usunięte / wyłączone = odmowa). Wygasłe sesje sprzątane przy `Utworz`.
6. `UstawHaslo`, `Usun` i wyłączenie konta w `Zmien` kończą sesje loginu we wszystkich magazynach `NoweSesje(k)`.
7. Nie można odebrać admina, wyłączyć ani usunąć ostatniego aktywnego admina (`ErrOstatniAdmin`).
8. `Lista`, `Uzytkownik` i `Zaloguj` zwracają kopie bez `HashHasla` (osobna mapa praw). `Dodaj` ignoruje
   `HashHasla` podany przez wywołującego.
9. Login: 1..`MaksDlugoscLoginu` znaków `[a-zA-Z0-9._-]`, nie od kropki, wielkość liter ma znaczenie.
   Hasło: `DlugoscHasla`..`MaksDlugoscHasla` znaków UTF-8.
10. Zegar wstrzykiwany przez nieeksportowane pole `Konta.teraz` (testy bez `time.Sleep`); sesje używają zegara kont.

## Pliki

`wewn/uslugi/konta/konta.go` (magazyn, logowanie), `hasla.go` (argon2id, PHC), `blokady.go` (liczniki),
`sesje.go` (sesje), testy `*_test.go` + `pomoc_test.go`; stałe `wewn/konfiguracja/konta.go`.

## Błędy pakietu (dodane obok kontraktu)

`ErrSlabeHaslo`, `ErrZbytDlugieHaslo`, `ErrOstatniAdmin`; poza tym `rdzen.ErrZleDane`, `ErrKontoZablokowane`,
`ErrNieZnaleziono`, `ErrJuzIstnieje`, `ErrZlaNazwa`.

## Testy

Hasło dobre / złe / zepsuty hash / format PHC z parametrami domyślnymi; długość hasła; logowanie (wyłączone,
nieistniejące z liczeniem hasha atrapy, za długie hasło); blokada loginu, IP, okno czasu, czyszczenie licznika,
sprzątanie; plik (zapis, odczyt, brak hasła jawnie, brak `.tmp`, błąd zapisu cofa stan, zły plik); ostatni admin
(`Zmien`, `Usun`, wyłączony admin się nie liczy); sesje (format tokenu, brak jawnego tokenu, wygasanie,
przedłużanie, sprzątanie, koniec po zmianie hasła / wyłączeniu / usunięciu, brak ożywienia po ponownym założeniu
konta); równoległość (16 gorutyn). Testy mutacyjne wykonane ręcznie: każda blokada zdjęta = test czerwony.

## Czego nie zrobiono

1. `-race` nie działa (brak cgo / gcc na tym komputerze); test równoległy jest, ale bez detektora wyścigów.
2. Brak górnego limitu całkowitego czasu sesji (tylko przesuwne `CzasSesji`) i limitu liczby sesji na konto.
3. Brak przeliczania hasha przy logowaniu, gdy zmienią się parametry argon2id.
4. Wartości praw (`rdzen.Prawo`) w `Dodaj` / `Zmien` nie są walidowane (np. nieznany udział, nieznane prawo).
5. Sesje giną przy restarcie (w pamięci, zgodnie z kontraktem).

# Fala 2: urządzenia

## Co robi

1. Urządzenia (`sesje.go`): `UtworzZOpisem(login, ip, przegladarka)` (opis obcinany do
   `MaksDlugoscOpisuUrzadzenia` znaków), `Urzadzenia(login)` od ostatnio używanej, `ZakonczUrzadzenie(login, id)`
   (tylko własne, inaczej `rdzen.ErrNieZnaleziono`), `IDSesji(token)` (funkcja pakietu) = pierwsze
   `DlugoscIDSesji` = 8 B SHA-256 tokenu w hex. Limit `MaksSesjiNaKonto` = 10: wypada sesja najdawniej używana.
2. Logowanie tylko loginem i hasłem (`Zaloguj`). Plik `dane/konta.json` z dawnym, nieużywanym polem `"dwa_fa"`
   wczytuje się dalej (nieznane pole pominięte), a pierwszy zapis je usuwa.

## Testy fali 2

Urządzenia (lista, kolejność, ID nie jest tokenem, cudze urządzenie, limit 10, obcinanie opisu); stary plik kont
z polem `"dwa_fa"` (wczytanie, logowanie samym hasłem, pole znika po zapisie). Mutacje (zdjęcie każdej blokady =
test czerwony): cudze urządzenie, limit sesji. `GOOS=linux go vet` czysty.
