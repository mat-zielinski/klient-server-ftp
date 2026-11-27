package konfiguracja

import "time"

// Stałe obszaru „transfery” (podgląd pobrań i wgrań na żywo).
const (
	// TransferyOknoPredkosciS: z ilu ostatnich sekund liczona jest średnia prędkość.
	TransferyOknoPredkosciS = 3
	// TransferyProbkiHistorii: ile próbek prędkości (co TransferyOkresProbkowania) trzyma historia transferu.
	TransferyProbkiHistorii = 60
	// TransferyOkresProbkowania: co ile gorutyna próbkująca liczy prędkość i wysyła postęp.
	TransferyOkresProbkowania = time.Second
	// TransferyOdstepPostepu: najkrótszy odstęp między zdarzeniami „postep” jednego transferu.
	TransferyOdstepPostepu = 500 * time.Millisecond
	// TransferyBuforSluchacza: pojemność kanału zdarzeń jednego słuchacza (SSE).
	TransferyBuforSluchacza = 64
	// TransferyBajtyID: ile losowych bajtów ma identyfikator transferu (zapis szesnastkowy).
	TransferyBajtyID = 8
)
