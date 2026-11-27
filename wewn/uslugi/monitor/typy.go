// Pakiet monitor: stan serwera na żywo (CPU, RAM, dysk I/O, sieć na interfejs, połączenia, transfery,
// gorutyny, pamięć procesu), historia 10 min co 1 s i 24 h co 1 min, słuchacze dla SSE.
// Właściciel: agent „monitor”.
//
// Odczyty systemowe są w plikach zrodlo_windows.go i zrodlo_linux.go (bez cgo); parsery plików /proc
// są w proc.go bez znacznika systemu, żeby testy szły też na Windows.
package monitor

import "time"

// Probka: jeden pomiar (JSON jak w docs/api.md, „Fala 2”). Prędkości w bajtach na sekundę.
type Probka struct {
	Czas          time.Time        `json:"czas"`
	CPUProc       float64          `json:"cpu_proc"`
	RAMUzyte      uint64           `json:"ram_uzyte"`
	RAMRazem      uint64           `json:"ram_razem"`
	DyskOdczytBps int64            `json:"dysk_odczyt_bps"`
	DyskZapisBps  int64            `json:"dysk_zapis_bps"`
	Siec          []SiecInterfejsu `json:"siec"`
	WysylanieBps  int64            `json:"wysylanie_bps"`
	OdbiorBps     int64            `json:"odbior_bps"`
	Polaczenia    int              `json:"polaczenia"`
	Transfery     int              `json:"transfery"`
	Gorutyny      int              `json:"gorutyny"`
	PamiecProcesu uint64           `json:"pamiec_procesu"`
}

// SiecInterfejsu: prędkości jednego interfejsu. Rodzaj: "lan" | "wireguard" | "petla".
type SiecInterfejsu struct {
	Interfejs    string `json:"interfejs"`
	Rodzaj       string `json:"rodzaj"`
	WysylanieBps int64  `json:"wysylanie_bps"`
	OdbiorBps    int64  `json:"odbior_bps"`
}

// System: stałe dane maszyny. CzasPracyS liczony od utworzenia monitora (czas pracy serwera Skrytka).
// DyskIODostepny (rozszerzenie kontraktu): false = system nie dał liczników dysku, prędkości dysku to 0.
type System struct {
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	CPURdzenie     int    `json:"cpu_rdzenie"`
	RAMRazem       uint64 `json:"ram_razem"`
	Host           string `json:"host"`
	CzasPracyS     int64  `json:"czas_pracy_s"`
	DyskIODostepny bool   `json:"dysk_io_dostepny"`
}

// Odczyt: surowe, narastające liczniki systemu w jednej chwili (źródło odczytów je zwraca).
type Odczyt struct {
	// CPURazem i CPUBezczynny: narastające takty procesora (jednostka dowolna, liczy się stosunek różnic).
	CPURazem     uint64
	CPUBezczynny uint64
	RAMUzyte     uint64
	RAMRazem     uint64
	// DyskOdczyt i DyskZapis: narastające bajty wszystkich dysków fizycznych.
	DyskOdczyt   uint64
	DyskZapis    uint64
	DyskDostepny bool
	Siec         []LicznikSieci
}

// LicznikSieci: narastające bajty jednego interfejsu.
type LicznikSieci struct {
	Interfejs string
	Opis      string // opis karty (Windows), pomaga rozpoznać WireGuard
	Petla     bool
	Wyslane   uint64
	Odebrane  uint64
}

// Zrodlo odczytów systemowych; testy podają atrapę (ZZrodlem).
type Zrodlo interface {
	Odczytaj() (Odczyt, error)
}

// Zajetosc dysku pod ścieżką (np. udział): bajty razem i wolne dla procesu.
type Zajetosc struct {
	Razem uint64 `json:"razem"`
	Wolne uint64 `json:"wolne"`
}
