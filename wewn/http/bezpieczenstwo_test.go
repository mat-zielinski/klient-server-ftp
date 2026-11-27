package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/dziennik"
)

func TestCSRF(t *testing.T) {
	sr := noweSrodowisko(t, true)
	przypadki := []struct {
		nazwa  string
		z      zadanie
		czy403 bool
	}{
		{"wylogowanie bez nagłówka", zadanie{metoda: "POST", adres: "/api/wylogowanie", login: "admin", bezCSRF: true}, true},
		{"folder bez nagłówka", zadanie{metoda: "POST", adres: "/api/folder?udzial=Wspolne&sciezka=/x", login: "jan", bezCSRF: true}, true},
		{"usuń bez nagłówka", zadanie{metoda: "DELETE", adres: "/api/plik?udzial=Wspolne&sciezka=/a.txt", login: "jan", bezCSRF: true}, true},
		{"wgraj bez nagłówka", zadanie{metoda: "PUT", adres: "/api/wgraj?udzial=Wspolne&sciezka=/b.txt&koniec=1", cialo: "x", login: "jan", bezCSRF: true}, true},
		{"logowanie bez nagłówka", zadanie{metoda: "POST", adres: "/api/logowanie", cialo: `{}`, bezCSRF: true}, true},
		{"GET bez nagłówka przechodzi", zadanie{metoda: "GET", adres: "/api/stan", bezCSRF: true}, false},
		{"folder z nagłówkiem przechodzi", zadanie{metoda: "POST", adres: "/api/folder?udzial=Wspolne&sciezka=/x", login: "jan"}, false},
	}
	sr.plik("Wspolne", "a.txt", "a")
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(p.z)
			if (w.Code == nethttp.StatusForbidden) != p.czy403 {
				t.Fatalf("kod %d, oczekiwano 403=%v: %s", w.Code, p.czy403, w.Body)
			}
		})
	}
	if !sr.istnieje("Wspolne", "a.txt") || sr.istnieje("Wspolne", "b.txt") {
		t.Fatal("żądanie bez nagłówka CSRF zmieniło dysk")
	}
}

func Test401BezSesji(t *testing.T) {
	sr := noweSrodowisko(t, true)
	adresy := []struct{ metoda, adres string }{
		{"GET", "/api/udzialy"}, {"GET", "/api/lista?udzial=Filmy"}, {"GET", "/api/szukaj?udzial=Filmy&q=a"},
		{"GET", "/api/pobierz?udzial=Filmy&sciezka=/a.txt"}, {"GET", "/api/zip?udzial=Filmy"},
		{"GET", "/api/protokoly?udzial=Filmy&sciezka=/a.txt"}, {"POST", "/api/folder?udzial=Filmy&sciezka=/x"},
		{"PUT", "/api/wgraj?udzial=Filmy&sciezka=/x"}, {"GET", "/api/transfery"}, {"GET", "/api/transfery/strumien"},
		{"GET", "/api/dziennik"}, {"GET", "/api/udostepnienia"}, {"GET", "/api/konta"}, {"GET", "/api/ustawienia"},
		{"GET", "/api/pulpit"}, {"DELETE", "/api/transfery?id=x"},
	}
	for _, a := range adresy {
		t.Run(a.metoda+" "+a.adres, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: a.metoda, adres: a.adres})
			if w.Code != nethttp.StatusUnauthorized {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
		})
	}
	t.Run("zły token", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/udzialy", nil)
		r.AddCookie(&nethttp.Cookie{Name: konfiguracja.NazwaCiasteczkaSesji, Value: "podrobiony"})
		w := httptest.NewRecorder()
		sr.h.ServeHTTP(w, r)
		if w.Code != nethttp.StatusUnauthorized {
			t.Fatalf("kod %d", w.Code)
		}
	})
}

func Test403BezPrawa(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Tajne", "sekret.txt", "tajne")
	sr.plik("Filmy", "film.txt", "film")
	sr.plik("Archiwum", "stary.txt", "stary")
	przypadki := []struct {
		nazwa string
		z     zadanie
	}{
		{"lista bez prawa", zadanie{metoda: "GET", adres: "/api/lista?udzial=Tajne"}},
		{"pobierz bez prawa", zadanie{metoda: "GET", adres: "/api/pobierz?udzial=Tajne&sciezka=/sekret.txt"}},
		{"zip bez prawa", zadanie{metoda: "GET", adres: "/api/zip?udzial=Tajne&sciezka=/"}},
		{"szukaj bez prawa", zadanie{metoda: "GET", adres: "/api/szukaj?udzial=Tajne&q=s"}},
		{"nieznany udział", zadanie{metoda: "GET", adres: "/api/lista?udzial=Nieistniejacy"}},
		{"folder z prawem odczytu", zadanie{metoda: "POST", adres: "/api/folder?udzial=Filmy&sciezka=/nowy"}},
		{"usuń z prawem odczytu", zadanie{metoda: "DELETE", adres: "/api/plik?udzial=Filmy&sciezka=/film.txt"}},
		{"zmień z prawem odczytu", zadanie{metoda: "POST", adres: "/api/zmien?udzial=Filmy", cialo: `{"z":"/film.txt","na":"/b.txt"}`}},
		{"wgraj do udziału tylko do odczytu", zadanie{metoda: "PUT", adres: "/api/wgraj?udzial=Archiwum&sciezka=/n.txt&koniec=1", cialo: "x"}},
		{"usuń w udziale tylko do odczytu", zadanie{metoda: "DELETE", adres: "/api/plik?udzial=Archiwum&sciezka=/stary.txt"}},
		{"link bez prawa", zadanie{metoda: "POST", adres: "/api/udostepnienia", cialo: `{"udzial":"Tajne","sciezka":"/sekret.txt"}`}},
		{"konta", zadanie{metoda: "GET", adres: "/api/konta"}},
		{"ustawienia", zadanie{metoda: "PUT", adres: "/api/ustawienia", cialo: `{}`}},
		{"statystyki dziennika", zadanie{metoda: "GET", adres: "/api/dziennik/statystyki"}},
		{"przerwij transfer", zadanie{metoda: "DELETE", adres: "/api/transfery?id=x"}},
		{"pulpit", zadanie{metoda: "GET", adres: "/api/pulpit"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			p.z.login = "jan"
			w := sr.wykonaj(p.z)
			if w.Code != nethttp.StatusForbidden {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
		})
	}
	if sr.istnieje("Filmy", "nowy") || !sr.istnieje("Filmy", "film.txt") || !sr.istnieje("Archiwum", "stary.txt") ||
		sr.istnieje("Archiwum", "n.txt") {
		t.Fatal("operacja bez prawa zmieniła dysk")
	}
}

func Test400UcieczkaSciezka(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "a.txt", "a")
	zle := []string{"/../Tajne/sekret.txt", "..", "/a/../../x", `..\..\x`, "C:/Windows/win.ini", "//serwer/udzial", "/a.txt:strumien"}
	for _, sc := range zle {
		for _, a := range []struct{ metoda, adres string }{
			{"GET", "/api/lista?udzial=Wspolne&sciezka="}, {"GET", "/api/pobierz?udzial=Wspolne&sciezka="},
			{"POST", "/api/folder?udzial=Wspolne&sciezka="}, {"PUT", "/api/wgraj?udzial=Wspolne&sciezka="},
		} {
			t.Run(a.metoda+" "+a.adres+sc, func(t *testing.T) {
				w := sr.wykonaj(zadanie{metoda: a.metoda, adres: a.adres + urlKoduj(sc), login: "jan", cialo: "x"})
				if w.Code != nethttp.StatusBadRequest {
					t.Fatalf("kod %d: %s", w.Code, w.Body)
				}
			})
		}
	}
	t.Run("zmiana nazwy poza udział", func(t *testing.T) {
		w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/zmien?udzial=Wspolne", login: "jan",
			cialo: `{"z":"/a.txt","na":"/../Tajne/a.txt"}`})
		if w.Code != nethttp.StatusBadRequest || !sr.istnieje("Wspolne", "a.txt") {
			t.Fatalf("kod %d: %s", w.Code, w.Body)
		}
	})
	t.Run("usunięcie korzenia", func(t *testing.T) {
		w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/plik?udzial=Wspolne&sciezka=/", login: "jan"})
		if w.Code != nethttp.StatusBadRequest {
			t.Fatalf("kod %d", w.Code)
		}
	})
}

func urlKoduj(s string) string {
	return strings.NewReplacer("%", "%25", "&", "%26", "#", "%23", "+", "%2B", " ", "%20", `\`, "%5C").Replace(s)
}

func TestNaglowkiBezpieczenstwa(t *testing.T) {
	sr := noweSrodowisko(t, true)
	for _, adres := range []string{"/api/stan", "/", "/api/nieznany"} {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: adres})
		h := w.Header()
		csp := h.Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "frame-ancestors 'self'") ||
			!strings.Contains(csp, "blob:") {
			t.Errorf("%s: CSP %q", adres, csp)
		}
		if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: brak nagłówków %v", adres, h)
		}
	}
}

func TestOdzyskiwaniePoPanice(t *testing.T) {
	sr := noweSrodowisko(t, false)
	h := sr.s.odzyskiwanie(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) { panic("awaria") }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/x", nil))
	if w.Code != nethttp.StatusInternalServerError {
		t.Fatalf("kod %d", w.Code)
	}
	wpisy := sr.dziennik.kopia()
	if len(wpisy) != 1 || wpisy[0].Rodzaj != dziennik.RodzajSerwer || !strings.Contains(wpisy[0].Opis, "awaria") {
		t.Fatalf("wpisy %+v", wpisy)
	}
}

func TestLimitCialaJSON(t *testing.T) {
	sr := noweSrodowisko(t, true)
	duze := `{"login":"` + strings.Repeat("a", konfiguracja.MaksCialoJSON) + `"}`
	w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/logowanie", cialo: duze})
	if w.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("kod %d", w.Code)
	}
	w = sr.wykonaj(zadanie{metoda: "POST", adres: "/api/logowanie", cialo: `{zły`})
	if w.Code != nethttp.StatusBadRequest {
		t.Fatalf("kod %d", w.Code)
	}
}

func TestLogowanie(t *testing.T) {
	sr := noweSrodowisko(t, true)
	przypadki := []struct {
		nazwa, cialo string
		kod          int
	}{
		{"dobre", `{"login":"jan","haslo":"` + hasloTestowe + `"}`, nethttp.StatusOK},
		{"złe hasło", `{"login":"jan","haslo":"zle-haslo-123"}`, nethttp.StatusUnauthorized},
		{"brak konta", `{"login":"nikt","haslo":"zle-haslo-123"}`, nethttp.StatusUnauthorized},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/logowanie", strings.NewReader(p.cialo))
			r.Header.Set(konfiguracja.NaglowekCSRF, konfiguracja.WartoscCSRF)
			r.Header.Set("X-Forwarded-For", "1.2.3.4")
			r.RemoteAddr = "10.10.10.7:4000"
			w := httptest.NewRecorder()
			sr.h.ServeHTTP(w, r)
			if w.Code != p.kod {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
			if strings.Contains(w.Body.String(), "hash") {
				t.Fatal("odpowiedź zawiera hash hasła")
			}
			if p.kod == nethttp.StatusOK {
				c := w.Result().Cookies()
				if len(c) != 1 || !c[0].HttpOnly || c[0].SameSite != nethttp.SameSiteStrictMode {
					t.Fatalf("ciasteczko %+v", c)
				}
			}
			wpisy := sr.dziennik.kopia()
			if ost := wpisy[len(wpisy)-1]; ost.IP != "10.10.10.7" {
				t.Fatalf("IP z X-Forwarded-For zamiast połączenia: %q", ost.IP)
			}
		})
	}
	t.Run("blokada 429", func(t *testing.T) {
		var w *httptest.ResponseRecorder
		for range konfiguracja.MaksBlednychLogowan + 1 {
			w = sr.wykonaj(zadanie{metoda: "POST", adres: "/api/logowanie", cialo: `{"login":"admin","haslo":"zle-haslo-123"}`})
		}
		if w.Code != nethttp.StatusTooManyRequests {
			t.Fatalf("kod %d", w.Code)
		}
	})
}

func TestKreatorTylkoGdyPusto(t *testing.T) {
	sr := noweSrodowisko(t, false)
	cialo := `{"login":"szef","haslo":"` + hasloTestowe + `","udzial":{"nazwa":"Dom","sciezka":"` +
		strings.ReplaceAll(sr.katalog, `\`, `\\`) + `","tylko_odczyt":false},"tryb_sieci":"vpn"}`
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/stan"})
	var stan map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &stan); err != nil || stan["kreator"] != true {
		t.Fatalf("stan %v %v", stan, err)
	}
	w = sr.wykonaj(zadanie{metoda: "POST", adres: "/api/kreator", cialo: cialo})
	if w.Code != nethttp.StatusOK || len(w.Result().Cookies()) != 1 {
		t.Fatalf("kod %d: %s", w.Code, w.Body)
	}
	if len(sr.zapisane) != 1 || sr.zapisane[0].Udzialy[0].Nazwa != "Dom" {
		t.Fatalf("konfiguracja %+v", sr.zapisane)
	}
	w = sr.wykonaj(zadanie{metoda: "POST", adres: "/api/kreator", cialo: strings.Replace(cialo, "szef", "intruz", 1)})
	if w.Code != nethttp.StatusForbidden {
		t.Fatalf("drugi kreator: kod %d", w.Code)
	}
	if _, err := sr.konta.Uzytkownik("intruz"); err == nil {
		t.Fatal("drugi kreator założył konto")
	}
}
