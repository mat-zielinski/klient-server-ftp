package rdzen

import (
	"errors"
	"testing"
)

func TestCzyWolno(t *testing.T) {
	const ud = "dokumenty"
	czytelnik := Uzytkownik{Login: "c", Prawa: map[string]Prawo{ud: PrawoOdczyt}}
	pisarz := Uzytkownik{Login: "p", Prawa: map[string]Prawo{ud: PrawoZapis}}
	brak := Uzytkownik{Login: "b", Prawa: map[string]Prawo{ud: PrawoBrak}}
	bezMapy := Uzytkownik{Login: "n"}
	dziwne := Uzytkownik{Login: "d", Prawa: map[string]Prawo{ud: "ZAPIS"}}
	admin := Uzytkownik{Login: "a", Admin: true}
	wylAdmin := Uzytkownik{Login: "wa", Admin: true, Wylaczony: true}
	wylPisarz := Uzytkownik{Login: "wp", Wylaczony: true, Prawa: map[string]Prawo{ud: PrawoZapis}}

	odczyty := []Operacja{OpOdczyt, OpPobranie, OpUdostepnienie}
	zapisy := []Operacja{OpWgranie, OpZmiana, OpUsuniecie}

	type przypadek struct {
		u      Uzytkownik
		op     Operacja
		udzial string
		tylko  bool
		chce   error
	}
	var pp []przypadek
	for _, op := range odczyty {
		pp = append(pp,
			przypadek{czytelnik, op, ud, false, nil},
			przypadek{czytelnik, op, ud, true, nil},
			przypadek{pisarz, op, ud, false, nil},
			przypadek{pisarz, op, ud, true, nil},
			przypadek{czytelnik, op, "inny", false, ErrBrakPrawa},
			przypadek{czytelnik, op, "", false, ErrBrakPrawa},
			przypadek{czytelnik, op, "Dokumenty", false, ErrBrakPrawa},
			przypadek{brak, op, ud, false, ErrBrakPrawa},
			przypadek{bezMapy, op, ud, false, ErrBrakPrawa},
			przypadek{dziwne, op, ud, false, ErrBrakPrawa},
			przypadek{admin, op, ud, false, nil},
			przypadek{admin, op, ud, true, nil},
			przypadek{admin, op, "dowolny", false, nil},
			przypadek{wylAdmin, op, ud, false, ErrBrakPrawa},
			przypadek{wylPisarz, op, ud, false, ErrBrakPrawa},
		)
	}
	for _, op := range zapisy {
		pp = append(pp,
			przypadek{pisarz, op, ud, false, nil},
			przypadek{pisarz, op, ud, true, ErrTylkoOdczyt},
			przypadek{czytelnik, op, ud, false, ErrBrakPrawa},
			przypadek{czytelnik, op, ud, true, ErrBrakPrawa},
			przypadek{pisarz, op, "inny", false, ErrBrakPrawa},
			przypadek{brak, op, ud, false, ErrBrakPrawa},
			przypadek{bezMapy, op, ud, false, ErrBrakPrawa},
			przypadek{dziwne, op, ud, false, ErrBrakPrawa},
			przypadek{admin, op, ud, false, nil},
			przypadek{admin, op, "dowolny", false, nil},
			przypadek{admin, op, ud, true, ErrTylkoOdczyt},
			przypadek{wylAdmin, op, ud, false, ErrBrakPrawa},
			przypadek{wylPisarz, op, ud, false, ErrBrakPrawa},
		)
	}
	pp = append(pp,
		przypadek{admin, OpAdministracja, "", false, nil},
		przypadek{admin, OpAdministracja, ud, true, nil},
		przypadek{wylAdmin, OpAdministracja, "", false, ErrBrakPrawa},
		przypadek{pisarz, OpAdministracja, ud, false, ErrBrakPrawa},
		przypadek{czytelnik, OpAdministracja, ud, false, ErrBrakPrawa},
		przypadek{admin, Operacja("nieznana"), ud, false, ErrBrakPrawa},
		przypadek{pisarz, Operacja(""), ud, false, ErrBrakPrawa},
		przypadek{pisarz, Operacja("ODCZYT"), ud, false, ErrBrakPrawa},
	)
	for _, p := range pp {
		err := CzyWolno(p.u, p.op, p.udzial, p.tylko)
		if p.chce == nil && err != nil || p.chce != nil && !errors.Is(err, p.chce) {
			t.Errorf("CzyWolno(%s, %s, %q, tylkoOdczyt=%v) = %v, chcę %v", p.u.Login, p.op, p.udzial, p.tylko, err, p.chce)
		}
	}
}
