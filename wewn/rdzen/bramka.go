package rdzen

// CzyWolno to bramka uprawnień, przez którą przechodzi każda operacja (WWW i FTPS):
//  1. konto wyłączone = ErrBrakPrawa (także admin);
//  2. OpAdministracja tylko dla admina;
//  3. admin może wszystko w każdym udziale, poza zapisem w udziale tylko do odczytu (ErrTylkoOdczyt);
//  4. OpOdczyt, OpPobranie, OpUdostepnienie wymagają PrawoOdczyt albo PrawoZapis w danym udziale;
//  5. OpWgranie, OpZmiana, OpUsuniecie wymagają PrawoZapis (inaczej ErrBrakPrawa), a udział nie może być tylko
//     do odczytu (ErrTylkoOdczyt);
//  6. nieznana operacja albo nieznana wartość prawa = ErrBrakPrawa (domyślnie odmowa).
func CzyWolno(u Uzytkownik, op Operacja, udzial string, tylkoOdczyt bool) error {
	if u.Wylaczony {
		return ErrBrakPrawa
	}
	if op == OpAdministracja {
		if u.Admin {
			return nil
		}
		return ErrBrakPrawa
	}
	zapis, znana := rodzajOperacji(op)
	if !znana {
		return ErrBrakPrawa
	}
	if u.Admin {
		if zapis && tylkoOdczyt {
			return ErrTylkoOdczyt
		}
		return nil
	}
	prawo := u.Prawa[udzial]
	if !zapis {
		if prawo == PrawoOdczyt || prawo == PrawoZapis {
			return nil
		}
		return ErrBrakPrawa
	}
	if prawo != PrawoZapis {
		return ErrBrakPrawa
	}
	if tylkoOdczyt {
		return ErrTylkoOdczyt
	}
	return nil
}

// rodzajOperacji: czy operacja zmienia udział (zapis) i czy bramka ją zna (operacje udziałów, bez administracji).
func rodzajOperacji(op Operacja) (zapis, znana bool) {
	switch op {
	case OpOdczyt, OpPobranie, OpUdostepnienie:
		return false, true
	case OpWgranie, OpZmiana, OpUsuniecie:
		return true, true
	}
	return false, false
}
