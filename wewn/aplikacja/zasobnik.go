package aplikacja

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/siec"
	"skrytka/wewn/uslugi/transfery"
)

// Ikona w zasobniku systemowym: czysta logika (dymek, ikona, podmenu, adres, decyzja o pokazaniu)
// i składanie z serwerem. Pętla zasobnika (fyne.io/systray) jest w zasobnik_menu.go, sprawy systemu
// (autostart, schowek, folder) w zasobnik_windows.go i zasobnik_linux.go.

// errPetlaZasobnika: pętla zasobnika zakończyła się błędem (serwer działa dalej bez ikony).
var errPetlaZasobnika = errors.New("Ikona w zasobniku przestała działać.")

// zasobnik: stan wspólny serwera i ikony.
type zasobnik struct {
	ctx    context.Context
	anuluj context.CancelFunc
	gotowa chan *Aplikacja
	a      atomic.Pointer[Aplikacja]
}

// ustawAplikacje (Opcje.PoStarcie): serwer nasłuchuje, ikona może zbudować menu.
func (z *zasobnik) ustawAplikacje(a *Aplikacja) {
	z.a.Store(a)
	select {
	case z.gotowa <- a:
	default: // celowo: aplikacja startuje raz
	}
}

// zglosBlad: wpis w dzienniku, póki serwer działa (potem dziennika już nie ma).
func (z *zasobnik) zglosBlad(gdzie string, err error) {
	a := z.a.Load()
	if a == nil || z.ctx.Err() != nil {
		return
	}
	a.zglosBlad(gdzie, err)
}

// uruchomZZasobnikiem: serwer w gorutynie, pętla zasobnika na wątku głównym (wymóg biblioteki).
// „Zakończ” w menu anuluje serwer; po jego łagodnym zatrzymaniu pętla zasobnika jest kończona.
func uruchomZZasobnikiem(ctx context.Context, o Opcje) error {
	ctx, anuluj := context.WithCancel(ctx)
	defer anuluj()
	z := &zasobnik{ctx: ctx, anuluj: anuluj, gotowa: make(chan *Aplikacja, 1)}
	o.PoStarcie = z.ustawAplikacje
	wynik := make(chan error, 1)
	koniecPetli := make(chan struct{})
	go func() {
		err := uruchomSerwer(ctx, o)
		wynik <- err
		zakonczPetleZasobnika()
		select {
		case <-koniecPetli:
		case <-time.After(konfiguracja.ZasobnikCzasWyjscia):
			// Pętla, której okno się nie utworzyło, nie kończy się na Quit: serwer jest już zatrzymany.
			if err != nil {
				pokazKomunikat("Serwer nie działa: " + err.Error())
			}
			os.Exit(kodWyjscia(err)) // celowo: wątek główny utknął w pętli biblioteki, nic więcej do zamknięcia
		}
	}()
	if err := petlaZasobnika(z); err != nil {
		z.zglosBlad("Zasobnik systemowy", err)
		close(koniecPetli)
		return <-wynik // serwer działa dalej bez ikony
	}
	close(koniecPetli)
	anuluj()
	return <-wynik
}

// kodWyjscia: KodOK albo KodBlad.
func kodWyjscia(err error) int {
	if err != nil {
		return konfiguracja.KodBlad
	}
	return konfiguracja.KodOK
}

// czyPokazacZasobnik: zwykłe uruchomienie (bez --konsola i --bez-zasobnika) na Windows albo na Linuksie
// z pulpitem (DISPLAY / WAYLAND_DISPLAY) i nie pod systemd. Usługa Windows tu nie dochodzi.
func czyPokazacZasobnik(p polecenie, system string, zmienna func(string) string) bool {
	if p.Konsola || p.BezZasobnika || p.Usluga != "" {
		return false
	}
	switch system {
	case konfiguracja.SystemWindows:
		return true
	case konfiguracja.SystemLinux:
		if zmienna(konfiguracja.ZmiennaSystemd) != "" {
			return false
		}
		return zmienna(konfiguracja.ZmiennaEkranuX11) != "" || zmienna(konfiguracja.ZmiennaEkranuWayland) != ""
	}
	return false
}

// tekstDymka: „Skrytka — działa · 192.168.1.5:8443” albo „Skrytka — 2 pobierania · 12,3 MB/s”;
// faza != "" (uruchamianie, zamykanie) ma pierwszeństwo. Skrócony do ZasobnikMaksDymka.
func tekstDymka(adres string, stany []transfery.Stan, faza string) string {
	t := konfiguracja.NazwaAplikacji + konfiguracja.ZasobnikSeparatorNazwy
	switch {
	case faza != "":
		t += faza
	case len(stany) > 0:
		t += opisTransferow(stany)
	default:
		t += konfiguracja.ZasobnikTekstDziala
		if adres != "" {
			t += konfiguracja.ZasobnikSeparatorPol + adres
		}
	}
	return przytnij(t, konfiguracja.ZasobnikMaksDymka)
}

// opisTransferow: „2 pobierania · 12,3 MB/s”, „1 pobranie, 1 wgranie · 3 MB/s”.
func opisTransferow(stany []transfery.Stan) string {
	pobrania, wgrania := 0, 0
	var predkosc int64
	for _, s := range stany {
		if s.Kierunek == transfery.KierunekWgranie {
			wgrania++
		} else {
			pobrania++
		}
		predkosc += s.PredkoscBps
	}
	czesci := []string{}
	if pobrania > 0 {
		czesci = append(czesci, liczebnik(pobrania, konfiguracja.ZasobnikFormyPobran))
	}
	if wgrania > 0 {
		czesci = append(czesci, liczebnik(wgrania, konfiguracja.ZasobnikFormyWgran))
	}
	return strings.Join(czesci, ", ") + konfiguracja.ZasobnikSeparatorPol + formatPredkosci(predkosc)
}

// liczebnik: „1 pobranie”, „3 pobrania”, „5 pobrań”, „22 pobrania”, „12 pobrań”.
func liczebnik(n int, formy [3]string) string {
	forma := formy[2]
	switch {
	case n == 1:
		forma = formy[0]
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		forma = formy[1]
	}
	return strconv.Itoa(n) + " " + forma
}

// formatPredkosci jak w interfejsie (stan/format.ts): podstawa 1024, do 2 miejsc poniżej 10, 1 poniżej 100,
// przecinek dziesiętny, bez zer końcowych: „0 B/s”, „850 KB/s”, „12,3 MB/s”.
func formatPredkosci(bps int64) string {
	if bps <= 0 {
		return "0 " + konfiguracja.ZasobnikJednostki[0] + "/s"
	}
	w := float64(bps)
	i := 0
	for w >= konfiguracja.ZasobnikKrokJednostek && i < len(konfiguracja.ZasobnikJednostki)-1 {
		w /= konfiguracja.ZasobnikKrokJednostek
		i++
	}
	miejsca := 0
	if i > 0 && w < 10 {
		miejsca = 2
	} else if i > 0 && w < 100 {
		miejsca = 1
	}
	t := strconv.FormatFloat(w, 'f', miejsca, 64)
	if strings.Contains(t, ".") {
		t = strings.TrimRight(strings.TrimRight(t, "0"), ".")
	}
	return strings.Replace(t, ".", ",", 1) + " " + konfiguracja.ZasobnikJednostki[i] + "/s"
}

// przytnij skraca tekst do max znaków (z wielokropkiem).
func przytnij(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-utf8.RuneCountInString(konfiguracja.ZasobnikWielokropek)]) + konfiguracja.ZasobnikWielokropek
}

// czyAktywne: są transfery (ikona ZasobnikAktywny).
func czyAktywne(stany []transfery.Stan) bool { return len(stany) > 0 }

// zestawIkon: jasne ikony (na ciemny pasek) i ciemne (na jasny pasek), zwykłe i aktywne.
type zestawIkon struct {
	jasna, jasnaAktywna, ciemna, ciemnaAktywna []byte
}

// wybierzIkone: wariant wg transferów i jasności paska; brak ciemnego wariantu = jasny.
func wybierzIkone(z zestawIkon, aktywna, jasnyPasek bool) []byte {
	jasna, ciemna := z.jasna, z.ciemna
	if aktywna {
		jasna, ciemna = z.jasnaAktywna, z.ciemnaAktywna
	}
	if jasnyPasek && len(ciemna) > 0 {
		return ciemna
	}
	return jasna
}

// pozycjeTransferow: do max pozycji podmenu „↓ kto · plik · prędkość” (kolejność jak w rejestrze).
func pozycjeTransferow(stany []transfery.Stan, max int) []string {
	wynik := []string{}
	for _, s := range stany {
		if len(wynik) >= max {
			break
		}
		znak := konfiguracja.ZasobnikZnakPobrania
		if s.Kierunek == transfery.KierunekWgranie {
			znak = konfiguracja.ZasobnikZnakWgrania
		}
		kto := s.Kto
		if kto == "" {
			kto = konfiguracja.ZasobnikKtoNieznany
		}
		plik := przytnij(path.Base(strings.ReplaceAll(s.Sciezka, "\\", "/")), konfiguracja.ZasobnikMaksNazwyPliku)
		wynik = append(wynik, znak+" "+kto+konfiguracja.ZasobnikSeparatorPol+plik+
			konfiguracja.ZasobnikSeparatorPol+formatPredkosci(s.PredkoscBps))
	}
	return wynik
}

// adresDlaKlientow: host:port, pod którym klienci widzą serwer. Kolejność: AdresNasluchu z konfiguracji
// (gdy konkretny), w trybie publicznym adres publiczny (gdy znany), pierwszy LAN, pierwszy WireGuard,
// na końcu pętla zwrotna.
func adresDlaKlientow(k konfiguracja.Konfiguracja, adresy []siec.Adres, publiczny string, port int) string {
	p := strconv.Itoa(port)
	if a := strings.TrimSpace(k.AdresNasluchu); a != "" {
		if ip := net.ParseIP(a); ip == nil || !ip.IsUnspecified() {
			return net.JoinHostPort(a, p)
		}
	}
	if k.TrybSieci == konfiguracja.TrybPubliczny && publiczny != "" {
		return net.JoinHostPort(publiczny, p)
	}
	for _, rodzaj := range []string{konfiguracja.RodzajLAN, konfiguracja.RodzajWireGuard} {
		for _, a := range adresy {
			if a.Rodzaj == rodzaj {
				return net.JoinHostPort(a.IP, p)
			}
		}
	}
	return net.JoinHostPort(konfiguracja.AdresPetli, p)
}

// pngZIco: największy obraz PNG zapisany w pliku ICO (StatusNotifier na Linuksie czyta PNG, nie ICO);
// nil, gdy plik nie jest ICO albo nie ma w nim PNG.
func pngZIco(ico []byte) []byte {
	const naglowek, wpis = 6, 16
	if len(ico) < naglowek || binary.LittleEndian.Uint16(ico[0:]) != 0 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil
	}
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	var najlepszy []byte
	najwiekszy := -1
	for i := 0; i < n; i++ {
		w := naglowek + i*wpis
		if w+wpis > len(ico) {
			return najlepszy
		}
		szer := int(ico[w])
		if szer == 0 {
			szer = 256
		}
		rozmiar := int(binary.LittleEndian.Uint32(ico[w+8:]))
		poczatek := int(binary.LittleEndian.Uint32(ico[w+12:]))
		if poczatek < 0 || rozmiar <= 0 || poczatek+rozmiar > len(ico) || poczatek+rozmiar < poczatek {
			continue
		}
		dane := ico[poczatek : poczatek+rozmiar]
		if bytes.HasPrefix(dane, []byte("\x89PNG\r\n\x1a\n")) && szer > najwiekszy {
			najwiekszy, najlepszy = szer, dane
		}
	}
	return najlepszy
}

// pierwszy element listy ("" dla pustej).
func pierwszy(lista []string) string {
	if len(lista) == 0 {
		return ""
	}
	return lista[0]
}

// wartoscAutostartu: polecenie zapisywane w kluczu Run (ścieżka w cudzysłowie, bo bywają spacje).
func wartoscAutostartu(exe string) string { return `"` + exe + `"` }

// czyNaszAutostart: wpis w Run wskazuje ten program (wielkość liter w ścieżkach Windows bez znaczenia).
func czyNaszAutostart(wartosc, exe string) bool {
	w := strings.Trim(strings.TrimSpace(wartosc), `"`)
	return w != "" && strings.EqualFold(w, exe)
}

// trescAutostartuLinux: plik .desktop dla ~/.config/autostart (Exec w cudzysłowie ze znakami ucieczki
// zgodnie ze specyfikacją Desktop Entry).
func trescAutostartuLinux(exe string) string {
	zamiana := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`)
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=" + konfiguracja.NazwaAplikacji + "\n" +
		"Comment=" + konfiguracja.OpisUslugi + "\n" +
		`Exec="` + zamiana.Replace(exe) + "\"\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
}
