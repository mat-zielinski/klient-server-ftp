package konfiguracja

import "time"

// Stałe obszaru „konta”: logowanie, blokada po złych hasłach, sesje, argon2id.

// Blokada po złych hasłach (liczona osobno na adres IP i na login).
const (
	// MaksBlednychLogowan: tyle złych prób w OknoBlednychLogowan zakłada blokadę.
	MaksBlednychLogowan = 5
	// OknoBlednychLogowan: okres, w którym liczą się złe próby.
	OknoBlednychLogowan = 15 * time.Minute
	// CzasBlokady: jak długo trwa blokada po przekroczeniu limitu.
	CzasBlokady = 15 * time.Minute
	// ProgSprzataniaBlokad: przy tylu wpisach licznika robione jest sprzątanie przeterminowanych.
	ProgSprzataniaBlokad = 1024
)

// Sesje.
const (
	// CzasSesji: ważność sesji od ostatniego użycia (przedłużana przy każdym sprawdzeniu).
	CzasSesji = 12 * time.Hour
	// DlugoscTokenuSesji: bajty losowe tokenu (base64url bez dopełnienia).
	DlugoscTokenuSesji = 32
)

// Loginy i hasła.
const (
	// DlugoscHasla: minimalna liczba znaków hasła.
	DlugoscHasla = 10
	// MaksDlugoscHasla: górny limit znaków hasła (ochrona przed kosztownym haszowaniem).
	MaksDlugoscHasla = 1024
	// MaksDlugoscLoginu: górny limit znaków loginu.
	MaksDlugoscLoginu = 64
)

// Parametry argon2id dla nowych haseł.
const (
	ArgonCzas          uint32 = 2
	ArgonPamiecKiB     uint32 = 64 * 1024
	ArgonWatki         uint8  = 2
	ArgonDlugoscSoli          = 16
	ArgonDlugoscKlucza uint32 = 32
	// ArgonMaksPamiecKiB i ArgonMaksCzas: górne granice przy odczycie hasha z pliku (uszkodzony plik
	// nie może wymusić ogromnego zużycia pamięci ani czasu).
	ArgonMaksPamiecKiB uint32 = 1024 * 1024
	ArgonMaksCzas      uint32 = 16
)

// Urządzenia (sesje z opisem).
const (
	// MaksSesjiNaKonto: przy przekroczeniu wypada sesja najdawniej używana.
	MaksSesjiNaKonto = 10
	// DlugoscIDSesji: bajty skrótu tokenu użyte jako identyfikator urządzenia (hex, 2× znaków).
	DlugoscIDSesji = 8
	// MaksDlugoscOpisuUrzadzenia: obcięcie adresu IP i nazwy przeglądarki.
	MaksDlugoscOpisuUrzadzenia = 200
)
