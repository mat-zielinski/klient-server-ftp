//go:build windows || linux

package aplikacja

import (
	"bytes"
	"fmt"
	"slices"
	"time"

	"fyne.io/systray"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/siec"
)

// petlaZasobnika: systray.Run na bieżącym wątku (główny: biblioteka blokuje go w init) do
// zakonczPetleZasobnika. Panika biblioteki (np. brak D-Bus przy zamykaniu na Linuksie) = błąd.
func petlaZasobnika(z *zasobnik) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w (%v)", errPetlaZasobnika, r)
		}
	}()
	systray.Run(func() { z.gotowe() }, nil)
	return nil
}

// zakonczPetleZasobnika: systray.Quit (bezpieczne wielokrotnie i z dowolnej gorutyny).
func zakonczPetleZasobnika() { systray.Quit() }

// menuZasobnika: pozycje menu i ostatnio pokazany stan (zmieniamy tylko to, co się zmieniło).
type menuZasobnika struct {
	otworz, kopiuj, transfery, brak, autostart, folder, zakoncz *systray.MenuItem
	pozycje                                                     []*systray.MenuItem

	adres         string
	adresCzas     time.Time
	dymek         string
	ikona         []byte // ostatnio ustawiona (nil = jeszcze żadna)
	jasnyPasek    bool
	motywCzas     time.Time
	tekstyPozycji []string
}

// gotowe (onReady, własna gorutyna biblioteki): ikona od razu, menu po starcie serwera, potem pętla
// zdarzeń menu i odświeżania co ZasobnikOkresOdswiezania do końca serwera.
func (z *zasobnik) gotowe() {
	systray.SetIcon(ikonaZasobnika(false, czyJasnyPasek()))
	systray.SetTitle(konfiguracja.NazwaAplikacji)
	systray.SetTooltip(tekstDymka("", nil, konfiguracja.ZasobnikTekstStartuje))
	var a *Aplikacja
	select {
	case a = <-z.gotowa:
	case <-z.ctx.Done():
		return
	}
	m := zbudujMenu()
	systray.SetOnTapped(func() { go z.otworzPanel(a) })
	m.odswiez(a, time.Now())
	z.obsluguj(a, m)
}

// zbudujMenu: nagłówek, akcje, podmenu transferów, autostart, folder, Zakończ.
func zbudujMenu() *menuZasobnika {
	m := &menuZasobnika{}
	naglowek := systray.AddMenuItem(konfiguracja.NazwaAplikacji+" "+wersjaDoPokazania(), "")
	naglowek.Disable()
	m.otworz = systray.AddMenuItem(konfiguracja.ZasobnikMenuOtworz, "")
	m.kopiuj = systray.AddMenuItem(konfiguracja.ZasobnikMenuKopiuj, "")
	m.transfery = systray.AddMenuItem(konfiguracja.ZasobnikMenuTransfery, "")
	m.brak = m.transfery.AddSubMenuItem(konfiguracja.ZasobnikMenuBrakTransfer, "")
	m.brak.Disable()
	for i := 0; i < konfiguracja.ZasobnikMaksTransferow; i++ {
		p := m.transfery.AddSubMenuItem("", "")
		p.Disable()
		p.Hide()
		m.pozycje = append(m.pozycje, p)
	}
	systray.AddSeparator()
	wlaczony, err := czyAutostart()
	m.autostart = systray.AddMenuItemCheckbox(etykietaAutostartu, "", err == nil && wlaczony)
	m.folder = systray.AddMenuItem(konfiguracja.ZasobnikMenuFolder, "")
	systray.AddSeparator()
	m.zakoncz = systray.AddMenuItem(konfiguracja.ZasobnikMenuZakoncz, "")
	return m
}

// obsluguj: jedna gorutyna na kliknięcia i odświeżanie.
func (z *zasobnik) obsluguj(a *Aplikacja, m *menuZasobnika) {
	t := time.NewTicker(konfiguracja.ZasobnikOkresOdswiezania)
	defer t.Stop()
	for {
		select {
		case <-z.ctx.Done():
			systray.SetTooltip(tekstDymka("", nil, konfiguracja.ZasobnikTekstZamykanie))
			return
		case teraz := <-t.C:
			m.odswiez(a, teraz)
		case <-m.otworz.ClickedCh:
			z.otworzPanel(a)
		case <-m.kopiuj.ClickedCh:
			m.adresCzas = time.Time{} // świeży adres (sieć mogła się zmienić)
			m.odswiez(a, time.Now())
			if err := kopiujDoSchowka(konfiguracja.SchematHTTPS + "://" + m.adres); err != nil {
				z.zglosBlad("Kopiowanie adresu do schowka", err)
			}
		case <-m.autostart.ClickedCh:
			z.przelaczAutostart(m.autostart)
		case <-m.folder.ClickedCh:
			if err := otworzFolder(konfiguracja.KatalogAplikacji()); err != nil {
				z.zglosBlad("Otwieranie folderu danych", err)
			}
		case <-m.zakoncz.ClickedCh:
			m.zakoncz.Disable()
			systray.SetTooltip(tekstDymka("", nil, konfiguracja.ZasobnikTekstZamykanie))
			z.anuluj() // serwer zamyka się łagodnie, potem pętla zasobnika kończy się sama
		}
	}
}

// odswiez: dymek, ikona i podmenu z rejestru transferów; adres co ZasobnikOkresAdresu.
func (m *menuZasobnika) odswiez(a *Aplikacja, teraz time.Time) {
	if m.adres == "" || teraz.Sub(m.adresCzas) >= konfiguracja.ZasobnikOkresAdresu {
		publiczny, _ := a.adresPubliczny.Load().(string) // celowo: brak wartości = adres nieznany
		m.adres = adresDlaKlientow(a.Konfiguracja(), siec.AdresyLokalne(), publiczny,
			portZAdresu(pierwszy(a.serwery.adresyHTTPS)))
		m.adresCzas = teraz
	}
	stany := a.transfery.Aktywne("")
	if d := tekstDymka(m.adres, stany, ""); d != m.dymek {
		m.dymek = d
		systray.SetTooltip(d)
	}
	if m.motywCzas.IsZero() || teraz.Sub(m.motywCzas) >= konfiguracja.ZasobnikOkresMotywu {
		m.jasnyPasek = czyJasnyPasek()
		m.motywCzas = teraz
	}
	if ikona := ikonaZasobnika(czyAktywne(stany), m.jasnyPasek); !bytes.Equal(ikona, m.ikona) {
		m.ikona = ikona
		systray.SetIcon(ikona)
	}
	m.ustawPozycje(pozycjeTransferow(stany, len(m.pozycje)))
}

// ustawPozycje pokazuje teksty w podmenu (puste = „Brak aktywnych transferów”).
func (m *menuZasobnika) ustawPozycje(teksty []string) {
	if m.tekstyPozycji != nil && slices.Equal(teksty, m.tekstyPozycji) {
		return
	}
	m.tekstyPozycji = teksty
	for i, p := range m.pozycje {
		if i < len(teksty) {
			p.SetTitle(teksty[i])
			p.Show()
		} else {
			p.Hide()
		}
	}
	if len(teksty) == 0 {
		m.brak.Show()
	} else {
		m.brak.Hide()
	}
}

// otworzPanel: strona serwera w przeglądarce (ta sama funkcja co kreator).
func (z *zasobnik) otworzPanel(a *Aplikacja) {
	adres := a.serwery.adresLokalnyHTTPS()
	if adres == "" {
		return
	}
	if err := a.opcje.OtworzPrzegladarke(adres + "/"); err != nil {
		z.zglosBlad("Otwieranie przeglądarki", err)
	}
}

// przelaczAutostart: włącza albo wyłącza uruchamianie po zalogowaniu i ustawia znaczek wg stanu po zmianie.
func (z *zasobnik) przelaczAutostart(p *systray.MenuItem) {
	if err := ustawAutostart(!p.Checked()); err != nil {
		z.zglosBlad("Uruchamianie z systemem", err)
	}
	wlaczony, err := czyAutostart()
	if err != nil {
		z.zglosBlad("Uruchamianie z systemem", err)
		return
	}
	if wlaczony {
		p.Check()
	} else {
		p.Uncheck()
	}
}
