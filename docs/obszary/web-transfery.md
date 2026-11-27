# Obszar web-transfery: Pulpit, Dziennik, wykresy SVG

## Co robi

1. **Wykresy własne w SVG** (`web/src/komponenty/wykresy/`), bez bibliotek, kolory i czcionki tylko z tokenów
   (`--kolor-siatka`, `--kolor-opis`, barwy serii `--kolor-niebieski`, `-zielony`, `-fioletowy`, `-pomaranczowy`,
   `-rozowy`, `-zolty`). Wygląd jak wykresy strony Prąd stacji (siatka, opisy osi mono) z wypełnieniem gradientem
   jak w UniFi.
   - `WykresLiniowy` `{punkty: PunktCzasu[], formatY, formatX?, formatDymka?, oknoMs?, barwa?, nazwa?, os?}`:
     gładka linia monotoniczna, gradient, siatka, dymek z kropką pod kursorem; z `oknoMs` oś czasu przesuwa się
     płynnie co klatkę (requestAnimationFrame), koniec osi opóźniony o 1 s, więc linia zawsze dochodzi do krawędzi.
     Maksimum osi zmienia się płynnie (`Tween`).
   - `WykresSlupkowy` `{punkty: PunktSlupka[], seriaA, seriaB}`: słupki serii A (oś lewa) i linia z kropkami serii B
     (oś prawa), podświetlenie kategorii i dymek z obiema wartościami, legenda.
   - `Pierscien` `{pozycje: Pozycja[], format, podpisSumy?}`: łuki z przerwami, środek pokazuje sumę albo procent
     wskazanego wycinka, legenda z liczbą i procentem.
   - `MiniWykres` `{wartosci: number[], barwa?, opis?}`: sparkline (viewBox rozciągany, linia o stałej grubości).
   - `ListaRankingu` `{pozycje: Pozycja[], format, barwa?, pusto?}`: miejsce, nazwa, wartość, pasek proporcjonalny.
   - `Dymek` (wspólny dymek wykresów, pozycja w ułamkach rozmiaru, bez px), `skala.ts` (skale, ładne podziałki
     z osią bajtów w krokach 1024 i osią liczb całkowitych, ścieżka monotoniczna, `obserwujRozmiar` na
     ResizeObserver), `stale.ts` (geometria w jednostkach SVG, okno na żywo 5 min, krok 1 s).
2. **Pulpit** (`strony/Pulpit.svelte`, tylko admin; inni widzą stan pusty): kafle (aktywne transfery z podziałem
   pobrania/wgrania, łączna prędkość ze szczytem z 5 min, dziś pobrano bajty i liczba, czas pracy liczony na żywo),
   wykres prędkości łącznej na żywo (zdarzenia SSE `suma`), tabela aktywnych transferów (kto, IP, protokół pigułką,
   plik z kierunkiem, PasekPostepu, prędkość, pozostało, mini wykres z `historia`, Przerwij = `DELETE
   /api/transfery?id`), karta adresów serwera (`GET /api/siec`: LAN / WireGuard / publiczny, adresy HTTPS i FTPS
   z kopiowaniem, porty ze stanem, tryb, odcisk certyfikatu) i zajętość udziałów (`GET /api/pulpit`, odświeżane co
   30 s i po każdym zakończonym transferze).
3. **Dziennik** (`strony/Dziennik.svelte`): segmenty zakresu (Dziś, 7 dni, 30 dni, Własny z polami dat) i krok
   (godzina/dzień, domyślny wg zakresu), przycisk Odśwież, kafle sum, `WykresSlupkowy` z `/api/dziennik/statystyki`
   (bajty + liczba pobrań), `Pierscien` bajtów wg protokołu (pusty protokół = „Nieznany”), top pliki, top klienci (tylko admin), tabela wpisów
   (`/api/dziennik`) z filtrem rodzaju (stała lista: pobranie, wgranie, zmiana, udostepnienie, logowanie, logowanie_blad, administracja, serwer, z polskimi nazwami), użytkownika (tylko admin) i szukaniem
   (zwłoka 350 ms), pigułki protokołu i wyniku, Eksport CSV (`/api/dziennik/eksport?od&do`). Gdy `!admin` interfejs
   chowa kolumnę „Kto”, filtr użytkownika, kafel i ranking klientów (filtruje serwer).

## Pliki

`web/src/strony/Pulpit.svelte`, `web/src/strony/Dziennik.svelte`,
`web/src/komponenty/transfery/`: `monitor.svelte.ts` (stan transferów i serii z SSE), `TabelaTransferow.svelte`,
`KartaAdresow.svelte`, `KartaUdzialow.svelte`, `WyborZakresu.svelte`, `TabelaDziennika.svelte`, `zakres.ts`
(granice zakresu, opisy punktów), `typy.ts` (`InfoSieci`, mapowanie pigułek, `odmiana`, `kopiuj`), `stale.ts`,
`web/src/komponenty/wykresy/`: jak wyżej.

## Stałe

`komponenty/transfery/stale.ts`: `PULPIT` (odświeżanie 30 s, 1,5 s po końcu transferu, progi zajętości 0,8 / 0,95),
`DZIENNIK` (limit 500 wpisów, 8 pozycji rankingu, zwłoka szukania 350 ms, 7 i 30 dni).
`komponenty/wykresy/stale.ts`: `WYKRES` (marginesy, podziałki, słupki, pierścień), `NA_ZYWO` (krok 1 s, opóźnienie
1 s, okno 5 min, podziałka czasu 1 min). Wysokości kontenerów: zmienne `--wykres-wysokosc`, `--pierscien-rozmiar`,
`--mini-wykres-wysokosc`, `--mini-wykres-szerokosc` z wartościami zapasowymi z istniejących tokenów (do dodania
w `tokeny.css` przez web-powloka).

## Testy

`npm run build` OK, `svelte-check` bez błędów i ostrzeżeń w plikach obszaru, strażnik stylu OK. Ręczne sprawdzenie
funkcji `podzialki`, `sciezkaGladka`, `najblizszy`, `granice`, `odmiana` (esbuild + node, poza projektem).

## Czego nie zrobiono

1. Brak testów jednostkowych interfejsu (projekt nie ma uruchamiacza testów JS).
2. Nie sprawdzono na żywo z serwerem (SSE, eksport CSV, przerwanie) ani zrzutów w obu motywach.
3. Statystyki liczą tylko pobrania (każdy wynik); `protokoly` to bajty wg protokołu (ustalenie z sesją główną).
4. Kafel „Klienci” liczy pozycje rankingu z serwera (kontrakt nie ma liczby wszystkich klientów).
