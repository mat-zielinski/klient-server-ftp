package http

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	nethttp "net/http"
	"strings"
	"testing"

	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

func TestPobieranieRange206(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "film.bin", "0123456789")
	adres := "/api/pobierz?udzial=Filmy&sciezka=/film.bin"
	przypadki := []struct {
		nazwa    string
		naglowki map[string]string
		kod      int
		tresc    string
	}{
		{"całość", nil, nethttp.StatusOK, "0123456789"},
		{"zakres", map[string]string{"Range": "bytes=2-5"}, nethttp.StatusPartialContent, "2345"},
		{"od końca", map[string]string{"Range": "bytes=-3"}, nethttp.StatusPartialContent, "789"},
		{"poza plikiem", map[string]string{"Range": "bytes=50-60"}, nethttp.StatusRequestedRangeNotSatisfiable, ""},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: "GET", adres: adres, login: "jan", naglowki: p.naglowki})
			if w.Code != p.kod {
				t.Fatalf("kod %d", w.Code)
			}
			if p.tresc != "" && w.Body.String() != p.tresc {
				t.Fatalf("treść %q", w.Body.String())
			}
		})
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: adres, login: "jan"})
	etag := w.Header().Get("ETag")
	if etag == "" || w.Header().Get("Last-Modified") == "" || w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("nagłówki %v", w.Header())
	}
	w = sr.wykonaj(zadanie{metoda: "GET", adres: adres, login: "jan", naglowki: map[string]string{"If-None-Match": etag}})
	if w.Code != nethttp.StatusNotModified {
		t.Fatalf("If-None-Match: kod %d", w.Code)
	}
	// Transfery: 200 i 206 (cztery), bez 304 i 416; każdy zakończony z wynikiem przez callback rejestru.
	stany, wyniki := sr.koniec.kopia()
	if len(stany) != 4 {
		t.Fatalf("zakończonych transferów %d, oczekiwano 4", len(stany))
	}
	for i, st := range stany {
		if st.Kto != "jan" || wyniki[i] != transfery.WynikOk || st.Protokol != "http" || st.Przeslano == 0 {
			t.Fatalf("transfer %+v %s", st, wyniki[i])
		}
	}
	for _, wp := range sr.dziennik.kopia() {
		if wp.Rodzaj == dziennik.RodzajPobranie {
			t.Fatalf("HTTP sam zapisał pobranie w dzienniku: %+v", wp)
		}
	}
	if len(sr.rejestr.Aktywne("")) != 0 {
		t.Fatal("transfer nie został zakończony")
	}
}

func TestNazwaUTF8(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "Żółw ąę.txt", "x")
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/pobierz?udzial=Filmy&sciezka=" + urlKoduj("/Żółw ąę.txt"), login: "jan"})
	cd := w.Header().Get("Content-Disposition")
	if w.Code != nethttp.StatusOK || cd != `attachment; filename="___w __.txt"; filename*=UTF-8''%C5%BB%C3%B3%C5%82w%20%C4%85%C4%99.txt` {
		t.Fatalf("kod %d, Content-Disposition %q", w.Code, cd)
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("typ %q", w.Header().Get("Content-Type"))
	}
}

func TestNaglowekNazwy(t *testing.T) {
	przypadki := []struct {
		zalacznik bool
		nazwa     string
		wynik     string
	}{
		{true, "a.txt", `attachment; filename="a.txt"; filename*=UTF-8''a.txt`},
		{false, `x"y\z.pdf`, `inline; filename="x_y_z.pdf"; filename*=UTF-8''x%22y%5Cz.pdf`},
		{true, "a;b\r\n.txt", `attachment; filename="a;b__.txt"; filename*=UTF-8''a%3Bb%0D%0A.txt`},
		{true, "zdjęcie 1.jpg", `attachment; filename="zdj_cie 1.jpg"; filename*=UTF-8''zdj%C4%99cie%201.jpg`},
	}
	for _, p := range przypadki {
		if got := naglowekNazwy(p.zalacznik, p.nazwa); got != p.wynik {
			t.Errorf("%q: %q, oczekiwano %q", p.nazwa, got, p.wynik)
		}
	}
}

func TestTypZawartosci(t *testing.T) {
	przypadki := []struct {
		nazwa   string
		podglad bool
		typ     string
		piaskow bool
	}{
		{"a.js", false, "application/octet-stream", false},
		{"a.js", true, "text/plain; charset=utf-8", false},
		{"a.mjs", true, "text/plain; charset=utf-8", false},
		{"a.html", true, "text/plain; charset=utf-8", true},
		{"a.svg", true, "image/svg+xml", true},
		{"a.png", true, "image/png", false},
		{"a.bez", true, "application/octet-stream", false},
	}
	for _, p := range przypadki {
		typ, piaskow := typZawartosci(p.nazwa, p.podglad)
		if typ != p.typ || piaskow != p.piaskow {
			t.Errorf("%s/%v: %q %v", p.nazwa, p.podglad, typ, piaskow)
		}
	}
}

func TestZipWieluGet(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "a.txt", "aaa")
	sr.plik("Filmy", "kat/b.txt", "bbb")
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/zip?udzial=Filmy&sciezka=/a.txt&sciezka=/kat", login: "jan"})
	if w.Code != nethttp.StatusOK || w.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("kod %d: %s", w.Code, w.Body)
	}
	z, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	nazwy := []string{}
	for _, f := range z.File {
		nazwy = append(nazwy, f.Name)
	}
	if got := strings.Join(nazwy, ","); !strings.Contains(got, "a.txt") || !strings.Contains(got, "b.txt") {
		t.Fatalf("pliki w ZIP: %s", got)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "Filmy.zip") {
		t.Fatalf("nazwa %q", w.Header().Get("Content-Disposition"))
	}
}

func TestWgrywanieWKawalkach(t *testing.T) {
	sr := noweSrodowisko(t, true)
	baza := "/api/wgraj?udzial=Wspolne&sciezka=" + urlKoduj("/nowy/pod/plik.txt")
	kroki := []struct {
		metoda, parametry, cialo string
		kod                      int
		przeslano                int64
	}{
		{"PUT", "&przesuniecie=0", "abc", nethttp.StatusOK, 3},
		{"GET", "", "", nethttp.StatusOK, 3},
		{"PUT", "&przesuniecie=10", "x", nethttp.StatusConflict, 0},
		{"PUT", "&przesuniecie=3&koniec=1", "def", nethttp.StatusOK, 6},
		{"PUT", "&przesuniecie=0&koniec=1", "nowe", nethttp.StatusConflict, 0},
	}
	for i, k := range kroki {
		w := sr.wykonaj(zadanie{metoda: k.metoda, adres: baza + k.parametry, cialo: k.cialo, login: "jan"})
		if w.Code != k.kod {
			t.Fatalf("krok %d: kod %d: %s", i, w.Code, w.Body)
		}
		if k.kod == nethttp.StatusOK {
			var odp struct{ Przeslano int64 }
			if err := json.Unmarshal(w.Body.Bytes(), &odp); err != nil || odp.Przeslano != k.przeslano {
				t.Fatalf("krok %d: %s", i, w.Body)
			}
		}
	}
	if !sr.istnieje("Wspolne", "nowy/pod/plik.txt") {
		t.Fatal("brak wgranego pliku")
	}
}

func TestProtokoly(t *testing.T) {
	sr := noweSrodowisko(t, true)
	sr.plik("Filmy", "a b.mkv", "x")
	przypadki := []struct {
		nazwa, login, host string
		ftps               bool
		adresFTPS          string
	}{
		{"admin z FTPS", "admin", "192.168.0.10:8080", true, "ftps://admin@192.168.0.10:2121/Filmy/a%20b.mkv"},
		{"konto bez FTPS", "jan", "10.10.10.1:8080", false, ""},
		{"zły host", "admin", "zly\"host", true, "ftps://admin@localhost:2121/Filmy/a%20b.mkv"},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/protokoly?udzial=Filmy&sciezka=" + urlKoduj("/a b.mkv"),
				login: p.login, naglowki: map[string]string{"Host": p.host}})
			var odp struct {
				Opcje             []opcjaProtokolu
				OdciskCertyfikatu string `json:"odcisk_certyfikatu"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &odp); err != nil || len(odp.Opcje) != 5 {
				t.Fatalf("kod %d: %s", w.Code, w.Body)
			}
			ftps := odp.Opcje[1]
			if ftps.ID != "ftps" || ftps.Dostepne != p.ftps || ftps.Adres != p.adresFTPS {
				t.Fatalf("ftps %+v", ftps)
			}
			if p.ftps && p.host != "zly\"host" && !strings.HasPrefix(odp.Opcje[0].Adres, "http://"+p.host+"/api/pobierz?") {
				t.Fatalf("https %+v", odp.Opcje[0])
			}
			if odp.OdciskCertyfikatu != "AB:CD" {
				t.Fatalf("odcisk %q", odp.OdciskCertyfikatu)
			}
		})
	}
}
