// Start aplikacji: tylko woła wewn/aplikacja.
package main

import (
	"os"

	"skrytka/wewn/aplikacja"
)

func main() {
	os.Exit(aplikacja.Uruchom(os.Args[1:]))
}
