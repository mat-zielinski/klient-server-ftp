package konfiguracja

import "time"

// Stałe dziennika serwera (logi/dziennik_RRRR-MM.jsonl). Właściciel: agent „dziennik”.
const (
	// DziennikPrefiks i DziennikRozszerzenie tworzą nazwę pliku miesiąca: dziennik_2026-11.jsonl.
	DziennikPrefiks      = "dziennik_"
	DziennikRozszerzenie = ".jsonl"
	// DziennikFormatMiesiaca: układ time.Format dla części RRRR-MM nazwy pliku.
	DziennikFormatMiesiaca = "2006-01"
	// DziennikBufor: pojemność kanału wpisów czekających na zapis (Zapisz nie blokuje).
	DziennikBufor = 4096
	// DziennikOkresSynchronizacji: co ile bufor trafia na dysk (fsync).
	DziennikOkresSynchronizacji = time.Second
	// DziennikOpisMax: najdłuższy opis wpisu w znakach (dłuższy jest przycinany).
	DziennikOpisMax = 500
	// DziennikTop: długość list „najczęstsze pliki” i „najaktywniejsi klienci”.
	DziennikTop = 10
	// DziennikLimitMaks: najwięcej wpisów zwracanych przez Czytaj (Limit 0 albo większy = ten limit).
	DziennikLimitMaks = 50000
	// DziennikMaksPunktowWykresu: ochrona przed zbyt drobnym krokiem wykresu.
	DziennikMaksPunktowWykresu = 10000
	// DziennikPamiecMiesiecy: ile zamkniętych miesięcy trzymać w pamięci podręcznej.
	DziennikPamiecMiesiecy = 24
	// DziennikProtokolNieznany: klucz w podziale na protokoły dla wpisów bez protokołu.
	DziennikProtokolNieznany = "nieznany"
	// DziennikSeparatorCSV: średnik, bo polski Excel tak czyta CSV.
	DziennikSeparatorCSV = ';'
	// DziennikFormatCzasuCSV: czas w eksporcie CSV.
	DziennikFormatCzasuCSV = "2006-01-02 15:04:05"
	// UprawnieniaPlikuDziennika: plik dziennika tylko dla właściciela.
	UprawnieniaPlikuDziennika = 0o600
)
