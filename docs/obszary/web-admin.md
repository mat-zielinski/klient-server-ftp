# Obszar web-admin (interfejs: logowanie, kreator, administracja, linki publiczne)

## Co robi

1. **Logowanie** (`POST /api/logowanie`): login, hasło, Enter wysyła formularz. Zły login lub hasło = czerwony
   komunikat z serwera; 429 = żółta ramka „Konto tymczasowo zablokowane” + komunikat serwera. Po sukcesie
   `sesja.odswiez()`, powłoka przełącza widok.
2. **Kreator** (`POST /api/kreator`): 4 kroki z paskiem postępu: konto administratora (hasło 2x, min. 10 znaków),
   pierwszy udział (nazwa, folder na dysku, tylko do odczytu), tryb sieci (VPN zalecany / Publiczny z ostrzeżeniem),
   podsumowanie. Błędy pól pokazywane dopiero po „Dalej”.
3. **Użytkownicy** (admin, `GET/POST/PUT/DELETE /api/konta`, `POST /api/konta/haslo`): nagłówek z kaflami (konta,
   administratorzy, wyłączone), tabela (login, rola, FTPS, prawa pigułkami, stan, akcje), okno dodawania i edycji
   z prawami per udział (segmenty Brak / Odczyt / Zapis; zapis zablokowany dla udziału tylko do odczytu), okno
   zmiany hasła, usuwanie z potwierdzeniem (własnego konta nie da się usunąć z interfejsu). Lista udziałów do praw
   z `GET /api/ustawienia`.
4. **Ustawienia** (admin, `GET/PUT /api/ustawienia`, `GET /api/siec`, `POST /api/siec/zapora`): układ stacji:
   kategorie po lewej (Udziały, Sieć, Certyfikat, Wygląd; wybrana w hashu `#/ustawienia?kategoria=siec`), karta po
   prawej, [Cofnij zmiany] i [Zapisz] na dole karty. Zapis całej konfiguracji naraz. Pola wymagające restartu
   (tryb, adres nasłuchu, porty, FTPS, certyfikat) dostają pigułkę „Wymaga restartu”, a po zapisie nagłówek strony
   pokazuje „Wymaga restartu serwera”. Sprawdzanie przed zapisem: porty 1..65535, różne porty HTTP/HTTPS/FTPS,
   zakres pasywny nieodwrócony, klucz przy certyfikacie z pliku.
   1. Udziały: tabela + okno dodaj/edytuj (nazwa bez `\ / : * ? " < > |`, bez powtórzeń), usuń z potwierdzeniem.
   2. Sieć: wybór trybu, adres nasłuchu z listy adresów `GET /api/siec` (plus 0.0.0.0), porty, porty pasywne,
      przełącznik FTPS, lista adresów z rodzajem, „Test portów” (ponowne `GET /api/siec`, pigułki
      `porty_otwarte`), „Dodaj regułę Zapory Windows” po potwierdzeniu w oknie.
   3. Certyfikat: własny / z pliku (ścieżki PEM) / Let's Encrypt (domena; ostrzeżenie w trybie VPN), odcisk
      z przyciskiem Kopiuj.
   4. Wygląd: motyw ciemny / jasny (od razu lokalnie, po zapisie domyślny serwera).
5. **Udostępnienia** (`GET/DELETE /api/udostepnienia`): tabela linków (ścieżka z pigułką „Wygasł” / „Limit
   wyczerpany”, autor tylko dla admina, wygasa względnie, hasło, pobrania / limit, adres), kopiowanie adresu,
   usuwanie z potwierdzeniem, sortowanie po kolumnach. Pusta lista podpowiada, jak utworzyć link na stronie Pliki.
6. **Publiczny** (`/s/<id>`, `GET /api/publiczny/<id>`): bez logowania. Plik: nazwa, rozmiar, ważność, limit,
   duży przycisk Pobierz. Folder: lista plików z pobieraniem pojedynczo (podfoldery tylko wyświetlane). 401/403 =
   formularz hasła (hasło w nagłówku `X-Haslo`, nigdy w adresie), 410 = „Link wygasł”, 404 = „Nie ma takiego linku”.

## Pliki

| plik | rola |
|---|---|
| `web/src/strony/Logowanie.svelte`, `Kreator.svelte`, `Uzytkownicy.svelte`, `Ustawienia.svelte`, `Udostepnienia.svelte`, `Publiczny.svelte` | strony |
| `web/src/komponenty/admin/stale.ts` | stałe obszaru: min. długość hasła (10), kody HTTP, domyślne wartości kreatora, opisy trybów sieci, opcje praw, nagłówek `X-Haslo`, czas zwolnienia blob |
| `…/admin/typy.ts` | typy spoza kontraktu: `InfoSieci`, `DaneKreatora`, `KontoZHaslem`, `WynikZapisu`, `InfoPubliczne`, `ZrodloCertyfikatu` |
| `…/admin/narzedzia.ts` | czyste funkcje: `bladHasla`, `bladPortu`, `poleWymagaRestartu`, `wymagaRestartu`, `zrodloCertyfikatu`, `adresLinku`, `maDate`, `wygasl`, `wyczerpany`, `kopiuj`, `kopiaKonfiguracji` |
| `…/admin/publiczny.ts` | fetch linku publicznego z nagłówkiem `X-Haslo`, adres pobrania, pobranie z hasłem przez blob |
| `…/admin/EkranSrodkowy.svelte` | ekran UniFi: granatowe tło, logo, wyśrodkowana karta, wersja |
| `…/admin/Komunikat.svelte` | ramka komunikatu ok / info / błąd / ostrzeżenie |
| `…/admin/KrokiKreatora.svelte`, `WyborTrybu.svelte` | pasek kroków; karty wyboru trybu sieci (Kreator i Ustawienia) |
| `…/admin/OknoKonta.svelte`, `OknoHasla.svelte`, `PrawaUdzialow.svelte`, `OknoPotwierdzenia.svelte` | okna strony Użytkownicy (potwierdzenie używane też w Udostępnieniach i Ustawieniach) |
| `…/admin/UstawieniaUdzialy.svelte`, `OknoUdzialu.svelte`, `UstawieniaSiec.svelte`, `UstawieniaCertyfikat.svelte`, `UstawieniaWyglad.svelte` | kategorie Ustawień |

## Testy

1. `npx svelte-check` 0 błędów, 0 ostrzeżeń; `npm run build` przechodzi; `node sprawdz_styl.mjs` OK (bez kolorów i px).
2. Podgląd w przeglądarce (vite dev + atrapa API na `127.0.0.1:8080`, poza projektem): Użytkownicy z oknem
   „Nowe konto”, Ustawienia → Sieć, Udostępnienia, Publiczny (folder i link wygasły 410), Logowanie z blokadą 429,
   Kreator (błąd hasła, krok 3 z ostrzeżeniem, motyw jasny). Bez prawdziwego serwera Go.
3. Brak testów jednostkowych `narzedzia.ts`: w `web/` nie ma uruchamiacza testów (vitest nie jest w `package.json`).

## Czego nie zrobiono

1. (Zrobione, patrz „Ekrany wejścia” p. 6) Pobieranie z linku z hasłem szło przez `fetch` + blob; teraz zwykły link z ciasteczkiem.
2. (Zrobione, patrz „Ekrany wejścia” p. 6) Podfoldery i „Pobierz wszystko (ZIP)” w linku publicznym.
3. Tworzenie linków publicznych jest na stronie Pliki (web-pliki), tu tylko lista i usuwanie.
4. Brak edycji istniejącego linku (API nie ma `PUT /api/udostepnienia`).

## Ekrany wejścia (logowanie, kreator, link publiczny): odświeżenie wyglądu

1. **`EkranSrodkowy`**: tło z poświatą (`--kolor-poswiata-1/2`) i wygaszaną siatką (`--kolor-siatka-tla`,
   `--siatka-tla-rozmiar`), karta półprzezroczysta z cieniem (`--kolor-karta-wejscia`, `--cien-karta-wejscia`),
   wjazd karty (szanuje `prefers-reduced-motion`). Od progu `waski` w górę dwie kolumny: marka (logo `ikony/logo.ts`,
   „Twoje pliki, Twój serwer.”, trzy zalety z ikonami), po prawej formularz; węziej tylko logo nad kartą. Prawy
   górny róg: `PrzelacznikMotywu` (Auto / Jasny / Ciemny, grupa radiowa ze strzałkami; na telefonie same ikony),
   działa przed zalogowaniem. Stopka: nazwa i wersja. Właściwość `marka?: boolean` wyłącza kolumnę marki.
2. **`wspolne/PoleHasla.svelte`** `{etykieta, value, autocomplete?, ikona?, podpowiedz?, blad?, opis?, autofocus?,
   pokaz? (bindable), sila?, minDlugosc?, generator?, onwygeneruj?, zgodne?, onblur?, onkeydown?}`: oko pokaż/ukryj
   (`aria-label`, `aria-pressed`, nie zabiera fokusu), „Włączony Caps Lock” (`getModifierState`), pasek siły
   (4 segmenty, Słabe/Średnie/Dobre/Silne: długość względem minimum, 14 i 20 znaków, klasy znaków, kara za
   powtórzenia), „Wygeneruj” (16 znaków z `crypto.getRandomValues` bez przekrzywienia, bez znaków mylących, każda
   klasa znaków) z przyciskiem Kopiuj. Czyste funkcje w `<script module>`: `ocenHaslo`, `generujHaslo`.
3. **`Pole`** dostało opcje (stare użycia bez zmian): `onblur`, `onfocus`, `inputmode`, `maxlength`, `duze`,
   `sufiks` (snippet), `aria-describedby` do opisu/błędu.
4. **Kreator**: login pusty z podpowiedzią „np. admin”, `autocomplete` username / new-password. Walidacja „na
   dotknięciu”: błąd po opuszczeniu pola albo po „Dalej”; „Hasła nie są takie same” także, gdy powtórzenie ma już
   długość hasła. „Dalej” zawsze aktywny, przy błędzie fokus na pierwsze złe pole (`input[aria-invalid]`). Generator
   wstawia hasło do obu pól i je pokazuje, zgodne powtórzenie = zielony znaczek. `KrokiKreatora`: tor postępu
   z animowanym wypełnieniem, ukończone z ptaszkiem i klikalne (`onwybierz`), na telefonie `zwarte` (podpis
   bieżącego kroku). Przejście kroków: wjazd z boku zależnie od kierunku (`--przesuniecie-krok`, `--czas-krok`).
   Krok 2: przykłady ścieżek Windows i Linux (klik wstawia). Krok 4: `PodsumowanieKreatora` (karta z ikonami,
   „Zmień” przy każdym wierszu) i „Uruchom serwer”. `WyborTrybu`: kółko wyboru po prawej.
5. **Logowanie**: tylko login i hasło (oko, Caps Lock), `api.post('/logowanie')` z `api/klient.ts`; złe dane =
   komunikat z serwera, hasło czyszczone i fokus na nie; 429 = „Konto tymczasowo zablokowane”. Bez „Zapamiętaj mnie”.
6. **Publiczny** (na prośbę sesji głównej): hasło w `PoleHasla`; po poprawnym haśle serwer ustawia ciasteczko linku,
   więc pobieranie i ZIP to zwykłe odnośniki `<a href download>` (`adresPobraniaLinku`, `adresZipLinku`), bez
   fetch + blob (`pobierzZHaslem` usunięte, `ZWOLNIENIE_POBRANIA_MS` w `stale.ts` nieużywane). Folder: okruszki
   (`okruszki()`), wchodzenie w podfoldery (`GET …?sciezka=/pod`), „Pobierz wszystko (ZIP)” dla bieżącego folderu.
   Wygasłe ciasteczko = 401 = ponownie formularz hasła.
7. Nowe tokeny: ekrany wejścia (`--kolor-poswiata-*`, `--kolor-siatka-tla`, `--kolor-karta-wejscia`,
   `--kolor-ramka-wejscia`, `--cien-karta-wejscia`, `--kolor-fokus-pola` w obu motywach; `--uklad-wejscie-*`,
   `--siatka-tla-rozmiar`, `--poswiata-rozmiar`, `--czcionka-haslo-marki`, `--wysokosc-pole-duze`,
   `--wysokosc-sila-hasla`, `--wysokosc-postep-krokow`, `--przesuniecie-*`, `--czas-krok`,
   `--czas-postep`, `--krzywa-wejscia`). Nowe ikony: `oko_zamkniete`, `caps_lock`, `generuj`, `rakieta`.
8. Sprawdzenie: na prośbę użytkownika BEZ testów i bez svelte-check; `npm run build` i `npm run straznik` OK. Podgląd
   (vite dev + atrapa API w scratchpadzie): kreator kroki 1–4 (błąd „Dalej” z fokusem, generator, powrót klikiem
   w krok, błąd serwera), logowanie (429), link publiczny z hasłem i podfolderem; motyw jasny i ciemny, szeroki
   ekran i telefon. Nie sprawdzono z prawdziwym serwerem Go ani czytnikiem ekranu.
