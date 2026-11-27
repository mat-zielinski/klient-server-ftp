package dziennik

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// naglowekCSV: kolumny eksportu w kolejności pól Wpis.
var naglowekCSV = []string{
	"czas", "rodzaj", "kto", "ip", "protokol", "udzial", "sciezka", "bajty", "czas_ms", "wynik", "opis",
}

// EksportCSV zapisuje wpisy wg filtru (od najnowszych) jako CSV ze średnikiem.
// Pola zaczynające się od = + - @ dostają apostrof, żeby arkusz nie wykonał ich jako formuły.
func (d *Dziennik) EksportCSV(w io.Writer, f Filtr) error {
	wpisy, err := d.Czytaj(f)
	if err != nil {
		return err
	}
	c := csv.NewWriter(w)
	c.Comma = konfiguracja.DziennikSeparatorCSV
	if err := c.Write(naglowekCSV); err != nil {
		return err
	}
	for _, x := range wpisy {
		if err := c.Write(wierszCSV(x)); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

func wierszCSV(w Wpis) []string {
	return []string{
		w.Czas.Local().Format(konfiguracja.DziennikFormatCzasuCSV),
		bezpiecznePole(w.Rodzaj), bezpiecznePole(w.Kto), bezpiecznePole(w.IP), bezpiecznePole(w.Protokol),
		bezpiecznePole(w.Udzial), bezpiecznePole(w.Sciezka),
		strconv.FormatInt(w.Bajty, 10), strconv.FormatInt(w.CzasMs, 10),
		bezpiecznePole(w.Wynik), bezpiecznePole(w.Opis),
	}
}

// bezpiecznePole chroni przed wstrzyknięciem formuły do arkusza (CSV injection).
func bezpiecznePole(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
