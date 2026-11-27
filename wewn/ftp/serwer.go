// Pakiet ftp: serwer FTPS (jawny TLS, AUTH TLS wymagane) na tym samym magazynie, kontach, bramce,
// transferach i dzienniku co HTTP. Biblioteka: github.com/fclairamb/ftpserverlib.
package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/konta"
	"skrytka/wewn/uslugi/transfery"
)

// Zaleznosci składane w wewn/aplikacja.
type Zaleznosci struct {
	Konfiguracja konfiguracja.Konfiguracja
	Adres        string // host:port
	TLS          *tls.Config
	Magazyn      *magazyn.Magazyn
	Konta        *konta.Konta
	Dziennik     *dziennik.Dziennik
	Transfery    *transfery.Rejestr
	// AdresPubliczny: IPv4 podawany klientom w odpowiedzi PASV (tryb publiczny za NAT); puste = adres lokalny połączenia.
	AdresPubliczny string
}

// Błędy serwera (komunikaty idą do klienta FTP).
var (
	errBrakTLS           = errors.New("Brak konfiguracji TLS, FTPS wymaga certyfikatu.")
	errBrakZaleznosci    = errors.New("Brak magazynu, kont albo dziennika.")
	errZaDuzoPolaczen    = errors.New("Zbyt wiele połączeń z tego adresu.")
	errNieObslugiwane    = errors.New("Operacja nie jest obsługiwana przez serwer.")
	errWznowienieWgrania = errors.New("Wznawianie wgrywania (REST, APPE) nie jest obsługiwane.")
)

// Interfejsy zależności: produkcyjnie typy z pakietów usług, w testach zaślepki.
type zrodloKont interface {
	Zaloguj(login, haslo, ip string) (rdzen.Uzytkownik, error)
	Uzytkownik(login string) (rdzen.Uzytkownik, error)
}

type magazynPlikow interface {
	Udzialy() []konfiguracja.Udzial
	Udzial(nazwa string) (konfiguracja.Udzial, error)
	Lista(udzial, sciezka string) ([]rdzen.Wpis, error)
	Info(udzial, sciezka string) (rdzen.Wpis, error)
	Otworz(udzial, sciezka string) (*os.File, rdzen.Wpis, error)
	Zapisz(udzial, sciezka string, r io.Reader, nadpisz bool) (int64, error)
	NowyFolder(udzial, sciezka string) error
	ZmienNazwe(udzial, z, na string) error
	DoKosza(udzial, sciezka string) error
}

type zapisDziennika interface {
	Zapisz(w dziennik.Wpis)
}

// uchwytTransferu: część *transfery.Transfer używana przez FTPS.
type uchwytTransferu interface {
	CzytajZ(src io.Reader) io.Reader
	PiszDo(dst io.Writer) io.Writer
	Zakoncz(wynik string)
}

// uslugi: wszystko, czego potrzebuje sterownik.
type uslugi struct {
	kfg            konfiguracja.Konfiguracja
	adresPubliczny string // adres IPv4 w odpowiedzi PASV; puste = adres lokalny połączenia
	tls            *tls.Config
	magazyn        magazynPlikow
	konta          zrodloKont
	dziennik       zapisDziennika
	rozpocznij     func(s transfery.Stan) uchwytTransferu
	czyWolno       func(u rdzen.Uzytkownik, op rdzen.Operacja, udzial string, tylkoOdczyt bool) error
	// licznik otwartych połączeń sterujących (produkcyjnie globalny dla procesu; nil = własny sterownika)
	licznik *atomic.Int64
}

// Uruchom serwer FTPS do anulowania ctx. Logowanie tylko kont z flagą FTPS; korzeń = lista udziałów
// użytkownika jako foldery; każda operacja przez rdzen.CzyWolno; transfer = transfery z protokołem "ftps".
func Uruchom(ctx context.Context, z Zaleznosci) error {
	if z.Magazyn == nil || z.Konta == nil || z.Dziennik == nil {
		return errBrakZaleznosci
	}
	u := uslugi{
		kfg: z.Konfiguracja, adresPubliczny: z.AdresPubliczny,
		tls: z.TLS, magazyn: z.Magazyn, konta: z.Konta, dziennik: z.Dziennik,
		czyWolno: rdzen.CzyWolno, rozpocznij: rozpoczynanie(z.Transfery), licznik: &polaczenia,
	}
	if u.tls == nil {
		return errBrakTLS
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", z.Adres)
	if err != nil {
		return err
	}
	return uruchomNa(ctx, u, ln)
}

// rozpoczynanie łączy rejestr transferów z interfejsem; brak rejestru = transfer bez podglądu.
func rozpoczynanie(r *transfery.Rejestr) func(s transfery.Stan) uchwytTransferu {
	if r == nil {
		return func(transfery.Stan) uchwytTransferu { return transferPusty{} }
	}
	return func(s transfery.Stan) uchwytTransferu { return r.Rozpocznij(s) }
}

// transferPusty: gdy aplikacja nie podała rejestru.
type transferPusty struct{}

func (transferPusty) CzytajZ(src io.Reader) io.Reader { return src }
func (transferPusty) PiszDo(dst io.Writer) io.Writer  { return dst }
func (transferPusty) Zakoncz(string)                  {}

// uruchomNa obsługuje gotowy nasłuch (testy podają port wolny) do anulowania ctx.
func uruchomNa(ctx context.Context, u uslugi, ln net.Listener) error {
	if u.tls == nil {
		if ln != nil {
			_ = ln.Close() // celowo: i tak zwracamy błąd braku TLS
		}
		return errBrakTLS
	}
	ster := nowySterownik(u, ln)
	srv := ftpserver.NewFtpServer(ster)
	if err := srv.Listen(); err != nil {
		_ = ln.Close() // celowo: zwracamy błąd startu
		return err
	}
	koniec := make(chan struct{})
	defer close(koniec)
	go func() {
		select {
		case <-ctx.Done():
		case <-koniec:
		}
		_ = srv.Stop() // celowo: drugie zamknięcie nasłuchu zwraca tylko „już zamknięty”
		ster.rozlaczWszystkich()
	}()
	err := srv.Serve()
	if ctx.Err() != nil {
		return nil
	}
	return err
}

// sterownik implementuje ftpserver.MainDriver.
type sterownik struct {
	u     uslugi
	ln    net.Listener
	mu    sync.Mutex
	naIP  map[string]int
	ipKl  map[uint32]string
	klien map[uint32]ftpserver.ClientContext
}

func nowySterownik(u uslugi, ln net.Listener) *sterownik {
	if u.licznik == nil {
		u.licznik = new(atomic.Int64)
	}
	return &sterownik{
		u: u, ln: ln,
		naIP:  map[string]int{},
		ipKl:  map[uint32]string{},
		klien: map[uint32]ftpserver.ClientContext{},
	}
}

// GetSettings: wymagane szyfrowanie sterowania i danych, tylko tryb pasywny, porty z konfiguracji.
func (s *sterownik) GetSettings() (*ftpserver.Settings, error) {
	ust := &ftpserver.Settings{
		Listener:               s.ln,
		Banner:                 konfiguracja.FTPBaner,
		IdleTimeout:            konfiguracja.FTPBezczynnoscS,
		ConnectionTimeout:      konfiguracja.FTPLimitPolaczeniaDanychS,
		TLSRequired:            ftpserver.MandatoryEncryption,
		DisableActiveMode:      true,
		DisableMFMT:            true,
		DisableSite:            true,
		DefaultTransferType:    ftpserver.TransferTypeBinary,
		PasvConnectionsCheck:   ftpserver.IPMatchRequired,
		ActiveConnectionsCheck: ftpserver.IPMatchRequired,
		PublicHost:             s.u.adresPubliczny,
	}
	if r := zakresPasywny(s.u.kfg.PortyPasywne); r != nil {
		ust.PassiveTransferPortRange = *r
	}
	return ust, nil
}

// zakresPasywny: poprawny zakres z konfiguracji albo nil (wtedy port losowy systemu).
func zakresPasywny(p [2]int) *ftpserver.PortRange {
	const maksPort = 65535
	if p[0] <= 0 || p[1] < p[0] || p[1] > maksPort {
		return nil
	}
	return &ftpserver.PortRange{Start: p[0], End: p[1]}
}

// GetTLSConfig: kopia konfiguracji z co najmniej TLS 1.2.
func (s *sterownik) GetTLSConfig() (*tls.Config, error) {
	if s.u.tls == nil {
		return nil, errBrakTLS
	}
	c := s.u.tls.Clone()
	if c.MinVersion < tls.VersionTLS12 {
		c.MinVersion = tls.VersionTLS12
	}
	return c, nil
}

// polaczenia: otwarte połączenia sterujące FTPS w całym procesie (wszystkie serwery i pokolenia po restarcie).
var polaczenia atomic.Int64

// Polaczenia: liczba otwartych połączeń sterujących FTPS w procesie (licznik dla monitora).
func Polaczenia() int { return int(polaczenia.Load()) }

// ClientConnected liczy połączenia z adresu; ponad limit = odmowa (ClientDisconnected i tak zdejmie licznik).
func (s *sterownik) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	ip := adresIP(cc.RemoteAddr())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.naIP[ip]++
	s.ipKl[cc.ID()] = ip
	s.klien[cc.ID()] = cc
	s.u.licznik.Add(1)
	if s.naIP[ip] > konfiguracja.FTPMaksPolaczenNaIP {
		return errZaDuzoPolaczen.Error(), errZaDuzoPolaczen
	}
	return konfiguracja.FTPPowitanie, nil
}

// ClientDisconnected zdejmuje licznik adresu.
func (s *sterownik) ClientDisconnected(cc ftpserver.ClientContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ip, ok := s.ipKl[cc.ID()]
	if !ok {
		return
	}
	delete(s.ipKl, cc.ID())
	delete(s.klien, cc.ID())
	s.u.licznik.Add(-1)
	s.naIP[ip]--
	if s.naIP[ip] <= 0 {
		delete(s.naIP, ip)
	}
}

// rozlaczWszystkich przy zamykaniu serwera.
func (s *sterownik) rozlaczWszystkich() {
	s.mu.Lock()
	lista := make([]ftpserver.ClientContext, 0, len(s.klien))
	for _, cc := range s.klien {
		lista = append(lista, cc)
	}
	s.mu.Unlock()
	for _, cc := range lista {
		_ = cc.Close() // celowo: klient mógł się już rozłączyć
	}
}

// AuthUser: konta.Zaloguj z IP (blokada po złych hasłach w kontach), tylko konta FTPS i niewyłączone.
// Klient dostaje zawsze ten sam komunikat; powód trafia do dziennika (bez hasła).
func (s *sterownik) AuthUser(cc ftpserver.ClientContext, login, haslo string) (ftpserver.ClientDriver, error) {
	ip := adresIP(cc.RemoteAddr())
	u, err := s.u.konta.Zaloguj(login, haslo, ip)
	if err == nil && (!u.FTPS || u.Wylaczony) {
		err = errBrakFTPS
	}
	if err != nil {
		s.u.dziennik.Zapisz(dziennik.Wpis{
			Rodzaj: dziennik.RodzajLogowanieBlad, Kto: login, IP: ip, Protokol: konfiguracja.FTPProtokol,
			Wynik: dziennik.WynikBlad, Opis: err.Error(),
		})
		if errors.Is(err, rdzen.ErrKontoZablokowane) {
			return nil, rdzen.ErrKontoZablokowane
		}
		return nil, rdzen.ErrZleDane
	}
	s.u.dziennik.Zapisz(dziennik.Wpis{
		Rodzaj: dziennik.RodzajLogowanie, Kto: u.Login, IP: ip, Protokol: konfiguracja.FTPProtokol,
		Wynik: dziennik.WynikOK,
	})
	cc.SetPath("/")
	return &klient{u: s.u, login: u.Login, ip: ip}, nil
}

var errBrakFTPS = errors.New("Konto bez dostępu FTPS albo wyłączone.")

// adresIP: sam adres (bez portu) zdalnej strony.
func adresIP(a net.Addr) string {
	if a == nil {
		return ""
	}
	if t, ok := a.(*net.TCPAddr); ok {
		return t.IP.String()
	}
	host, _, err := net.SplitHostPort(a.String())
	if err != nil {
		return a.String()
	}
	return host
}
