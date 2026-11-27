# Do zrobienia — Skrytka

Lista otwartych punktów: do poprawy w kodzie, do sprawdzenia na żywo, do decyzji usera. Historia zmian:
[zmiany.md](zmiany.md), plan sprintów: [PLAN.md](PLAN.md), stan wydań: [docs/wydania.md](docs/wydania.md).

## 1. Do poprawy w kodzie

**Z pierwszego uruchomienia (2026-11-27)**
- [ ] Pliki: pasek narzędzi łamie się na drugi rząd przy zaznaczeniu (przycisk „Wgraj” spada niżej); zasada stacji
      4a: zwężać, nie zawijać
- [ ] Pulpit: w trybie VPN karta adresów pokazuje adres publiczny jak dostępny; oznaczyć „niedostępny w trybie VPN”
- [ ] Dziennik: odświeżać statystyki same po zdarzeniu „koniec” transferu (teraz dopiero po ponownym wejściu)
- [ ] Kreator: pole loginu ma wpisaną wartość „admin” zamiast podpowiedzi; pisanie dokleja tekst
- [ ] FTPS: `curl` kończy pobranie kodem 56 mimo poprawnej treści (zamykanie TLS kanału danych); sprawdzić z FileZillą
- [ ] Wgrywanie przez HTTP nie pokazuje się w transferach na żywo (tylko wpis po zatwierdzeniu)
- [ ] Monitor: liczba połączeń nie obejmuje FTPS (pakiet ftp nie ma licznika)
- [ ] Link publiczny z hasłem: pobieranie przez blob w pamięci przeglądarki; ciasteczko po haśle i zwykły link;
      wejście do podfolderów i ZIP całości
- [ ] FTPS przy wielu adresach nasłuchu: porty pasywne wiązane z IP nasłuchu
- [ ] Let's Encrypt HTTP-01 na porcie 80 (dziś tylko TLS-ALPN-01 na 443)
- [ ] Kosz: przeglądanie, przywracanie, opróżnianie
- [ ] Testy jednostkowe interfejsu (vitest): `format.ts`, nawigacja, koder QR, rozmyte dopasowanie
- [ ] `rdzen.Zwin` (usuwanie ogonków) eksportowane, zamiast kopii tabeli w indeksie

**Dokumenty**
- [ ] Zrzuty ekranu w `README.md`
- [ ] `INSTALACJA.txt` i `docs/dla_klienta.md`: sprawdzić nazwy przycisków i komunikatów z gotowym interfejsem

## 2. Do sprawdzenia na żywo

- [ ] Wejście przez WireGuard z telefonu i laptopa (tunel z `Dozwolone adresy IP` obejmującymi LAN), adres LAN w menu
      „Pobierz przez…” poprawny przez tunel
- [ ] Ostrzeżenie o certyfikacie i zgodność odcisku SHA-256 w Chrome, Edge, Firefox i na telefonie
- [ ] FTPS w FileZilli, WinSCP i Total Commanderze: jawny TLS, tryb pasywny przez WireGuard i przez router
- [ ] Pobranie pliku 10+ GB z przerwaniem i wznowieniem (HTTPS i FTPS), kilka pobrań naraz
- [ ] Tryb publiczny: przekierowanie portów na routerze, test z sieci komórkowej, blokada po złych hasłach
- [ ] Reguła Zapory Windows dodana z Ustawień (okno UAC)
- [ ] Linux: uruchomienie binarki, `instaluj-linux.sh`, usługa systemd, monitor z /proc (dotąd tylko kompilacja i vet)
- [ ] Usługa Windows (`--usluga instaluj`) i druga instancja (okienko komunikatu)
- [ ] PWA: instalacja na telefonie; kod QR zeskanowany telefonem

## 3. Do decyzji usera

- [ ] Nazwa aplikacji (robocza „Skrytka”; `NazwaAplikacji` w `wewn/konfiguracja/wersja.go`)
- [ ] FTPS a 2FA: konto z 2FA nie zaloguje się przez FTPS (brak miejsca na kod); propozycja: osobne „hasła
      aplikacji” do programów FTP, jak w dużych usługach
- [ ] Ikona w zasobniku (biblioteka `fyne.io/systray`) czy wystarczy usługa Windows
- [ ] Let's Encrypt w trybie publicznym: czy jest domena; bez niej certyfikat własny i sprawdzanie odcisku
- [ ] Przekazywanie klientom plików tunelu WireGuard: kto je robi (router UniFi) i czy Skrytka ma tylko o nich
      przypominać
