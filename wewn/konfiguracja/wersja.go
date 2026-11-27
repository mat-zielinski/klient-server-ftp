// Pakiet konfiguracja trzyma WSZYSTKIE stałe, ścieżki i wersję aplikacji.
// Każdy obszar dopisuje swoje stałe we własnym pliku (konta.go, ftp.go, siec.go …).
package konfiguracja

// Wersja aplikacji: podbijana przy każdej zmianie, wpis w zmiany.md.
const Wersja = "0.2"

// NazwaAplikacji pokazywana w tytule, stopce i dzienniku.
const NazwaAplikacji = "Skrytka"

// WersjaBudowy ustawia budowanie wydania (-ldflags "-X skrytka/wewn/konfiguracja.WersjaBudowy=v0.1.0");
// pusta = budowa lokalna, pokazywana jest Wersja.
var WersjaBudowy string
