//go:build windows

package magazyn

import "golang.org/x/sys/windows"

// dlugaNazwa rozwija krótkie nazwy 8.3 (np. RUNNER~1) do pełnych. Korzeń udziału jest już w pełnej postaci
// (filepath.EvalSymlinks), więc cel dowiązania zapisany krótką nazwą musi przejść to samo, inaczej porównanie
// z korzeniem uzna go za ścieżkę poza udziałem. Nieistniejąca ścieżka albo błąd = bez zmian.
func dlugaNazwa(sciezka string) string {
	wej, err := windows.UTF16PtrFromString(sciezka)
	if err != nil {
		return sciezka
	}
	bufor := make([]uint16, windows.MAX_PATH)
	for {
		n, err := windows.GetLongPathName(wej, &bufor[0], uint32(len(bufor)))
		if err != nil || n == 0 {
			return sciezka
		}
		if int(n) < len(bufor) {
			return windows.UTF16ToString(bufor[:n])
		}
		bufor = make([]uint16, n)
	}
}
