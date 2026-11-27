package ftp

import (
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/transfery"
)

// infoPliku: os.FileInfo z wpisu magazynu.
type infoPliku struct{ w rdzen.Wpis }

func (i infoPliku) Name() string       { return i.w.Nazwa }
func (i infoPliku) Size() int64        { return i.w.Rozmiar }
func (i infoPliku) ModTime() time.Time { return i.w.Zmieniono }
func (i infoPliku) IsDir() bool        { return i.w.Folder }
func (i infoPliku) Sys() any           { return nil }
func (i infoPliku) Mode() os.FileMode {
	if i.w.Folder {
		return os.ModeDir | konfiguracja.FTPPrawaFolderu
	}
	return konfiguracja.FTPPrawaPliku
}

// folderWirtualny: korzeń albo udział widziany jako folder.
func folderWirtualny(nazwa string) os.FileInfo {
	return infoPliku{w: rdzen.Wpis{Nazwa: nazwa, Sciezka: "/", Folder: true, Rodzaj: rdzen.RodzajFolder}}
}

// GetHandle (rozszerzenie ftpserverlib): O_RDONLY = pobranie (REST przez Seek), inaczej wgranie.
func (k *klient) GetHandle(nazwa string, flagi int, przesuniecie int64) (ftpserver.FileTransfer, error) {
	c, err := rozloz(nazwa)
	if err != nil {
		return nil, tlumacz(err)
	}
	if flagi&(os.O_WRONLY|os.O_RDWR) == 0 {
		return k.pobierz(c)
	}
	if flagi&os.O_APPEND != 0 || przesuniecie != 0 {
		return nil, errWznowienieWgrania
	}
	return k.wgraj(c)
}

// stan transferu do rejestru.
func (k *klient) stan(c cel, kierunek string, rozmiar int64) transfery.Stan {
	return transfery.Stan{
		Kto: k.login, IP: k.ip, Protokol: konfiguracja.FTPProtokol, Kierunek: kierunek,
		Udzial: c.udzial, Sciezka: c.sciezka, Rozmiar: rozmiar,
	}
}

func (k *klient) pobierz(c cel) (ftpserver.FileTransfer, error) {
	if c.udzial == "" || c.sciezka == "/" {
		return nil, tlumacz(rdzen.ErrNieZnaleziono)
	}
	if _, err := k.wolno(rdzen.OpPobranie, c); err != nil {
		return nil, tlumacz(err)
	}
	f, w, err := k.u.magazyn.Otworz(c.udzial, c.sciezka)
	if err != nil {
		return nil, tlumacz(err)
	}
	t := k.u.rozpocznij(k.stan(c, transfery.KierunekPobranie, w.Rozmiar))
	return &pobieranie{f: f, r: t.CzytajZ(f), t: t}, nil
}

// pobieranie: plik z magazynu liczony przez transfer; Seek obsługuje REST.
type pobieranie struct {
	f      *os.File
	r      io.Reader
	t      uchwytTransferu
	mu     sync.Mutex
	blad   bool
	koniec bool
}

func (p *pobieranie) Read(b []byte) (int, error) { return p.r.Read(b) }
func (p *pobieranie) Write([]byte) (int, error)  { return 0, errNieObslugiwane }
func (p *pobieranie) Seek(przes int64, skad int) (int64, error) {
	return p.f.Seek(przes, skad)
}

// TransferError: klient przerwał (ABOR, zerwane połączenie danych).
func (p *pobieranie) TransferError(error) {
	p.mu.Lock()
	p.blad = true
	p.mu.Unlock()
}

// Close zamyka plik i kończy transfer (wpis do dziennika robi rejestr transferów).
func (p *pobieranie) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.koniec {
		return nil
	}
	p.koniec = true
	wynik := wynikTransferu(p.blad, nil)
	err := p.f.Close()
	p.t.Zakoncz(wynik)
	return err
}

// wynikTransferu: przerwany przez klienta, błąd zapisu albo sukces.
func wynikTransferu(przerwano bool, err error) string {
	switch {
	case przerwano:
		return transfery.WynikPrzerwane
	case err != nil:
		return transfery.WynikBlad
	}
	return transfery.WynikOk
}

func (k *klient) wgraj(c cel) (ftpserver.FileTransfer, error) {
	if err := k.wolnoZapis(rdzen.OpWgranie, c); err != nil {
		return nil, tlumacz(err)
	}
	pr, pw := io.Pipe()
	t := k.u.rozpocznij(k.stan(c, transfery.KierunekWgranie, 0))
	w := &wgrywanie{pw: pw, w: t.PiszDo(pw), t: t, gotowe: make(chan error, 1)}
	go func() {
		_, err := k.u.magazyn.Zapisz(c.udzial, c.sciezka, pr, true)
		w.magazynSkonczyl.Store(true)
		if err != nil {
			pr.CloseWithError(err) // odblokowuje Write, gdy magazyn skończył błędem przed końcem danych
		} else {
			pr.CloseWithError(io.ErrClosedPipe)
		}
		w.gotowe <- err
	}()
	return w, nil
}

// errPrzerwanoWgrywanie przekazywany magazynowi, żeby porzucił plik tymczasowy.
var errPrzerwanoWgrywanie = errors.New("Wgrywanie przerwane przez klienta.")

// wgrywanie: strumień do magazyn.Zapisz przez rurę (zapis do pliku tymczasowego, zamiana na końcu).
type wgrywanie struct {
	pw              *io.PipeWriter
	w               io.Writer
	t               uchwytTransferu
	gotowe          chan error
	magazynSkonczyl atomic.Bool
	mu              sync.Mutex
	przerwano       bool
	koniec          bool
}

func (w *wgrywanie) Write(b []byte) (int, error) { return w.w.Write(b) }
func (w *wgrywanie) Read([]byte) (int, error)    { return 0, errNieObslugiwane }
func (w *wgrywanie) Seek(przes int64, skad int) (int64, error) {
	if przes == 0 && skad == io.SeekStart {
		return 0, nil
	}
	return 0, errWznowienieWgrania
}

// TransferError: przerwane wgrywanie nie może zostawić niepełnego pliku. Gdy to magazyn skończył
// wcześniej (np. błąd dysku), wynikiem jest jego błąd, nie przerwanie.
func (w *wgrywanie) TransferError(error) {
	if w.magazynSkonczyl.Load() {
		return
	}
	w.mu.Lock()
	w.przerwano = true
	w.mu.Unlock()
	w.pw.CloseWithError(errPrzerwanoWgrywanie)
}

// Close czeka na magazyn i kończy transfer.
func (w *wgrywanie) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.koniec {
		return nil
	}
	w.koniec = true
	if !w.przerwano {
		_ = w.pw.Close() // celowo: PipeWriter.Close zawsze zwraca nil
	}
	err := <-w.gotowe
	w.t.Zakoncz(wynikTransferu(w.przerwano, err))
	if w.przerwano {
		return nil
	}
	return err
}
