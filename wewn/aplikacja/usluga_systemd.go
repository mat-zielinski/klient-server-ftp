package aplikacja

import (
	"path"
	"strings"

	"skrytka/wewn/konfiguracja"
)

// plikUslugiSystemd: treść jednostki systemd dla programu exe (czysta funkcja, testowana na każdym
// systemie). uzytkownik pusty = usługa jako root. Program działa z --konsola: bez przeglądarki,
// wpisy serwera na stderr, czyli w journalctl -u skrytka.
func plikUslugiSystemd(exe, uzytkownik string) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=" + konfiguracja.OpisUslugi + "\n")
	b.WriteString("After=network-online.target\nWants=network-online.target\n\n")
	b.WriteString("[Service]\nType=simple\n")
	b.WriteString("ExecStart=" + cytatSystemd(exe) + " " + konfiguracja.ArgumentKonsola + "\n")
	b.WriteString("WorkingDirectory=" + cytatSystemd(path.Dir(exe)) + "\n")
	if uzytkownik != "" {
		b.WriteString("User=" + uzytkownik + "\nGroup=" + uzytkownik + "\n")
		// Porty poniżej 1024 (np. 443 w trybie publicznym) bez praw roota.
		b.WriteString("AmbientCapabilities=CAP_NET_BIND_SERVICE\n")
	}
	b.WriteString("Restart=on-failure\nNoNewPrivileges=true\n\n")
	b.WriteString("[Install]\nWantedBy=multi-user.target\n")
	return b.String()
}

// cytatSystemd: ścieżka w cudzysłowie, gdy ma spacje (składnia systemd).
func cytatSystemd(s string) string {
	if !strings.ContainsAny(s, " \t") {
		return s
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\`), `"`, `\"`) + `"`
}
