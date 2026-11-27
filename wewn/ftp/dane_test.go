package ftp

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

// paraTCP: połączenie serwera (z nasłuchu owiniętego jak kanał danych) i klienta.
func paraTCP(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	st := &sterownik{}
	owiniety, err := st.WrapPassiveListener(ln)
	if err != nil {
		t.Fatal(err)
	}
	defer owiniety.Close()
	kl, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = kl.Close() }) // celowo: sprzątanie testu
	sv, err := owiniety.Accept()
	if err != nil {
		t.Fatal(err)
	}
	return sv, kl
}

// Łagodne zamknięcie: klient dostaje całość i zwykły EOF (bez RST) mimo nieprzeczytanych bajtów
// po stronie serwera; drugie Close nic nie psuje.
func TestZamknijLagodnie(t *testing.T) {
	testy := []struct {
		nazwa   string
		resztka int // bajty, które klient wysyła, a serwer ich nie czyta
		dane    int
	}{
		{"bez resztek", 0, 1000},
		{"mała resztka", 7, 64 << 10},
	}
	for _, tt := range testy {
		t.Run(tt.nazwa, func(t *testing.T) {
			sv, kl := paraTCP(t)
			if _, ok := sv.(*polaczenieDanych); !ok {
				t.Fatalf("połączenie nieowinięte: %T", sv)
			}
			wyslane := make(chan error, 1)
			go func() {
				_, err := kl.Write(bytes.Repeat([]byte("r"), tt.resztka))
				wyslane <- err
			}()
			if tt.resztka > 0 {
				if err := <-wyslane; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := sv.Write(bytes.Repeat([]byte("d"), tt.dane)); err != nil {
				t.Fatal(err)
			}
			if err := sv.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if err := sv.Close(); err != nil {
				t.Fatalf("drugie Close: %v", err)
			}
			if err := kl.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(kl)
			if err != nil || len(b) != tt.dane {
				t.Fatalf("klient: %d z %d bajtów, błąd %v", len(b), tt.dane, err)
			}
		})
	}
}
