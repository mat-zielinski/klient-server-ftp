# Zmiany — Skrytka

Co wnosi i co poprawia każda wersja, od najnowszej. Numer bieżący: `Wersja` w `wewn/konfiguracja/wersja.go`
(stopka interfejsu). Wpis powstaje w tej samej zmianie co podbicie wersji. Piszemy, co widzi użytkownik albo
programista, bez listy plików.

## 0.1 (2026-11-27)

**Nowe**
- Serwer plików w jednym pliku (`skrytka.exe`, Linux amd64 i arm64): pobieranie przez przeglądarkę (HTTPS,
  wznawianie, ZIP strumieniem) i FTPS (wymagany TLS, wznawianie), wgrywanie w kawałkach z wznawianiem, kosz, linki
  publiczne (ważność, hasło, limit pobrań, kod QR).
- Menu pod prawym przyciskiem „Pobierz przez…” z wyborem protokołu: przeglądarka, klient FTP (instrukcja, odcisk
  certyfikatu, QR), link bezpośredni, ZIP, link publiczny.
- Transfery na żywo po obu stronach: klient widzi „Moje pobrania” (prędkość, postęp, pozostały czas, mini wykres),
  admin na Pulpicie wszystkie transfery z przerwaniem i wykresem prędkości łącznej.
- Dziennik pobrań z wykresami (godziny/dni, protokoły, najczęściej pobierane, klienci, eksport CSV) i mapą ciepła.
- Monitor serwera: CPU, RAM, dyski, sieć na kartę (w tym WireGuard), połączenia, historia 10 min i 24 h.
- Wygląd jak stacja Programator_Czarek: pasek boczny po lewej, tokeny kolorów stacji, motyw ciemny, jasny i
  automatyczny, paleta poleceń Ctrl+K, skróty, galeria z miniaturami, płynne listy przy dziesiątkach tysięcy plików,
  aplikacja na telefon (PWA).
- Bezpieczeństwo: hasła argon2id, blokada po złych hasłach, 2FA (TOTP), lista zalogowanych urządzeń, bramka
  uprawnień przy każdej operacji, ochrona przed wyjściem poza udział (także dowiązania i junction), CSRF, CSP.
- Tryby sieci VPN (WireGuard, LAN) i publiczny (wymuszone HTTPS), certyfikat własny, z pliku albo Let's Encrypt,
  reguła Zapory Windows z Ustawień, kreator pierwszego uruchomienia, usługa Windows i systemd.
- Jakość: strażnik (warstwy, kod Windows/Linux, limity), mutacje blokad, `sprawdz.bat`/`sprawdz.sh`, haki git,
  GitHub Actions (testy Windows i Linux, wydanie z binarkami przy tagu `v*`).
- Plan projektu (Go + Svelte 5, przeglądarka i FTPS, tryb WireGuard albo publiczny, wygląd jak stacja
  Programator_Czarek i UniFi) i podział na sprinty.
- Szkielet warstw `konfiguracja ← rdzen ← magazyn ← uslugi ← http, ftp, siec ← aplikacja ← cmd` z kontraktami
  (`kontrakt.go` w każdym pakiecie), kontrakt API HTTP (`docs/api.md`) i interfejsu (`docs/interfejs.md`).
- Konfiguracja: domyślne porty (HTTP 8080, HTTPS 8443, FTPS 2121, pasywne 50000–50100), tryb VPN domyślnie, zapis
  atomowy plików stanu, stałe obszarów (konta, dziennik, transfery, linki, sieć, rdzeń).
- Go 1.27.4 przenośne w `_go-portable\`, interfejs Svelte 5 + Vite z zaślepką `web/dist/index.html` wkompilowaną
  w `.exe`.
- Dokumenty: `CLAUDE.md`, skill `skrytka`, `README.md`, `INSTALACJA.txt`, instrukcja dla klienta, architektura,
  testy, wydania, lista do zrobienia; `.gitignore` chroniący dane, hasła, certyfikaty i logi przed publicznym repo.
