# Obszar web-monitor: strona Monitor (zasoby serwera na żywo, analityka)

## Co robi

Strona **Monitor** (`#/monitor`, tylko admin, w pasku bocznym zaraz po Pulpicie, ikona `wykres`). Wygląd jak strona
Prąd stacji Programator_Czarek (wykresy na żywo, kafle liczb) połączony z dashboardem UniFi (półokrągłe wskaźniki,
mapa aktywności). Kolory i rozmiary tylko z `tokeny.css`.

1. **Pasek systemu** (`PasekSystemu`): host, system i architektura, rdzenie, RAM, czas pracy liczony na żywo,
   pigułka „Na żywo” / „Łączenie”.
2. **Obciążenie** (`KartaObciazenia`): trzy `Wskaznik` (CPU %, RAM % z bieżącej próbki, zajętość dysków udziałów
   jako suma `GET /api/pulpit` → `udzialy`, odświeżana co minutę); barwa łuku: seria → żółty od 70% → czerwony od 90%.
   Obok kafle (`Kafel`): połączenia, aktywne transfery, gorutyny, pamięć procesu.
3. **Wykresy** z przełącznikiem okresu (`Segmenty`): 10 min (historia 1 s, przesuwanie na żywo) albo 24 h (historia
   1 min). Procesor i RAM = `WykresLiniowy`; sieć (wysyłanie/odbiór) i dysk (odczyt/zapis) = `WykresDwuSerii`
   (dwie serie na wspólnej osi, legenda z bieżącą wartością, dymek z obiema). Sieć osobno dla każdego interfejsu
   (`KartaInterfejsow`, bez pętli zwrotnej) z pigułką rodzaju (LAN, WireGuard, Publiczny).
4. **Analityka**: `KartaMapy` = `MapaCiepla` pobrań z `GET /api/dziennik/mapa?od&do` (7 / 30 / 90 dni, pon..niedz ×
   godzina, kolor od `--kolor-karta` do `--kolor-niebieski` przez `color-mix`, dymek z bajtami, legenda);
   `KartyIndeksu` = zajętość wg rodzaju plików (`Pierscien`, wybór udziału) + ranking udziałów (`ListaRankingu`)
   i stan indeksu każdego udziału (zindeksowano kiedy, pliki, foldery, bajty) z `GET /api/indeks/statystyki`.
5. **Stany**: ładowanie = szkielety (`Szkielet`, pulsowanie wyłączone przy `prefers-reduced-motion`); adres
   jeszcze nieistniejący na serwerze (404/405/501) = `Pusto` z opisem, reszta strony działa (każda karta analityki
   ładuje się i psuje niezależnie). SSE wznawia połączenie z rosnącą przerwą (1 s → 30 s); stara lub powtórzona
   ramka jest pomijana.
6. **Responsywnie**: siatki `auto-fill` (wykresy i interfejsy od `--uklad-okno-w`, kafle od `--uklad-kafel-min-w`),
   rzędy dwukolumnowe przechodzą w jedną kolumnę poniżej progu `--prog-waski` (`uklad.waski`).

## Pliki

- `web/src/strony/Monitor.svelte` (strona),
- `web/src/api/monitor.ts` (typy `Probka`, `SiecInterfejsu`, `InfoSystemu`, `StanMonitoraApi`, `StatystykiIndeksu`,
  `StatystykiUdzialu`, `MapaPobran`; `pobierzMonitor`, `pobierzStatystykiIndeksu`, `pobierzMape`, `sluchajMonitora`),
- `web/src/komponenty/monitor/`: `stan.svelte.ts` (klasa `StanMonitora`, serie do wykresów), `stale.ts`,
  `WykresDwuSerii.svelte`, `PasekSystemu.svelte`, `KartaObciazenia.svelte`, `KartaInterfejsow.svelte`,
  `KartaMapy.svelte`, `KartyIndeksu.svelte`, `Segmenty.svelte`, `Szkielet.svelte`,
- `web/src/komponenty/wykresy/Wskaznik.svelte` `{wartosc: 0..1 | null, etykieta, tekst?, opis?, progi?, barwa?}`,
- `web/src/komponenty/wykresy/MapaCiepla.svelte` `{komorki: number[7][24], format, nazwa?}`,
- `web/src/App.svelte`: jedna pozycja `monitor` w rejestrze `STRONY`.

## Stałe

`komponenty/monitor/stale.ts`: `MONITOR` (okno 10 min i 24 h, zapas 5 s, krok minutowy, odświeżanie dysków 60 s
i indeksu 5 min, progi wskaźników 0,7 / 0,9, 8 pozycji rankingu), `OKRESY`, `ZAKRESY_MAPY` (7, 30, 90 dni),
`NAZWY_RODZAJOW`, `RODZAJE_INTERFEJSU`. Geometria wskaźnika i mapy w jednostkach viewBox / siatki w plikach
komponentów.

## Testy

`npm run build` OK, `npm run straznik` OK, `svelte-check`: 0 błędów i 0 ostrzeżeń w plikach obszaru (2 błędy
w `strony/Pliki.svelte`, obszar web-pliki). Nie sprawdzono w przeglądarce z serwerem (adresy Fali 2 jeszcze
nie istnieją).

## Czego nie zrobiono

1. Brak testów jednostkowych interfejsu (projekt nie ma uruchamiacza testów JS).
2. `cpu_proc` traktowany jako procent 0..100 (wg nazwy); gdyby serwer podawał ułamek 0..1, poprawić w
   `KartaObciazenia` i wykresie procesora.
3. `WykresLiniowy` ma stałą podziałkę czasu 1 min na żywo: w oknie 10 min na wąskiej karcie opisy osi mogą się
   zagęszczać (`WykresDwuSerii` dobiera krok do szerokości).
