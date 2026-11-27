package aplikacja

import (
	"fmt"
	nethttp "net/http"
	"reflect"
	"sync/atomic"

	widokhttp "skrytka/wewn/http"
	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
)

// obslugaPrzelaczana: obsługa HTTP podmieniana w locie (nowy magazyn po zmianie udziałów).
type obslugaPrzelaczana struct {
	biezaca atomic.Pointer[nethttp.Handler]
}

// ServeHTTP przekazuje żądanie do bieżącej obsługi.
func (o *obslugaPrzelaczana) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	h := o.biezaca.Load()
	if h == nil {
		nethttp.Error(w, "Serwer się uruchamia.", nethttp.StatusServiceUnavailable)
		return
	}
	(*h).ServeHTTP(w, r)
}

// ustaw podmienia obsługę.
func (o *obslugaPrzelaczana) ustaw(h nethttp.Handler) { o.biezaca.Store(&h) }

// Konfiguracja zwraca kopię bieżącej konfiguracji (bezpieczne dla wielu gorutyn).
func (a *Aplikacja) Konfiguracja() konfiguracja.Konfiguracja {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return kopiaKonfiguracji(a.konf)
}

// kopiaKonfiguracji: osobny wycinek udziałów, żeby wołający nie zmienił stanu aplikacji.
func kopiaKonfiguracji(k konfiguracja.Konfiguracja) konfiguracja.Konfiguracja {
	k.Udzialy = append([]konfiguracja.Udzial{}, k.Udzialy...)
	return k
}

// ZapiszKonfiguracje zapisuje plik i stan w pamięci. Zmiana udziałów działa od razu (nowy magazyn,
// nowa obsługa HTTP, restart FTPS); porty, tryb sieci i certyfikat działają po restarcie programu.
func (a *Aplikacja) ZapiszKonfiguracje(nowa konfiguracja.Konfiguracja) error {
	nowa = kopiaKonfiguracji(nowa)
	a.zapisMu.Lock()
	defer a.zapisMu.Unlock()
	stara := a.Konfiguracja()
	zmianaUdzialow := !reflect.DeepEqual(stara.Udzialy, nowa.Udzialy)
	var m *magazyn.Magazyn
	if zmianaUdzialow {
		var err error
		if m, err = magazyn.Nowy(nowa.Udzialy); err != nil {
			return fmt.Errorf("Udziały nie zostały zapisane: %w", err)
		}
	}
	if err := konfiguracja.Zapisz(a.plikKonf, nowa); err != nil {
		return fmt.Errorf("Nie można zapisać konfiguracji: %w", err)
	}
	a.mu.Lock()
	a.konf = nowa
	a.mu.Unlock()
	if zmianaUdzialow {
		a.ustawMagazyn(m)
		a.zglos(fmt.Sprintf("Zmieniono udziały (%d).", len(nowa.Udzialy)))
	}
	if wymagaRestartu(stara, nowa) {
		a.zglos("Zmiana portów, trybu sieci albo certyfikatu zacznie działać po ponownym uruchomieniu.")
	}
	return nil
}

// wymagaRestartu: pola konfiguracji czytane tylko przy starcie.
func wymagaRestartu(s, n konfiguracja.Konfiguracja) bool {
	return s.TrybSieci != n.TrybSieci || s.AdresNasluchu != n.AdresNasluchu ||
		s.PortHTTP != n.PortHTTP || s.PortHTTPS != n.PortHTTPS || s.PortFTPS != n.PortFTPS ||
		s.PortyPasywne != n.PortyPasywne || s.FTPSWlaczone != n.FTPSWlaczone ||
		s.Certyfikat != n.Certyfikat || s.Klucz != n.Klucz || s.DomenaLetsEncrypt != n.DomenaLetsEncrypt
}

// ustawMagazyn buduje usługi magazynu (indeks, miniatury, sumy) i obsługę HTTP, podmienia je, kończy
// indeksowanie starego magazynu; działający FTPS jest restartowany.
func (a *Aplikacja) ustawMagazyn(m *magazyn.Magazyn) {
	u := a.noweUslugiMagazynu(m)
	a.magazyn.Store(m)
	stare := a.uslugiMag.Swap(u)
	a.obsluga.ustaw(widokhttp.Nowy(widokhttp.Zaleznosci{
		Monitor:            a.monitor,
		Indeks:             u.indeks,
		Miniatury:          u.miniatury,
		Sumy:               u.sumy,
		Konfiguracja:       a.Konfiguracja,
		ZapiszKonfiguracje: a.ZapiszKonfiguracje,
		Magazyn:            m,
		Konta:              a.konta,
		Sesje:              a.sesje,
		Dziennik:           a.dziennik,
		Transfery:          a.transfery,
		Udostepnienia:      a.udost,
		Statyczne:          a.statyczne,
		OdciskCertyfikatu:  a.odcisk,
	}))
	if stare != nil {
		stare.zatrzymaj()
	}
	if a.ftps != nil {
		go a.ftps.przelacz() // restart FTPS trwa; żądanie HTTP nie czeka (bierze najnowszy magazyn)
	}
}
