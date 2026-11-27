package aplikacja

import (
	"context"
	"net"
	nethttp "net/http"

	"skrytka/wewn/ftp"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/uslugi/indeks"
	"skrytka/wewn/uslugi/miniatury"
	"skrytka/wewn/uslugi/sumy"
)

// uslugiMagazynu: usługi zależne od jednego magazynu; odtwarzane razem z nim przy zmianie udziałów.
type uslugiMagazynu struct {
	indeks    *indeks.Indeks
	miniatury *miniatury.Miniatury
	sumy      *sumy.Sumy
	// anuluj kończy indeksowanie tego magazynu (nil = nic do zatrzymania).
	anuluj context.CancelFunc
}

// noweUslugiMagazynu tworzy indeks (od razu uruchomiony w tle, do końca programu albo podmiany),
// miniatury i sumy dla magazynu.
func (a *Aplikacja) noweUslugiMagazynu(m *magazyn.Magazyn) *uslugiMagazynu {
	ctx, anuluj := context.WithCancel(a.ctx)
	x := indeks.Nowy(m)
	x.Uruchom(ctx)
	return &uslugiMagazynu{indeks: x, miniatury: miniatury.Nowe(m, ""), sumy: sumy.Nowe(m), anuluj: anuluj}
}

// zatrzymaj kończy indeksowanie starego magazynu.
func (u *uslugiMagazynu) zatrzymaj() {
	if u.anuluj != nil {
		u.anuluj()
	}
}

// liczniki dla monitora: otwarte połączenia HTTP/HTTPS i sterujące FTPS oraz aktywne transfery (HTTP i FTPS).
func (a *Aplikacja) liczniki() (polaczenia, transfery int) {
	return int(a.polaczeniaHTTP.Load()) + ftp.Polaczenia(), len(a.transfery.Aktywne(""))
}

// stanPolaczenia liczy połączenia HTTP (hak http.Server.ConnState).
func (a *Aplikacja) stanPolaczenia(_ net.Conn, s nethttp.ConnState) {
	switch s {
	case nethttp.StateNew:
		a.polaczeniaHTTP.Add(1)
	case nethttp.StateClosed, nethttp.StateHijacked:
		a.polaczeniaHTTP.Add(-1)
	}
}

// sprzatajMiniatury: raz przy starcie, w tle (limit pamięci podręcznej miniatur).
func (a *Aplikacja) sprzatajMiniatury() {
	a.serwery.wg.Add(1)
	go func() {
		defer a.serwery.wg.Done()
		if err := a.uslugiMag.Load().miniatury.Sprzataj(); err != nil {
			a.zglosBlad("Sprzątanie miniatur", err)
		}
	}()
}
