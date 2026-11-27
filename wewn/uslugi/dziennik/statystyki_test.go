package dziennik

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

func TestStatystykiZPustymiPrzedzialami(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	od := data(2026, 11, 27, 8, 0)
	pob := func(g, min int, kto, sciezka, protokol string, bajty, ms int64) {
		d.Zapisz(Wpis{Czas: data(2026, 11, 27, g, min), Rodzaj: RodzajPobranie, Kto: kto, Udzial: "Filmy",
			Sciezka: sciezka, Protokol: protokol, Bajty: bajty, CzasMs: ms, Wynik: WynikOK})
	}
	pob(8, 10, "jan", "/a.mkv", "https", 1000, 1000)
	pob(8, 20, "jan", "/a.mkv", "ftps", 3000, 1000)
	pob(10, 30, "ola", "/b.mkv", "", 500, 0)
	pob(7, 59, "jan", "/poza.mkv", "https", 99, 1)  // przed zakresem
	pob(11, 0, "jan", "/poza2.mkv", "https", 99, 1) // równo z końcem: poza (do wyłącznie)
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 9, 0), Rodzaj: RodzajWgranie, Kto: "jan", Bajty: 777})

	st, err := d.Statystyki(od, data(2026, 11, 27, 11, 0), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Wykres) != 3 {
		t.Fatalf("%d punktów wykresu, oczekiwano 3", len(st.Wykres))
	}
	oczek := []PunktWykresu{
		{Czas: od, Bajty: 4000, Pobrania: 2, SredniaBps: 2000},
		{Czas: od.Add(time.Hour)},
		{Czas: od.Add(2 * time.Hour), Bajty: 500, Pobrania: 1},
	}
	for i, p := range oczek {
		g := st.Wykres[i]
		if !g.Czas.Equal(p.Czas) || g.Bajty != p.Bajty || g.Pobrania != p.Pobrania || g.SredniaBps != p.SredniaBps {
			t.Errorf("punkt %d = %+v, oczekiwano %+v", i, g, p)
		}
	}
	if st.SumaBajtow != 4500 || st.SumaPobran != 3 {
		t.Errorf("sumy: %d B, %d pobrań", st.SumaBajtow, st.SumaPobran)
	}
	if st.Protokoly["https"] != 1000 || st.Protokoly["ftps"] != 3000 ||
		st.Protokoly[konfiguracja.DziennikProtokolNieznany] != 500 {
		t.Errorf("protokoły: %v", st.Protokoly)
	}
	if len(st.NajczestszePliki) != 2 || st.NajczestszePliki[0].Nazwa != "Filmy/a.mkv" || st.NajczestszePliki[0].Liczba != 2 {
		t.Errorf("top pliki: %+v", st.NajczestszePliki)
	}
	if len(st.NajaktywniejsiKlienci) != 2 || st.NajaktywniejsiKlienci[0].Nazwa != "jan" ||
		st.NajaktywniejsiKlienci[0].Bajty != 4000 {
		t.Errorf("top klienci: %+v", st.NajaktywniejsiKlienci)
	}
}

func TestStatystykiTopDziesiec(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	for i := range konfiguracja.DziennikTop + 5 {
		d.Zapisz(Wpis{Czas: data(2026, 11, 27, 9, i), Rodzaj: RodzajPobranie, Kto: fmt.Sprintf("k%02d", i),
			Sciezka: fmt.Sprintf("/p%02d", i), Bajty: int64(i)})
	}
	st, err := d.Statystyki(data(2026, 11, 27, 0, 0), data(2026, 11, 28, 0, 0), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.NajczestszePliki) != konfiguracja.DziennikTop || len(st.NajaktywniejsiKlienci) != konfiguracja.DziennikTop {
		t.Errorf("długość top: %d, %d", len(st.NajczestszePliki), len(st.NajaktywniejsiKlienci))
	}
	if st.NajaktywniejsiKlienci[0].Nazwa != "k14" {
		t.Errorf("pierwszy klient wg bajtów: %s", st.NajaktywniejsiKlienci[0].Nazwa)
	}
	if st.SumaPobran != konfiguracja.DziennikTop+5 {
		t.Errorf("suma pobrań %d", st.SumaPobran)
	}
}

func TestStatystykiZleArgumenty(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	od, do := data(2026, 11, 1, 0, 0), data(2026, 11, 27, 0, 0)
	przypadki := []struct {
		nazwa  string
		od, do time.Time
		krok   time.Duration
		blad   error
	}{
		{"od puste", time.Time{}, do, time.Hour, ErrZlyZakres},
		{"od po do", do, od, time.Hour, ErrZlyZakres},
		{"krok zero", od, do, 0, ErrZlyKrok},
		{"krok ujemny", od, do, -time.Hour, ErrZlyKrok},
		{"za drobny krok", od, do, time.Second, ErrZaDuzoPunktow},
		{"poprawne, do puste = teraz", od, time.Time{}, 24 * time.Hour, nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			_, err := d.Statystyki(p.od, p.do, p.krok)
			if !errors.Is(err, p.blad) {
				t.Errorf("błąd %v, oczekiwano %v", err, p.blad)
			}
		})
	}
}

func TestEksportCSV(t *testing.T) {
	z := &zegar{t: data(2026, 11, 27, 12, 0)}
	d, _ := nowy(t, z, Opcje{})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 9, 0), Rodzaj: RodzajPobranie, Kto: "jan", Sciezka: "/a;b.txt",
		Bajty: 10, CzasMs: 5, Wynik: WynikOK})
	d.Zapisz(Wpis{Czas: data(2026, 11, 27, 10, 0), Rodzaj: RodzajLogowanieBlad, Kto: "=HYPERLINK(\"x\")",
		Opis: "Złe hasło"})
	var b bytes.Buffer
	if err := d.EksportCSV(&b, Filtr{}); err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(&b)
	r.Comma = konfiguracja.DziennikSeparatorCSV
	wiersze, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(wiersze) != 3 || wiersze[0][0] != "czas" || len(wiersze[0]) != len(naglowekCSV) {
		t.Fatalf("wiersze: %v", wiersze)
	}
	if wiersze[1][2] != "'=HYPERLINK(\"x\")" {
		t.Errorf("formuła nie zneutralizowana: %q", wiersze[1][2])
	}
	if wiersze[2][6] != "/a;b.txt" || wiersze[2][7] != "10" || wiersze[2][0] != "2026-11-27 09:00:00" {
		t.Errorf("wiersz pobrania: %v", wiersze[2])
	}
}

func TestBezpiecznePole(t *testing.T) {
	przypadki := map[string]string{
		"":         "",
		"jan":      "jan",
		"=1+1":     "'=1+1",
		"+48":      "'+48",
		"-2":       "'-2",
		"@SUM(A1)": "'@SUM(A1)",
		"a=b":      "a=b",
	}
	for wej, wyj := range przypadki {
		if got := bezpiecznePole(wej); got != wyj {
			t.Errorf("bezpiecznePole(%q) = %q, oczekiwano %q", wej, got, wyj)
		}
	}
}
