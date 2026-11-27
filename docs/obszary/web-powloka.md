# Obszar: web-powloka (fundament interfejsu)

## Co robi

Powłoka aplikacji `web/` (Svelte 5 runes + TS + Vite) w wyglądzie stacji Programator_Czarek: tokeny, style globalne,
ikony, klient API, SSE transferów, stan (sesja, nawigacja, powiadomienia, motyw, układ, pasek), wspólne komponenty,
pasek boczny, nagłówek strony, wybór widoku i dynamiczne ładowanie stron.

## Pliki

| plik | rola |
|---|---|
| `web/src/styl/tokeny.css` | JEDYNE miejsce z kolorami i px: `KOLOR` (ciemny) i `_KOLORY_JASNE` (`[data-motyw="jasny"]`), `CZCIONKA`, `ODSTEP` (`--odstep-xs…sekcja`, skróty `--odstep-1…7`), `PROMIEN`, `WYSOKOSC`, `UKLAD`, kolory przycisków nieaktywnych (`WYLACZONY` ze stacji, przeliczone), szerokości kolumn `--kolumna-xs…xl`, progi `--prog-telefon/waski/sredni`, czasy i warstwy |
| `web/src/styl/globalne.css` | reset, typografia, przewijanie jak w stacji, zaznaczenie, klasy `.opis .etykieta .licznik .skroc .wiersz .siatka-kafli .tylko-czytnik` |
| `web/src/ikony/ikony.ts` | `IKONY` (SVG 24×24, kreska 2): ikony stacji pod polskimi nazwami + dorysowane (folder, plik, obraz, wideo, dzwiek, pdf, tekst, archiwum, pobierz, wgraj, link, udostepnij, uzytkownik(cy), ustawienia, dziennik, pulpit, szukaj, kosz, olowek, kopiuj, siec, tarcza, wykres, wyloguj, menu, zamknij, strzalka*, odswiez, kafelki, lista, ftp, serwer, globus, klucz, klodka, ksiezyc, slonce…); `NazwaIkony`, `IKONA_RODZAJU` (Rodzaj → ikona) |
| `web/src/api/klient.ts` | `api.get/post/put/usun`, `BladApi(kod, komunikat)`, `X-Skrytka: 1`, `adresPobrania`, `zapytanie`, `adresApi`, `trescBledu`; 401 → odświeżenie sesji |
| `web/src/api/transfery.ts` | `sluchajTransferow(naZdarzenie, naSume)` na EventSource; po zamknięciu przez przeglądarkę wznawia z przerwą 1 s…30 s |
| `web/src/stan/sesja.svelte.ts` | `sesja.stan`, `wczytano`, `blad`, `zalogowany`, `admin`, `odswiez()`, `wyloguj()` |
| `web/src/stan/nawigacja.svelte.ts` | `nawigacja.strona/parametry/publiczny`, `idz()`, `zastap()`; czyste `czytajHash`, `zbudujHash` (`#/pliki/<udzial>/<sciezka>`) |
| `web/src/stan/powiadomienia.svelte.ts` | `powiadom(tekst, rodzaj)`, lista toastów (czas z `--czas-toast`, błąd 2×) |
| `web/src/stan/format.ts` | `rozmiar`, `predkosc`, `czasTrwania`, `data`, `dataWzgledna` (+ `procent`, `liczbaCala`) po polsku |
| `web/src/stan/motyw.svelte.ts` | `motyw.biezacy`, `przelacz()`, `ustaw()`; wybór w localStorage, bez wyboru motyw z serwera |
| `web/src/stan/uklad.svelte.ts` | `uklad.szerokosc`, `ponizej(prog)`, `telefon`, `waski` (progi czytane z tokenów) |
| `web/src/stan/pasek.svelte.ts` | `pasek.zwiniety` (pamiętany), `szuflada` (telefon) |
| `web/src/komponenty/wspolne/*` | `Ikona`, `Przycisk`, `Karta`, `Pigulka`, `Kafel`, `Okno`, `MenuKontekstowe` (+ typ `PozycjaMenu`), `Pole`, `Przelacznik`, `Wybor` (+ `OpcjaWyboru`), `Tabela` (+ `KolumnaTabeli`), `PasekPostepu`, `Pusto`, `Powiadomienia` |
| `web/src/komponenty/PasekBoczny.svelte`, `NaglowekStrony.svelte` | pasek boczny stacji i nagłówek strony (`.ph`) |
| `web/src/App.svelte`, `web/src/main.ts` | rejestr stron `STRONY`, wybór widoku, `import.meta.glob('./strony/*.svelte')` |
| `web/sprawdz_styl.mjs` | strażnik stylu (`npm run straznik`) |

## Umowy dla innych agentów (uzupełnienie `docs/interfejs.md`)

1. **Strona renderuje własny `NaglowekStrony`** (`{tytul, podtytul?, ikona?, srodek? (snippet, kafle), akcje? (snippet)}`);
   App pokazuje nagłówek tylko dla strony, której plik jeszcze nie istnieje. Na telefonie nagłówek sam pokazuje przycisk
   szuflady menu.
2. **`Tabela`**: snippet `wiersz(w, i)` renderuje CAŁY `<tr>` z `<td>` w kolejności kolumn (zdarzenia i zaznaczenie po
   stronie strony; klasy `td.prawo/.srodek/.mono/.opis`, `tr.zaznaczony`). `szerokosc`: `'xs'|'s'|'m'|'l'|'xl'` albo
   wartość CSS bez px (`'30%'`). `ukryjPonizej`: `'telefon'|'waski'|'sredni'`. Opcjonalnie `sortKlucz`, `malejaco`
   (bindable), `pusto` (snippet).
3. **`Przycisk`** dodatkowo: `rozmiar?: 'maly'|'duzy'`, `typ?: 'button'|'submit'`, `tytul?` (dymek; wymagany przy samej ikonie).
4. **`Pole`**: `typ`, `podpowiedz` (placeholder), `blad`, `opis`, `ikona`, `wylaczone`, `wymagane`, `mono`,
   `autocomplete`, `ukryjEtykiete`, `autofocus`. **`Przelacznik`** `{etykieta, value, opis?, wylaczony?, onchange?}`.
   **`Wybor`** `{etykieta, value, opcje: (string|{wartosc, etykieta, wylaczona?})[], onchange?}`.
5. **`Okno`**: dodatkowo `szerokie?`, `naZamknij?`. **`Karta`**: `bezMarginesu?` (tabela do krawędzi).
   **`Pigulka`**: `kropka?`. **`Pusto`**: opcjonalne `children` (akcje).
6. Kolory stanów: `ok` zielony, `info` niebieski, `blad` czerwony, `ostrz` żółty, `wyl` szary (jak `KOLOR_STANU`).
7. Media queries nie przyjmują `var()`: responsywność przez `uklad.ponizej(prog)` albo `uklad.telefon/waski`.

## Strażnik stylu

`web/sprawdz_styl.mjs` (`npm run straznik`): błąd, gdy w `src/` poza `styl/tokeny.css` jest `#hex`, `rgb(`/`rgba(`,
`hsl(`/`hsla(` albo liczba z `px`/`rem` różna od zera (komentarze też). Sprawdzony wstrzyknięciem (plik z `#fff`, `rgb(`,
`10px`, `1.5rem` = 4 naruszenia, `0px` przepuszczone).

## Testy

`npm run build` OK, `npm run straznik` OK, `svelte-check` 0 błędów (2 ostrzeżenia w cudzym `KafelkiPlikow.svelte`).
Brak testów jednostkowych TS (projekt nie ma uruchamiacza testów, np. vitest); czyste funkcje do pokrycia:
`format.ts`, `czytajHash`/`zbudujHash`, `zapytanie`/`adresApi`.

## Czego nie zrobiono

1. Brak zrzutów obu motywów i porównania ze stacją (wymaga działającego serwera albo atrapy API).
2. Brak testów jednostkowych (brak vitest w `package.json`).
3. Zwinięty pasek nie wysuwa się pod myszą (jak 8.13 w stacji): tylko dymki z podpisem; rozwinięcie przyciskiem
   „Rozwiń menu”. Poniżej progu `--prog-waski` pasek zawsze zwinięty.
4. Motyw zapisany tylko w przeglądarce (nie wysyłany do `PUT /api/ustawienia`).

## Fala 2 (standard „bigtech”, PLAN.md sekcja H)

### Paleta poleceń i skróty (`web/src/stan/skroty.svelte.ts`, `web/src/komponenty/paleta/*`)

1. `zarejestrujPolecenie({id, nazwa, ikona?, skrot?, akcja, widoczne?, grupa?, slowa?}): () => void`, czyli dodanie
   polecenia (to samo `id` zastępuje poprzednie). Zwraca funkcję wyrejestrowania, którą strona woła przy zniknięciu, np.
   `$effect(() => zarejestrujPolecenie({...}))` (efekt zwraca funkcję sprzątającą). `widoczne()` ogranicza polecenie
   (np. tylko przy prawie zapisu), `grupa` grupuje w palecie i w pomocy, `slowa` to dodatkowe słowa wyszukiwania.
2. Skrót: `'Ctrl+K'`, `'Shift+N'`, `'Alt+1'`, `'Delete'`, `'F2'`, `'N'` (Ctrl = Ctrl albo Cmd). Skrót bez Ctrl/Alt nie
   działa, gdy fokus jest w polu tekstowym. Gdy otwarte jest okno modalne (`dialog[open]`), działa tylko Ctrl+K.
3. Wbudowane: **Ctrl+K** paleta, **/** szukanie (uruchamia polecenie o id `SZUKAJ` = `'szukaj'`, które rejestruje strona
   z wyszukiwarką, a bez niego otwiera paletę), **?** okno pomocy skrótów, **Esc** zamyka.
4. Polecenia powłoki (App.svelte): `Przejdź: <strona>` (Alt+1…6), trzy motywy, „Zwiń lub rozwiń menu” (Ctrl+B),
   „Skróty klawiszowe”, „Wyloguj”. Polecenia stron (np. „Nowy folder”, „Wgraj”) rejestrują ich właściciele.
5. `Paleta.svelte`: rozmyte dopasowanie (`paleta/rozmyte.ts`: kolejność liter, bez wielkości liter i polskich znaków,
   premie za ciąg, początek słowa i tekstu), podświetlenie liter, strzałki, Enter, klik. `PomocSkrotow.svelte`: okno „?”
   z grupami. `Klawisze.svelte`: `{skrot}` jako `<kbd>`.
6. `skroty.paleta`, `skroty.pomoc`: stan otwarcia (np. przycisk „Szukaj” w nagłówku może zrobić `skroty.otworzPalete()`).

### Motyw automatyczny (`stan/motyw.svelte.ts`)

`motyw.wybor`: `'auto' | 'ciemny' | 'jasny'` (domyślnie `auto`, wg `prefers-color-scheme`, reaguje na zmianę w systemie);
`motyw.biezacy` to motyw wynikowy (ten trafia do `data-motyw`). `przelacz()` przechodzi kolejno: automatyczny, ciemny,
jasny. `NAZWY_MOTYWOW` zawiera podpisy. Meta `theme-color` = `--kolor-pasek`.

### PWA (`web/public/`)

1. `manifest.webmanifest` (nazwa Skrytka, `standalone`, kolory = `--kolor-okno` i `--kolor-pasek` wpisane jako hex,
   jedyny wyjątek strażnika poza tokenami), ikony PNG `ikona-32/192/512.png`, `ikona-maskowalna-512.png`,
   `apple-touch-icon.png` (niebieski kwadrat `--kolor-niebieski-tlo` z białym dyskiem w stylu ikon). Ikony wygenerował
   skrypt node bez bibliotek (nie ma go w projekcie; zmiana ikony = nowy generator albo edytor grafiki).
2. `sw.js`: zapisuje TYLKO powłokę (`./`, `index.html`, manifest, ikony, `/assets/*.js|css`). **Nigdy** nie przechwytuje
   `/api/` (dane, pobrania, SSE) ani innych adresów, nie obsługuje też metod innych niż GET. Nawigacja: najpierw sieć,
   bez sieci zapisany `index.html`. Zasoby z hashem: najpierw pamięć. Nowa wersja pamięci = zmiana `PAMIEC`.
3. `main.ts` rejestruje `/sw.js` tylko w wersji zbudowanej (`import.meta.env.PROD`) i w bezpiecznym kontekście (HTTPS
   albo localhost). Dokłada też `<link rel="manifest">`, ikonę strony i `apple-touch-icon`, gdy nie ma ich w `index.html`.

### Kod QR (`komponenty/wspolne/KodQR.svelte`, `kodqr.ts`)

`<KodQR tekst="https://…" rozmiar?: 's'|'m'|'l' etykieta? />`. Własny koder: tryb bajtowy (UTF-8), korekcja M,
wersje 1–10 (do 213 bajtów; dłuższy tekst = stan pusty z komunikatem), maska o najmniejszej karze (reguły N1–N4).
Rysunek SVG ze strefą ciszy 4 moduły, kolory `--kolor-qr-tlo` i `--kolor-qr-modul` (ciemne na jasnym w obu motywach).
Sprawdzenie (skrypt node, bo w projekcie nie ma uruchamiacza testów):
- Reed-Solomon na wektorze „HELLO WORLD” 1-M z normy (dane 32,91,…,17 dają korekcję 196,35,39,119,235,215,231,226,93,23);
- 15-bitowa informacja o formacie dla poziomu M i 8 masek oraz 18-bitowa informacja o wersji dla v7–v10, zgodne
  z tabelami normy;
- kody danych trybu bajtowego dla „hello”;
- zapis i odczyt w obie strony (zdjęcie maski i odczyt w kolejności zapisu) dla 6 tekstów (1…213 B, w tym UTF-8)
  i wszystkich 8 masek;
- rozmiary wersji, odrzucenie 214 B, wzory pozycji i ciemny moduł.

Wynik: wszystko OK. Nie sprawdzono skanowania prawdziwym telefonem.

### Szkielet ładowania (`komponenty/wspolne/Szkielet.svelte`)

`{linie?: number = 3, wysokosc?: 'wiersz' | 'kafel' | 'wykres'}`: pulsujące paski (ostatni krótszy), `aria-busy`.

### Nowe tokeny i ikony

Tokeny: `--wykres-wysokosc`, `--pierscien-rozmiar`, `--mini-wykres-wysokosc/szerokosc`, `--kolor-qr-tlo/modul`,
`--qr-s/m/l`, `--uklad-paleta-w/gora/lista-h`. Ikony: `motyw_auto`, `qr`, `polecenie`, `enter`.

### Strażnik

Sprawdza teraz `src/` i `public/` (pliki .svelte, .ts, .css, .js, .html, .svg, .webmanifest). Wyjątki to
`src/styl/tokeny.css` i `public/manifest.webmanifest`. Sprawdzone wstrzyknięciem (`public/_p.svg` z `#abc` = błąd).
`src/vite-env.d.ts` zawiera typy Vite i Svelte (odwołanie z `main.ts` usunięte).

### Poprawka: pętla efektów przy rejestracji poleceń

Objaw: przejście między stronami kończyło się `effect_update_depth_exceeded`, a nowa strona zostawała na
„Wczytywanie…”. Przyczyna: `zarejestrujPolecenie` czytało i zapisywało tablicę `$state` poleceń, więc efekt, który je
wołał, zaczynał zależeć od rejestru i sam się restartował. Teraz rejestr to zwykła `Map`, a zmianę ogłasza licznik
`wersja`, zwiększany w `untrack()`. Rejestracja nic nie czyta reaktywnie i można ją bezpiecznie wołać z `$effect`.
Wyrejestrowanie usuwa tylko własne polecenie (nie nowsze o tym samym `id`).

Sprawdzenie: `npm run dev` z atrapą API na 127.0.0.1:8080 (skrypt node w scratchpadzie: `/api/stan` = zalogowany admin,
reszta pusta) i przełączanie stron kliknięciami w pasku w przeglądarce (8–12 przejść, w tym Pliki ↔ Pulpit ↔ Dziennik).
Wynik: 0 błędów, żadna strona nie zostaje na „Wczytywanie”, Ctrl+K otwiera paletę. Wstrzyknięcie (zapis licznika bez
`untrack`) odtwarza błąd: 5 × `effect_update_depth_exceeded` i strona na „Wczytywanie”.
