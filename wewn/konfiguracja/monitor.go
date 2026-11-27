package konfiguracja

import "time"

// Stałe obszaru „monitor” (CPU, RAM, dysk, sieć serwera na żywo i w historii).
const (
	// MonitorOkresProbkowania: co ile monitor zbiera próbkę.
	MonitorOkresProbkowania = time.Second
	// MonitorProbkiHistorii1s: ile próbek co sekundę trzyma historia krótka (10 min).
	MonitorProbkiHistorii1s = 600
	// MonitorProbkiHistorii1m: ile średnich minutowych trzyma historia długa (24 h).
	MonitorProbkiHistorii1m = 1440
	// MonitorOkresHistorii1m: okres uśredniania historii długiej.
	MonitorOkresHistorii1m = time.Minute
	// MonitorBuforSluchacza: pojemność kanału jednego słuchacza (SSE); przy pełnym próbka przepada.
	MonitorBuforSluchacza = 4
	// MonitorProcentMaks: górna granica zajętości CPU w procentach.
	MonitorProcentMaks = 100.0
)

// Liczniki systemowe (przepełnienie i reset).
const (
	// MonitorGranicaLicznika32: licznik 32-bitowy przekręca się po tej wartości (np. /proc/net/dev na 32 bitach).
	MonitorGranicaLicznika32 = 1<<32 - 1
	// MonitorProgPrzekreceniaLicznika32: spadek licznika, który wcześniej był powyżej tej wartości (połowa
	// zakresu 32 bitów), liczymy jako przekręcenie 32-bitowe; niższy = reset licznika (prędkość 0).
	MonitorProgPrzekreceniaLicznika32 = 1 << 31
)

// Odczyty systemu Linux.
const (
	MonitorPlikStatystykCPU         = "/proc/stat"
	MonitorPlikPamieci              = "/proc/meminfo"
	MonitorPlikSieci                = "/proc/net/dev"
	MonitorPlikDyskow               = "/proc/diskstats"
	MonitorKatalogUrzadzenBlokowych = "/sys/block"
	// MonitorPodkatalogUrzadzenia: tylko dyski z /sys/block/<nazwa>/device są fizyczne (bez loop, ram, dm, partycji).
	MonitorPodkatalogUrzadzenia = "device"
	// MonitorBajtySektora: /proc/diskstats liczy w sektorach po 512 B niezależnie od dysku.
	MonitorBajtySektora = 512
	// MonitorBajtyKB: jednostka „kB” w /proc/meminfo.
	MonitorBajtyKB = 1024
	// MonitorNazwaPetliLinux: interfejs pętli zwrotnej w Linuksie.
	MonitorNazwaPetliLinux = "lo"
)

// Odczyty systemu Windows.
const (
	// MonitorPrzedrostekDyskuFizycznego: dyski pytane o liczniki wydajności (\\.\PhysicalDrive0 ...).
	MonitorPrzedrostekDyskuFizycznego = `\\.\PhysicalDrive`
	// MonitorMaksDyskowFizycznych: tyle numerów dysków fizycznych sprawdzamy.
	MonitorMaksDyskowFizycznych = 16
	// MonitorKodIoctlWydajnosciDysku: IOCTL_DISK_PERFORMANCE.
	MonitorKodIoctlWydajnosciDysku = 0x70020
)
