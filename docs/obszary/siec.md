# Obszar: siec

## Co robi

Pakiet `wewn/siec`: adresy interfejsów, adresy nasłuchu wg trybu, konfiguracja TLS (plik / Let's Encrypt /
certyfikat własny), odcisk certyfikatu, test portu, reguła Zapory Windows, adres IP publiczny. Dane dla
`GET /api/siec` i `POST /api/siec/zapora`.

- `AdresyLokalne()`: IPv4 interfejsów w górze (bez link-local 169.254.x). Rodzaj: `petla` (127.x / interfejs
  pętli), `wireguard` (nazwa albo opis karty zawiera „wireguard” lub nazwa zaczyna się od „wg”; opis karty na
  Windows z `GetAdaptersAddresses`, np. „WireGuard Tunnel”), `lan` (adres prywatny), `publiczny` (reszta).
  Adres z 10.10.10.0/24 na karcie bez takiej nazwy to zwykły `lan` (rozpoznanie tylko po nazwie/opisie).
  Błąd systemu = pusta lista.
- `AdresyNasluchu(k)`: same IP bez portu. `AdresNasluchu` z konfiguracji ma pierwszeństwo; tryb publiczny =
  `0.0.0.0`; tryb VPN (i pusty tryb) = adresy `lan` + `wireguard` + `127.0.0.1`, bez powtórzeń. Adres
  publiczny na karcie nigdy nie trafia do nasłuchu w trybie VPN. Użytkownik z WireGuardem (10.10.10.6/32,
  dozwolone 192.168.0.0/24) dochodzi do serwera przez adres LAN 192.168.0.x, bo serwer WireGuard stoi na routerze.
- `KonfiguracjaTLS(k)`: kolejność: (1) `Certyfikat` + `Klucz` z konfiguracji (oba wymagane), (2) tryb publiczny
  z `DomenaLetsEncrypt` = `autocert` z pamięcią w `dane/certyfikaty/acme`, (3) certyfikat własny ECDSA P-256 w
  `dane/certyfikaty/wlasny.crt` + `wlasny.key` (zapis atomowy, 0600). Certyfikat własny jest używany ponownie;
  odnawiany, gdy brakuje któregoś bieżącego adresu lokalnego lub `localhost` w SAN albo do końca ważności
  zostało mniej niż 30 dni. Ważność 10 lat. TLS min. 1.2, w TLS 1.2 tylko ECDHE + AES-GCM/ChaCha20,
  krzywe X25519 i P-256.
- **Let's Encrypt**: autocert używa wyzwania TLS-ALPN-01, więc serwer HTTPS musi być osiągalny z internetu na
  **porcie 443** (przekierowanie 443 z routera na `PortHTTPS`). `NextProtos` zawiera `acme-tls/1`, więc
  serwer HTTP musi używać zwróconej konfiguracji bez nadpisywania `NextProtos`. Bez portu 443 certyfikat się
  nie wyda (HTTP-01 na porcie 80 nie jest podłączone).
- `OdciskCertyfikatu(c)`: SHA-256 certyfikatu liścia, format `AB:CD:…` (32 bajty). Przy Let's Encrypt (brak
  stałego certyfikatu w konfiguracji) zwraca `""`.
- `SprawdzPort(port)`: połączenie próbne na 127.0.0.1 (limit 1 s); gdy nie odpowiada, próbny `net.Listen`,
  żeby rozróżnić „wolny” od „zajęty/zablokowany”. Sprawdza tylko lokalnie, nie z internetu.
- `RegulaZapory(nazwa, porty)`: `netsh advfirewall firewall delete rule name=X` (błąd pomijany), potem
  `add rule name=X dir=in action=allow protocol=TCP localport=… profile=any`. Porty sortowane, kolejne zwijane
  w zakresy (`50000-50100`). Nazwa: tylko litery, cyfry, `_ . -`, do 64 znaków (pusta = „Skrytka”), więc nic
  nie wstrzykniemy do wiersza poleceń. Wyjście netsh dekodowane jawnie: UTF-8 albo CP852 (OEM polskiego Windows).
  Wymaga uprawnień administratora.
- `AdresPubliczny()`: pyta po kolei usługi (ipify, icanhazip, checkip.amazonaws), limit 3 s na zapytanie,
  czyta najwyżej 64 bajty, wynik musi być poprawnym IP; błąd = `""`.

## Pliki

- `wewn/siec/adresy.go`, `tls.go`, `narzedzia.go`, `opisy_windows.go`, `opisy_inne.go` (+ testy `_test.go`).
- `wewn/konfiguracja/siec.go`: stałe (usługi IP publicznego, limity czasu, nazwa reguły, pliki certyfikatu,
  ważność, wzorce nazw WireGuard, rodzaje adresów, `AdresPetli`, `AdresWszystkie`).

## Testy

`GOTOOLCHAIN=local _go-portable/bin/go.exe test ./wewn/siec/...`: klasyfikacja adresów i WireGuarda (atrapa
interfejsów), adresy nasłuchu dla trybów (w tym: VPN bez adresu publicznego i bez 0.0.0.0), generowanie,
ponowne użycie i odnowienie certyfikatu (`SKRYTKA_KATALOG` = `t.TempDir()`, zegar wstrzyknięty), certyfikat
z pliku i błędy, konfiguracja autocert (bez łączenia z Let's Encrypt), format odcisku, test portu na
127.0.0.1:0, polecenia netsh przez atrapę wykonawcy (netsh nie jest uruchamiany), odrzucanie złych nazw reguł,
dekodowanie CP852, IP publiczne na `httptest` (śmieci, błąd HTTP, za długa odpowiedź, limit czasu).
Sprawdzone wstrzyknięciem: bez klasyfikacji „publiczny” i bez walidacji nazwy reguły testy są czerwone.

## Czego nie zrobiono

- Brak wyzwania HTTP-01 (port 80) dla Let's Encrypt, tylko TLS-ALPN-01 na 443.
- Brak reguły Zapory dla UDP i dla IPv6; `SprawdzPort` nie sprawdza dostępności z internetu.
- Stała nazwy reguły nazywa się `DomyslnaNazwaRegulyZapory`, bo `konfiguracja/http.go` ma już `NazwaRegulyZapory`
  (ta sama wartość „Skrytka”).
