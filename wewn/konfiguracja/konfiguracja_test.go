package konfiguracja

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWczytajBrakPlikuDajeDomyslna(t *testing.T) {
	k, err := Wczytaj(filepath.Join(t.TempDir(), "brak.json"))
	if err != nil {
		t.Fatalf("błąd: %v", err)
	}
	if k.TrybSieci != TrybVPN || k.PortHTTPS != DomyslnyPortHTTPS || !k.FTPSWlaczone {
		t.Errorf("nie domyślna: %+v", k)
	}
}

func TestZapiszIWczytaj(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pod", PlikKonfiguracjiNazwa)
	k := Domyslna()
	k.TrybSieci = TrybPubliczny
	k.Udzialy = []Udzial{{Nazwa: "Filmy", Sciezka: "D:/Filmy", TylkoOdczyt: true}}
	if err := Zapisz(p, k); err != nil {
		t.Fatalf("zapis: %v", err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("został plik tymczasowy")
	}
	w, err := Wczytaj(p)
	if err != nil {
		t.Fatalf("odczyt: %v", err)
	}
	if w.TrybSieci != TrybPubliczny || len(w.Udzialy) != 1 || !w.Udzialy[0].TylkoOdczyt {
		t.Errorf("różnica po odczycie: %+v", w)
	}
}

func TestWczytajZepsutyPlik(t *testing.T) {
	p := filepath.Join(t.TempDir(), "zly.json")
	if err := os.WriteFile(p, []byte("{nie json"), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := Wczytaj(p)
	if err == nil {
		t.Fatal("oczekiwano błędu")
	}
	if k.TrybSieci != TrybVPN {
		t.Errorf("przy błędzie ma być domyślna, jest %+v", k)
	}
}

func TestKatalogAplikacjiZeZmiennej(t *testing.T) {
	d := t.TempDir()
	t.Setenv(ZmiennaKataloguAplikacji, d)
	if KatalogAplikacji() != d || PlikKont() != filepath.Join(d, KatalogDanychNazwa, PlikKontNazwa) {
		t.Errorf("ścieżki nie z SKRYTKA_KATALOG: %s", PlikKont())
	}
}
