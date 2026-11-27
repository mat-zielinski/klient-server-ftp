package http

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/udostepnienia"
)

func TestSPA(t *testing.T) {
	sr := noweSrodowisko(t, false)
	przypadki := []struct {
		adres, gzip  string
		kod          int
		tresc, cache string
		kodowanie    string
	}{
		{"/", "", 200, "<!doctype html>", konfiguracja.CacheBez, ""},
		{"/pliki/Filmy/a", "", 200, "<!doctype html>", konfiguracja.CacheBez, ""},
		{"/s/abcdef", "", 200, "<!doctype html>", konfiguracja.CacheBez, ""},
		{"/brak/obrazek.png", "", 200, "<!doctype html>", konfiguracja.CacheBez, ""},
		{"/assets/app-AbCdEf12.js", "", 200, "console.log", konfiguracja.CacheZasobowZHashem, ""},
		{"/assets/app-AbCdEf12.js", "gzip, br", 200, "console.log", konfiguracja.CacheZasobowZHashem, "gzip"},
		{"/assets/styl-Zz99Yy88.css", "gzip", 200, "body", konfiguracja.CacheZasobowZHashem, "gzip"},
		{"/favicon.svg", "", 200, "<svg", konfiguracja.CacheBez, ""},
		{"/api/nieznany", "", 404, `"blad"`, "no-store", ""},
	}
	for _, p := range przypadki {
		t.Run(p.adres+" "+p.gzip, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres, naglowki: map[string]string{"Accept-Encoding": p.gzip}})
			if w.Code != p.kod || w.Header().Get("Cache-Control") != p.cache ||
				w.Header().Get("Content-Encoding") != p.kodowanie {
				t.Fatalf("kod %d, nagłówki %v", w.Code, w.Header())
			}
			tresc := w.Body.Bytes()
			if p.kodowanie == "gzip" {
				z, err := gzip.NewReader(bytes.NewReader(tresc))
				if err != nil {
					t.Fatal(err)
				}
				if tresc, err = io.ReadAll(z); err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(string(tresc), p.tresc) {
				t.Fatalf("treść %q", tresc)
			}
		})
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/"})
	w2 := sr.wykonaj(zadanie{metoda: "GET", adres: "/", naglowki: map[string]string{"If-None-Match": w.Header().Get("ETag")}})
	if w2.Code != nethttp.StatusNotModified {
		t.Fatalf("ETag index.html: kod %d", w2.Code)
	}
}

func TestKodBledu(t *testing.T) {
	przypadki := []struct {
		err error
		kod int
	}{
		{rdzen.ErrNieZnaleziono, 404}, {rdzen.ErrBrakPrawa, 403}, {rdzen.ErrTylkoOdczyt, 403},
		{rdzen.ErrPozaUdzialem, 400}, {rdzen.ErrZlaNazwa, 400}, {rdzen.ErrZleDane, 400},
		{rdzen.ErrKontoZablokowane, 429}, {rdzen.ErrJuzIstnieje, 409}, {rdzen.ErrWygaslo, 410},
		{udostepnienia.ErrZlyLink, 400}, {magazyn.ErrZlePrzesuniecie, 409}, {konta.ErrOstatniAdmin, 409},
		{konta.ErrSlabeHaslo, 400}, {errors.Join(errors.New("x"), rdzen.ErrNieZnaleziono), 404},
		{errBrakSesji, 401}, {errors.New("nieznany"), 500},
	}
	for _, p := range przypadki {
		if kod, kom := kodBledu(p.err); kod != p.kod || kom == "" {
			t.Errorf("%v: %d %q", p.err, kod, kom)
		}
	}
}

func TestWewnatrzLinku(t *testing.T) {
	przypadki := []struct {
		korzen, pod, wynik string
		blad               bool
	}{
		{"/kat", "/", "/kat", false},
		{"/kat", "/a/b.txt", "/kat/a/b.txt", false},
		{"/", "/a", "/a", false},
		{"/kat", "/../inny", "", true},
	}
	for _, p := range przypadki {
		got, err := wewnatrzLinku(p.korzen, p.pod)
		if (err != nil) != p.blad || got != p.wynik {
			t.Errorf("%s + %s: %q %v", p.korzen, p.pod, got, err)
		}
	}
}

func TestLinkPubliczny(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "kat/a.txt", "AAA")
	sr.plik("Wspolne", "kat/pod/b.txt", "BBB")
	sr.plik("Wspolne", "poza.txt", "tajne")
	w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/udostepnienia", login: "jan",
		cialo: `{"udzial":"Wspolne","sciezka":"/kat","haslo":"haslo-linku","limit_pobran":2}`})
	var l widokLinku
	if err := json.Unmarshal(w.Body.Bytes(), &l); err != nil || w.Code != 200 || !strings.HasSuffix(l.Adres, "/s/"+l.ID) {
		t.Fatalf("kod %d: %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "hash_hasla") || l.Wygasa.IsZero() {
		t.Fatalf("link %s", w.Body)
	}
	haslo := map[string]string{konfiguracja.NaglowekHaslaLinku: "haslo-linku"}
	baza := "/api/publiczny/" + l.ID
	przypadki := []struct {
		nazwa, adres string
		naglowki     map[string]string
		kod          int
		zawiera      string
	}{
		{"bez hasła", baza, nil, 401, ""},
		{"złe hasło", baza, map[string]string{konfiguracja.NaglowekHaslaLinku: "zle"}, 401, ""},
		{"lista", baza, haslo, 200, `"sciezka":"/a.txt"`},
		{"podfolder", baza + "?sciezka=/pod", haslo, 200, `"sciezka":"/pod/b.txt"`},
		{"ucieczka", baza + "?sciezka=/../poza.txt", haslo, 400, ""},
		{"pobierz plik", baza + "/pobierz?sciezka=/a.txt", haslo, 200, "AAA"},
		{"pobierz ucieczka", baza + "/pobierz?sciezka=/../poza.txt", haslo, 400, ""},
		{"nieznany link", "/api/publiczny/nieznany", nil, 404, ""},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: "GET", adres: p.adres, naglowki: p.naglowki, bezCSRF: true})
			if w.Code != p.kod || !strings.Contains(w.Body.String(), p.zawiera) {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
			if strings.Contains(w.Body.String(), "/kat/") {
				t.Fatalf("ujawniona ścieżka powyżej linku: %s", w.Body)
			}
		})
	}
	// Limit 2: jedno pobranie już było (zip folderu jako drugie), trzecie = 410.
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: baza + "/pobierz", naglowki: haslo}); w.Code != 200 {
		t.Fatalf("zip folderu: kod %d", w.Code)
	}
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: baza + "/pobierz?sciezka=/a.txt", naglowki: haslo}); w.Code != 410 {
		t.Fatalf("po limicie: kod %d", w.Code)
	}
}

func TestUsunLinkTylkoWlasny(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "a.txt", "a")
	sr.dodajKonto(rdzen.Uzytkownik{Login: "ola", Prawa: map[string]rdzen.Prawo{"Wspolne": rdzen.PrawoOdczyt}})
	l, err := sr.linki.Utworz(udostepnienia.Link{Udzial: "Wspolne", Sciezka: "/a.txt", Autor: "jan"}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []struct {
		login string
		kod   int
	}{{"ola", 403}, {"jan", 200}, {"admin", 404}} {
		w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/udostepnienia?id=" + l.ID, login: p.login})
		if w.Code != p.kod {
			t.Fatalf("%s: kod %d", p.login, w.Code)
		}
	}
}

func TestLinkAutorBezPrawa(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Wspolne", "a.txt", "a")
	l, err := sr.linki.Utworz(udostepnienia.Link{Udzial: "Wspolne", Sciezka: "/a.txt", Autor: "jan"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := sr.konta.Zmien(rdzen.Uzytkownik{Login: "jan", Prawa: map[string]rdzen.Prawo{}}); err != nil {
		t.Fatal(err)
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/publiczny/" + l.ID + "/pobierz"})
	if w.Code != nethttp.StatusForbidden {
		t.Fatalf("kod %d", w.Code)
	}
}

func TestAdministracjaKont(t *testing.T) {
	sr := noweSrodowisko(t, true)
	kroki := []struct {
		metoda, adres, cialo string
		kod                  int
	}{
		{"POST", "/api/konta", `{"login":"ola","haslo":"` + hasloTestowe + `","prawa":{"Filmy":"odczyt"}}`, 200},
		{"POST", "/api/konta", `{"login":"ola","haslo":"` + hasloTestowe + `"}`, 409},
		{"POST", "/api/konta", `{"login":"ewa","haslo":"krotkie"}`, 400},
		{"POST", "/api/konta", `{"login":"ewa","haslo":"` + hasloTestowe + `","prawa":{"Filmy":"wszystko"}}`, 400},
		{"PUT", "/api/konta?login=ola", `{"wylaczony":true}`, 200},
		{"POST", "/api/konta/haslo?login=ola", `{"haslo":"` + hasloTestowe + `x"}`, 200},
		{"DELETE", "/api/konta?login=admin", "", 400},
		{"PUT", "/api/konta?login=admin", `{"admin":false}`, 409},
		{"DELETE", "/api/konta?login=ola", "", 200},
	}
	for i, k := range kroki {
		w := sr.wykonaj(zadanie{metoda: k.metoda, adres: k.adres, cialo: k.cialo, login: "admin"})
		if w.Code != k.kod {
			t.Fatalf("krok %d: kod %d: %s", i, w.Code, w.Body)
		}
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/konta", login: "admin"})
	if strings.Contains(w.Body.String(), "hash") || strings.Contains(w.Body.String(), `"ola"`) {
		t.Fatalf("lista %s", w.Body)
	}
}

func TestUstawienia(t *testing.T) {
	sr := noweSrodowisko(t, true)
	przypadki := []struct {
		nazwa, cialo string
		kod          int
		restart      bool
	}{
		{"motyw", `{"motyw":"jasny"}`, 200, false},
		{"port", `{"port_http":9090}`, 200, true},
		{"zły port", `{"port_http":70000}`, 400, false},
		{"zły tryb", `{"tryb_sieci":"internet"}`, 400, false},
		{"zła nazwa udziału", `{"udzialy":[{"nazwa":"a/b","sciezka":"x"}]}`, 400, false},
		{"powtórzony udział", `{"udzialy":[{"nazwa":"A","sciezka":"x"},{"nazwa":"a","sciezka":"y"}]}`, 409, false},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: "PUT", adres: "/api/ustawienia", cialo: p.cialo, login: "admin"})
			if w.Code != p.kod {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
			if p.kod == 200 && strings.Contains(w.Body.String(), `"restart":true`) != p.restart {
				t.Fatalf("restart: %s", w.Body)
			}
		})
	}
	if sr.konfig.Motyw != "jasny" || sr.konfig.PortHTTP != 9090 || len(sr.konfig.Udzialy) != 4 {
		t.Fatalf("konfiguracja %+v", sr.konfig)
	}
}

func TestPulpitISiec(t *testing.T) {
	sr := noweSrodowisko(t, true)
	for _, adres := range []string{"/api/pulpit", "/api/siec", "/api/stan", "/api/zdrowie", "/api/udzialy"} {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: adres, login: "admin"})
		if w.Code != 200 || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("%s: kod %d: %s", adres, w.Code, w.Body)
		}
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/udzialy", login: "jan"})
	if strings.Contains(w.Body.String(), "Tajne") || !strings.Contains(w.Body.String(), `"prawo":"zapis"`) {
		t.Fatalf("udziały jana: %s", w.Body)
	}
	if w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/siec/zapora", login: "admin"}); w.Code != 200 {
		t.Fatalf("zapora: kod %d", w.Code)
	}
}
