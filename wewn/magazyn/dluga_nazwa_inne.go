//go:build !windows

package magazyn

// dlugaNazwa: poza Windows nie ma krótkich nazw 8.3, ścieżka zostaje bez zmian.
func dlugaNazwa(sciezka string) string { return sciezka }
