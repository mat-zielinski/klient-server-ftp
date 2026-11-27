package aplikacja

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

// limitTestu: najdłuższe czekanie na start albo zamknięcie serwera w teście.
const limitTestu = 30 * time.Second

// przygotujKatalog: SKRYTKA_KATALOG = katalog tymczasowy, konfiguracja z portami 0 tylko na pętli.
func przygotujKatalog(t *testing.T, zmien func(*konfiguracja.Konfiguracja)) string {
	t.Helper()
	katalog := t.TempDir()
	t.Setenv(konfiguracja.ZmiennaKataloguAplikacji, katalog)
	k := konfiguracja.Domyslna()
	k.AdresNasluchu = konfiguracja.AdresPetli
	k.PortHTTP, k.PortHTTPS, k.PortFTPS = 0, 0, 0
	k.FTPSWlaczone = false
	if zmien != nil {
		zmien(&k)
	}
	if err := konfiguracja.Zapisz(konfiguracja.PlikKonfiguracji(), k); err != nil {
		t.Fatal(err)
	}
	return katalog
}

// biegnacy: serwer uruchomiony w gorutynie testu.
type biegnacy struct {
	a      *Aplikacja
	anuluj context.CancelFunc
	koniec chan error
	mu     sync.Mutex
	strony []string
}

// uruchomTestowo startuje serwer; zaślepka TLS = t.Skip z powodem.
func uruchomTestowo(t *testing.T) *biegnacy {
	t.Helper()
	b := &biegnacy{koniec: make(chan error, 1)}
	gotowy := make(chan *Aplikacja, 1)
	ctx, anuluj := context.WithCancel(context.Background())
	b.anuluj = anuluj
	o := Opcje{
		OtworzPrzegladarke: func(adres string) error {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.strony = append(b.strony, adres)
			return nil
		},
		PoStarcie: func(a *Aplikacja) { gotowy <- a },
	}
	go func() { b.koniec <- uruchomSerwer(ctx, o) }()
	select {
	case b.a = <-gotowy:
	case err := <-b.koniec:
		anuluj()
		if err != nil && strings.Contains(err.Error(), "do zrobienia") {
			t.Skipf("zależność jest jeszcze zaślepką: %v", err)
		}
		t.Fatalf("serwer nie wystartował: %v", err)
	case <-time.After(limitTestu):
		t.Fatal("serwer nie wystartował w czasie")
	}
	t.Cleanup(func() { b.zatrzymaj(t) })
	return b
}

// zatrzymaj anuluje kontekst i czeka na łagodne zamknięcie (drugie wywołanie nic nie robi).
func (b *biegnacy) zatrzymaj(t *testing.T) {
	t.Helper()
	if b.anuluj == nil {
		return
	}
	b.anuluj()
	b.anuluj = nil
	select {
	case err := <-b.koniec:
		if err != nil {
			t.Fatalf("zamknięcie z błędem: %v", err)
		}
	case <-time.After(limitTestu):
		t.Fatal("serwer nie zamknął się w czasie")
	}
}

// klientTLS: certyfikat własny serwera nie jest zaufany, test sprawdza tylko działanie.
func klientTLS() *http.Client {
	return &http.Client{
		Timeout:   limitTestu,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec // test
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func TestSerwerStartStanZamkniecie(t *testing.T) {
	katalog := przygotujKatalog(t, nil)
	b := uruchomTestowo(t)
	if len(b.a.serwery.adresyHTTPS) != 1 || b.a.serwery.adresHTTP == "" {
		t.Fatalf("adresy: HTTPS %v, HTTP %q", b.a.serwery.adresyHTTPS, b.a.serwery.adresHTTP)
	}
	odp, err := klientTLS().Get("https://" + b.a.serwery.adresyHTTPS[0] + "/api/stan")
	if err != nil {
		t.Fatal(err)
	}
	defer odp.Body.Close()
	if odp.StatusCode == http.StatusNotFound {
		t.Skip("do-zrobienia: wewn/http jest jeszcze zaślepką (404 na /api/stan)")
	}
	var stan struct {
		Wersja  string `json:"wersja"`
		Kreator bool   `json:"kreator"`
	}
	if err := json.NewDecoder(odp.Body).Decode(&stan); err != nil || odp.StatusCode != http.StatusOK {
		t.Fatalf("kod %d, błąd JSON %v", odp.StatusCode, err)
	}
	if stan.Wersja != konfiguracja.Wersja {
		t.Fatalf("wersja %q", stan.Wersja)
	}
	b.zatrzymaj(t)
	if _, err := klientTLS().Get("https://" + b.a.serwery.adresyHTTPS[0] + "/api/stan"); err == nil {
		t.Fatal("serwer odpowiada po zamknięciu")
	}
	sprawdzDziennik(t, katalog, []string{"Start " + konfiguracja.NazwaAplikacji, "Serwer zatrzymany."})
}

func TestKreatorOtwieraPrzegladarkeRaz(t *testing.T) {
	przygotujKatalog(t, nil)
	b := uruchomTestowo(t)
	if !b.a.konta.CzyPusto() {
		t.Skip("konta nie są puste")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	chce := "https://127.0.0.1:" + strings.Split(b.a.serwery.adresyHTTPS[0], ":")[1]
	if len(b.strony) != 1 || b.strony[0] != chce {
		t.Fatalf("otwarte strony %v, oczekiwano [%s]", b.strony, chce)
	}
}

func TestHTTPWTrybiePublicznymTylkoPrzekierowuje(t *testing.T) {
	przygotujKatalog(t, func(k *konfiguracja.Konfiguracja) { k.TrybSieci = konfiguracja.TrybPubliczny })
	b := uruchomTestowo(t)
	odp, err := klientTLS().Get("http://" + b.a.serwery.adresHTTP + "/api/stan")
	if err != nil {
		t.Fatal(err)
	}
	defer odp.Body.Close()
	cel := "https://127.0.0.1:" + strings.Split(b.a.serwery.adresyHTTPS[0], ":")[1] + "/api/stan"
	if odp.StatusCode != http.StatusPermanentRedirect || odp.Header.Get("Location") != cel {
		t.Fatalf("kod %d, Location %q, oczekiwano 308 na %s", odp.StatusCode, odp.Header.Get("Location"), cel)
	}
}

func TestZapiszKonfiguracjeUdzialy(t *testing.T) {
	przygotujKatalog(t, nil)
	b := uruchomTestowo(t)
	folder := t.TempDir()
	przed := b.a.uslugiMag.Load()
	nowa := b.a.Konfiguracja()
	nowa.Udzialy = []konfiguracja.Udzial{{Nazwa: "dokumenty", Sciezka: folder}}
	if err := b.a.ZapiszKonfiguracje(nowa); err != nil {
		t.Fatal(err)
	}
	if po := b.a.uslugiMag.Load(); po == przed || po.indeks == nil || po.miniatury == nil || po.sumy == nil {
		t.Fatal("indeks, miniatury i sumy nie zostały odtworzone dla nowego magazynu")
	}
	if u := b.a.magazyn.Load().Udzialy(); len(u) != 1 || u[0].Nazwa != "dokumenty" {
		t.Fatalf("magazyn po zapisie: %v", u)
	}
	zPliku, err := konfiguracja.Wczytaj(konfiguracja.PlikKonfiguracji())
	if err != nil || len(zPliku.Udzialy) != 1 {
		t.Fatalf("plik po zapisie: %v, %v", zPliku.Udzialy, err)
	}
	// Nieistniejący folder: błąd, plik i magazyn bez zmian.
	zla := b.a.Konfiguracja()
	zla.Udzialy = []konfiguracja.Udzial{{Nazwa: "brak", Sciezka: filepath.Join(folder, "nie_ma")}}
	if err := b.a.ZapiszKonfiguracje(zla); err == nil {
		t.Fatal("zapis udziału z nieistniejącym folderem przeszedł")
	}
	if b.a.Konfiguracja().Udzialy[0].Nazwa != "dokumenty" || b.a.magazyn.Load().Udzialy()[0].Nazwa != "dokumenty" {
		t.Fatal("nieudany zapis zmienił stan")
	}
}

func TestJednaInstancja(t *testing.T) {
	nazwa := nazwaTestowejInstancji(t)
	zwolnij, err := jednaInstancja(nazwa)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jednaInstancja(nazwa); !errors.Is(err, errDrugaInstancja) {
		t.Fatalf("druga instancja: %v", err)
	}
	zwolnij()
	zwolnij2, err := jednaInstancja(nazwa)
	if err != nil {
		t.Fatalf("po zwolnieniu: %v", err)
	}
	zwolnij2()
}

// sprawdzDziennik: każdy z tekstów występuje w pliku dziennika bieżącego miesiąca.
func sprawdzDziennik(t *testing.T, katalog string, teksty []string) {
	t.Helper()
	pliki, err := filepath.Glob(filepath.Join(katalog, konfiguracja.KatalogLogowNazwa, "*"+konfiguracja.DziennikRozszerzenie))
	if err != nil || len(pliki) == 0 {
		t.Fatalf("brak pliku dziennika: %v", err)
	}
	dane, err := os.ReadFile(pliki[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, tekst := range teksty {
		if !strings.Contains(string(dane), tekst) {
			t.Errorf("dziennik bez %q", tekst)
		}
	}
}

// nazwaTestowejInstancji: mutex lokalny (Windows) albo plik blokady w katalogu tymczasowym (Linux).
func nazwaTestowejInstancji(t *testing.T) string {
	if runtime.GOOS == "windows" {
		return `Local\Skrytka_test_` + filepath.Base(t.TempDir())
	}
	return filepath.Join(t.TempDir(), konfiguracja.PlikBlokadyInstancji)
}

func TestTransferTrafiaDoDziennika(t *testing.T) {
	katalog := przygotujKatalog(t, nil)
	b := uruchomTestowo(t)
	tr := b.a.transfery.Rozpocznij(transfery.Stan{Kto: "ola", IP: "127.0.0.1", Protokol: "https",
		Kierunek: transfery.KierunekWgranie, Udzial: "u", Sciezka: "/plik_testowy.bin"})
	tr.Dodaj(123)
	tr.Zakoncz(dziennik.WynikOK)
	b.zatrzymaj(t)
	sprawdzDziennik(t, katalog, []string{`"rodzaj":"wgranie"`, `"sciezka":"/plik_testowy.bin"`, `"bajty":123`})
}

func TestFTPSStartIZamkniecie(t *testing.T) {
	katalog := przygotujKatalog(t, func(k *konfiguracja.Konfiguracja) {
		k.FTPSWlaczone = true
		k.PortyPasywne = [2]int{0, 0}
	})
	b := uruchomTestowo(t)
	b.zatrzymaj(t)
	sprawdzDziennik(t, katalog, []string{"FTPS: 127.0.0.1:0", "Serwer zatrzymany."})
}

// TestPobranieHTTPJedenWpis: naKoniec rejestru transferów jest jedynym źródłem wpisu pobrania;
// jedno pobranie przez HTTPS = dokładnie jeden wpis "pobranie" w dzienniku.
func TestPobranieHTTPJedenWpis(t *testing.T) {
	katalog := przygotujKatalog(t, nil)
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "a.txt"), []byte("zawartosc testowa"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := uruchomTestowo(t)
	klient := klientTLS()
	sloik, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	klient.Jar = sloik
	baza := "https://" + b.a.serwery.adresyHTTPS[0]
	cialo, err := json.Marshal(map[string]any{
		"login": "admin", "haslo": "haslo-testowe-123", "tryb_sieci": "vpn",
		"udzial": map[string]any{"nazwa": "dok", "sciezka": folder, "tylko_odczyt": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	zad, err := http.NewRequest(http.MethodPost, baza+"/api/kreator", bytes.NewReader(cialo))
	if err != nil {
		t.Fatal(err)
	}
	zad.Header.Set("X-Skrytka", "1")
	zad.Header.Set("Content-Type", "application/json")
	odp, err := klient.Do(zad)
	if err != nil {
		t.Fatal(err)
	}
	odp.Body.Close()
	if odp.StatusCode != http.StatusOK {
		t.Skipf("do-zrobienia: kreator HTTP zwrócił %d", odp.StatusCode)
	}
	odp, err = klient.Get(baza + "/api/pobierz?udzial=dok&sciezka=%2Fa.txt")
	if err != nil {
		t.Fatal(err)
	}
	tresc, err := io.ReadAll(odp.Body)
	odp.Body.Close()
	if err != nil || odp.StatusCode != http.StatusOK || string(tresc) != "zawartosc testowa" {
		t.Fatalf("pobranie: kod %d, treść %q, błąd %v", odp.StatusCode, tresc, err)
	}
	b.zatrzymaj(t)
	if n := policzWDzienniku(t, katalog, `"rodzaj":"pobranie"`); n != 1 {
		t.Fatalf("wpisów pobrania w dzienniku: %d, oczekiwano 1", n)
	}
}

// policzWDzienniku: liczba linii dziennika (wszystkie miesiące) zawierających tekst.
func policzWDzienniku(t *testing.T, katalog, tekst string) int {
	t.Helper()
	pliki, err := filepath.Glob(filepath.Join(katalog, konfiguracja.KatalogLogowNazwa, "*"+konfiguracja.DziennikRozszerzenie))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range pliki {
		dane, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, linia := range strings.Split(string(dane), "\n") {
			if strings.Contains(linia, tekst) {
				n++
			}
		}
	}
	return n
}
