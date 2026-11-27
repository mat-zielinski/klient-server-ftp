package monitor

import (
	"strings"
	"testing"
)

func TestParsujStat(t *testing.T) {
	przypadki := []struct {
		nazwa            string
		tekst            string
		razem, bezczynny uint64
		blad             bool
	}{
		{"pełny", "cpu  10 20 30 400 50 6 7 8 9 10\ncpu0 1 2 3 4 5 6 7 8 0 0\n", 531, 450, false},
		{"stare jądro", "cpu  10 20 30 400\n", 460, 400, false},
		{"brak wiersza", "intr 1 2 3\n", 0, 0, true},
		{"śmieci", "cpu  a b c\n", 0, 0, true},
	}
	for _, p := range przypadki {
		r, b, err := parsujStat(strings.NewReader(p.tekst))
		if (err != nil) != p.blad || r != p.razem || b != p.bezczynny {
			t.Errorf("%s: %d %d %v", p.nazwa, r, b, err)
		}
	}
}

func TestParsujMeminfo(t *testing.T) {
	przypadki := []struct {
		nazwa        string
		tekst        string
		uzyte, razem uint64
		blad         bool
	}{
		{"zwykły", "MemTotal:       1000 kB\nMemFree:  100 kB\nMemAvailable:    250 kB\n", 750 * 1024, 1000 * 1024, false},
		{"brak MemAvailable", "MemTotal:       1000 kB\n", 0, 0, true},
		{"dostępne > razem", "MemTotal: 10 kB\nMemAvailable: 20 kB\n", 0, 10 * 1024, false},
	}
	for _, p := range przypadki {
		u, r, err := parsujMeminfo(strings.NewReader(p.tekst))
		if (err != nil) != p.blad || u != p.uzyte || r != p.razem {
			t.Errorf("%s: %d %d %v", p.nazwa, u, r, err)
		}
	}
}

func TestParsujNetDev(t *testing.T) {
	tekst := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo:  1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0
  eth0: 123456     100    0    0    0     0          0         0   654321     200    0    0    0     0       0          0
   wg0:5 1 0 0 0 0 0 0 7 1 0 0 0 0 0 0
  zly: 1 2 3
`
	lista, err := parsujNetDev(strings.NewReader(tekst))
	if err != nil || len(lista) != 3 {
		t.Fatalf("%v %+v", err, lista)
	}
	if !lista[0].Petla || lista[0].Interfejs != "lo" {
		t.Errorf("lo: %+v", lista[0])
	}
	if lista[1].Interfejs != "eth0" || lista[1].Odebrane != 123456 || lista[1].Wyslane != 654321 || lista[1].Petla {
		t.Errorf("eth0: %+v", lista[1])
	}
	if lista[2].Interfejs != "wg0" || lista[2].Odebrane != 5 || lista[2].Wyslane != 7 {
		t.Errorf("wg0: %+v", lista[2])
	}
}

func TestParsujDiskstats(t *testing.T) {
	tekst := `   8       0 sda 100 0 2000 0 50 0 4000 0 0 0 0
   8       1 sda1 90 0 1900 0 40 0 3900 0 0 0 0
   7       0 loop0 5 0 10 0 0 0 0 0 0 0 0
 259       0 nvme0n1 1 0 8 0 1 0 16 0 0 0 0
   1       0 krotki 1 2
`
	fizyczne := map[string]bool{"sda": true, "nvme0n1": true}
	o, z, err := parsujDiskstats(strings.NewReader(tekst), func(n string) bool { return fizyczne[n] })
	if err != nil || o != (2000+8)*512 || z != (4000+16)*512 {
		t.Errorf("odczyt %d zapis %d %v", o, z, err)
	}
}
