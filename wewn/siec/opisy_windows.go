//go:build windows

package siec

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"

	"skrytka/wewn/konfiguracja"
)

// opisyInterfejsow zwraca opisy kart sieciowych (np. "WireGuard Tunnel") wg numeru interfejsu.
// Błąd systemu = pusta mapa (rozpoznanie WireGuard działa wtedy tylko po nazwie).
func opisyInterfejsow() map[int]string {
	wynik := map[int]string{}
	rozmiar := uint32(konfiguracja.RozmiarBuforaAdapterow)
	for i := 0; i < konfiguracja.ProbyBuforaAdapterow; i++ {
		bufor := make([]byte, rozmiar)
		pierwszy := (*windows.IpAdapterAddresses)(unsafe.Pointer(&bufor[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, pierwszy, &rozmiar)
		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			continue
		}
		if err != nil {
			return wynik
		}
		for a := pierwszy; a != nil; a = a.Next {
			indeks := int(a.IfIndex)
			if indeks == 0 {
				indeks = int(a.Ipv6IfIndex)
			}
			wynik[indeks] = windows.UTF16PtrToString(a.Description)
		}
		return wynik
	}
	return wynik
}
