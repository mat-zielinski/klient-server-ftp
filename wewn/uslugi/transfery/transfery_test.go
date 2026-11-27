package transfery

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
)

// zegar to sztuczny zegar do testów.
type zegar struct {
	mu sync.Mutex
	t  time.Time
}

func (z *zegar) teraz() time.Time {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.t
}

func (z *zegar) przesun(d time.Duration) time.Time {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.t = z.t.Add(d)
	return z.t
}

// nowyTestowy: rejestr bez gorutyny próbkującej (próbki ręcznie przez probkuj).
func nowyTestowy(t *testing.T, naKoniec func(Stan, string)) (*Rejestr, *zegar) {
	t.Helper()
	z := &zegar{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	r := NowyZOpcjami(naKoniec, ZZegarem(z.teraz), ZTykaczem(nil))
	t.Cleanup(r.Zamknij)
	return r, z
}

func TestPredkoscZOkna(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	tr := r.Rozpocznij(Stan{Kto: "ala", Rozmiar: 20000})
	kroki := []struct {
		dodaj      int64
		predkosc   int64
		pozostalo  int64
		dlHistorii int
	}{
		{1000, 1000, 19, 1}, // (1000-0)/1 s
		{2000, 1500, 12, 2}, // (3000-0)/2 s
		{3000, 2000, 7, 3},  // (6000-0)/3 s
		{0, 1666, 9, 4},     // okno 3 s: (6000-1000)/3 s
		{0, 1000, 14, 5},    // (6000-3000)/3 s
		{14000, 4666, 0, 6}, // (20000-6000)/3 s, wszystko przesłane
	}
	for i, k := range kroki {
		tr.Dodaj(k.dodaj)
		r.probkuj(z.przesun(time.Second))
		s := r.Aktywne("")[0]
		if s.PredkoscBps != k.predkosc || s.PozostaloS != k.pozostalo || len(s.Historia) != k.dlHistorii {
			t.Fatalf("krok %d: prędkość %d pozostało %d historia %d, chcę %d %d %d",
				i, s.PredkoscBps, s.PozostaloS, len(s.Historia), k.predkosc, k.pozostalo, k.dlHistorii)
		}
	}
	if got := r.PredkoscLaczna(); got != 4666 {
		t.Fatalf("PredkoscLaczna = %d", got)
	}
}

func TestHistoriaOgraniczona(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	tr := r.Rozpocznij(Stan{Kto: "ala"})
	for i := 1; i <= konfiguracja.TransferyProbkiHistorii+10; i++ {
		tr.Dodaj(int64(i))
		r.probkuj(z.przesun(time.Second))
	}
	h := r.Aktywne("")[0].Historia
	if len(h) != konfiguracja.TransferyProbkiHistorii {
		t.Fatalf("długość historii %d", len(h))
	}
	if h[0] != 11 || h[len(h)-1] != int64(konfiguracja.TransferyProbkiHistorii+10) {
		t.Fatalf("historia nie przesuwa się: pierwsza %d ostatnia %d", h[0], h[len(h)-1])
	}
	// Migawka to kopia: zmiana nie psuje stanu w rejestrze.
	h[0] = -5
	if r.Aktywne("")[0].Historia[0] != 11 {
		t.Fatal("Aktywne oddaje współdzieloną historię")
	}
}

func TestPozostaloITempo(t *testing.T) {
	przypadki := []struct {
		rozmiar, przeslano, predkosc, chce int64
	}{
		{0, 100, 10, -1},
		{100, 100, 0, 0},
		{100, 50, 0, -1},
		{100, 50, 10, 5},
		{100, 49, 10, 6},
	}
	for _, p := range przypadki {
		if got := pozostalo(p.rozmiar, p.przeslano, p.predkosc); got != p.chce {
			t.Errorf("pozostalo(%d,%d,%d) = %d, chcę %d", p.rozmiar, p.przeslano, p.predkosc, got, p.chce)
		}
	}
	t0 := time.Unix(0, 0)
	tempa := []struct {
		od, do probka
		chce   int64
	}{
		{probka{t0, 0}, probka{t0, 100}, 0},
		{probka{t0, 0}, probka{t0.Add(2 * time.Second), 100}, 50},
		{probka{t0, 0}, probka{t0.Add(500 * time.Millisecond), 100}, 200},
		{probka{t0, 100}, probka{t0.Add(time.Second), 100}, 0},
	}
	for _, p := range tempa {
		if got := tempo(p.od, p.do); got != p.chce {
			t.Errorf("tempo = %d, chcę %d", got, p.chce)
		}
	}
}

func TestPostepNieCzesciejNizOdstep(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	k, wypisz := r.Sluchaj("")
	defer wypisz()
	r.Rozpocznij(Stan{Kto: "ala"})
	<-k // start
	r.probkuj(z.przesun(konfiguracja.TransferyOdstepPostepu / 2))
	r.probkuj(z.przesun(konfiguracja.TransferyOdstepPostepu / 2))
	r.probkuj(z.przesun(konfiguracja.TransferyOdstepPostepu / 4))
	if len(k) != 1 {
		t.Fatalf("zdarzeń postępu %d, chcę 1", len(k))
	}
	if zd := <-k; zd.Typ != ZdarzeniePostep {
		t.Fatalf("typ %q", zd.Typ)
	}
}

func TestFiltrUzytkownika(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	kAla, wAla := r.Sluchaj("ala")
	kWszyscy, wWszyscy := r.Sluchaj("")
	defer wAla()
	defer wWszyscy()

	a := r.Rozpocznij(Stan{Kto: "ala"})
	b := r.Rozpocznij(Stan{Kto: "bolek"})
	r.probkuj(z.przesun(time.Second))
	a.Zakoncz(WynikOk)
	b.Zakoncz(WynikOk)

	przypadki := []struct {
		nazwa string
		k     <-chan Zdarzenie
		chce  int
	}{
		{"ala", kAla, 3},
		{"wszyscy", kWszyscy, 6},
	}
	for _, p := range przypadki {
		if len(p.k) != p.chce {
			t.Errorf("%s: zdarzeń %d, chcę %d", p.nazwa, len(p.k), p.chce)
		}
		for len(p.k) > 0 {
			if zd := <-p.k; p.nazwa == "ala" && zd.Stan.Kto != "ala" {
				t.Errorf("ala widzi transfer użytkownika %q", zd.Stan.Kto)
			}
		}
	}
	r.Rozpocznij(Stan{Kto: "ala"})
	r.Rozpocznij(Stan{Kto: "bolek"})
	if n := len(r.Aktywne("ala")); n != 1 {
		t.Errorf("Aktywne(ala) = %d", n)
	}
	if n := len(r.Aktywne("")); n != 2 {
		t.Errorf("Aktywne() = %d", n)
	}
	if n := len(r.Aktywne("nikt")); n != 0 {
		t.Errorf("Aktywne(nikt) = %d", n)
	}
}

func TestWolnySluchacz(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	kWolny, wWolny := r.Sluchaj("")
	defer wWolny()
	kSzybki, wSzybki := r.Sluchaj("")
	defer wSzybki()

	n := konfiguracja.TransferyBuforSluchacza * 2
	odebrane := make(chan int)
	go func() {
		wazne := 0
		for z := range kSzybki {
			if z.Typ != ZdarzeniePostep {
				wazne++
			}
			if wazne == 2*n {
				break
			}
		}
		odebrane <- wazne
	}()

	transfery := make([]*Transfer, n)
	for i := range transfery {
		transfery[i] = r.Rozpocznij(Stan{Kto: "ala"})
	}
	for i := 0; i < 3; i++ {
		r.probkuj(z.przesun(time.Second)) // postęp przy pełnym kanale ma przepaść
	}
	for _, tr := range transfery {
		tr.Zakoncz(WynikOk)
	}
	// Rejestr nie zablokował się na wolnym słuchaczu. Teraz wolny czyta wszystko.
	starty, konce, postepy := 0, 0, 0
	widziane := map[string]bool{}
	for starty+konce < 2*n {
		select {
		case zd := <-kWolny:
			switch zd.Typ {
			case ZdarzenieStart:
				starty++
				widziane[zd.Stan.ID] = true
			case ZdarzenieKoniec:
				konce++
				if !widziane[zd.Stan.ID] {
					t.Fatalf("koniec %s przed startem", zd.Stan.ID)
				}
			default:
				postepy++
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("wolny dostał tylko %d startów i %d końców z %d", starty, konce, n)
		}
	}
	if starty != n || konce != n || postepy != 0 {
		t.Fatalf("wolny: starty %d końce %d postępy %d", starty, konce, postepy)
	}
	select {
	case w := <-odebrane:
		if w != 2*n {
			t.Fatalf("szybki dostał %d zdarzeń start/koniec", w)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("szybki nie dostał wszystkich zdarzeń start/koniec")
	}
}

func TestZakonczDokladnieRaz(t *testing.T) {
	var wywolania atomic.Int32
	var wynikKoncowy atomic.Value
	r, _ := nowyTestowy(t, func(s Stan, wynik string) {
		wywolania.Add(1)
		wynikKoncowy.Store(wynik)
	})
	k, wypisz := r.Sluchaj("")
	defer wypisz()
	tr := r.Rozpocznij(Stan{Kto: "ala"})
	tr.Dodaj(42)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Zakoncz(WynikOk)
		}()
	}
	wg.Wait()
	tr.Zakoncz(WynikBlad)
	if wywolania.Load() != 1 || wynikKoncowy.Load() != WynikOk {
		t.Fatalf("naKoniec wołane %d razy, wynik %v", wywolania.Load(), wynikKoncowy.Load())
	}
	if len(r.Aktywne("")) != 0 {
		t.Fatal("transfer został w rejestrze")
	}
	konce := 0
	for len(k) > 0 {
		if zd := <-k; zd.Typ == ZdarzenieKoniec {
			konce++
			if zd.Stan.Przeslano != 42 || zd.Wynik != WynikOk {
				t.Errorf("koniec: %+v", zd)
			}
		}
	}
	if konce != 1 {
		t.Fatalf("zdarzeń koniec %d", konce)
	}
}

func TestPrzerwij(t *testing.T) {
	var wynik atomic.Value
	r, _ := nowyTestowy(t, func(s Stan, w string) { wynik.Store(w) })
	if r.Przerwij("brak") {
		t.Fatal("Przerwij nieistniejącego zwróciło true")
	}
	tr := r.Rozpocznij(Stan{Kto: "ala"})
	czyt := tr.CzytajZ(strings.NewReader(strings.Repeat("x", 100)))
	var bufor bytes.Buffer
	pis := tr.PiszDo(&bufor)
	p := make([]byte, 10)
	if _, err := czyt.Read(p); err != nil {
		t.Fatalf("odczyt przed przerwaniem: %v", err)
	}
	if _, err := pis.Write(p); err != nil {
		t.Fatalf("zapis przed przerwaniem: %v", err)
	}
	if tr.Przerwany() {
		t.Fatal("przerwany przed Przerwij")
	}
	if !r.Przerwij(tr.migawkaID()) {
		t.Fatal("Przerwij zwróciło false")
	}
	if _, err := czyt.Read(p); !errors.Is(err, ErrPrzerwany) {
		t.Fatalf("odczyt po przerwaniu: %v", err)
	}
	if _, err := pis.Write(p); !errors.Is(err, ErrPrzerwany) {
		t.Fatalf("zapis po przerwaniu: %v", err)
	}
	if _, err := io.Copy(io.Discard, czyt); !errors.Is(err, ErrPrzerwany) {
		t.Fatalf("io.Copy po przerwaniu: %v", err)
	}
	if !tr.Przerwany() {
		t.Fatal("Przerwany() = false")
	}
	tr.Zakoncz(WynikBlad)
	if wynik.Load() != WynikPrzerwane {
		t.Fatalf("wynik %v, chcę przerwane", wynik.Load())
	}
	if r.Przerwij(tr.migawkaID()) {
		t.Fatal("Przerwij zakończonego zwróciło true")
	}
}

// migawkaID: ID transferu (pole stałe od Rozpocznij).
func (t *Transfer) migawkaID() string {
	t.r.mu.Lock()
	defer t.r.mu.Unlock()
	return t.stan.ID
}

func TestLiczenieBajtow(t *testing.T) {
	r, _ := nowyTestowy(t, nil)
	tr := r.Rozpocznij(Stan{Kto: "ala", Sciezka: "/a.bin"})
	dane := strings.Repeat("abc", 1000)
	var cel bytes.Buffer
	if _, err := io.Copy(tr.PiszDo(&cel), tr.CzytajZ(strings.NewReader(dane))); err != nil {
		t.Fatal(err)
	}
	tr.Dodaj(-5) // ujemne ignorowane
	tr.Dodaj(7)
	s := r.Aktywne("")[0]
	if s.Przeslano != int64(2*len(dane)+7) || cel.String() != dane {
		t.Fatalf("przesłano %d", s.Przeslano)
	}
	if s.ID == "" || len(s.ID) != 2*konfiguracja.TransferyBajtyID || s.PozostaloS != -1 || s.Sciezka != "/a.bin" {
		t.Fatalf("stan: %+v", s)
	}
}

func TestRownoleglosc(t *testing.T) {
	r, z := nowyTestowy(t, nil)
	k, wypisz := r.Sluchaj("")
	go func() {
		for range k { // celowo: tylko opróżnia kanał
		}
	}()
	const gorutyny, kawalki, rozmiar = 16, 500, 1024
	var wg sync.WaitGroup
	transfery := make([]*Transfer, gorutyny)
	for i := range transfery {
		transfery[i] = r.Rozpocznij(Stan{Kto: "ala"})
	}
	stop := make(chan struct{})
	var tlo sync.WaitGroup
	tlo.Add(1)
	go func() {
		defer tlo.Done()
		for {
			select {
			case <-stop:
				return
			default:
				r.probkuj(z.przesun(time.Second))
				r.Aktywne("")
				r.PredkoscLaczna()
				_, w := r.Sluchaj("ala")
				w()
			}
		}
	}()
	for i := 0; i < gorutyny; i++ {
		wg.Add(1)
		go func(tr *Transfer) {
			defer wg.Done()
			czyt := tr.CzytajZ(bytes.NewReader(make([]byte, kawalki*rozmiar)))
			if _, err := io.CopyBuffer(io.Discard, czyt, make([]byte, rozmiar)); err != nil {
				t.Error(err)
			}
		}(transfery[i])
		// Ten sam transfer liczony też z innej gorutyny.
		wg.Add(1)
		go func(tr *Transfer) {
			defer wg.Done()
			for j := 0; j < kawalki; j++ {
				tr.Dodaj(1)
			}
		}(transfery[i])
	}
	wg.Wait()
	close(stop)
	tlo.Wait()
	for _, s := range r.Aktywne("") {
		if s.Przeslano != kawalki*rozmiar+kawalki {
			t.Fatalf("przesłano %d", s.Przeslano)
		}
	}
	for _, tr := range transfery {
		tr.Zakoncz(WynikOk)
	}
	wypisz()
	wypisz() // drugie wypisanie nic nie robi
}

func TestGorutynaProbkujaca(t *testing.T) {
	z := &zegar{t: time.Unix(1000, 0)}
	takt := make(chan time.Time)
	zrobione := make(chan struct{}, 1)
	r := NowyZOpcjami(nil, ZZegarem(z.teraz), ZTykaczem(takt))
	r.mu.Lock()
	r.poProbce = func() { zrobione <- struct{}{} }
	r.mu.Unlock()
	defer r.Zamknij()

	tr := r.Rozpocznij(Stan{Kto: "ala"})
	tr.Dodaj(2048)
	z.przesun(2 * time.Second)
	takt <- time.Time{}
	<-zrobione
	if got := r.Aktywne("")[0].PredkoscBps; got != 1024 {
		t.Fatalf("prędkość %d", got)
	}
	r.Zamknij()
	r.Zamknij() // drugie zamknięcie nic nie robi
}

func TestDomyslnyRejestr(t *testing.T) {
	r := Nowy(nil)
	defer r.Zamknij()
	tr := r.Rozpocznij(Stan{Kto: "ala", Kierunek: KierunekPobranie})
	tr.Zakoncz(WynikOk) // naKoniec == nil nie może paść
	if len(r.Aktywne("")) != 0 {
		t.Fatal("transfer został")
	}
}

// zrodloZer: nieskończone źródło do benchmarku.
type zrodloZer struct{}

func (zrodloZer) Read(p []byte) (int, error) { return len(p), nil }

func BenchmarkCzytajZ(b *testing.B) {
	r := NowyZOpcjami(nil, ZTykaczem(nil))
	defer r.Zamknij()
	tr := r.Rozpocznij(Stan{Kto: "ala"})
	czyt := tr.CzytajZ(zrodloZer{})
	bufor := make([]byte, 32*1024)
	b.SetBytes(int64(len(bufor)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := czyt.Read(bufor); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCzytajZRownolegle(b *testing.B) {
	r := NowyZOpcjami(nil, ZTykaczem(nil))
	defer r.Zamknij()
	tr := r.Rozpocznij(Stan{Kto: "ala"})
	b.SetBytes(32 * 1024)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		czyt := tr.CzytajZ(zrodloZer{})
		bufor := make([]byte, 32*1024)
		for pb.Next() {
			if _, err := czyt.Read(bufor); err != nil {
				b.Fatal(err)
			}
		}
	})
}
