// Stałe strony Pliki (jedno miejsce na liczby i limity obszaru web-pliki).

/** Rozmiar kawałka wgrywania (zgodny z limitem PUT /api/wgraj). */
export const KAWALEK_WGRYWANIA = 8 * 1024 * 1024
/** Ile plików wgrywa się naraz. */
export const ROWNOLEGLE_WGRYWANIA = 2
/** Ile razy ponawiać kawałek po błędzie sieci, zanim plik dostanie błąd. */
export const PONOWIENIA_KAWALKA = 3
/** Przerwa przed ponowieniem kawałka (ms). */
export const PRZERWA_PONOWIENIA_MS = 1500
/** Po ilu ms zakończony wpis znika z kolejki wgrywania i z panelu „Moje pobrania”. */
export const ZNIKANIE_ZAKONCZONYCH_MS = 6000
/** Opóźnienie wyszukiwania po ostatnim naciśnięciu klawisza (ms). */
export const OPOZNIENIE_SZUKANIA_MS = 200
/** Największy plik tekstowy pokazywany w podglądzie (bajty). */
export const LIMIT_PODGLADU_TEKSTU = 1024 * 1024
/** Długość historii prędkości w mini wykresie (próbki co 1 s). */
export const DLUGOSC_HISTORII = 60
/** Domyślna ważność linku publicznego (dni). */
export const WAZNOSC_LINKU_DNI = 7
/** Klucz pamięci przeglądarki dla wybranego widoku (lista / kafelki). */
export const KLUCZ_WIDOKU = 'skrytka.pliki.widok'
/** Klucz pamięci przeglądarki dla zwinięcia panelu „Moje pobrania”. */
export const KLUCZ_PANELU = 'skrytka.pliki.panel_zwiniety'
/** Ile wierszy ponad i pod widocznym pasem renderuje wirtualizacja. */
export const BUFOR_WIRTUALIZACJI = 8
/** Ile wpisów renderować przed pierwszym pomiarem. */
export const POCZATKOWE_WIERSZE = 60
/** Najwięcej wyników szukania z indeksu. */
export const LIMIT_WYNIKOW_SZUKANIA = 500
/** Przybliżanie obrazu w galerii: krok kółka, największe przybliżenie, przybliżenie po podwójnym kliknięciu. */
export const KROK_PRZYBLIZENIA = 1.2
export const MAKS_PRZYBLIZENIE = 8
export const PRZYBLIZENIE_DWUKLIK = 2.5
/** Ważność linku publicznego do wyboru (dni); najdłużej rok (ustalenie sesji głównej). */
export const WAZNOSCI_LINKU_DNI = [1, 7, 30, 365]
export const MAKS_WAZNOSC_LINKU_DNI = 365
