# Obszar web-pliki: strona Pliki

Strona przeglądania plików (wzór działania FileBrowser Quantum, wygląd stacji Programator_Czarek + UniFi).
Położenie w hashu `#/pliki/<udzial>/<sciezka>` (przez `stan/nawigacja.svelte.ts`); brak lub nieznany udział = pierwszy
dostępny (bez wpisu w historii).

## Co robi

1. **Układ**: lewa kolumna udziałów (`ListaUdzialow`, pigułka „Odczyt” bez prawa zapisu); poniżej progu `waski`
   udział wybiera się listą w nagłówku. Okruszki ścieżki, przycisk „folder wyżej”. Szkielety ładowania
   (`wspolne/Szkielet`) zamiast pustego ekranu przy wczytywaniu udziałów, folderu i szukania.
2. **Pasek narzędzi** (`PasekNarzedzi`): jeden rząd o stałej wysokości, który się nie zawija. Bez zaznaczenia:
   szukaj, sortowanie (foldery zawsze pierwsze, porządek naturalny nazw), widok lista / kafelki (pamiętany
   w `localStorage`), tryb „Zaznacz” (pola wyboru), odśwież, nowy folder, wgraj. Przy zaznaczeniu pasek zmienia się
   w pasek zaznaczenia: licznik, Pobierz ZIP, Usuń, Odznacz. Przy braku miejsca (pasek sam mierzy, czy treść się
   mieści) kolejno: szukajka się zwęża, przyciski zostają same z ikoną (dymek i etykieta dostępności), a sortowanie,
   widok, Zaznacz, Odśwież i Nowy folder przechodzą do menu „…” (`MenuKontekstowe`). Bez prawa zapisu przyciski
   i pozycje menu wyszarzone z powodem w dymku.
3. **Szukanie**: GET `/api/indeks/szukaj?udzial&q&limit` (cały udział, wynik zawężany do bieżącego folderu),
   przy 404 / 503 zapasowo GET `/api/szukaj`; opóźnienie 200 ms; pasujące fragmenty nazw podświetlone
   (`Podswietlenie`), w liście dodatkowo ścieżka wpisu.
4. **Lista i kafelki z wirtualizacją** (`wirtualizacja.svelte.ts`): renderowane są tylko widoczne wiersze + 8 wierszy
   bufora, resztę zastępują odstępy (w tabeli wiersze-odstępy, w siatce kafelków marginesy). Wysokość wiersza
   mierzona z wyrenderowanych elementów (pierwszy pomiar z tokenu `--wysokosc-wiersz-tabeli`), liczba kolumn siatki
   z `grid-template-columns`. Działa z dowolnym przewijanym rodzicem; `przewinDo` przewija do wpisu spoza
   wyrenderowanych (strzałki). Długości w CSS jako `calc(var(--odstep-ramka) * n)` (bez „px” poza tokenami).
   Kolumny listy zależą od szerokości samej listy: nazwa zostaje zawsze (wielokropek i dymek ze ścieżką), kolumna
   Zmieniono wymaga miejsca na nazwę co najmniej `--kolumna-xl`, a kolumna Rozmiar co najmniej `--kolumna-l`.
   Ukryte dane schodzą do drugiego wiersza pod nazwą (np. „2 KB · 3 dni temu”). Kolumny znikają naraz
   z `<colgroup>`, nagłówka i wierszy, więc nagłówek i wiersze mają jedną siatkę.
5. **Miniatury** (`Miniatura`): GET `/api/miniatura?udzial&sciezka&rozmiar=s|m` (kafelki `s`, galeria `m`),
   wczytywane dopiero w polu widzenia (IntersectionObserver), do czasu wczytania i przy błędzie ikona rodzaju.
6. **Zaznaczanie i klawiatura** (`zaznaczenie.svelte.ts`, `klawiatura.ts`): klik, Ctrl+klik, Shift+klik; strzałki
   (w kafelkach także w bok), Shift+strzałki, Home/End, Enter, Delete, F2, Backspace, Esc, Ctrl+A, klawisz menu
   i Shift+F10.
7. **Paleta Ctrl+K** (`polecenia.ts`, `stan/skroty.svelte.ts`): Nowy folder (Shift+N), Wgraj pliki (Shift+U),
   Szukaj plików („/”, fokus w polu szukania), Widok: lista (Shift+L), Widok: kafelki (Shift+K), Zaznacz wszystko.
   Polecenia znikają z rejestru po wyjściu ze strony.
8. **Podgląd i galeria** (`Podglad`, `PodgladObrazu`): obraz, wideo, dźwięk, PDF w `iframe`, tekst do 1 MB.
   Obrazy: najpierw miniatura `m` (rozmyta), potem pełny plik; przybliżanie kółkiem (do 8×) i podwójnym kliknięciem
   (2,5×), przesuwanie przeciąganiem, pełny ekran (przycisk lub F, w pełnym ekranie strzałki i licznik),
   wstępne ładowanie sąsiednich obrazów. Strzałki ← → i licznik „3 / 120” po plikach z podglądem.
9. **Akcje** (`AkcjePlikow` + `menu.ts` + wspólne `MenuKontekstowe`): Otwórz / Podgląd, **Pobierz przez… ▸**
   (GET `/api/protokoly`, najpierw „Wczytywanie…”; niedostępne wyszarzone z `powod`), Kopiuj ścieżkę, Zmień nazwę,
   Usuń, Właściwości; kilka zaznaczonych: ZIP i Usuń; puste miejsce: Nowy folder, Wgraj, Odśwież, Zaznacz wszystko.
   - Przeglądarka HTTPS i Jako ZIP: pobranie adresu z odpowiedzi.
   - Klient FTP (FTPS): `OknoFtps` z adresem `ftps://` i Kopiuj, serwerem / portem / loginem / ścieżką, kodem QR
     adresu, instrukcją dla FileZilli, Total Commandera i WinSCP, odciskiem certyfikatu i innymi adresami serwera.
   - Kopiuj link bezpośredni: schowek (zapasowo przez zaznaczenie, gdy brak HTTPS).
   - Link publiczny…: `OknoLinkuPublicznego`: ważność 1 dzień / 7 dni (domyślnie) / 30 dni / 1 rok / własna data
     (najdłużej rok, bez opcji „bez terminu”), hasło, limit pobrań → POST `/api/udostepnienia`, potem adres
     z Kopiuj i kod QR.
   - ZIP: zwykły link GET `/api/zip?udzial&sciezka=..&sciezka=..` (strumień prosto do pliku, bez Blob w pamięci).
   - Zmiana nazwy: POST `/api/zmien` z pełnymi ścieżkami `{"z","na"}`.
   - Właściwości (`OknoWlasciwosci`): także „Oblicz SHA-256” (GET `/api/suma`) z kopiowaniem.
10. **Wgrywanie** (`wgrywanie.svelte.ts`, `upuszczanie.ts`, `ListaWgrywania`, `StrefaUpuszczania`): przeciągnij
    i upuść (także całe foldery; serwer tworzy podfoldery) i przycisk Wgraj. Kawałki 8 MB przez PUT `/api/wgraj`
    (XHR, postęp wewnątrz kawałka), wznowienie z GET `/api/wgraj`, 2 pliki naraz, 3 ponowienia kawałka po błędzie
    sieci. 409 = „Istnieje” z Nadpisz / Pomiń; błąd = Wznów. Udane wgranie odświeża listę.
11. **Moje pobrania** (`MojePobrania`, `MiniWykres`): `sluchajTransferow`, tylko transfery z
    `kto === sesja.stan.uzytkownik.login`; nazwa, pigułka protokołu, `PasekPostepu`, prędkość, pozostały czas,
    mini wykres SVG z 60 s. Zakończone pokazują wynik i znikają po 6 s. Panel zwijany, widoczny tylko przy transferach.
12. **Stany puste i błędy**: brak udziałów, błąd udziałów (Ponów), folder 404, 403, inny błąd (Ponów), pusty folder,
    brak wyników. Komunikaty operacji przez `powiadom()`.

## Pliki

`web/src/strony/Pliki.svelte` (stan strony, wczytywanie, układ) oraz `web/src/komponenty/pliki/`:
komponenty `ListaUdzialow`, `Okruszki`, `PasekNarzedzi`, `TabelaPlikow`, `KafelkiPlikow`, `Miniatura`,
`Podswietlenie`, `Podglad`, `PodgladObrazu`, `AkcjePlikow`, `OknoFtps`, `OknoLinkuPublicznego`, `OknoNazwy`,
`OknoWlasciwosci`, `OknoUsuwania`, `StrefaUpuszczania`, `ListaWgrywania`, `MojePobrania`, `MiniWykres`;
moduły `operacje.ts` (API, sortowanie, adresy ZIP i miniatur), `menu.ts`, `klawiatura.ts`, `zaznaczenie.svelte.ts`,
`wirtualizacja.svelte.ts`, `polecenia.ts`, `wgrywanie.svelte.ts`, `upuszczanie.ts`, `narzedzia.ts` (ścieżki, schowek,
pobieranie, podświetlenie), `rodzaje.ts`, `zdarzenia.ts`, `stale.ts`.

## Stałe (`komponenty/pliki/stale.ts`)

Kawałek wgrywania 8 MB, 2 wgrania naraz, 3 ponowienia co 1,5 s, znikanie zakończonych 6 s, opóźnienie szukania
200 ms, limit wyników 500, limit podglądu tekstu 1 MB, historia wykresu 60 próbek, bufor wirtualizacji 8 wierszy,
60 wpisów przed pierwszym pomiarem, przybliżanie (krok 1,2, maks. 8, dwuklik 2,5), ważności linku 1 / 7 / 30 / 365 dni
(domyślnie 7, najdłużej 365), klucze `localStorage`. Kolory i rozmiary wyłącznie z `styl/tokeny.css`.

## Testy

`npm run build` (OK), `npx svelte-check` (0 błędów, 0 ostrzeżeń), `node sprawdz_styl.mjs` (OK). Brak testów
jednostkowych interfejsu (w `web/` nie ma uruchamiacza testów JS). Strona nie była sprawdzona na żywo z serwerem.

## Czego nie zrobiono

1. Brak testów jednostkowych (`uporzadkuj`, `fragmentyPodswietlenia`, `Zaznaczenie`, `obslugaKlawiszy`,
   `Wirtualizacja`): w `web/` nie ma vitest.
2. Wirtualizacja zakłada równą wysokość wierszy (w wynikach szukania wszystkie wiersze mają ścieżkę, więc są równe).
3. Szukanie w indeksie zwraca do 500 trafień z całego udziału; po zawężeniu do folderu może ich zostać mniej,
   niż jest w folderze naprawdę.
