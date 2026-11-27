package konfiguracja

import "time"

// Stałe obszaru „udostepnienia” (linki publiczne).
const (
	// DlugoscIDLinku: liczba losowych bajtów identyfikatora linku (zapis base64url bez dopełnienia).
	DlugoscIDLinku = 16
	// MaksWaznoscLinku: najdłuższy dozwolony czas życia linku; dalsza data jest przycinana.
	MaksWaznoscLinku = 365 * 24 * time.Hour
	// DomyslnaWaznoscLinku: używana, gdy przy tworzeniu nie podano daty wygaśnięcia.
	DomyslnaWaznoscLinku = 7 * 24 * time.Hour
	// LimitProbHaslaLinku: tyle złych haseł w oknie blokuje link (rdzen.ErrKontoZablokowane).
	LimitProbHaslaLinku = 5
	// OknoProbHaslaLinku: okno liczenia złych prób i czas blokady linku.
	OknoProbHaslaLinku = 15 * time.Minute
	// MaksDlugoscHaslaLinku: dłuższe hasło jest odrzucane (ochrona przed kosztownym hashowaniem).
	MaksDlugoscHaslaLinku = 256

	// Parametry argon2id haseł linków (zalecenie OWASP: m=19 MiB, t=2, p=1).
	Argon2LinkPamiecKiB     = 19 * 1024
	Argon2LinkIteracje      = 2
	Argon2LinkWatki         = 1
	Argon2LinkDlugoscSoli   = 16
	Argon2LinkDlugoscKlucza = 32
)
