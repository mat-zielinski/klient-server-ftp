package aplikacja

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

func TestCzytajArgumenty(t *testing.T) {
	testy := []struct {
		nazwa  string
		arg    []string
		wynik  polecenie
		zlyArg bool
	}{
		{"brak", nil, polecenie{}, false},
		{"konsola", []string{"--konsola"}, polecenie{Konsola: true}, false},
		{"instaluj", []string{"--usluga", "instaluj"}, polecenie{Usluga: "instaluj"}, false},
		{"usun", []string{"--usluga", "usun"}, polecenie{Usluga: "usun"}, false},
		{"start", []string{"--usluga", "start"}, polecenie{Usluga: "start"}, false},
		{"stop", []string{"--usluga", "stop"}, polecenie{Usluga: "stop"}, false},
		{"usluga bez polecenia", []string{"--usluga"}, polecenie{}, true},
		{"nieznane polecenie", []string{"--usluga", "restart"}, polecenie{}, true},
		{"dwa polecenia", []string{"--usluga", "start", "--usluga", "stop"}, polecenie{}, true},
		{"usluga i konsola", []string{"--usluga", "start", "--konsola"}, polecenie{}, true},
		{"nieznany argument", []string{"--cos"}, polecenie{}, true},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			p, err := czytajArgumenty(tt.arg)
			if tt.zlyArg != (err != nil) {
				t.Fatalf("błąd = %v, oczekiwano błędu: %v", err, tt.zlyArg)
			}
			if err != nil && !errors.Is(err, errZleArgumenty) {
				t.Fatalf("błąd bez opisu użycia: %v", err)
			}
			if p != tt.wynik {
				t.Fatalf("polecenie = %+v, oczekiwano %+v", p, tt.wynik)
			}
		})
	}
}

func TestWpisTransferu(t *testing.T) {
	start := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	teraz := start.Add(1500 * time.Millisecond)
	stan := transfery.Stan{Kto: "ola", IP: "10.0.0.2", Protokol: "ftps", Udzial: "u", Sciezka: "/a.bin",
		Przeslano: 4096, Start: start}
	testy := []struct {
		nazwa    string
		kierunek string
		start    time.Time
		rodzaj   string
		ms       int64
	}{
		{"pobranie", transfery.KierunekPobranie, start, dziennik.RodzajPobranie, 1500},
		{"wgranie", transfery.KierunekWgranie, start, dziennik.RodzajWgranie, 1500},
		{"bez startu", transfery.KierunekPobranie, time.Time{}, dziennik.RodzajPobranie, 0},
		{"start w przyszłości", transfery.KierunekPobranie, teraz.Add(time.Hour), dziennik.RodzajPobranie, 0},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			s := stan
			s.Kierunek, s.Start = tt.kierunek, tt.start
			w := wpisTransferu(s, dziennik.WynikPrzerwane, teraz)
			chce := dziennik.Wpis{Czas: teraz, Rodzaj: tt.rodzaj, Kto: "ola", IP: "10.0.0.2", Protokol: "ftps",
				Udzial: "u", Sciezka: "/a.bin", Bajty: 4096, CzasMs: tt.ms, Wynik: dziennik.WynikPrzerwane}
			if w != chce {
				t.Fatalf("wpis = %+v, oczekiwano %+v", w, chce)
			}
		})
	}
}

func TestPrzekierowanieHTTPS(t *testing.T) {
	testy := []struct {
		nazwa, host, adres, cel string
	}{
		{"pętla z portem", "127.0.0.1:8080", "/pliki?u=a", "https://127.0.0.1:8443/pliki?u=a"},
		{"localhost", "localhost:8080", "/", "https://localhost:8443/"},
		{"IPv6", "[::1]:8080", "/x", "https://[::1]:8443/x"},
		{"bez portu", "127.0.0.1", "/", "https://127.0.0.1:8443/"},
		// Bezpieczeństwo: obca nazwa z nagłówka Host nie może przekierować w obce miejsce.
		{"obca nazwa", "zly.example:8080", "/", "https://127.0.0.1:8443/"},
		{"pusty host", "", "/", "https://127.0.0.1:8443/"},
	}
	h := przekierowanieHTTPS(8443)
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.adres, nil)
			r.Host = tt.host
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusPermanentRedirect {
				t.Fatalf("kod %d", w.Code)
			}
			if got := w.Header().Get("Location"); got != tt.cel {
				t.Fatalf("Location = %q, oczekiwano %q", got, tt.cel)
			}
		})
	}
}

func TestAdresKreatora(t *testing.T) {
	testy := []struct {
		nazwa  string
		adresy []string
		chce   string
	}{
		{"brak", nil, ""},
		{"pętla", []string{"192.168.0.5:8443", "127.0.0.1:8443"}, "https://127.0.0.1:8443"},
		{"wszystkie", []string{"0.0.0.0:9443"}, "https://127.0.0.1:9443"},
		{"tylko LAN", []string{"192.168.0.5:8443"}, "https://192.168.0.5:8443"},
		{"zły adres", []string{"bez-portu"}, "https://bez-portu"},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			if got := adresKreatora(tt.adresy); got != tt.chce {
				t.Fatalf("%q, oczekiwano %q", got, tt.chce)
			}
		})
	}
}

func TestPortZAdresu(t *testing.T) {
	testy := []struct {
		adres string
		chce  int
	}{{"127.0.0.1:8443", 8443}, {"[::1]:1", 1}, {"bez-portu", 0}, {"h:abc", 0}}
	for _, tt := range testy {
		if got := portZAdresu(tt.adres); got != tt.chce {
			t.Errorf("%q: %d, oczekiwano %d", tt.adres, got, tt.chce)
		}
	}
}

func TestZPortem(t *testing.T) {
	got := zPortem([]string{"127.0.0.1", "::1"}, 2121)
	if !reflect.DeepEqual(got, []string{"127.0.0.1:2121", "[::1]:2121"}) {
		t.Fatalf("%v", got)
	}
}

func TestWymagaRestartu(t *testing.T) {
	baza := konfiguracja.Domyslna()
	testy := []struct {
		nazwa string
		zmien func(*konfiguracja.Konfiguracja)
		chce  bool
	}{
		{"bez zmian", func(*konfiguracja.Konfiguracja) {}, false},
		{"udziały", func(k *konfiguracja.Konfiguracja) { k.Udzialy = []konfiguracja.Udzial{{Nazwa: "a"}} }, false},
		{"motyw", func(k *konfiguracja.Konfiguracja) { k.Motyw = "jasny" }, false},
		{"port HTTPS", func(k *konfiguracja.Konfiguracja) { k.PortHTTPS++ }, true},
		{"port HTTP", func(k *konfiguracja.Konfiguracja) { k.PortHTTP++ }, true},
		{"port FTPS", func(k *konfiguracja.Konfiguracja) { k.PortFTPS++ }, true},
		{"porty pasywne", func(k *konfiguracja.Konfiguracja) { k.PortyPasywne[1]++ }, true},
		{"tryb", func(k *konfiguracja.Konfiguracja) { k.TrybSieci = konfiguracja.TrybPubliczny }, true},
		{"adres", func(k *konfiguracja.Konfiguracja) { k.AdresNasluchu = "10.0.0.1" }, true},
		{"FTPS", func(k *konfiguracja.Konfiguracja) { k.FTPSWlaczone = !k.FTPSWlaczone }, true},
		{"certyfikat", func(k *konfiguracja.Konfiguracja) { k.Certyfikat = "a.crt" }, true},
		{"klucz", func(k *konfiguracja.Konfiguracja) { k.Klucz = "a.key" }, true},
		{"domena", func(k *konfiguracja.Konfiguracja) { k.DomenaLetsEncrypt = "a.example" }, true},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			n := kopiaKonfiguracji(baza)
			tt.zmien(&n)
			if got := wymagaRestartu(baza, n); got != tt.chce {
				t.Fatalf("%v, oczekiwano %v", got, tt.chce)
			}
		})
	}
}

func TestKopiaKonfiguracjiNieDzieliUdzialow(t *testing.T) {
	k := konfiguracja.Domyslna()
	k.Udzialy = []konfiguracja.Udzial{{Nazwa: "a"}}
	kopia := kopiaKonfiguracji(k)
	kopia.Udzialy[0].Nazwa = "zmienione"
	if k.Udzialy[0].Nazwa != "a" {
		t.Fatal("kopia dzieli wycinek udziałów z oryginałem")
	}
}

func TestPlikUslugiSystemd(t *testing.T) {
	testy := []struct {
		nazwa, exe, uzytkownik string
		zawiera, brak          []string
	}{
		{"z użytkownikiem", "/opt/skrytka/skrytka", "skrytka",
			[]string{"ExecStart=/opt/skrytka/skrytka --konsola\n", "WorkingDirectory=/opt/skrytka\n",
				"User=skrytka\n", "Group=skrytka\n", "AmbientCapabilities=CAP_NET_BIND_SERVICE\n",
				"WantedBy=multi-user.target\n", "Restart=on-failure\n"}, nil},
		{"root", "/opt/skrytka/skrytka", "", []string{"ExecStart=/opt/skrytka/skrytka --konsola\n"},
			[]string{"User=", "AmbientCapabilities"}},
		{"spacja w ścieżce", "/opt/moja skrytka/skrytka", "",
			[]string{`ExecStart="/opt/moja skrytka/skrytka" --konsola`, `WorkingDirectory="/opt/moja skrytka"`}, nil},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			tresc := plikUslugiSystemd(tt.exe, tt.uzytkownik)
			for _, z := range tt.zawiera {
				if !strings.Contains(tresc, z) {
					t.Errorf("brak %q w:\n%s", z, tresc)
				}
			}
			for _, b := range tt.brak {
				if strings.Contains(tresc, b) {
					t.Errorf("niepotrzebne %q w:\n%s", b, tresc)
				}
			}
		})
	}
}

func TestWersjaDoPokazania(t *testing.T) {
	stara := konfiguracja.WersjaBudowy
	t.Cleanup(func() { konfiguracja.WersjaBudowy = stara })
	testy := []struct{ budowy, chce string }{
		{"", konfiguracja.Wersja},
		{"v9.9.9", "v9.9.9"},
	}
	for _, tt := range testy {
		konfiguracja.WersjaBudowy = tt.budowy
		if got := wersjaDoPokazania(); got != tt.chce {
			t.Errorf("WersjaBudowy %q: %q, oczekiwano %q", tt.budowy, got, tt.chce)
		}
	}
}

func TestStanPolaczenia(t *testing.T) {
	testy := []struct {
		nazwa string
		stany []http.ConnState
		chce  int64
	}{
		{"nowe", []http.ConnState{http.StateNew, http.StateNew}, 2},
		{"zamknięte", []http.ConnState{http.StateNew, http.StateClosed}, 0},
		{"przejęte (SSE)", []http.ConnState{http.StateNew, http.StateHijacked}, 0},
		{"aktywne i bezczynne bez zmian", []http.ConnState{http.StateNew, http.StateActive, http.StateIdle}, 1},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			a := &Aplikacja{}
			for _, s := range tt.stany {
				a.stanPolaczenia(nil, s)
			}
			if got := a.polaczeniaHTTP.Load(); got != tt.chce {
				t.Fatalf("%d, oczekiwano %d", got, tt.chce)
			}
		})
	}
}
