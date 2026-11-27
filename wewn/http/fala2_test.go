package http

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/indeks"
	"skrytka/wewn/uslugi/monitor"
)

func TestUrzadzenia(t *testing.T) {
	sr := noweSrodowisko(t, true)
	w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/logowanie", cialo: `{"login":"jan","haslo":"` + hasloTestowe + `"}`,
		naglowki: map[string]string{"User-Agent": "Testowa przegladarka"}})
	ciasteczko := w.Result().Cookies()[0].Value
	inna := sr.token("jan")
	w = sr.wykonaj(zadanie{metoda: "GET", adres: "/api/konta/urzadzenia", ciasteczko: ciasteczko})
	var lista []widokUrzadzenia
	if err := json.Unmarshal(w.Body.Bytes(), &lista); err != nil || len(lista) != 2 {
		t.Fatalf("lista: %s", w.Body)
	}
	var biezace, cudzeID string
	for _, u := range lista {
		if u.Biezaca {
			biezace = u.Przegladarka
		} else {
			cudzeID = u.ID
		}
	}
	if biezace != "Testowa przegladarka" || cudzeID == "" {
		t.Fatalf("lista %+v", lista)
	}
	przypadki := []struct {
		nazwa, login, id string
		kod              int
	}{
		{"cudze konto", "admin", cudzeID, 404},
		{"nieznane", "jan", "brak", 404},
		{"własne", "jan", cudzeID, 200},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/konta/urzadzenia?id=" + p.id, login: p.login})
		if w.Code != p.kod {
			t.Fatalf("%s: kod %d", p.nazwa, w.Code)
		}
	}
	if _, ok := sr.sesje.Sprawdz(inna); ok {
		t.Fatal("usunięte urządzenie dalej zalogowane")
	}
}

// zrodloAtrapa: stałe odczyty systemu do monitora.
type zrodloAtrapa struct{}

func (zrodloAtrapa) Odczytaj() (monitor.Odczyt, error) {
	return monitor.Odczyt{RAMRazem: 100, RAMUzyte: 50}, nil
}

func TestMonitor(t *testing.T) {
	sr := noweSrodowisko(t, true)
	for _, a := range []string{"/api/monitor", "/api/monitor/strumien"} {
		if w := sr.wykonaj(zadanie{metoda: "GET", adres: a, login: "admin"}); w.Code != nethttp.StatusServiceUnavailable {
			t.Fatalf("%s bez monitora: kod %d", a, w.Code)
		}
	}
	tyk := make(chan time.Time)
	m := monitor.Nowy(monitor.ZZrodlem(zrodloAtrapa{}), monitor.ZTykaczem(tyk))
	sr.s.monitor = m
	for _, p := range []struct {
		login string
		kod   int
	}{{"jan", 403}, {"admin", 200}} {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/monitor", login: p.login})
		if w.Code != p.kod || (p.kod == 200 && !strings.Contains(w.Body.String(), `"system"`)) {
			t.Fatalf("%s: kod %d: %s", p.login, w.Code, w.Body)
		}
	}
	serw := httptest.NewServer(sr.h)
	defer serw.Close()
	ctx, anuluj := context.WithTimeout(context.Background(), 20*time.Second)
	defer anuluj()
	r, _ := nethttp.NewRequestWithContext(ctx, "GET", serw.URL+"/api/monitor/strumien", nil)
	r.AddCookie(&nethttp.Cookie{Name: konfiguracja.NazwaCiasteczkaSesji, Value: sr.token("admin")})
	odp, err := serw.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = odp.Body.Close() }() // celowo: zamknięcie w teście, błąd bez znaczenia
	m.Uruchom(ctx)                          // pierwsza próbka od razu trafia do słuchacza
	linie := bufio.NewScanner(odp.Body)
	for linie.Scan() {
		if linie.Text() == "event: probka" {
			return
		}
	}
	t.Fatal("brak zdarzenia probka")
}

// atrapaIndeksu zapamiętuje prośby o odświeżenie.
type atrapaIndeksu struct {
	mu      sync.Mutex
	odswiez []string
	gotowy  bool
	szukano int
}

func (a *atrapaIndeksu) Gotowy(string) bool { return a.gotowy }
func (a *atrapaIndeksu) Szukaj(_, _, _ string, _ int) []rdzen.Wpis {
	a.szukano++
	return []rdzen.Wpis{{Nazwa: "z_indeksu"}}
}
func (a *atrapaIndeksu) Statystyki() []indeks.StatystykaUdzialu { return nil }
func (a *atrapaIndeksu) Odswiez(u string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.odswiez = append(a.odswiez, u)
}

func TestIndeksSzukaj(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "kat/Żółw.mkv", "x")
	przypadki := []struct {
		nazwa, adres, login string
		kod                 int
		zawiera             string
	}{
		{"przed indeksem (magazyn)", "/api/indeks/szukaj?udzial=Filmy&q=zolw", "jan", 200, "Żółw.mkv"},
		{"bez prawa", "/api/indeks/szukaj?udzial=Tajne&q=a", "jan", 403, ""},
		{"puste zapytanie", "/api/indeks/szukaj?udzial=Filmy&q=", "jan", 400, ""},
		{"zły limit", "/api/indeks/szukaj?udzial=Filmy&q=a&limit=-1", "jan", 400, ""},
		{"ucieczka", "/api/indeks/szukaj?udzial=Filmy&q=a&sciezka=/..", "jan", 400, ""},
		{"statystyki nie-admin", "/api/indeks/statystyki", "jan", 403, ""},
		{"statystyki", "/api/indeks/statystyki", "admin", 200, `"udzialy":[`},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres, login: p.login})
		if w.Code != p.kod || !strings.Contains(w.Body.String(), p.zawiera) {
			t.Fatalf("%s: kod %d: %s", p.nazwa, w.Code, w.Body)
		}
	}
	prawdziwy := sr.s.indeks.(*indeks.Indeks)
	if err := prawdziwy.Przeindeksuj(context.Background(), "Filmy"); err != nil || !prawdziwy.Gotowy("Filmy") {
		t.Fatal(err)
	}
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/indeks/szukaj?udzial=Filmy&q=zolw", login: "jan"}); !strings.Contains(w.Body.String(), "Żółw.mkv") {
		t.Fatalf("z indeksu: %s", w.Body)
	}
	a := &atrapaIndeksu{gotowy: true}
	sr.s.indeks = a
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/indeks/szukaj?udzial=Filmy&q=zolw", login: "jan"})
	if !strings.Contains(w.Body.String(), "z_indeksu") || a.szukano != 1 {
		t.Fatalf("gotowy indeks nieużyty: %s", w.Body)
	}
	sr.s.indeks = nil
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/indeks/statystyki", login: "admin"}); w.Code != 503 {
		t.Fatalf("bez indeksu: kod %d", w.Code)
	}
}

func TestOdswiezIndeksuPoZmianie(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "a.txt", "a")
	a := &atrapaIndeksu{}
	sr.s.indeks = a
	kroki := []zadanie{
		{metoda: "POST", adres: "/api/folder?udzial=Wspolne&sciezka=/nowy"},
		{metoda: "POST", adres: "/api/zmien?udzial=Wspolne", cialo: `{"z":"/a.txt","na":"/b.txt"}`},
		{metoda: "PUT", adres: "/api/wgraj?udzial=Wspolne&sciezka=/c.txt&koniec=1", cialo: "c"},
		{metoda: "DELETE", adres: "/api/plik?udzial=Wspolne&sciezka=/b.txt"},
		{metoda: "PUT", adres: "/api/wgraj?udzial=Wspolne&sciezka=/d.txt", cialo: "d"}, // bez koniec: bez odświeżenia
	}
	for i, k := range kroki {
		k.login = "jan"
		if w := sr.wykonaj(k); w.Code != 200 {
			t.Fatalf("krok %d: kod %d: %s", i, w.Code, w.Body)
		}
	}
	if strings.Join(a.odswiez, ",") != "Wspolne,Wspolne,Wspolne,Wspolne" {
		t.Fatalf("odświeżenia %v", a.odswiez)
	}
}

func zapiszPNG(t *testing.T, sr *srodowisko, udzial, nazwa string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for x := range 40 {
		img.Set(x, x%30, color.RGBA{R: 200, A: 255})
	}
	f, err := os.Create(filepath.Join(sr.katalog, udzial, nazwa))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() // celowo: zamknięcie w teście, błąd bez znaczenia
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestMiniatura(t *testing.T) {
	sr := noweSrodowisko(t, true)
	zapiszPNG(t, sr, "Filmy", "obraz.png")
	zapiszPNG(t, sr, "Tajne", "obraz.png")
	sr.plik("Filmy", "tekst.txt", "x")
	przypadki := []struct {
		nazwa, adres string
		kod          int
	}{
		{"obraz", "/api/miniatura?udzial=Filmy&sciezka=/obraz.png&rozmiar=s", 200},
		{"nie obraz", "/api/miniatura?udzial=Filmy&sciezka=/tekst.txt&rozmiar=s", 404},
		{"zły rozmiar", "/api/miniatura?udzial=Filmy&sciezka=/obraz.png&rozmiar=xl", 400},
		{"bez prawa", "/api/miniatura?udzial=Tajne&sciezka=/obraz.png&rozmiar=s", 403},
		{"ucieczka", "/api/miniatura?udzial=Filmy&sciezka=/../Tajne/obraz.png&rozmiar=s", 400},
		{"brak pliku", "/api/miniatura?udzial=Filmy&sciezka=/brak.png&rozmiar=m", 404},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres, login: "jan"})
		if w.Code != p.kod {
			t.Fatalf("%s: kod %d: %s", p.nazwa, w.Code, w.Body)
		}
		if p.kod == 200 && (w.Header().Get("Content-Type") != "image/jpeg" ||
			w.Header().Get("Cache-Control") != konfiguracja.CacheMiniatur) {
			t.Fatalf("nagłówki %v", w.Header())
		}
	}
	if sr.koniec.ile() != 0 || len(sr.dziennik.kopia()) != 0 {
		t.Fatal("miniatura zarejestrowana jako transfer albo w dzienniku")
	}
	sr.s.miniatury = nil
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: przypadki[0].adres, login: "jan"}); w.Code != 503 {
		t.Fatalf("bez usługi: kod %d", w.Code)
	}
}

func TestSuma(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "a.txt", "skrytka")
	sr.plik("Tajne", "a.txt", "tajne")
	oczek := sha256.Sum256([]byte("skrytka"))
	przypadki := []struct {
		adres   string
		kod     int
		zawiera string
	}{
		{"/api/suma?udzial=Filmy&sciezka=/a.txt", 200, hex.EncodeToString(oczek[:])},
		{"/api/suma?udzial=Tajne&sciezka=/a.txt", 403, ""},
		{"/api/suma?udzial=Filmy&sciezka=/", 404, ""},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres, login: "jan"})
		if w.Code != p.kod || !strings.Contains(w.Body.String(), p.zawiera) {
			t.Fatalf("%s: kod %d: %s", p.adres, w.Code, w.Body)
		}
	}
	sr.s.sumy = nil
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: przypadki[0].adres, login: "jan"}); w.Code != 503 {
		t.Fatalf("bez usługi: kod %d", w.Code)
	}
}

func TestMapaDziennika(t *testing.T) {
	sr := noweSrodowisko(t, true)
	pon := time.Date(2026, 11, 23, 10, 15, 0, 0, time.Local) // poniedziałek
	nie := time.Date(2026, 11, 29, 23, 5, 0, 0, time.Local)  // niedziela
	for _, w := range []dziennik.Wpis{
		{Czas: pon, Rodzaj: dziennik.RodzajPobranie, Kto: "jan", Bajty: 100},
		{Czas: pon, Rodzaj: dziennik.RodzajPobranie, Kto: "jan", Bajty: 5},
		{Czas: nie, Rodzaj: dziennik.RodzajPobranie, Kto: "admin", Bajty: 50},
		{Czas: nie, Rodzaj: dziennik.RodzajLogowanie, Kto: "jan", Bajty: 999},
	} {
		sr.dziennik.Zapisz(w)
	}
	przypadki := []struct {
		login        string
		pon10, nie23 int64
	}{{"jan", 105, 0}, {"admin", 105, 50}}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/dziennik/mapa", login: p.login})
		var odp struct{ Komorki [][]int64 }
		if err := json.Unmarshal(w.Body.Bytes(), &odp); err != nil || len(odp.Komorki) != 7 || len(odp.Komorki[0]) != 24 {
			t.Fatalf("%s: %s", p.login, w.Body)
		}
		if odp.Komorki[0][10] != p.pon10 || odp.Komorki[6][23] != p.nie23 {
			t.Fatalf("%s: %v", p.login, odp.Komorki)
		}
	}
}

func TestPulpitZajetosc(t *testing.T) {
	sr := noweSrodowisko(t, true)
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/pulpit", login: "admin"})
	var odp struct {
		Udzialy []miejsceUdzialu
	}
	if err := json.Unmarshal(w.Body.Bytes(), &odp); err != nil || len(odp.Udzialy) != 4 {
		t.Fatalf("%s", w.Body)
	}
	for _, u := range odp.Udzialy {
		if u.Razem == 0 || u.Wolne == 0 || u.Wolne > u.Razem {
			t.Fatalf("udział %+v", u)
		}
	}
}

func TestWersjaBudowyIStatyczne(t *testing.T) {
	sr := noweSrodowisko(t, false)
	poprzednia := konfiguracja.WersjaBudowy
	konfiguracja.WersjaBudowy = "v9.9.9"
	t.Cleanup(func() { konfiguracja.WersjaBudowy = poprzednia })
	for _, a := range []string{"/api/stan", "/api/zdrowie"} {
		if w := sr.wykonaj(zadanie{metoda: "GET", adres: a}); !strings.Contains(w.Body.String(), `"wersja":"v9.9.9"`) {
			t.Fatalf("%s: %s", a, w.Body)
		}
	}
	przypadki := []struct{ adres, typ, cache string }{
		{"/sw.js", "text/javascript; charset=utf-8", konfiguracja.CacheBez},
		{"/manifest.webmanifest", "application/manifest+json", konfiguracja.CacheBez},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres})
		if w.Code != 200 || w.Header().Get("Content-Type") != p.typ || w.Header().Get("Cache-Control") != p.cache {
			t.Fatalf("%s: %d %v", p.adres, w.Code, w.Header())
		}
	}
}

func TestDodatkiNil(t *testing.T) {
	d := dodatkiZ(Zaleznosci{})
	if d.monitor != nil || d.indeks != nil || d.miniatury != nil || d.sumy != nil {
		t.Fatalf("wyłączone usługi nie są nil: %+v", d)
	}
}
