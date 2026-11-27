package aplikacja

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/siec"
)

// serwery: HTTPS (na adresach z trybu sieci) i HTTP (tylko pętla zwrotna).
type serwery struct {
	https       *nethttp.Server
	http        *nethttp.Server
	adresyHTTPS []string // faktyczne host:port (port 0 w konfiguracji = wybrany przez system)
	adresHTTP   string
	bledy       chan error
	wg          sync.WaitGroup
}

// nowySerwerHTTP: limity czasu na nagłówki i bezczynność; bez WriteTimeout (duże pobrania).
func (a *Aplikacja) nowySerwerHTTP(h nethttp.Handler) *nethttp.Server {
	wyjscie := io.Discard
	if a.konsola != nil {
		wyjscie = a.konsola.Writer()
	}
	return &nethttp.Server{
		Handler:           h,
		ReadHeaderTimeout: konfiguracja.CzasNaglowkaHTTP,
		IdleTimeout:       konfiguracja.CzasBezczynnosciHTTP,
		MaxHeaderBytes:    konfiguracja.MaksBajtowNaglowkaHTTP,
		ConnState:         a.stanPolaczenia,
		// Błędy uzgadniania TLS od skanerów zalałyby dziennik; widoczne tylko w trybie konsoli.
		ErrorLog: log.New(wyjscie, "", log.LstdFlags),
	}
}

// start otwiera nasłuch HTTPS, HTTP i FTPS oraz zapisuje wpis startu.
func (a *Aplikacja) start(ctx context.Context) error {
	k := a.Konfiguracja()
	s := &a.serwery
	s.bledy = make(chan error, 1)
	s.https = a.nowySerwerHTTP(&a.obsluga)
	// Kopia: net/http dopisuje do NextProtos "h2"/"http/1.1"; wspólna konfiguracja (FTPS) zostaje
	// nietknięta, a protokoły z siec (np. acme-tls/1 dla Let's Encrypt) są zachowane.
	s.https.TLSConfig = a.tls.Clone()
	for _, ip := range siec.AdresyNasluchu(k) {
		ln, err := net.Listen("tcp", net.JoinHostPort(ip, strconv.Itoa(k.PortHTTPS)))
		if err != nil {
			a.zglosBlad("Nasłuch HTTPS na "+ip, err)
			continue
		}
		s.adresyHTTPS = append(s.adresyHTTPS, ln.Addr().String())
		s.obsluz(func() error { return s.https.ServeTLS(ln, "", "") })
	}
	if len(s.adresyHTTPS) == 0 {
		return errBrakNasluchu
	}
	a.startHTTP(k)
	if k.FTPSWlaczone {
		a.ftps = nowyBiegFTPS(ctx, a, zPortem(adresyFTPS(k), k.PortFTPS))
		a.ftps.przelacz()
		if k.TrybSieci == konfiguracja.TrybPubliczny {
			a.ustalAdresPubliczny(ctx)
		}
	}
	a.monitor.Uruchom(ctx)
	a.sprzatajMiniatury()
	a.uruchomSprzatanie(ctx, konfiguracja.OkresSprzataniaUdostepnien)
	a.zglos(opisStartu(k, s.adresyHTTPS, s.adresHTTP))
	return nil
}

// ustalAdresPubliczny: raz przy starcie, w tle, pyta o IP zewnętrzne; znane = restart FTPS z tym adresem
// (tryb pasywny za routerem). Błąd = adres pusty, FTPS działa dalej bez niego.
func (a *Aplikacja) ustalAdresPubliczny(ctx context.Context) {
	a.serwery.wg.Add(1)
	go func() {
		defer a.serwery.wg.Done()
		ip := a.opcje.AdresPubliczny()
		if ip == "" {
			a.zglosBlad("Adres publiczny dla FTPS", errBrakAdresuPublicznego)
			return
		}
		if ctx.Err() != nil {
			return
		}
		a.adresPubliczny.Store(ip)
		a.zglos("Adres publiczny dla FTPS: " + ip + ".")
		a.ftps.przelacz()
	}()
}

// uruchomSprzatanie: gorutyna usuwająca wygasłe linki co okres, do końca ctx.
func (a *Aplikacja) uruchomSprzatanie(ctx context.Context, okres time.Duration) {
	a.serwery.wg.Add(1)
	go func() {
		defer a.serwery.wg.Done()
		zegar := time.NewTicker(okres)
		defer zegar.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-zegar.C:
				a.sprzatajUdostepnienia()
			}
		}
	}()
}

// sprzatajUdostepnienia: jedno sprzątanie z wpisem w dzienniku (gdy coś usunięto albo był błąd).
func (a *Aplikacja) sprzatajUdostepnienia() {
	n, err := a.udost.Sprzataj()
	if err != nil {
		a.zglosBlad("Sprzątanie linków publicznych", err)
		return
	}
	if n > 0 {
		a.zglos(fmt.Sprintf("Usunięto wygasłe linki publiczne: %d.", n))
	}
}

// startHTTP: zwykłe HTTP tylko na 127.0.0.1; w trybie publicznym wyłącznie przekierowanie na HTTPS.
func (a *Aplikacja) startHTTP(k konfiguracja.Konfiguracja) {
	s := &a.serwery
	var h nethttp.Handler = &a.obsluga
	if k.TrybSieci == konfiguracja.TrybPubliczny {
		h = przekierowanieHTTPS(portZAdresu(s.adresyHTTPS[0]))
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(konfiguracja.AdresLokalnyHTTP, strconv.Itoa(k.PortHTTP)))
	if err != nil {
		a.zglosBlad("Nasłuch HTTP", err) // HTTP jest pomocnicze: serwer działa dalej na HTTPS
		return
	}
	s.adresHTTP = ln.Addr().String()
	s.http = a.nowySerwerHTTP(h)
	s.obsluz(func() error { return s.http.Serve(ln) })
}

// zPortem dokleja port do każdego IP.
func zPortem(ip []string, port int) []string {
	wynik := make([]string, 0, len(ip))
	for _, a := range ip {
		wynik = append(wynik, net.JoinHostPort(a, strconv.Itoa(port)))
	}
	return wynik
}

// obsluz uruchamia serwer w gorutynie; błąd inny niż zamknięcie trafia do kanału bledy.
func (s *serwery) obsluz(f func() error) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := f(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			select {
			case s.bledy <- err:
			default: // celowo: wystarczy pierwszy błąd, serwer i tak się zamyka
			}
		}
	}()
}

// zatrzymaj zamyka łagodnie serwery (limit CzasZamykania, potem zerwanie połączeń) i FTPS.
func (a *Aplikacja) zatrzymaj() {
	ctx, anuluj := context.WithTimeout(context.Background(), konfiguracja.CzasZamykania)
	defer anuluj()
	for _, srv := range []*nethttp.Server{a.serwery.https, a.serwery.http} {
		if srv == nil {
			continue
		}
		if err := srv.Shutdown(ctx); err != nil {
			a.zglosBlad("Zamykanie serwera HTTP", err)
			if err := srv.Close(); err != nil {
				a.zglosBlad("Zrywanie połączeń HTTP", err)
			}
		}
	}
	if a.ftps != nil {
		a.ftps.zatrzymaj()
	}
	a.serwery.wg.Wait()
	a.transfery.Zamknij()
	a.zglos("Serwer zatrzymany.")
}

// adresLokalnyHTTPS: strona otwierana przy pierwszym uruchomieniu.
func (s *serwery) adresLokalnyHTTPS() string { return adresKreatora(s.adresyHTTPS) }

// adresKreatora: https://127.0.0.1:<port>, gdy słuchamy na pętli albo wszystkich adresach;
// inaczej pierwszy adres nasłuchu.
func adresKreatora(adresy []string) string {
	if len(adresy) == 0 {
		return ""
	}
	for _, a := range adresy {
		host, port, err := net.SplitHostPort(a)
		if err != nil {
			continue
		}
		ip := net.ParseIP(host)
		if ip != nil && (ip.IsLoopback() || ip.IsUnspecified()) {
			return konfiguracja.SchematHTTPS + "://" + net.JoinHostPort(konfiguracja.AdresPetli, port)
		}
	}
	return konfiguracja.SchematHTTPS + "://" + adresy[0]
}

// portZAdresu: port z host:port (0 przy błędzie).
func portZAdresu(adres string) int {
	_, p, err := net.SplitHostPort(adres)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	return n
}

// przekierowanieHTTPS: 308 na ten sam host (tylko IP albo localhost, inaczej 127.0.0.1) z portem HTTPS.
func przekierowanieHTTPS(portHTTPS int) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		cel := url.URL{
			Scheme:   konfiguracja.SchematHTTPS,
			Host:     net.JoinHostPort(hostPrzekierowania(r.Host), strconv.Itoa(portHTTPS)),
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		nethttp.Redirect(w, r, cel.String(), nethttp.StatusPermanentRedirect)
	})
}

// hostPrzekierowania: nagłówek Host bez portu, jeśli to adres IP albo localhost; inaczej pętla zwrotna
// (HTTP słucha tylko na 127.0.0.1, więc dowolna nazwa z nagłówka nie może kierować w obce miejsce).
func hostPrzekierowania(naglowek string) string {
	host := naglowek
	if h, _, err := net.SplitHostPort(naglowek); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if net.ParseIP(host) != nil || strings.EqualFold(host, konfiguracja.NazwaHostaLokalnego) {
		return host
	}
	return konfiguracja.AdresLokalnyHTTP
}

// opisStartu: wpis dziennika ze startem (wersja i adresy).
func opisStartu(k konfiguracja.Konfiguracja, https []string, http string) string {
	opis := fmt.Sprintf("Start %s %s, tryb %s, HTTPS: %s", konfiguracja.NazwaAplikacji, wersjaDoPokazania(),
		k.TrybSieci, strings.Join(https, ", "))
	if http != "" {
		opis += ", HTTP: " + http
	}
	if k.FTPSWlaczone {
		opis += ", FTPS: " + strings.Join(zPortem(adresyFTPS(k), k.PortFTPS), ", ")
	}
	return opis + "."
}

// wersjaDoPokazania: WersjaBudowy (wydanie), a przy budowie lokalnej Wersja.
func wersjaDoPokazania() string {
	if konfiguracja.WersjaBudowy != "" {
		return konfiguracja.WersjaBudowy
	}
	return konfiguracja.Wersja
}
