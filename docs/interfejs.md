# Interfejs (web/): podział plików i wspólne komponenty

Svelte 5 (runes) + TypeScript + Vite. Wzór wyglądu: stacja Programator_Czarek (`C:\Users\mateu\Desktop\Programator_Czarek`,
`programator/qt/motyw.py`, `programator/qt/okno.py`, `programator/qt/arkusz.py`, `programator/styl/ikony_svg.py`)
połączona z charakterem UniFi: granatowe tło, karty, **pasek boczny po lewej jak w stacji** (ikona + podpis,
zaznaczona pozycja tłem `aktywny` i akcentem), nagłówek strony z tytułem i kaflami, zwarte tabele, pigułki stanów.

## Zasady (jak w stacji)

1. Żaden komponent nie wpisuje koloru ani rozmiaru na sztywno: tylko zmienne CSS z `src/styl/tokeny.css`
   (`var(--kolor-panel)`, `var(--odstep-2)` …). Strażnik odrzuca `#rrggbb`, `rgb(`, `px` poza `tokeny.css`
   (wyjątek: `0`, `1px` ramki przez `var(--ramka)`).
2. Dwa motywy: ciemny (domyślny, kolory z `KOLOR` stacji) i jasny (`_KOLORY_JASNE`), atrybut `data-motyw` na `<html>`.
3. Ikony tylko z `src/ikony/ikony.ts` (SVG liniowe 24×24, styl stacji); bez emoji.
4. Teksty po polsku, zdania wielką literą, bez długich myślników.
5. Responsywność bez przeskakiwania: pasek boczny zwęża się do samych ikon (podpis w dymku), tabela chowa kolumny
   wg progów z tokenów; na telefonie pasek boczny jako szuflada z przyciskiem.
6. Plik ≤ 600 linii, komponent robi jedną rzecz.

## Właściciele plików (agenci nie ruszają cudzych plików)

| agent | pliki |
|---|---|
| **web-powloka** | `src/main.ts`, `src/App.svelte`, `src/styl/*`, `src/ikony/*`, `src/api/klient.ts`, `src/api/transfery.ts` (SSE), `src/stan/*`, `src/komponenty/wspolne/*`, `src/komponenty/PasekBoczny.svelte`, `src/komponenty/NaglowekStrony.svelte` |
| **web-pliki** | `src/strony/Pliki.svelte`, `src/komponenty/pliki/*` (tabela, kafelki, okruszki, podgląd, menu kontekstowe z „Pobierz przez…”, wgrywanie przeciągnij-upuść z wznawianiem, panel „Moje pobrania”) |
| **web-transfery** | `src/strony/Pulpit.svelte`, `src/strony/Dziennik.svelte`, `src/komponenty/transfery/*`, `src/komponenty/wykresy/*` (własne wykresy SVG, bez bibliotek) |
| **web-admin** | `src/strony/Logowanie.svelte`, `src/strony/Kreator.svelte`, `src/strony/Uzytkownicy.svelte`, `src/strony/Ustawienia.svelte`, `src/strony/Udostepnienia.svelte`, `src/strony/Publiczny.svelte`, `src/komponenty/admin/*` |

`src/api/typy.ts` = kontrakt (sesja główna).

## Wspólne komponenty i moduły (robi web-powloka; inni używają według tych sygnatur)

| plik | eksport / właściwości |
|---|---|
| `src/api/klient.ts` | `api.get<T>(adres, parametry?)`, `api.post<T>(adres, cialo?, parametry?)`, `api.put<T>(…)`, `api.usun<T>(adres, parametry?)`; błąd = `throw new BladApi(kod, komunikat)`; nagłówek `X-Skrytka: 1`; `adresPobrania(udzial, sciezka, podglad?)` |
| `src/api/transfery.ts` | `sluchajTransferow(naZdarzenie: (z: ZdarzenieTransferu) => void, naSume: (s: SumaTransferow) => void): () => void` (EventSource, samo wznawia) |
| `src/stan/sesja.svelte.ts` | `sesja` (runes `$state`): `stan: StanAplikacji \| null`, `odswiez()`, `wyloguj()` |
| `src/stan/nawigacja.svelte.ts` | `nawigacja.strona: string`, `nawigacja.parametry`, `idz(strona, parametry?)` (hash `#/pliki/<udzial>/<sciezka>`) |
| `src/stan/powiadomienia.svelte.ts` | `powiadom(tekst, rodzaj: 'ok' \| 'info' \| 'blad' \| 'ostrz')` |
| `src/stan/format.ts` | `rozmiar(b)`, `predkosc(bps)`, `czasTrwania(s)`, `data(iso)`, `dataWzgledna(iso)` (po polsku) |
| `src/ikony/ikony.ts` | `IKONY: Record<NazwaIkony, string>` (ścieżki SVG); `Ikona.svelte` w `komponenty/wspolne` |
| `src/komponenty/wspolne/Ikona.svelte` | `{nazwa, rozmiar?: 's'\|'m'\|'l'}` |
| `…/Przycisk.svelte` | `{rodzaj?: 'zwykly'\|'glowny'\|'niebezpieczny'\|'duch', ikona?, wylaczony?, onclick, children}` |
| `…/Karta.svelte` | `{tytul?, ikona?, akcje? (snippet), children}` |
| `…/Pigulka.svelte` | `{stan: 'ok'\|'info'\|'blad'\|'ostrz'\|'wyl', children}` |
| `…/Kafel.svelte` | `{etykieta, wartosc, opis?, ikona?, stan?}` (kafel liczby jak w stacji) |
| `…/Okno.svelte` | dialog modalny `{tytul, otwarte (bindable), children, stopka? (snippet)}` |
| `…/MenuKontekstowe.svelte` | `{x, y, pozycje: {etykieta, ikona?, opis?, onclick, wylaczona?, podmenu?: pozycje[], separator?}[], zamknij}` |
| `…/Pole.svelte`, `…/Przelacznik.svelte`, `…/Wybor.svelte` | pola formularzy `{etykieta, value (bindable), …}` |
| `…/Tabela.svelte` | `{kolumny: {klucz, etykieta, szerokosc?, wyrownanie?, ukryjPonizej?}[], wiersze, wiersz (snippet), naSortuj?}` |
| `…/PasekPostepu.svelte` | `{wartosc 0..1 \| null (nieokreślony), stan?}` |
| `…/Pusto.svelte` | stan pusty `{ikona, tytul, opis?}` |
| `src/komponenty/NaglowekStrony.svelte` | `{tytul, podtytul?, ikona? (przed podtytułem), srodek? (snippet: kafle, chowany poniżej progu `waski`), akcje? (snippet: prawy obszar, łamie się pod tytuł)}`; każda strona renderuje go sama u góry; na telefonie sam dokłada przycisk szuflady menu |

Strony (klucz nawigacji → plik): `pulpit` Pulpit (admin), `pliki` Pliki, `dziennik` Dziennik, `udostepnienia`
Udostepnienia, `uzytkownicy` Uzytkownicy (admin), `ustawienia` Ustawienia (admin). Bez sesji: `Logowanie`,
w trybie kreatora `Kreator`, pod `/s/<id>` `Publiczny`. Każda strona: `export default`, bez właściwości.

## Pobieranie przez wybrany protokół (prawy przycisk)

Menu kontekstowe pliku (web-pliki): Otwórz / Podgląd, **Pobierz przez… ▸** (podmenu z `GET /api/protokoly`:
Przeglądarka HTTPS, Klient FTP (FTPS) z adresem do skopiowania i instrukcją, Kopiuj link bezpośredni, Jako ZIP,
Link publiczny…), Kopiuj ścieżkę, Zmień nazwę, Usuń, Właściwości. Opcja niedostępna = wyszarzona z powodem w opisie.

## Podgląd transferów (obowiązkowo po obu stronach)

1. Klient: pływający panel „Moje pobrania” (prawy dół, zwijany) na stronie Pliki: każdy transfer z nazwą,
   protokołem, paskiem postępu, prędkością, pozostałym czasem i mini wykresem prędkości (60 s).
2. Serwer (admin): Pulpit z kaflami (aktywne, łączna prędkość, dziś pobrano), wykresem prędkości łącznej na żywo
   (5 min), tabelą aktywnych transferów (kto, IP, protokół, plik, postęp, prędkość, przerwij) i adresami serwera.
3. Dziennik pobrań: wykres słupkowy bajtów i liczby pobrań w czasie (godziny / dni), pierścień protokołów, top pliki,
   top klienci, tabela wpisów z filtrem i eksportem CSV.
