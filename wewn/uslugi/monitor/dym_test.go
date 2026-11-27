package monitor

import (
	"testing"
	"time"
)

// TestDymSystem: prawdziwe źródło odczytów (Windows albo Linux), wartości w rozsądnych granicach.
func TestDymSystem(t *testing.T) {
	zr := NoweZrodloSystemowe()
	o, err := zr.Odczytaj()
	if err != nil {
		t.Fatalf("odczyt systemu: %v", err)
	}
	if o.RAMRazem == 0 || o.RAMUzyte == 0 || o.RAMUzyte > o.RAMRazem {
		t.Errorf("RAM: %d / %d", o.RAMUzyte, o.RAMRazem)
	}
	if o.CPURazem == 0 || o.CPUBezczynny > o.CPURazem {
		t.Errorf("CPU: %d / %d", o.CPUBezczynny, o.CPURazem)
	}
	if o.Siec == nil {
		t.Error("Siec nie powinna być nil")
	}
	t.Logf("dysk I/O dostępny: %v, interfejsów: %d", o.DyskDostepny, len(o.Siec))

	// Dwie próbki przez Monitor z prawdziwym źródłem (czas z zegara testowego).
	z := &zegarTestowy{t: time.Now()}
	m := Nowy(ZZegarem(z.teraz))
	m.probkuj()
	z.przesun(time.Second)
	m.probkuj()
	p := m.Teraz()
	if p.CPUProc < 0 || p.CPUProc > 100 || p.WysylanieBps < 0 || p.OdbiorBps < 0 || p.DyskOdczytBps < 0 {
		t.Errorf("próbka poza granicami: %+v", p)
	}
	if p.Gorutyny < 1 || p.PamiecProcesu == 0 || p.RAMRazem == 0 {
		t.Errorf("próbka: %+v", p)
	}

	zaj, err := ZajetoscDysku(t.TempDir())
	if err != nil || zaj.Razem == 0 || zaj.Wolne > zaj.Razem {
		t.Errorf("zajętość dysku: %+v %v", zaj, err)
	}
	if _, err := ZajetoscDysku(t.TempDir() + "/nie/ma/takiego"); err == nil {
		t.Error("nieistniejąca ścieżka powinna dać błąd")
	}
}
