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

# Fala 2: 2FA (TOTP) i urządzenia

## Co robi

1. TOTP wg RFC 6238 bez bibliotek (`totp.go`): HMAC-SHA1, `TOTPCyfry` = 6, `TOTPOkres` = 30 s, okno ±`TOTPOkno`,
   sekret `TOTPDlugoscSekretu` = 20 B w base32 bez dopełnienia, kod porównywany w stałym czasie.
2. `Start2FA(login)` → sekret i adres `otpauth://totp/Skrytka:login?secret=…&issuer=Skrytka&algorithm=SHA1&digits=6&period=30`;
   sekret czeka w pamięci (nie w pliku) do `Wlacz2FA(login, kod)`. Przy włączonym 2FA `Start2FA` = `Err2FAJuzWlaczone`.
3. Włączone sekrety są w `dane/konta.json` w osobnym polu `"dwa_fa": {login: sekret}`, poza `rdzen.Uzytkownik`
   (nie wyciekają przez `Lista` / API). Usunięcie konta usuwa jego 2FA.
4. `ZalogujZKodem(login, haslo, kod, ip)`: kolejno blokada, hasło, wyłączone konto, 2FA. Brak kodu przy dobrym
   haśle = `ErrWymaganyKod` (nie liczy się do blokady); zły kod = `ErrZlyKod` i próba liczona na IP i login.
   Ochrona przed powtórzeniem: kod musi być z kroku czasu późniejszego niż ostatnio użyty (pamięć).
   `Zaloguj` = `ZalogujZKodem` bez kodu, więc przy 2FA zwraca `ErrWymaganyKod`.
5. `Wylacz2FA(login, haslo)`: wymaga hasła, złe hasło liczy się do blokady loginu; `Err2FANieWlaczone`.
   `Ma2FA(login)`.
6. Urządzenia (`sesje.go`): `UtworzZOpisem(login, ip, przegladarka)` (opis obcinany do
   `MaksDlugoscOpisuUrzadzenia` znaków), `Urzadzenia(login)` od ostatnio używanej, `ZakonczUrzadzenie(login, id)`
   (tylko własne, inaczej `rdzen.ErrNieZnaleziono`), `IDSesji(token)` (funkcja pakietu) = pierwsze
   `DlugoscIDSesji` = 8 B SHA-256 tokenu w hex. Limit `MaksSesjiNaKonto` = 10: wypada sesja najdawniej używana.

## Testy fali 2

Wektory RFC 6238 (SHA1, 8 i 6 cyfr), okno ±1 (±2 odrzucone), format kodu, adres otpauth; włączanie (bez startu,
zły kod, ponowny start); logowanie z kodem, powtórzenie kodu, starszy kod po nowszym, blokada po złych kodach,
brak kodu nie blokuje; wyłączanie (złe hasło, blokada); plik (zapis, odczyt, zły sekret, brak w liście,
usunięcie konta); urządzenia (lista, kolejność, ID nie jest tokenem, cudze urządzenie, limit 10, obcinanie opisu).
Mutacje (zdjęcie każdej blokady = test czerwony): 2FA w logowaniu, blokada złych kodów, powtórzenie kodu, brak
kodu, hasło i blokada przy wyłączaniu, kod przy włączaniu, szerokość okna, obcięcie RFC, cudze urządzenie,
limit sesji, usunięcie 2FA z kontem. `GOOS=linux go vet` czysty.

## Czego nie zrobiono (fala 2)

1. Sekret TOTP jest w pliku jawnie (plik 0600); brak szyfrowania kluczem z innego miejsca.
2. Brak kodów zapasowych (odzyskiwania) i wyłączania 2FA przez admina dla innego konta (zgubiony telefon:
   dziś tylko ręczna edycja pliku albo usunięcie i założenie konta od nowa).
3. Ostatni użyty krok i oczekujące sekrety giną przy restarcie (powtórzenie kodu możliwe tylko w ~90 s po restarcie).
4. Złe kody w `Wlacz2FA` nie liczą się do blokady (wymaga zalogowanej sesji).
