//go:build windows

package monitor

import (
	"net"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"

	"skrytka/wewn/konfiguracja"
)

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

// stanPamieci: MEMORYSTATUSEX.
type stanPamieci struct {
	Dlugosc               uint32
	Obciazenie            uint32
	FizyczneRazem         uint64
	FizyczneDostepne      uint64
	StronicowanieRazem    uint64
	StronicowanieDostepne uint64
	WirtualneRazem        uint64
	WirtualneDostepne     uint64
	RozszerzoneDostepne   uint64
}

// wydajnoscDysku: DISK_PERFORMANCE (odpowiedź IOCTL_DISK_PERFORMANCE).
type wydajnoscDysku struct {
	BajtyOdczytane   int64
	BajtyZapisane    int64
	CzasOdczytu      int64
	CzasZapisu       int64
	CzasBezczynnosci int64
	LiczbaOdczytow   uint32
	LiczbaZapisow    uint32
	Kolejka          uint32
	Podzielone       uint32
	CzasZapytania    int64
	NumerUrzadzenia  uint32
	NazwaMenedzera   [8]uint16
}

// zrodloWindows czyta liczniki przez kernel32 i iphlpapi.
type zrodloWindows struct{}

// NoweZrodloSystemowe zwraca źródło odczytów prawdziwego systemu.
func NoweZrodloSystemowe() Zrodlo { return zrodloWindows{} }

// Odczytaj: CPU i RAM są wymagane; dysk i sieć najlepiej jak się da.
func (zrodloWindows) Odczytaj() (Odczyt, error) {
	var o Odczyt
	var bezczynny, jadro, uzytkownik windows.Filetime
	r, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&bezczynny)), uintptr(unsafe.Pointer(&jadro)), uintptr(unsafe.Pointer(&uzytkownik)))
	if r == 0 {
		return Odczyt{}, err
	}
	// Czas jądra zawiera czas bezczynności.
	o.CPUBezczynny = takty(bezczynny)
	o.CPURazem = takty(jadro) + takty(uzytkownik)

	pam := stanPamieci{}
	pam.Dlugosc = uint32(unsafe.Sizeof(pam))
	if r, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&pam))); r == 0 {
		return Odczyt{}, err
	}
	o.RAMRazem = pam.FizyczneRazem
	o.RAMUzyte = pam.FizyczneRazem - min(pam.FizyczneDostepne, pam.FizyczneRazem)

	o.DyskOdczyt, o.DyskZapis, o.DyskDostepny = dyskiWindows()
	o.Siec = sieciWindows()
	return o, nil
}

func takty(f windows.Filetime) uint64 {
	return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime)
}

// dyskiWindows sumuje liczniki IOCTL_DISK_PERFORMANCE dysków fizycznych (otwartych bez praw odczytu,
// więc bez uprawnień administratora). dostepny = false, gdy żaden dysk nie odpowiedział.
func dyskiWindows() (odczyt, zapis uint64, dostepny bool) {
	for i := 0; i < konfiguracja.MonitorMaksDyskowFizycznych; i++ {
		w, ok := wydajnoscJednegoDysku(konfiguracja.MonitorPrzedrostekDyskuFizycznego + strconv.Itoa(i))
		if !ok {
			continue
		}
		odczyt += uint64(max(w.BajtyOdczytane, 0))
		zapis += uint64(max(w.BajtyZapisane, 0))
		dostepny = true
	}
	return odczyt, zapis, dostepny
}

func wydajnoscJednegoDysku(sciezka string) (wydajnoscDysku, bool) {
	nazwa, err := windows.UTF16PtrFromString(sciezka)
	if err != nil {
		return wydajnoscDysku{}, false
	}
	uchwyt, err := windows.CreateFile(nazwa, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil,
		windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return wydajnoscDysku{}, false
	}
	defer windows.CloseHandle(uchwyt) // celowo: uchwyt tylko do zapytania, błąd zamknięcia bez znaczenia
	var w wydajnoscDysku
	var zwrocono uint32
	err = windows.DeviceIoControl(uchwyt, konfiguracja.MonitorKodIoctlWydajnosciDysku, nil, 0,
		(*byte)(unsafe.Pointer(&w)), uint32(unsafe.Sizeof(w)), &zwrocono, nil)
	return w, err == nil
}

// sieciWindows: liczniki MIB_IF_ROW2 interfejsów w górze z net.Interfaces (bez filtrów i kart ukrytych).
func sieciWindows() []LicznikSieci {
	interfejsy, err := net.Interfaces()
	if err != nil {
		return []LicznikSieci{}
	}
	wynik := make([]LicznikSieci, 0, len(interfejsy))
	for _, inf := range interfejsy {
		if inf.Flags&net.FlagUp == 0 {
			continue
		}
		wiersz := windows.MibIfRow2{InterfaceIndex: uint32(inf.Index)}
		if err := windows.GetIfEntry2Ex(windows.MibIfTableNormal, &wiersz); err != nil {
			continue // celowo: interfejs bez liczników pomijamy
		}
		wynik = append(wynik, LicznikSieci{
			Interfejs: inf.Name,
			Opis:      windows.UTF16ToString(wiersz.Description[:]),
			Petla:     inf.Flags&net.FlagLoopback != 0,
			Wyslane:   wiersz.OutOctets,
			Odebrane:  wiersz.InOctets,
		})
	}
	return wynik
}

// ZajetoscDysku: bajty razem i wolne (dla procesu) na woluminie zawierającym ścieżkę.
func ZajetoscDysku(sciezka string) (Zajetosc, error) {
	p, err := windows.UTF16PtrFromString(sciezka)
	if err != nil {
		return Zajetosc{}, err
	}
	var wolne, razem, wolneRazem uint64
	if err := windows.GetDiskFreeSpaceEx(p, &wolne, &razem, &wolneRazem); err != nil {
		return Zajetosc{}, err
	}
	return Zajetosc{Razem: razem, Wolne: wolne}, nil
}
