// Pakiet ikony osadza ikony aplikacji: zasobnik Windows (zwykla i aktywna) oraz ikona PNG aplikacji.
// Pliki generuje skrypt skrypty/ikona.mjs (zrodlo projektu: assets/ikona); nie edytuj ich recznie.
package ikony

import _ "embed"

// Zasobnik to ikona zasobnika systemowego (ICO 16/20/24/32/48, biala na przezroczystym).
//
//go:embed zasobnik.ico
var Zasobnik []byte

// ZasobnikAktywny to ikona zasobnika w czasie trwania transferu (biala z zielona kropka).
//
//go:embed zasobnik-aktywny.ico
var ZasobnikAktywny []byte

// Aplikacja to pelna kolorowa ikona aplikacji (PNG 256x256).
//
//go:embed skrytka.png
var Aplikacja []byte

// ZasobnikCiemny to ikona zasobnika na jasny pasek zadan (granatowa na przezroczystym).
//
//go:embed zasobnik-ciemny.ico
var ZasobnikCiemny []byte

// ZasobnikCiemnyAktywny to ciemna ikona zasobnika w czasie transferu (z zielona kropka).
//
//go:embed zasobnik-ciemny-aktywny.ico
var ZasobnikCiemnyAktywny []byte
