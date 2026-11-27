package http

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// kodTOTP: RFC 6238 (HMAC-SHA1, 6 cyfr) do testu logowania z 2FA.
func kodTOTP(t *testing.T, sekret string, czas time.Time) string {
	t.Helper()
	klucz, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(sekret))
	if err != nil {
		t.Fatal(err)
	}
	var licznik [8]byte
	binary.BigEndian.PutUint64(licznik[:], uint64(czas.Unix()/int64(konfiguracja.TOTPOkres/time.Second)))
	m := hmac.New(sha1.New, klucz)
	m.Write(licznik[:])
	h := m.Sum(nil)
	o := h[len(h)-1] & 0x0f
	v := binary.BigEndian.Uint32(h[o:o+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000)
}

func TestDwaFA(t *testing.T) {
	sr := noweSrodowisko(t, true)
	w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/konta/2fa/start", login: "jan"})
	var start struct {
		Sekret string `json:"sekret"`
		Adres  string `json:"adres_otpauth"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &start); err != nil || start.Sekret == "" ||
		!strings.HasPrefix(start.Adres, "otpauth://") {
		t.Fatalf("start: %d %s", w.Code, w.Body)
	}
	t0 := time.Now()
	logowanie := func(kod string) string {
		return `{"login":"jan","haslo":"` + hasloTestowe + `","kod":"` + kod + `"}`
	}
	kroki := []struct {
		nazwa, metoda, adres, cialo, login string
		kod                                int
		zawiera                            string
	}{
		{"zły kod przy włączaniu", "POST", "/api/konta/2fa/wlacz", `{"kod":"000000x"}`, "jan", 400, ""},
		{"włącz", "POST", "/api/konta/2fa/wlacz", `{"kod":"` + kodTOTP(t, start.Sekret, t0) + `"}`, "jan", 200, ""},
		{"ponowny start", "POST", "/api/konta/2fa/start", "", "jan", 409, ""},
		{"stan", "GET", "/api/stan", "", "jan", 200, `"ma_2fa":true`},
		{"logowanie bez kodu", "POST", "/api/logowanie", logowanie(""), "", 401, `"wymaga_kodu":true`},
		{"logowanie złym kodem", "POST", "/api/logowanie", logowanie("123456x"), "", 401, "kod"},
		{"logowanie dobrym kodem", "POST", "/api/logowanie",
			logowanie(kodTOTP(t, start.Sekret, t0.Add(konfiguracja.TOTPOkres))), "", 200, `"login":"jan"`},
		{"wyłącz złym hasłem", "POST", "/api/konta/2fa/wylacz", `{"haslo":"zle-haslo-123"}`, "jan", 403, ""},
		{"wyłącz", "POST", "/api/konta/2fa/wylacz", `{"haslo":"` + hasloTestowe + `"}`, "jan", 200, ""},
		{"wyłącz ponownie", "POST", "/api/konta/2fa/wylacz", `{"haslo":"` + hasloTestowe + `"}`, "jan", 409, ""},
		{"bez sesji", "POST", "/api/konta/2fa/start", "", "", 401, ""},
	}
	for _, k := range kroki {
		w := sr.wykonaj(zadanie{metoda: k.metoda, adres: k.adres, cialo: k.cialo, login: k.login})
		if w.Code != k.kod || !strings.Contains(w.Body.String(), k.zawiera) {
			t.Fatalf("%s: kod %d: %s", k.nazwa, w.Code, w.Body)
		}
		if k.nazwa == "logowanie złym kodem" && strings.Contains(w.Body.String(), "wymaga_kodu") {
			t.Fatalf("zły kod opisany jako brak kodu: %s", w.Body)
		}
	}
}

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
