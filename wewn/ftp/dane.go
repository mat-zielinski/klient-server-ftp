package ftp

import (
	"io"
	"net"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// Zamykanie kanału danych bez zrywania (RST).
//
// Biblioteka po transferze woła tls.Conn.Close: close_notify, potem zamknięcie gniazda. Jeśli klient zdążył
// coś dosłać na kanał danych (np. własny close_notify albo rekord TLS po uzgadnianiu), a serwer tego nie
// przeczytał, system (zwłaszcza Windows) zamyka gniazdo przez RST, a RST po stronie klienta wyrzuca
// dane jeszcze nieodebrane z bufora. Objaw: curl ze Schannel kończy się kodem 56 („Recv failure: Connection
// was aborted”), część pliku nie dociera. Dlatego surowe połączenie TCP pod TLS zamykamy „z ociąganiem”:
// najpierw tylko kierunek wysyłania (FIN po close_notify), a w tle czytamy i wyrzucamy to, co jeszcze
// przyśle klient, aż sam zamknie (albo do limitu czasu i bajtów), dopiero wtedy pełne zamknięcie.

// WrapPassiveListener (rozszerzenie ftpserverlib): biblioteka nakłada TLS na nasz nasłuch, więc
// owijamy surowe połączenia TCP pod warstwą TLS.
func (s *sterownik) WrapPassiveListener(ln net.Listener) (net.Listener, error) {
	return nasluchDanych{Listener: ln}, nil
}

// nasluchDanych: nasłuch pasywny, którego połączenia zamykają się łagodnie.
type nasluchDanych struct{ net.Listener }

func (n nasluchDanych) Accept() (net.Conn, error) {
	c, err := n.Listener.Accept()
	if err != nil {
		return nil, err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		return &polaczenieDanych{TCPConn: tc}, nil
	}
	return c, nil
}

// polaczenieDanych: połączenie TCP kanału danych z łagodnym zamknięciem.
type polaczenieDanych struct {
	*net.TCPConn
	raz sync.Once
}

// Close: FIN zamiast RST; reszta od klienta czytana w tle (limit czasu i bajtów), potem pełne zamknięcie.
func (p *polaczenieDanych) Close() error {
	var err error
	p.raz.Do(func() {
		err = zamknijLagodnie(p.TCPConn, konfiguracja.FTPZamykanieDanych, konfiguracja.FTPZamykanieDanychBajty)
	})
	return err
}

// zamknijLagodnie: CloseWrite, w tle odczyt do EOF/limitu, potem Close. Błąd CloseWrite = od razu Close.
func zamknijLagodnie(c *net.TCPConn, limit time.Duration, maksBajtow int64) error {
	if err := c.CloseWrite(); err != nil {
		return c.Close()
	}
	if err := c.SetReadDeadline(time.Now().Add(limit)); err != nil {
		return c.Close()
	}
	go func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(c, maksBajtow)) // celowo: resztki od klienta są bez znaczenia
		_ = c.Close()                                             // celowo: gniazdo i tak kończy życie
	}()
	return nil
}
