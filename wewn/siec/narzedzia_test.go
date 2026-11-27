package siec

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAdresPublicznyZ(t *testing.T) {
	odpowiedz := func(kod int, tresc string) *httptest.Server {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(kod)
			fmt.Fprint(w, tresc)
		}))
		t.Cleanup(s.Close)
		return s
	}
	dobry := odpowiedz(http.StatusOK, "83.1.2.3\n")
	smieci := odpowiedz(http.StatusOK, "<html>nie IP</html>")
	blad := odpowiedz(http.StatusInternalServerError, "1.2.3.4")
	dlugi := odpowiedz(http.StatusOK, strings.Repeat("9", 1000))
	klient := &http.Client{Timeout: time.Second}
	przypadki := []struct {
		nazwa  string
		uslugi []string
		chce   string
	}{
		{"pierwsza dobra", []string{dobry.URL}, "83.1.2.3"},
		{"śmieci potem dobra", []string{smieci.URL, dobry.URL}, "83.1.2.3"},
		{"błąd HTTP pomijany", []string{blad.URL}, ""},
		{"za długa odpowiedź", []string{dlugi.URL}, ""},
		{"brak usług", nil, ""},
		{"niedostępna", []string{"http://127.0.0.1:1"}, ""},
	}
	for _, p := range przypadki {
		if got := adresPublicznyZ(klient, p.uslugi); got != p.chce {
			t.Errorf("%s: adresPublicznyZ = %q, chcę %q", p.nazwa, got, p.chce)
		}
	}
}

// Limit czasu: wolna usługa nie blokuje dłużej niż limit klienta.
func TestAdresPublicznyLimitCzasu(t *testing.T) {
	zwolnij := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-zwolnij:
		case <-r.Context().Done():
		}
	}))
	defer s.Close()
	defer close(zwolnij)
	klient := &http.Client{Timeout: 50 * time.Millisecond}
	if got := adresPublicznyZ(klient, []string{s.URL}); got != "" {
		t.Fatalf("po przekroczeniu limitu chcę pusty wynik, jest %q", got)
	}
}

func TestSprawdzPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	zajety := l.Addr().(*net.TCPAddr).Port
	if ok, opis := SprawdzPort(zajety); !ok || opis == "" {
		t.Fatalf("nasłuchujący port %d: %v %q", zajety, ok, opis)
	}
	l.Close()
	if ok, _ := SprawdzPort(zajety); ok {
		t.Fatalf("zamknięty port %d zgłoszony jako nasłuchujący", zajety)
	}
	for _, zly := range []int{0, -1, 70000} {
		if ok, opis := SprawdzPort(zly); ok || !strings.Contains(opis, "Niepoprawny") {
			t.Errorf("port %d: %v %q", zly, ok, opis)
		}
	}
}

func TestListaPortow(t *testing.T) {
	przypadki := []struct {
		porty []int
		chce  string
		blad  bool
	}{
		{[]int{8443}, "8443", false},
		{[]int{8443, 8080, 2121}, "2121,8080,8443", false},
		{[]int{50002, 50000, 50001, 8443, 8443}, "8443,50000-50002", false},
		{nil, "", true},
		{[]int{0}, "", true},
		{[]int{8080, 65536}, "", true},
	}
	for _, p := range przypadki {
		got, err := listaPortow(p.porty)
		if (err != nil) != p.blad || got != p.chce {
			t.Errorf("listaPortow(%v) = %q, %v; chcę %q, błąd %v", p.porty, got, err, p.chce, p.blad)
		}
	}
}

type wywolanie struct {
	program string
	args    []string
}

func atrapaWykonawcy(t *testing.T, bladDodania error, wyjscie []byte) *[]wywolanie {
	t.Helper()
	zapis := &[]wywolanie{}
	stary := wykonawcaPolecen
	wykonawcaPolecen = func(program string, args ...string) ([]byte, error) {
		*zapis = append(*zapis, wywolanie{program, args})
		if len(args) > 2 && args[2] == "add" {
			return wyjscie, bladDodania
		}
		return nil, errors.New("Brak reguły")
	}
	t.Cleanup(func() { wykonawcaPolecen = stary })
	return zapis
}

func TestRegulaZaporyPolecenie(t *testing.T) {
	zapis := atrapaWykonawcy(t, nil, nil)
	if err := RegulaZapory("", []int{8443, 2121, 50000, 50001}); err != nil {
		t.Fatal(err)
	}
	chce := []wywolanie{
		{"netsh", []string{"advfirewall", "firewall", "delete", "rule", "name=Skrytka"}},
		{"netsh", []string{"advfirewall", "firewall", "add", "rule", "name=Skrytka", "dir=in", "action=allow",
			"protocol=TCP", "localport=2121,8443,50000-50001", "profile=any"}},
	}
	if !reflect.DeepEqual(*zapis, chce) {
		t.Fatalf("polecenia %+v, chcę %+v", *zapis, chce)
	}
}

func TestRegulaZaporyBledy(t *testing.T) {
	// CP852: „Żądana operacja wymaga podniesienia uprawnień” zaczyna się od bajtu 0xBD (Ż).
	wyjscie := []byte{0xBD, 'a', 'd', 'a', 'n', 'a'}
	zapis := atrapaWykonawcy(t, errors.New("exit status 1"), wyjscie)
	err := RegulaZapory("Skrytka", []int{8443})
	if err == nil || !strings.Contains(err.Error(), "Żadana") {
		t.Fatalf("chcę błąd z odkodowanym wyjściem netsh, jest %v", err)
	}
	// Bezpieczeństwo: nazwa z cudzysłowem, spacją lub znakami powłoki nie trafia do netsh.
	*zapis = nil
	for _, zla := range []string{`a" dir=out`, "a b", "a&calc", strings.Repeat("x", 65)} {
		if err := RegulaZapory(zla, []int{8443}); err == nil {
			t.Errorf("nazwa %q przyjęta", zla)
		}
	}
	if err := RegulaZapory("Skrytka", nil); err == nil {
		t.Error("brak portów przyjęty")
	}
	if len(*zapis) != 0 {
		t.Fatalf("przy złych danych uruchomiono netsh: %+v", *zapis)
	}
}

func TestDekodujWyjscie(t *testing.T) {
	przypadki := []struct {
		dane []byte
		chce string
	}{
		{[]byte("Ok.\r\n"), "Ok."},
		{[]byte("Żółw"), "Żółw"},
		{[]byte{0xA5, 0x86}, "ąć"},
	}
	for _, p := range przypadki {
		if got := dekodujWyjscie(p.dane); got != p.chce {
			t.Errorf("dekodujWyjscie(%v) = %q, chcę %q", p.dane, got, p.chce)
		}
	}
}
