package http

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/udostepnienia"
)

// zadanieLinku: GET bez sesji, z opcjonalnym hasłem i ciasteczkami linku.
func (sr *srodowisko) zadanieLinku(adres, haslo string, ciasteczka ...*nethttp.Cookie) *httptest.ResponseRecorder {
	sr.t.Helper()
	r := httptest.NewRequest("GET", adres, nil)
	r.RemoteAddr = "192.168.0.60:52000"
	if haslo != "" {
		r.Header.Set(konfiguracja.NaglowekHaslaLinku, haslo)
	}
	for _, c := range ciasteczka {
		r.AddCookie(&nethttp.Cookie{Name: c.Name, Value: c.Value})
	}
	w := httptest.NewRecorder()
	sr.h.ServeHTTP(w, r)
	return w
}

// nowyLinkTestowy: link jana do folderu (udział Wspolne) z hasłem.
func (sr *srodowisko) nowyLinkTestowy(sciezka, haslo string) udostepnienia.Link {
	sr.t.Helper()
	l, err := sr.linki.Utworz(udostepnienia.Link{Udzial: "Wspolne", Sciezka: sciezka, Autor: "jan"}, haslo)
	if err != nil {
		sr.t.Fatal(err)
	}
	return l
}

// ciasteczkoLinku: GET info z hasłem i ciasteczko z odpowiedzi.
func (sr *srodowisko) ciasteczkoLinku(id, haslo string) *nethttp.Cookie {
	sr.t.Helper()
	w := sr.zadanieLinku(konfiguracja.PrefiksAPILinku+id, haslo)
	for _, c := range w.Result().Cookies() {
		if c.Name == konfiguracja.NazwaCiasteczkaLinku {
			return c
		}
	}
	sr.t.Fatalf("brak ciasteczka linku (kod %d)", w.Code)
	return nil
}

func srodowiskoLinkow(t *testing.T) (*srodowisko, *zegarTestowy) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "kat/a.txt", "AAA")
	sr.plik("Wspolne", "kat/pod/b.txt", "BBB")
	sr.plik("Wspolne", "poza.txt", "tajne")
	sr.plik("Wspolne", "inny/c.txt", "CCC")
	return sr, nowyZegar().ustawW(sr.s)
}

func TestCiasteczkoLinkuAtrybuty(t *testing.T) {
	sr, zegar := srodowiskoLinkow(t)
	l := sr.nowyLinkTestowy("/kat", "haslo-linku")
	c := sr.ciasteczkoLinku(l.ID, "haslo-linku")
	if !c.HttpOnly || c.SameSite != nethttp.SameSiteStrictMode || c.Path != "/api/publiczny/"+l.ID {
		t.Fatalf("atrybuty %+v", c)
	}
	if c.MaxAge != int(konfiguracja.WaznoscCiasteczkaLinku/time.Second) ||
		!c.Expires.Equal(zegar.teraz().Add(konfiguracja.WaznoscCiasteczkaLinku).Truncate(time.Second)) {
		t.Fatalf("ważność %d / %v", c.MaxAge, c.Expires)
	}
	// Bez hasła, ze złym hasłem i z samym ciasteczkiem: nowego ciasteczka nie ma (nie przedłuża się samo).
	for _, w := range []*httptest.ResponseRecorder{
		sr.zadanieLinku("/api/publiczny/"+l.ID, ""),
		sr.zadanieLinku("/api/publiczny/"+l.ID, "zle"),
		sr.zadanieLinku("/api/publiczny/"+l.ID, "", c),
	} {
		if len(w.Result().Cookies()) != 0 {
			t.Fatalf("kod %d: niespodziewane ciasteczko %v", w.Code, w.Result().Cookies())
		}
	}
	// Link bez hasła: ciasteczko niepotrzebne.
	bez := sr.nowyLinkTestowy("/inny", "")
	if w := sr.zadanieLinku("/api/publiczny/"+bez.ID, ""); w.Code != 200 || len(w.Result().Cookies()) != 0 {
		t.Fatalf("link bez hasła: kod %d, ciasteczka %v", w.Code, w.Result().Cookies())
	}
}

func TestCiasteczkoLinkuPobieranie(t *testing.T) {
	sr, zegar := srodowiskoLinkow(t)
	l := sr.nowyLinkTestowy("/kat", "haslo-linku")
	inny := sr.nowyLinkTestowy("/inny", "inne-haslo")
	dobre := sr.ciasteczkoLinku(l.ID, "haslo-linku")
	cudze := sr.ciasteczkoLinku(inny.ID, "inne-haslo")
	czas, podpis, _ := strings.Cut(dobre.Value, ".")
	unix, _ := strconv.ParseInt(czas, 10, 64)
	falsz := func(v string) *nethttp.Cookie {
		return &nethttp.Cookie{Name: konfiguracja.NazwaCiasteczkaLinku, Value: v}
	}
	przypadki := []struct {
		nazwa      string
		haslo      string
		ciasteczko *nethttp.Cookie
		przed      time.Duration
		kod        int
	}{
		{"dobre ciasteczko", "", dobre, 0, 200},
		{"dobre ciasteczko tuż przed wygaśnięciem", "", dobre, konfiguracja.WaznoscCiasteczkaLinku - time.Second, 200},
		{"hasło bez ciasteczka", "haslo-linku", nil, 0, 200},
		{"bez hasła i ciasteczka", "", nil, 0, 401},
		{"złe hasło bez ciasteczka", "zle", nil, 0, 401},
		{"śmieci", "", falsz("abc"), 0, 401},
		{"pusty podpis", "", falsz(czas + "."), 0, 401},
		{"wygasłe", "", dobre, konfiguracja.WaznoscCiasteczkaLinku, 401},
		{"podrobiony czas", "", falsz(strconv.FormatInt(unix+60, 10) + "." + podpis), 0, 401},
		{"podrobiony podpis", "", falsz(czas + "." + strings.Repeat("A", len(podpis))), 0, 401},
		{"ciasteczko innego linku", "", cudze, 0, 401},
		{"podpis innego linku z tym czasem", "", falsz(czas + "." + sr.s.podpisLinku(inny.ID, unix)), 0, 401},
		{"czas z daleka w przyszłości (nawet z dobrym podpisem)", "",
			falsz(strconv.FormatInt(unix+86400, 10) + "." + sr.s.podpisLinku(l.ID, unix+86400)), 0, 401},
	}
	start := zegar.teraz()
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			zegar.ns.Store(start.Add(p.przed).UnixNano())
			var c []*nethttp.Cookie
			if p.ciasteczko != nil {
				c = append(c, p.ciasteczko)
			}
			w := sr.zadanieLinku("/api/publiczny/"+l.ID+"/pobierz?sciezka=/a.txt", p.haslo, c...)
			if w.Code != p.kod || (p.kod == 200 && w.Body.String() != "AAA") {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
		})
	}
	// Złe ciasteczka nie liczą się jako próby hasła: poprawne hasło nadal działa (bez blokady 429).
	zegar.ns.Store(start.UnixNano())
	if w := sr.zadanieLinku("/api/publiczny/"+l.ID+"/pobierz?sciezka=/a.txt", "haslo-linku"); w.Code != 200 {
		t.Fatalf("po złych ciasteczkach: kod %d", w.Code)
	}
}

// TestCiasteczkoLinkuWygaslyLink: ciasteczko nie ożywia wygasłego linku ani nie omija limitu pobrań.
func TestCiasteczkoLinkuWygaslyLink(t *testing.T) {
	sr, _ := srodowiskoLinkow(t)
	l, err := sr.linki.Utworz(udostepnienia.Link{Udzial: "Wspolne", Sciezka: "/kat", Autor: "jan", LimitPobran: 1}, "h")
	if err != nil {
		t.Fatal(err)
	}
	c := sr.ciasteczkoLinku(l.ID, "h")
	adres := "/api/publiczny/" + l.ID + "/pobierz?sciezka=/a.txt"
	for i, kod := range []int{200, 410, 410} {
		if w := sr.zadanieLinku(adres, "", c); w.Code != kod {
			t.Fatalf("pobranie %d: kod %d", i, w.Code)
		}
	}
	if w := sr.zadanieLinku("/api/publiczny/"+l.ID, "", c); w.Code != 410 {
		t.Fatalf("info po limicie: kod %d", w.Code)
	}
}

// nazwyWZip: posortowane nazwy plików w archiwum.
func nazwyWZip(t *testing.T, b []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var n []string
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, "/") {
			n = append(n, f.Name)
		}
	}
	sort.Strings(n)
	return n
}

func TestLinkPodfolderyIZip(t *testing.T) {
	sr, _ := srodowiskoLinkow(t)
	l := sr.nowyLinkTestowy("/kat", "haslo-linku")
	c := sr.ciasteczkoLinku(l.ID, "haslo-linku")
	baza := "/api/publiczny/" + l.ID
	przypadki := []struct {
		nazwa, adres string
		ciasteczko   bool
		kod          int
		zawiera      []string
		brak         string
	}{
		{"lista podfolderu", baza + "?sciezka=/pod", true, 200, []string{`"sciezka":"/pod/b.txt"`}, "a.txt"},
		{"podfolder poza linkiem", baza + "?sciezka=/../inny", true, 400, nil, ""},
		{"zip całego folderu", baza + "/zip", true, 200, []string{"a.txt", "b.txt"}, "c.txt"},
		{"zip podfolderu", baza + "/zip?sciezka=/pod", true, 200, []string{"b.txt"}, "a.txt"},
		{"zip poza linkiem", baza + "/zip?sciezka=/../inny", true, 400, nil, ""},
		{"zip pliku = 400", baza + "/zip?sciezka=/a.txt", true, 400, nil, ""},
		{"zip bez hasła i ciasteczka", baza + "/zip", false, 401, nil, ""},
		{"zip nieistniejącego", baza + "/zip?sciezka=/brak", true, 404, nil, ""},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			var cs []*nethttp.Cookie
			if p.ciasteczko {
				cs = append(cs, c)
			}
			w := sr.zadanieLinku(p.adres, "", cs...)
			if w.Code != p.kod {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
			tekst := w.Body.String()
			if w.Header().Get("Content-Type") == "application/zip" {
				tekst = strings.Join(nazwyWZip(t, w.Body.Bytes()), "\n")
			}
			for _, z := range p.zawiera {
				if !strings.Contains(tekst, z) {
					t.Fatalf("brak %q w %s", z, tekst)
				}
			}
			if p.brak != "" && strings.Contains(tekst, p.brak) {
				t.Fatalf("niespodziewane %q w %s", p.brak, tekst)
			}
		})
	}
	// Dwa udane ZIP-y = dwa pobrania i dwa transfery link:<id> (każdy ZIP liczony dokładnie raz).
	if got, err := sr.linki.Pobierz(l.ID); err != nil || got.Pobrano != 2 {
		t.Fatalf("pobrano %d (%v)", got.Pobrano, err)
	}
	stany, wyniki := sr.koniec.kopia()
	if len(stany) != 2 || stany[0].Kto != konfiguracja.PrefiksKtoLinku+l.ID || wyniki[0] != "ok" {
		t.Fatalf("transfery %+v %v", stany, wyniki)
	}
	var info struct{ Wpisy []struct{ Sciezka string } }
	w := sr.zadanieLinku(baza+"?sciezka=/pod", "", c)
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil || len(info.Wpisy) != 1 {
		t.Fatalf("lista: %s", w.Body)
	}
}
