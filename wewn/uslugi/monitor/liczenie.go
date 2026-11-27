package monitor

import (
	"math"
	"sort"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// roznicaLicznika: przyrost narastającego licznika. Spadek wartości = przekręcenie licznika 32-bitowego
// (gdy poprzednia wartość była blisko końca zakresu 32 bitów) albo reset (przyrost 0).
func roznicaLicznika(poprzedni, biezacy uint64) uint64 {
	if biezacy >= poprzedni {
		return biezacy - poprzedni
	}
	if poprzedni <= konfiguracja.MonitorGranicaLicznika32 && poprzedni >= konfiguracja.MonitorProgPrzekreceniaLicznika32 {
		return konfiguracja.MonitorGranicaLicznika32 - poprzedni + 1 + biezacy
	}
	return 0
}

// predkosc: bajty na sekundę z przyrostu licznika w czasie dt (sekundy); dt <= 0 daje 0.
func predkosc(poprzedni, biezacy uint64, dt float64) int64 {
	if dt <= 0 {
		return 0
	}
	return int64(math.Round(float64(roznicaLicznika(poprzedni, biezacy)) / dt))
}

// procentCPU: zajętość z różnic taktów; brak przyrostu albo reset liczników daje 0.
func procentCPU(poprzedni, biezacy Odczyt) float64 {
	if biezacy.CPURazem <= poprzedni.CPURazem || biezacy.CPUBezczynny < poprzedni.CPUBezczynny {
		return 0
	}
	razem := float64(biezacy.CPURazem - poprzedni.CPURazem)
	bezczynny := float64(biezacy.CPUBezczynny - poprzedni.CPUBezczynny)
	p := (razem - bezczynny) / razem * konfiguracja.MonitorProcentMaks
	switch {
	case p < 0:
		return 0
	case p > konfiguracja.MonitorProcentMaks:
		return konfiguracja.MonitorProcentMaks
	}
	return math.Round(p*10) / 10
}

// rodzajInterfejsu: pętla, WireGuard (po nazwie lub opisie, wzorce jak w pakiecie siec), reszta lan.
func rodzajInterfejsu(l LicznikSieci) string {
	if l.Petla || l.Interfejs == konfiguracja.MonitorNazwaPetliLinux {
		return konfiguracja.RodzajPetla
	}
	n, o := strings.ToLower(l.Interfejs), strings.ToLower(l.Opis)
	for _, w := range konfiguracja.WzorceWireGuardZawiera {
		if strings.Contains(n, w) || strings.Contains(o, w) {
			return konfiguracja.RodzajWireGuard
		}
	}
	for _, w := range konfiguracja.WzorceWireGuardPrzedrostek {
		if strings.HasPrefix(n, w) {
			return konfiguracja.RodzajWireGuard
		}
	}
	return konfiguracja.RodzajLAN
}

// predkosciSieci: prędkości interfejsów (posortowane wg nazwy) i sumy bez pętli zwrotnej.
// Interfejs nowy (bez poprzedniego odczytu) ma prędkość 0.
func predkosciSieci(poprzedni, biezacy []LicznikSieci, dt float64) ([]SiecInterfejsu, int64, int64) {
	stare := make(map[string]LicznikSieci, len(poprzedni))
	for _, l := range poprzedni {
		stare[l.Interfejs] = l
	}
	wynik := make([]SiecInterfejsu, 0, len(biezacy))
	var wys, odb int64
	for _, l := range biezacy {
		s := SiecInterfejsu{Interfejs: l.Interfejs, Rodzaj: rodzajInterfejsu(l)}
		if p, ok := stare[l.Interfejs]; ok {
			s.WysylanieBps = predkosc(p.Wyslane, l.Wyslane, dt)
			s.OdbiorBps = predkosc(p.Odebrane, l.Odebrane, dt)
		}
		if s.Rodzaj != konfiguracja.RodzajPetla {
			wys += s.WysylanieBps
			odb += s.OdbiorBps
		}
		wynik = append(wynik, s)
	}
	sort.Slice(wynik, func(i, j int) bool { return wynik[i].Interfejs < wynik[j].Interfejs })
	return wynik, wys, odb
}
