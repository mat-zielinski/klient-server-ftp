# Obszar: ikona aplikacji

## Co robi

Jedna ikona Skrytki we wszystkich miejscach: plik .exe (Windows), zasobnik systemowy, PWA i favicon,
Linux (.desktop), logo w interfejsie. Projekt: zaokrąglony kwadrat z gradientem niebieski → granat (jak tokeny
`--kolor-niebieski-tlo`), na nim biały sejf (zawiasy z lewej, nóżki) z wyciętą strzałką pobierania nad tacką.
Do 24 px symbol jest uproszczony (bez zawiasów i tacki, większa strzałka), żeby był czytelny w 16×16.

Warianty:

- pełna kolorowa (`skrytka.svg`, do 24 px `skrytka-mala.svg`),
- maskowalna PWA (`skrytka-maskowalna.svg`): tło na cały kwadrat, symbol w strefie bezpiecznej (koło 80%),
- apple-touch-icon: tło na cały kwadrat (iOS sam zaokrągla), symbol większy,
- zasobnik (`zasobnik.svg`): biały symbol na przezroczystym,
- zasobnik aktywny (`zasobnik-aktywny.svg`): jak wyżej + zielona kropka (`--kolor-zielony`) z przezroczystą
  szczeliną, gdy trwa transfer,
- zasobnik ciemny (`zasobnik-ciemny.svg`, `zasobnik-ciemny-aktywny.svg`): ten sam symbol granatowy (#0f1a2e)
  na jasny pasek zadań (motyw jasny Windows), aktywny z tą samą zieloną kropką.

## Pliki

- `skrypty/ikona.mjs`: generator (Node, bez bibliotek). Projekt jest w kodzie (stała `SYMBOL`), z niego powstają
  SVG i rasteryzacja. Własny rasteryzator: wielokąty z zaokrąglonymi narożami i okręgi spłaszczone do odcinków,
  linie skanowania z nadpróbkowaniem 4×4, reguły nonzero/evenodd, gradient liniowy, tryb „wymaż” (szczelina wokół
  kropki); PNG przez `zlib.deflateSync` + własne CRC32; ICO z osadzonymi PNG.
  Uruchomienie: `node skrypty/ikona.mjs` (opcjonalnie `--podglad <katalog>` zapisuje PNG wszystkich wariantów
  w rozmiarach 16–256 do obejrzenia).
- `assets/ikona/`: źródła SVG, `skrytka.ico` (16, 24, 32, 48, 64, 128, 256), `skrytka-256.png`,
  `zasobnik.ico`, `zasobnik-aktywny.ico`, `zasobnik-ciemny.ico`, `zasobnik-ciemny-aktywny.ico` (16, 20, 24, 32, 48).
- `assets/linux/skrytka.png` (256) i `assets/linux/skrytka.desktop` (Icon=skrytka, Exec=xdg-open https://127.0.0.1:8443).
- `web/public/`: `ikona-32.png`, `ikona-192.png`, `ikona-512.png`, `ikona-maskowalna-512.png`,
  `apple-touch-icon.png` (180). Nazwy i manifest bez zmian.
- `web/src/ikony/logo.ts`: `LOGO_SVG` (wygenerowany), kolory tylko z tokenów: gradient
  `--kolor-niebieski` → `--kolor-niebieski-tlo`, przyciemnienie `--kolor-cien`, symbol `--kolor-na-ciemnym`.
- `wewn/aplikacja/ikony/`: pakiet `ikony` z `//go:embed`: `Zasobnik`, `ZasobnikAktywny` (białe, ciemny pasek),
  `ZasobnikCiemny`, `ZasobnikCiemnyAktywny` (granatowe, jasny pasek), `Aplikacja` (PNG 256).
  Wybór wariantu według motywu paska (rejestr `SystemUsesLightTheme`) należy do zasobnika.
  Pliki w tym katalogu też zapisuje generator.
- `cmd/skrytka/winres/winres.json`: zasoby Windows (ikona `APP`, wersja: ProductName Skrytka, FileDescription
  „Skrytka — serwer plików”, CompanyName pusty; manifest: dpi „per monitor v2”, „as invoker”, long-path-aware,
  common controls v6).
- `cmd/skrytka/rsrc_windows_amd64.syso`, `rsrc_windows_arm64.syso`: wygenerowane zasoby.

## Odświeżanie

```
node skrypty/ikona.mjs
GOTOOLCHAIN=local _go-portable/bin/go.exe tool go-winres make --in cmd/skrytka/winres/winres.json --out cmd/skrytka/rsrc --arch amd64,arm64
```

Przy zmianie wersji programu (`konfiguracja.Wersja`) trzeba zmienić wersję w `winres.json` (albo podać
`--file-version` / `--product-version` przy `make`) i odtworzyć .syso.

## Testy i sprawdzenie

Bez testów jednostkowych (zgodnie z poleceniem). Sprawdzono ręcznie: podgląd 16/20/24/32/48/256 na jasnym i ciemnym
tle, ikona i informacje o wersji w zbudowanym .exe (`ExtractAssociatedIcon`, `VersionInfo`).

## Czego nie zrobiono

- Logo nie jest podłączone w interfejsie (robi to sesja główna).
- Wersja w `winres.json` wpisana ręcznie (0.1), nie jest czytana z `konfiguracja.Wersja`.
