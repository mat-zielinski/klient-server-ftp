package http

import (
	"bufio"
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

// czytnikSSE: otwarty strumień i odczyt kolejnych zdarzeń.
type czytnikSSE struct {
	odp    *nethttp.Response
	linie  *bufio.Scanner
	anuluj context.CancelFunc
}

func otworzSSE(t *testing.T, sr *srodowisko, login string) *czytnikSSE {
	t.Helper()
	serw := httptest.NewServer(sr.h)
	t.Cleanup(serw.Close)
	// Limit czasu tylko jako bezpiecznik przed zawieszeniem testu (bez czekania w normalnym przebiegu).
	ctx, anuluj := context.WithTimeout(context.Background(), 20*time.Second)
	r, err := nethttp.NewRequestWithContext(ctx, "GET", serw.URL+"/api/transfery/strumien", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.AddCookie(&nethttp.Cookie{Name: konfiguracja.NazwaCiasteczkaSesji, Value: sr.token(login)})
	odp, err := serw.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	c := &czytnikSSE{odp: odp, linie: bufio.NewScanner(odp.Body), anuluj: anuluj}
	t.Cleanup(func() { anuluj(); _ = odp.Body.Close() }) // celowo: zamknięcie w teście, błąd bez znaczenia
	return c
}

// nastepne zwraca typ i dane następnego zdarzenia (komentarze pomijane).
func (c *czytnikSSE) nastepne(t *testing.T) (string, string) {
	t.Helper()
	var typ, dane string
	for c.linie.Scan() {
		l := c.linie.Text()
		switch {
		case strings.HasPrefix(l, "event: "):
			typ = strings.TrimPrefix(l, "event: ")
		case strings.HasPrefix(l, "data: "):
			dane = strings.TrimPrefix(l, "data: ")
		case l == "" && typ != "":
			return typ, dane
		case strings.HasPrefix(l, ":"):
			return "ping", ""
		}
	}
	t.Fatalf("strumień zamknięty: %v", c.linie.Err())
	return "", ""
}

func TestSSE(t *testing.T) {
	sr := noweSrodowisko(t, true)
	admin := otworzSSE(t, sr, "admin")
	h := admin.odp.Header
	if admin.odp.StatusCode != 200 || !strings.HasPrefix(h.Get("Content-Type"), "text/event-stream") ||
		h.Get("Cache-Control") != "no-cache" {
		t.Fatalf("odpowiedź %d %v", admin.odp.StatusCode, h)
	}
	tr := sr.rejestr.Rozpocznij(transfery.Stan{Kto: "jan", Protokol: "https", Kierunek: transfery.KierunekPobranie,
		Sciezka: "/a.txt", Rozmiar: 10})
	typ, dane := admin.nastepne(t)
	var z transfery.Zdarzenie
	if typ != transfery.ZdarzenieStart || json.Unmarshal([]byte(dane), &z) != nil || z.Stan.Sciezka != "/a.txt" {
		t.Fatalf("start: %s %s", typ, dane)
	}
	sr.tykacz.kanal(konfiguracja.OkresSumySSE) <- time.Time{}
	if typ, dane := admin.nastepne(t); typ != "suma" || !strings.Contains(dane, `"aktywne":1`) {
		t.Fatalf("suma: %s %s", typ, dane)
	}
	tr.Zakoncz(transfery.WynikOk)
	if typ, dane := admin.nastepne(t); typ != transfery.ZdarzenieKoniec || !strings.Contains(dane, `"wynik":"ok"`) {
		t.Fatalf("koniec: %s %s", typ, dane)
	}
	sr.tykacz.kanal(konfiguracja.OkresPingSSE) <- time.Time{}
	if typ, _ := admin.nastepne(t); typ != "ping" {
		t.Fatalf("ping: %s", typ)
	}
}

func TestSSETylkoWlasne(t *testing.T) {
	sr := noweSrodowisko(t, true)
	jan := otworzSSE(t, sr, "jan")
	tr := sr.rejestr.Rozpocznij(transfery.Stan{Kto: "admin", Sciezka: "/cudzy.txt"})
	defer tr.Zakoncz(transfery.WynikOk)
	// Cudzy start nie trafia do strumienia jana: pierwszym zdarzeniem jest suma po takcie.
	sr.tykacz.kanal(konfiguracja.OkresSumySSE) <- time.Time{}
	if typ, dane := jan.nastepne(t); typ != "suma" || !strings.Contains(dane, `"aktywne":0`) {
		t.Fatalf("jan: %s %s", typ, dane)
	}
}

func TestPrzerwijTransfer(t *testing.T) {
	sr := noweSrodowisko(t, true)
	tr := sr.rejestr.Rozpocznij(transfery.Stan{Kto: "jan", Sciezka: "/a"})
	id := sr.rejestr.Aktywne("")[0].ID
	przypadki := []struct {
		nazwa, login, id string
		kod              int
	}{
		{"nie-admin", "jan", id, nethttp.StatusForbidden},
		{"nieznany", "admin", "brak", nethttp.StatusNotFound},
		{"admin", "admin", id, nethttp.StatusOK},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/transfery?id=" + p.id, login: p.login})
		if w.Code != p.kod {
			t.Fatalf("%s: kod %d", p.nazwa, w.Code)
		}
	}
	if _, err := tr.PiszDo(nil).Write([]byte("x")); err == nil {
		t.Fatal("transfer nie został przerwany")
	}
	tr.Zakoncz(transfery.WynikPrzerwane)
}

func TestTransferyTylkoWlasne(t *testing.T) {
	sr := noweSrodowisko(t, true)
	a := sr.rejestr.Rozpocznij(transfery.Stan{Kto: "admin", Sciezka: "/a"})
	b := sr.rejestr.Rozpocznij(transfery.Stan{Kto: "jan", Sciezka: "/b"})
	defer a.Zakoncz(transfery.WynikOk)
	defer b.Zakoncz(transfery.WynikOk)
	for _, p := range []struct {
		login string
		ile   int
	}{{"admin", 2}, {"jan", 1}} {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/transfery", login: p.login})
		var stany []transfery.Stan
		if err := json.Unmarshal(w.Body.Bytes(), &stany); err != nil || len(stany) != p.ile {
			t.Fatalf("%s: %s", p.login, w.Body)
		}
	}
}

func TestDziennikFiltrNieAdmina(t *testing.T) {
	sr := noweSrodowisko(t, true)
	przypadki := []struct {
		login, rodzaj, kto string
	}{
		{"jan", dziennik.RodzajPobranie, "jan"},
		{"admin", "logowanie", "ktos"},
	}
	for _, p := range przypadki {
		w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/dziennik?kto=ktos&rodzaj=logowanie&limit=999999", login: p.login})
		if w.Code != nethttp.StatusOK {
			t.Fatalf("kod %d: %s", w.Code, w.Body)
		}
		f := sr.dziennik.ostatniFiltr()
		if f.Kto != p.kto || f.Rodzaj != p.rodzaj || f.Limit != konfiguracja.MaksLimitDziennika {
			t.Fatalf("%s: filtr %+v", p.login, f)
		}
	}
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/dziennik/eksport?kto=ktos", login: "jan"})
	if f := sr.dziennik.ostatniFiltr(); w.Code != 200 || f.Kto != "jan" ||
		!strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("eksport: %d %+v", w.Code, f)
	}
	if w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/dziennik?od=wczoraj", login: "admin"}); w.Code != 400 {
		t.Fatalf("zła data: kod %d", w.Code)
	}
}
