package main

// Limity i nazwy strażnika (jedno miejsce; opis reguł: narzedzia_dev/README.md).
const (
	PlikMaxLinii        = 600 // każdy plik .go, także testy
	FunkcjaMaxLinii     = 80  // funkcja / metoda poza testami
	ZlozonoscMax        = 15  // złożoność cyklomatyczna funkcji poza testami
	ArgumentyMax        = 8   // parametry funkcji (bez odbiorcy)
	PlikWyjatkow        = "narzedzia_dev/wyjatki.json"
	PlikStyluWeb        = "web/sprawdz_styl.mjs"
	PolecenieNode       = "node"
	KatalogWewn         = "wewn"
	KatalogCmd          = "cmd"
	KatalogUslug        = "wewn/uslugi"
	PlikKontraktu       = "kontrakt.go"
	MinDlugoscPowodu    = 5 // znaków uzasadnienia wyjątku
	KodOK               = 0
	KodNaruszenia       = 1
	KodBladUruchomienia = 2
)

// KatalogiPomijane: katalogi bez kodu Go projektu (poza tym każdy zaczynający się od „.” albo „_”).
var KatalogiPomijane = []string{"node_modules", "testdata", "vendor", "dist"}

// Warstwy: poziom warstwy; import wolno tylko do warstwy o NIŻSZYM poziomie (docs/zasady_agentow.md pkt 4).
var Warstwy = map[string]int{
	"konfiguracja": 0,
	"web":          0, // osadzony interfejs (web/osadz.go), liść bez importów projektu
	"rdzen":        1,
	"magazyn":      2,
	"uslugi":       3,
	"siec":         4, // pod http i ftp (decyzja sesji głównej)
	"http":         5,
	"ftp":          5,
	"aplikacja":    6,
	"cmd":          7,
}

// ZakazaneWRdzeniu: importy, których rdzeń nie może mieć (dysk, sieć, procesy). Prefiks "x/" = cała gałąź.
var ZakazaneWRdzeniu = []string{"os", "os/", "net", "net/", "io/ioutil", "syscall"}
