// Pakiet web: zbudowany interfejs (web/dist) wkompilowany w plik .exe.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Pliki zwraca zawartość web/dist (index.html w korzeniu).
func Pliki() fs.FS {
	f, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // celowo: brak dist to błąd budowania, nie działania
	}
	return f
}
