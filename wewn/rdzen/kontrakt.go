package rdzen

// KONTRAKT rdzenia: sygnatury, których używają inne warstwy. Właściciel: agent „rdzen”.
// Implementacje w osobnych plikach (sciezki.go, sortowanie.go, szukanie.go, rodzaje.go, bramka.go).
//
//	func OczyscSciezke(wzgledna string) (string, error)
//	    Normalizuje ścieżkę w udziale do postaci "/a/b" ("" i "." = "/"); "..", znaki sterujące,
//	    dyski ("C:"), ścieżki UNC i strumienie NTFS (":") = ErrPozaUdzialem / ErrZlaNazwa.
//	func SciezkaNaDysku(korzen, wzgledna string) (string, error)
//	    Łączy korzeń udziału ze ścieżką po OczyscSciezke; wynik zawsze wewnątrz korzenia.
//	func SprawdzNazwe(nazwa string) error
//	    Nazwa pliku/folderu: bez "/", "\\", ":", "*?\"<>|", nazw zarezerwowanych Windows (CON, NUL…),
//	    bez kropki/spacji na końcu, nie KatalogKosza.
//	func RodzajPliku(nazwa string, folder bool) string
//	func Sortuj(w []Wpis, klucz string, malejaco bool)   // klucz: "nazwa" | "rozmiar" | "zmieniono"; foldery zawsze pierwsze; nazwa naturalnie ("a2" < "a10"), bez wielkości liter
//	func Pasuje(nazwa, zapytanie string) bool              // bez wielkości liter i bez ogonków ("zolw" pasuje do "Żółw")
//	func CzyWolno(u Uzytkownik, op Operacja, udzial string, tylkoOdczyt bool) error
//	    Bramka uprawnień: wyłączony = ErrBrakPrawa; admin = wszystko (poza zapisem w udziale tylko do odczytu);
//	    OpOdczyt/OpPobranie wymagają PrawoOdczyt|PrawoZapis; OpWgranie/OpZmiana/OpUsuniecie wymagają PrawoZapis
//	    i !tylkoOdczyt (wtedy ErrTylkoOdczyt); OpUdostepnienie = PrawoOdczyt+; OpAdministracja tylko admin.
