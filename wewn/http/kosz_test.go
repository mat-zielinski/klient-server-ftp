package http

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"skrytka/wewn/uslugi/dziennik"
)

// wpisyKosza: GET /api/kosz jako admin (do sprawdzeń stanu).
func (sr *srodowisko) wpisyKosza(udzial string) []widokWpisuKosza {
	sr.t.Helper()
	w := sr.wykonaj(zadanie{metoda: "GET", adres: "/api/kosz?udzial=" + udzial, login: "admin"})
	if w.Code != 200 {
		sr.t.Fatalf("lista kosza: %d %s", w.Code, w.Body)
	}
	var l []widokWpisuKosza
	if err := json.Unmarshal(w.Body.Bytes(), &l); err != nil {
		sr.t.Fatal(err)
	}
	return l
}

// doKosza: plik w udziale i usunięcie przez API (jak użytkownik).
func (sr *srodowisko) doKosza(udzial, sciezka string) string {
	sr.t.Helper()
	sr.plik(udzial, sciezka, "treść")
	if w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/plik?udzial=" + udzial + "&sciezka=" + sciezka, login: "admin"}); w.Code != 200 {
		sr.t.Fatalf("do kosza: %d %s", w.Code, w.Body)
	}
	for _, e := range sr.wpisyKosza(udzial) {
		if e.Sciezka == sciezka {
			return e.ID
		}
	}
	sr.t.Fatalf("brak %s w koszu", sciezka)
	return ""
}

func TestKoszPrzebieg(t *testing.T) {
	sr := noweSrodowisko(t, true)
	a := &atrapaIndeksu{}
	sr.s.indeks = a
	id := sr.doKosza("Wspolne", "/k/a.txt")
	l := sr.wpisyKosza("Wspolne")
	if len(l) != 1 || l[0].Nazwa != "a.txt" || l[0].Rozmiar != int64(len("treść")) || l[0].Folder || l[0].Usunieto.IsZero() {
		t.Fatalf("lista: %+v", l)
	}
	cialo := func(id string, nadpisz bool) string {
		b, _ := json.Marshal(map[string]any{"id": id, "nadpisz": nadpisz}) // celowo: mapa zawsze się koduje
		return string(b)
	}
	sr.plik("Wspolne", "/k/a.txt", "nowy")
	if w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/kosz/przywroc?udzial=Wspolne", cialo: cialo(id, false), login: "jan"}); w.Code != 409 {
		t.Fatalf("kolizja: %d %s", w.Code, w.Body)
	}
	if w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/kosz/przywroc?udzial=Wspolne", cialo: cialo(id, true), login: "jan"}); w.Code != 200 {
		t.Fatalf("przywrócenie z nadpisaniem: %d %s", w.Code, w.Body)
	}
	if !sr.istnieje("Wspolne", "/k/a.txt") || len(sr.wpisyKosza("Wspolne")) != 1 {
		t.Fatal("przywrócony plik albo nadpisany element w koszu: źle")
	}
	nowy := sr.wpisyKosza("Wspolne")[0].ID
	if w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/kosz?udzial=Wspolne&id=" + nowy, login: "jan"}); w.Code != 200 {
		t.Fatalf("usunięcie trwałe: %d %s", w.Code, w.Body)
	}
	if len(sr.wpisyKosza("Wspolne")) != 0 {
		t.Fatal("wpis został po usunięciu trwałym")
	}
	sr.doKosza("Wspolne", "/x.txt")
	sr.doKosza("Wspolne", "/y.txt")
	w := sr.wykonaj(zadanie{metoda: "DELETE", adres: "/api/kosz/wszystko?udzial=Wspolne", login: "admin"})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"usuniete":2`) || len(sr.wpisyKosza("Wspolne")) != 0 {
		t.Fatalf("opróżnianie: %d %s", w.Code, w.Body)
	}
	zmiany := 0
	for _, e := range sr.dziennik.kopia() {
		if e.Rodzaj == dziennik.RodzajZmiana && (strings.Contains(e.Opis, "kosz") || strings.Contains(e.Opis, "Kosz")) {
			zmiany++
		}
	}
	if zmiany < 3+3 { // 3× do kosza + przywrócenie + usunięcie trwałe + opróżnienie (+2 do kosza)
		t.Errorf("za mało wpisów dziennika: %d", zmiany)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.odswiez) < 6 {
		t.Errorf("indeks nieodświeżony po zmianach kosza: %v", a.odswiez)
	}
}

func TestKoszPrawa(t *testing.T) {
	sr := noweSrodowisko(t, true)
	id := sr.doKosza("Wspolne", "/a.txt")
	idFilmy := sr.doKosza("Filmy", "/f.txt")
	ucieczka := base64.RawURLEncoding.EncodeToString([]byte("2026-11-27_10-30-00/../../x"))
	przywroc := `{"id":"` + id + `"}`
	przypadki := []struct {
		nazwa, metoda, adres, cialo, login string
		kod                                int
	}{
		{"lista bez sesji", "GET", "/api/kosz?udzial=Wspolne", "", "", 401},
		{"lista z prawem odczytu", "GET", "/api/kosz?udzial=Filmy", "", "jan", 403},
		{"lista bez prawa", "GET", "/api/kosz?udzial=Tajne", "", "jan", 403},
		{"lista nieznanego udziału", "GET", "/api/kosz?udzial=Nieznany", "", "jan", 403},
		{"lista udziału tylko do odczytu", "GET", "/api/kosz?udzial=Archiwum", "", "jan", 403},
		{"przywrócenie z prawem odczytu", "POST", "/api/kosz/przywroc?udzial=Filmy", `{"id":"` + idFilmy + `"}`, "jan", 403},
		{"usunięcie z prawem odczytu", "DELETE", "/api/kosz?udzial=Filmy&id=" + idFilmy, "", "jan", 403},
		{"opróżnienie nie-admin", "DELETE", "/api/kosz/wszystko?udzial=Wspolne", "", "jan", 403},
		{"opróżnienie tylko do odczytu", "DELETE", "/api/kosz/wszystko?udzial=Archiwum", "", "admin", 403},
		{"zły identyfikator", "POST", "/api/kosz/przywroc?udzial=Wspolne", `{"id":"!!!"}`, "jan", 400},
		{"ucieczka identyfikatorem", "DELETE", "/api/kosz?udzial=Wspolne&id=" + ucieczka, "", "jan", 400},
		{"zły JSON", "POST", "/api/kosz/przywroc?udzial=Wspolne", `{`, "jan", 400},
		{"lista z prawem zapisu", "GET", "/api/kosz?udzial=Wspolne", "", "jan", 200},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			w := sr.wykonaj(zadanie{metoda: p.metoda, adres: p.adres, cialo: p.cialo, login: p.login})
			if w.Code != p.kod {
				t.Errorf("kod %d, oczekiwano %d: %s", w.Code, p.kod, w.Body)
			}
		})
	}
	if len(sr.wpisyKosza("Wspolne")) != 1 || len(sr.wpisyKosza("Filmy")) != 1 {
		t.Fatal("odrzucone żądanie zmieniło kosz")
	}
	if w := sr.wykonaj(zadanie{metoda: "POST", adres: "/api/kosz/przywroc?udzial=Wspolne", cialo: przywroc, login: "jan"}); w.Code != 200 {
		t.Fatalf("przywrócenie z prawem zapisu: %d %s", w.Code, w.Body)
	}
}

func TestKoszCSRF(t *testing.T) {
	sr := noweSrodowisko(t, true)
	id := sr.doKosza("Wspolne", "/a.txt")
	for _, z := range []zadanie{
		{metoda: "POST", adres: "/api/kosz/przywroc?udzial=Wspolne", cialo: `{"id":"` + id + `"}`},
		{metoda: "DELETE", adres: "/api/kosz?udzial=Wspolne&id=" + id},
		{metoda: "DELETE", adres: "/api/kosz/wszystko?udzial=Wspolne"},
	} {
		z.login, z.bezCSRF = "admin", true
		if w := sr.wykonaj(z); w.Code != 403 {
			t.Errorf("%s %s bez nagłówka CSRF: %d", z.metoda, z.adres, w.Code)
		}
	}
	if len(sr.wpisyKosza("Wspolne")) != 1 || sr.istnieje("Wspolne", "/a.txt") {
		t.Fatal("żądanie bez CSRF zmieniło kosz")
	}
}
