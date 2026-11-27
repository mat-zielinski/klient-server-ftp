// Stałe geometrii wykresów SVG (jednostki układu SVG = piksele ekranu, bo viewBox = rozmiar elementu).
// Kolory, czcionki i wysokości kontenerów są w tokenach CSS; tu tylko liczby potrzebne do rachunku.

export const WYKRES = {
  /** Margines obszaru danych: miejsce na opisy osi. */
  margines: { gora: 10, prawo: 14, dol: 24, lewo: 64 },
  /** Prawy margines, gdy jest druga oś (wykres słupkowy). */
  marginesPrawaOs: 44,
  /** Odstęp opisu osi od obszaru danych. */
  odstepOpisu: 8,
  /** Ile mniej więcej podziałek na osi wartości. */
  liczbaPodzialek: 4,
  /** Dopuszczalne „ładne” mnożniki kroku podziałki. */
  ladneMnozniki: [1, 2, 2.5, 5, 10],
  /** Podstawa jednostek osi bajtów (KB = 1024 B, jak w stan/format.ts). */
  podstawaBajtow: 1024,
  /** Najmniejsza szerokość obszaru danych na jeden opis osi czasu. */
  minOdstepOpisowCzasu: 72,
  /** Promień kropki pod kursorem. */
  promienKropki: 4,
  /** Część szerokości kategorii zajęta przez słupek. */
  wypelnienieSlupka: 0.62,
  /** Zaokrąglenie wierzchołka słupka. */
  zaokraglenieSlupka: 2,
  /** Pierścień: grubość jako część promienia i przerwa między łukami (radiany). */
  gruboscPierscienia: 0.28,
  przerwaPierscienia: 0.02,
  /** Mini wykres: margines pionowy, żeby linia nie dotykała krawędzi. */
  marginesMini: 2,
} as const

/** Wykres na żywo: krok czasu zdarzeń „suma” i opóźnienie końca osi (koniec linii zawsze dochodzi do krawędzi). */
export const NA_ZYWO = {
  krokMs: 1000,
  opoznienieMs: 1000,
  oknoMs: 5 * 60 * 1000,
  podzialkaCzasuMs: 60 * 1000,
} as const
