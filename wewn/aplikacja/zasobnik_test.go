package aplikacja

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"skrytka/wewn/aplikacja/ikony"
	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/siec"
	"skrytka/wewn/uslugi/transfery"
)

func TestArgumentBezZasobnika(t *testing.T) {
	testy := []struct {
		nazwa  string
		arg    []string
		wynik  polecenie
		zlyArg bool
	}{
		{"bez zasobnika", []string{"--bez-zasobnika"}, polecenie{BezZasobnika: true}, false},
		{"z konsolą", []string{"--konsola", "--bez-zasobnika"}, polecenie{Konsola: true, BezZasobnika: true}, false},
		{"z usługą", []string{"--usluga", "start", "--bez-zasobnika"}, polecenie{}, true},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			p, err := czytajArgumenty(tt.arg)
			if tt.zlyArg != (err != nil) || (err != nil && !errors.Is(err, errZleArgumenty)) {
				t.Fatalf("błąd = %v, oczekiwano błędu: %v", err, tt.zlyArg)
			}
			if p != tt.wynik {
				t.Fatalf("polecenie = %+v, oczekiwano %+v", p, tt.wynik)
			}
		})
	}
}

func TestCzyPokazacZasobnik(t *testing.T) {
	testy := []struct {
		nazwa  string
		p      polecenie
		system string
		env    map[string]string
		wynik  bool
	}{
		{"windows zwykle", polecenie{}, "windows", nil, true},
		{"windows konsola", polecenie{Konsola: true}, "windows", nil, false},
		{"windows bez zasobnika", polecenie{BezZasobnika: true}, "windows", nil, false},
		{"windows usługa", polecenie{Usluga: "start"}, "windows", nil, false},
		{"linux bez pulpitu", polecenie{}, "linux", nil, false},
		{"linux X11", polecenie{}, "linux", map[string]string{"DISPLAY": ":0"}, true},
		{"linux Wayland", polecenie{}, "linux", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, true},
		{"linux systemd z ekranem", polecenie{}, "linux",
			map[string]string{"DISPLAY": ":0", "INVOCATION_ID": "abc"}, false},
		{"linux konsola z ekranem", polecenie{Konsola: true}, "linux", map[string]string{"DISPLAY": ":0"}, false},
		{"inny system", polecenie{}, "darwin", map[string]string{"DISPLAY": ":0"}, false},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			env := func(k string) string { return tt.env[k] }
			if w := czyPokazacZasobnik(tt.p, tt.system, env); w != tt.wynik {
				t.Fatalf("= %v, oczekiwano %v", w, tt.wynik)
			}
		})
	}
}

func stan(kierunek, kto, sciezka string, bps int64) transfery.Stan {
	return transfery.Stan{Kierunek: kierunek, Kto: kto, Sciezka: sciezka, PredkoscBps: bps}
}

func TestTekstDymka(t *testing.T) {
	pob, wgr := transfery.KierunekPobranie, transfery.KierunekWgranie
	mb := int64(1 << 20)
	testy := []struct {
		nazwa string
		adres string
		stany []transfery.Stan
		faza  string
		wynik string
	}{
		{"działa", "192.168.1.5:8443", nil, "", "Skrytka — działa · 192.168.1.5:8443"},
		{"działa bez adresu", "", nil, "", "Skrytka — działa"},
		{"uruchamianie", "1.2.3.4:1", nil, "uruchamianie", "Skrytka — uruchamianie"},
		{"zamykanie ma pierwszeństwo", "", []transfery.Stan{stan(pob, "a", "x", 1)}, "zamykanie", "Skrytka — zamykanie"},
		{"dwa pobrania", "a:1", []transfery.Stan{stan(pob, "a", "x", 6*mb+mb*3/10), stan(pob, "b", "y", 6*mb)},
			"", "Skrytka — 2 pobrania · 12,3 MB/s"},
		{"jedno pobranie", "", []transfery.Stan{stan(pob, "a", "x", 0)}, "", "Skrytka — 1 pobranie · 0 B/s"},
		{"mieszane", "", []transfery.Stan{stan(pob, "a", "x", mb), stan(wgr, "b", "y", 2*mb)},
			"", "Skrytka — 1 pobranie, 1 wgranie · 3 MB/s"},
		{"pięć wgrań", "", []transfery.Stan{stan(wgr, "", "", 0), stan(wgr, "", "", 0), stan(wgr, "", "", 0),
			stan(wgr, "", "", 0), stan(wgr, "", "", 0)}, "", "Skrytka — 5 wgrań · 0 B/s"},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := tekstDymka(tt.adres, tt.stany, tt.faza); w != tt.wynik {
				t.Fatalf("= %q, oczekiwano %q", w, tt.wynik)
			}
		})
	}
	dlugi := tekstDymka(strings.Repeat("ą", 300), nil, "")
	if n := utf8.RuneCountInString(dlugi); n != konfiguracja.ZasobnikMaksDymka || !strings.HasSuffix(dlugi, "…") {
		t.Fatalf("długi dymek: %d znaków, %q", n, dlugi)
	}
}

func TestLiczebnik(t *testing.T) {
	testy := map[int]string{1: "1 pobranie", 2: "2 pobrania", 4: "4 pobrania", 5: "5 pobrań", 11: "11 pobrań",
		12: "12 pobrań", 14: "14 pobrań", 22: "22 pobrania", 25: "25 pobrań", 0: "0 pobrań", 112: "112 pobrań"}
	for n, oczek := range testy {
		if w := liczebnik(n, konfiguracja.ZasobnikFormyPobran); w != oczek {
			t.Errorf("liczebnik(%d) = %q, oczekiwano %q", n, w, oczek)
		}
	}
}

func TestFormatPredkosci(t *testing.T) {
	testy := []struct {
		bps   int64
		wynik string
	}{
		{-5, "0 B/s"}, {0, "0 B/s"}, {512, "512 B/s"}, {1023, "1023 B/s"}, {1024, "1 KB/s"},
		{1536, "1,5 KB/s"}, {850 * 1024, "850 KB/s"}, {12*1024*1024 + 300*1024, "12,3 MB/s"},
		{1100 * 1024 * 1024, "1,07 GB/s"}, {1 << 50, "1024 TB/s"},
	}
	for _, tt := range testy {
		if w := formatPredkosci(tt.bps); w != tt.wynik {
			t.Errorf("formatPredkosci(%d) = %q, oczekiwano %q", tt.bps, w, tt.wynik)
		}
	}
}

func TestPozycjeTransferow(t *testing.T) {
	pob, wgr := transfery.KierunekPobranie, transfery.KierunekWgranie
	dlugaNazwa := strings.Repeat("x", 60) + ".mkv"
	testy := []struct {
		nazwa string
		stany []transfery.Stan
		max   int
		wynik []string
	}{
		{"pusto", nil, 5, []string{}},
		{"pobranie i wgranie", []transfery.Stan{stan(pob, "jan", "Filmy/a.mkv", 1024), stan(wgr, "", `b\c.txt`, 0)}, 5,
			[]string{"↓ jan · a.mkv · 1 KB/s", "↑ gość · c.txt · 0 B/s"}},
		{"limit", []transfery.Stan{stan(pob, "a", "1", 0), stan(pob, "b", "2", 0), stan(pob, "c", "3", 0)}, 2,
			[]string{"↓ a · 1 · 0 B/s", "↓ b · 2 · 0 B/s"}},
		{"długa nazwa", []transfery.Stan{stan(pob, "a", dlugaNazwa, 0)}, 5,
			[]string{"↓ a · " + strings.Repeat("x", 39) + "… · 0 B/s"}},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := pozycjeTransferow(tt.stany, tt.max); !slices.Equal(w, tt.wynik) {
				t.Fatalf("= %q, oczekiwano %q", w, tt.wynik)
			}
		})
	}
}

func TestAdresDlaKlientow(t *testing.T) {
	lan := siec.Adres{IP: "192.168.1.5", Rodzaj: konfiguracja.RodzajLAN}
	wg := siec.Adres{IP: "10.8.0.1", Rodzaj: konfiguracja.RodzajWireGuard}
	petla := siec.Adres{IP: "127.0.0.1", Rodzaj: konfiguracja.RodzajPetla}
	testy := []struct {
		nazwa     string
		k         konfiguracja.Konfiguracja
		adresy    []siec.Adres
		publiczny string
		wynik     string
	}{
		{"LAN przed WireGuard", konfiguracja.Konfiguracja{}, []siec.Adres{petla, wg, lan}, "", "192.168.1.5:8443"},
		{"tylko WireGuard", konfiguracja.Konfiguracja{}, []siec.Adres{petla, wg}, "", "10.8.0.1:8443"},
		{"nic = pętla", konfiguracja.Konfiguracja{}, nil, "", "127.0.0.1:8443"},
		{"adres z konfiguracji", konfiguracja.Konfiguracja{AdresNasluchu: "127.0.0.1"}, []siec.Adres{lan}, "",
			"127.0.0.1:8443"},
		{"0.0.0.0 z konfiguracji pomijany", konfiguracja.Konfiguracja{AdresNasluchu: "0.0.0.0"}, []siec.Adres{lan}, "",
			"192.168.1.5:8443"},
		{"publiczny znany", konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybPubliczny}, []siec.Adres{lan},
			"203.0.113.7", "203.0.113.7:8443"},
		{"publiczny nieznany", konfiguracja.Konfiguracja{TrybSieci: konfiguracja.TrybPubliczny}, []siec.Adres{lan},
			"", "192.168.1.5:8443"},
		{"publiczny w trybie vpn pomijany", konfiguracja.Konfiguracja{}, []siec.Adres{lan}, "203.0.113.7",
			"192.168.1.5:8443"},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := adresDlaKlientow(tt.k, tt.adresy, tt.publiczny, 8443); w != tt.wynik {
				t.Fatalf("= %q, oczekiwano %q", w, tt.wynik)
			}
		})
	}
}

// ico składa plik ICO z wpisów (szerokość, dane).
func ico(wpisy ...[]byte) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.LittleEndian, [3]uint16{0, 1, uint16(len(wpisy))}) // celowo: bufor w pamięci
	poczatek := 6 + 16*len(wpisy)
	for i, d := range wpisy {
		szer := byte(16 * (i + 1))
		_ = binary.Write(&b, binary.LittleEndian, struct { // celowo: bufor w pamięci
			Szer, Wys, Kolory, Zero byte
			Plany, Bity             uint16
			Rozmiar, Poczatek       uint32
		}{szer, szer, 0, 0, 1, 32, uint32(len(d)), uint32(poczatek)})
		poczatek += len(d)
	}
	for _, d := range wpisy {
		b.Write(d)
	}
	return b.Bytes()
}

func TestPngZIco(t *testing.T) {
	png1 := []byte("\x89PNG\r\n\x1a\nmaly")
	png2 := []byte("\x89PNG\r\n\x1a\nduzy")
	bmp := []byte("BMPdanebezpng")
	zly := ico(png1)
	binary.LittleEndian.PutUint32(zly[6+8:], 1000) // rozmiar poza plikiem
	testy := []struct {
		nazwa string
		dane  []byte
		wynik []byte
	}{
		{"pusto", nil, nil},
		{"nie ICO", []byte("\x89PNG\r\n\x1a\n"), nil},
		{"jeden PNG", ico(png1), png1},
		{"największy PNG", ico(png1, png2), png2},
		{"BMP pomijany", ico(png1, bmp), png1},
		{"tylko BMP", ico(bmp), nil},
		{"rozmiar poza plikiem", zly, nil},
		{"ucięty katalog", ico(png1, png2)[:20], nil},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := pngZIco(tt.dane); !bytes.Equal(w, tt.wynik) {
				t.Fatalf("= %q, oczekiwano %q", w, tt.wynik)
			}
		})
	}
}

func TestWybierzIkone(t *testing.T) {
	pelny := zestawIkon{jasna: []byte("J"), jasnaAktywna: []byte("JA"), ciemna: []byte("C"), ciemnaAktywna: []byte("CA")}
	bezCiemnych := zestawIkon{jasna: []byte("J"), jasnaAktywna: []byte("JA")}
	testy := []struct {
		nazwa         string
		z             zestawIkon
		aktywna, jasn bool
		wynik         string
	}{
		{"ciemny pasek", pelny, false, false, "J"},
		{"ciemny pasek, transfer", pelny, true, false, "JA"},
		{"jasny pasek", pelny, false, true, "C"},
		{"jasny pasek, transfer", pelny, true, true, "CA"},
		{"jasny pasek bez ciemnych", bezCiemnych, false, true, "J"},
		{"jasny pasek bez ciemnych, transfer", bezCiemnych, true, true, "JA"},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := string(wybierzIkone(tt.z, tt.aktywna, tt.jasn)); w != tt.wynik {
				t.Fatalf("= %q, oczekiwano %q", w, tt.wynik)
			}
		})
	}
}

func TestCzyNaszAutostart(t *testing.T) {
	exe := `C:\Program Files\Skrytka\skrytka.exe`
	testy := []struct {
		nazwa, wartosc string
		wynik          bool
	}{
		{"nasz w cudzysłowie", wartoscAutostartu(exe), true},
		{"inna wielkość liter", `"c:\program files\skrytka\SKRYTKA.EXE"`, true},
		{"bez cudzysłowu", exe, true},
		{"inny program", `"C:\Inne\skrytka.exe"`, false},
		{"pusty", `""`, false},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if w := czyNaszAutostart(tt.wartosc, exe); w != tt.wynik {
				t.Fatalf("= %v, oczekiwano %v", w, tt.wynik)
			}
		})
	}
}

func TestTrescAutostartuLinux(t *testing.T) {
	testy := []struct {
		exe, exec string
	}{
		{"/opt/skrytka/skrytka", `Exec="/opt/skrytka/skrytka"`},
		{"/home/a b/skrytka", `Exec="/home/a b/skrytka"`},
		{`/x/"$y` + "`" + `\z`, `Exec="/x/\\"\\$y\\` + "`" + `\\\\z"`},
	}
	for _, tt := range testy {
		t.Run(tt.exe, func(t *testing.T) {
			tresc := trescAutostartuLinux(tt.exe)
			if !strings.Contains(tresc, "\n"+tt.exec+"\n") || !strings.HasPrefix(tresc, "[Desktop Entry]\n") {
				t.Fatalf("treść:\n%s\nbrak linii %s", tresc, tt.exec)
			}
		})
	}
}

// Linux (StatusNotifier) czyta PNG: osadzone ikony zasobnika muszą mieć PNG w środku ICO.
func TestIkonyZasobnikaMajaPNG(t *testing.T) {
	for nazwa, dane := range map[string][]byte{
		"Zasobnik": ikony.Zasobnik, "ZasobnikAktywny": ikony.ZasobnikAktywny,
		"ZasobnikCiemny": ikony.ZasobnikCiemny, "ZasobnikCiemnyAktywny": ikony.ZasobnikCiemnyAktywny,
	} {
		if pngZIco(dane) == nil {
			t.Errorf("%s: brak PNG w ICO", nazwa)
		}
	}
}
