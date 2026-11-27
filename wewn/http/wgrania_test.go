package http

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

// zegarTestowy: czas przesuwany z testu, czytany także przez gorutynę sprzątającą (atomowo).
type zegarTestowy struct{ ns atomic.Int64 }

func nowyZegar() *zegarTestowy {
	z := &zegarTestowy{}
	z.ns.Store(time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC).UnixNano())
	return z
}

func (z *zegarTestowy) teraz() time.Time               { return time.Unix(0, z.ns.Load()) }
func (z *zegarTestowy) przesun(d time.Duration)        { z.ns.Add(int64(d)) }
func (z *zegarTestowy) ustawW(s *serwer) *zegarTestowy { s.teraz = z.teraz; return z }

// krokWgrania: żądanie PUT (adres "" = bez żądania), przesunięcie zegara przed nim, takt sprzątacza po nim.
type krokWgrania struct {
	przed     time.Duration
	parametry string
	cialo     string
	rozmiar   string
	kod       int
	aktywne   int
	takt      bool
}

const bazaWgraniaTestu = "/api/wgraj?udzial=Wspolne&sciezka=/plik.bin"

func TestWgranieWTransferach(t *testing.T) {
	przypadki := []struct {
		nazwa     string
		kroki     []krokWgrania
		wyniki    []string
		przeslano []int64
		rozmiary  []int64
	}{
		{"dwa kawałki od pierwszego do ostatniego", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", rozmiar: "6", kod: 200, aktywne: 1},
			{przed: 59 * time.Second, parametry: "&przesuniecie=3&koniec=1", cialo: "def", rozmiar: "6", kod: 200},
		}, []string{transfery.WynikOk}, []int64{6}, []int64{6}},
		{"jeden kawałek bez nagłówka = rozmiar 0", []krokWgrania{
			{parametry: "&koniec=1", cialo: "abc", kod: 200},
		}, []string{transfery.WynikOk}, []int64{3}, []int64{0}},
		{"porzucone, sprzątanie przy następnym żądaniu", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", rozmiar: "9", kod: 200, aktywne: 1},
			{przed: konfiguracja.CzasBezczynnosciWgrania, parametry: "&koniec=1&x=", cialo: "z", kod: 200},
		}, []string{transfery.WynikPrzerwane, transfery.WynikOk}, []int64{3, 1}, []int64{9, 0}},
		{"porzucone, sprzątanie tykaczem", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", rozmiar: "9", kod: 200, aktywne: 1},
			{przed: konfiguracja.CzasBezczynnosciWgrania, takt: true},
		}, []string{transfery.WynikPrzerwane}, []int64{3}, []int64{9}},
		{"start od zera zamyka poprzedni", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", kod: 200, aktywne: 1},
			{parametry: "&przesuniecie=0&koniec=1", cialo: "xyz", kod: 200},
		}, []string{transfery.WynikPrzerwane, transfery.WynikOk}, []int64{3, 3}, []int64{0, 0}},
		{"wznowienie bez aktywnego = nowy transfer z resztą rozmiaru", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", rozmiar: "6", kod: 200, aktywne: 1},
			{przed: konfiguracja.CzasBezczynnosciWgrania, takt: true},
			{parametry: "&przesuniecie=3&koniec=1", cialo: "def", rozmiar: "6", kod: 200},
		}, []string{transfery.WynikPrzerwane, transfery.WynikOk}, []int64{3, 3}, []int64{6, 3}},
		{"zły kawałek kończy transfer błędem", []krokWgrania{
			{parametry: "&przesuniecie=0", cialo: "abc", kod: 200, aktywne: 1},
			{parametry: "&przesuniecie=10", cialo: "x", kod: 409},
		}, []string{transfery.WynikBlad}, []int64{3}, []int64{0}},
		{"zły nagłówek rozmiaru = 400 bez transferu", []krokWgrania{
			{parametry: "&koniec=1", cialo: "abc", rozmiar: "-5", kod: 400},
		}, nil, nil, nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			sr := noweSrodowisko(t, true)
			zegar := nowyZegar().ustawW(sr.s)
			zdarzenia, wypisz := sr.rejestr.Sluchaj("")
			defer wypisz()
			for i, k := range p.kroki {
				zegar.przesun(k.przed)
				if k.takt {
					sr.tykacz.kanal(konfiguracja.OkresSprzataniaWgran) <- zegar.teraz()
					czekajNaKoniec(t, zdarzenia)
					continue
				}
				w := wgrajKawalek(sr, k)
				if w.Code != k.kod {
					t.Fatalf("krok %d: kod %d: %s", i, w.Code, w.Body)
				}
				if a := sr.rejestr.Aktywne("jan"); len(a) != k.aktywne {
					t.Fatalf("krok %d: aktywne %d, chcę %d", i, len(a), k.aktywne)
				} else if k.aktywne == 1 && (a[0].Kierunek != transfery.KierunekWgranie || a[0].Rozmiar != p.rozmiary[0]) {
					t.Fatalf("krok %d: stan %+v", i, a[0])
				}
			}
			sprawdzZakonczoneWgrania(t, sr, p.wyniki, p.przeslano, p.rozmiary)
		})
	}
}

// wgrajKawalek: PUT kawałka (parametr x= podmienia plik, żeby sprawdzić sprzątanie innego wgrania).
func wgrajKawalek(sr *srodowisko, k krokWgrania) *httptest.ResponseRecorder {
	adres := bazaWgraniaTestu + k.parametry
	if strings.Contains(k.parametry, "&x=") {
		adres = strings.Replace(adres, "/plik.bin", "/inny.bin", 1)
	}
	z := zadanie{metoda: "PUT", adres: adres, cialo: k.cialo, login: "jan"}
	if k.rozmiar != "" {
		z.naglowki = map[string]string{konfiguracja.NaglowekRozmiaruCalkowitego: k.rozmiar}
	}
	return sr.wykonaj(z)
}

// czekajNaKoniec: zdarzenie „koniec” z rejestru (gorutyna sprzątająca), bez time.Sleep.
func czekajNaKoniec(t *testing.T, zdarzenia <-chan transfery.Zdarzenie) {
	t.Helper()
	limit := time.After(5 * time.Second)
	for {
		select {
		case z := <-zdarzenia:
			if z.Typ == transfery.ZdarzenieKoniec {
				return
			}
		case <-limit:
			t.Fatal("brak zdarzenia koniec po takcie sprzątacza")
		}
	}
}

// sprawdzZakonczoneWgrania: wyniki z callbacku naKoniec (jedyne źródło wpisu dziennika) i brak
// własnego wpisu „wgranie” z HTTP (inaczej w aplikacji byłyby dwa wpisy).
func sprawdzZakonczoneWgrania(t *testing.T, sr *srodowisko, wyniki []string, przeslano, rozmiary []int64) {
	t.Helper()
	stany, got := sr.koniec.kopia()
	if strings.Join(got, ",") != strings.Join(wyniki, ",") {
		t.Fatalf("wyniki %v, chcę %v", got, wyniki)
	}
	for i, s := range stany {
		if s.Kierunek != transfery.KierunekWgranie || s.Kto != "jan" || s.Przeslano != przeslano[i] ||
			(rozmiary != nil && s.Rozmiar != rozmiary[i]) {
			t.Fatalf("transfer %d: %+v, chcę przesłano %d", i, s, przeslano[i])
		}
	}
	for _, w := range sr.dziennik.kopia() {
		if w.Rodzaj == dziennik.RodzajWgranie {
			t.Fatalf("HTTP zapisał własny wpis wgrania: %+v", w)
		}
	}
	if a := sr.rejestr.Aktywne(""); len(a) != 0 {
		t.Fatalf("zostały aktywne: %+v", a)
	}
}

func TestRozmiarDoWyslania(t *testing.T) {
	for _, p := range []struct{ calkowity, przesuniecie, chce int64 }{
		{0, 0, 0}, {100, 0, 100}, {100, 40, 60}, {100, 100, 0}, {10, 40, 0},
	} {
		if got := rozmiarDoWyslania(p.calkowity, p.przesuniecie); got != p.chce {
			t.Fatalf("%d/%d: %d", p.calkowity, p.przesuniecie, got)
		}
	}
}

func TestWgranieAdminPrzerywa(t *testing.T) {
	sr := noweSrodowisko(t, true)
	nowyZegar().ustawW(sr.s)
	if w := wgrajKawalek(sr, krokWgrania{parametry: "&przesuniecie=0", cialo: "abc"}); w.Code != 200 {
		t.Fatalf("kod %d", w.Code)
	}
	a := sr.rejestr.Aktywne("jan")
	if len(a) != 1 || !sr.rejestr.Przerwij(a[0].ID) {
		t.Fatalf("aktywne %+v", a)
	}
	w := wgrajKawalek(sr, krokWgrania{parametry: "&przesuniecie=3&koniec=1", cialo: "def"})
	if w.Code == nethttp.StatusOK {
		t.Fatalf("przerwane wgranie przyjęte: %s", w.Body)
	}
	sprawdzZakonczoneWgrania(t, sr, []string{transfery.WynikPrzerwane}, []int64{3}, nil)
}
