# Skrytka: instrukcja dla osoby pobierającej

Ktoś udostępnił Ci pliki przez Skrytkę. Od niego dostajesz: **login i hasło**, **adres serwera** (np.
`https://192.168.0.10:8443`), **odcisk certyfikatu** (ciąg w postaci `AB:CD:12:…`) i, jeśli wchodzisz przez VPN,
**plik tunelu WireGuard** (`.conf`) albo kod QR. Adresy niżej są przykładowe; używaj tych, które dostałeś.

Plik tunelu i hasło traktuj jak klucz do mieszkania: nie przesyłaj ich dalej.

## 1. Wejście przez WireGuard (VPN)

Ten sposób działa, gdy właściciel serwera dał Ci tunel WireGuard. Serwer nie jest wtedy widoczny z internetu, tylko
przez tunel.

1. Zainstaluj klienta WireGuard: Windows i macOS ze strony wireguard.com, telefon ze sklepu (Google Play, App Store),
   aplikacja „WireGuard”.
2. Zaimportuj tunel:
   1. Windows: „Importuj tunel(e) z pliku” i wskaż otrzymany plik `.conf`.
   2. Telefon: „+” → „Skanuj kod QR” albo „Importuj z pliku”.
3. Sprawdź w szczegółach tunelu pole **Dozwolone adresy IP** (AllowedIPs): musi obejmować sieć serwera, np.
   `192.168.0.0/24`. Bez tego adres serwera nie odpowie.
4. Włącz tunel („Aktywuj”). Status ma być „Aktywny”, a licznik „Odebrano” powinien rosnąć.
5. Otwórz przeglądarkę i wpisz adres serwera **z `https://` i portem**, np. `https://192.168.0.10:8443`.
6. Przy pierwszym wejściu przeglądarka ostrzeże, że połączenie nie jest prywatne. To normalne: serwer ma własny
   certyfikat, a nie wystawiony przez urząd certyfikacji. **Zanim przejdziesz dalej, sprawdź odcisk:**
   1. Chrome, Edge: kliknij „Niezabezpieczona” na lewo od adresu → „Certyfikat jest nieprawidłowy” → odcisk
      **SHA-256**.
   2. Firefox: „Zaawansowane” → „Wyświetl certyfikat” → „Odciski” → SHA-256.
   3. Porównaj z odciskiem od właściciela serwera (wystarczy kilka pierwszych i ostatnich par znaków). Zgodny =
      „Zaawansowane” → „Przejdź do strony”. **Niezgodny = nie wchodź i daj znać właścicielowi.**
7. Zaloguj się loginem i hasłem.

Po pobraniu możesz wyłączyć tunel.

## 2. Wejście przez adres publiczny

Ten sposób działa, gdy właściciel serwera udostępnił go w internecie. Dostajesz adres w postaci
`https://example.com:8443` albo `https://203.0.113.5:8443` (przykłady).

1. Wpisz adres w przeglądarce, z `https://` i portem.
2. Jeśli serwer ma certyfikat Let's Encrypt (adres z nazwą domeny), przeglądarka nie ostrzega. Jeśli ostrzega,
   sprawdź odcisk jak w punkcie 1.6.
3. Zaloguj się. Po kilku złych hasłach serwer blokuje próby na kwadrans: odczekaj, nie zgaduj dalej.

## 3. Pobieranie w przeglądarce

1. Strona **Pliki**: udziały po lewej, foldery i pliki w tabeli albo w kafelkach, ścieżka u góry. Szukanie: pole
   „Szukaj” (bez wielkości liter i bez polskich znaków: „zolw” znajdzie „Żółw”).
2. Kliknięcie pliku otwiera **podgląd** (obrazy, PDF, wideo, tekst). Pobranie: przycisk „Pobierz” albo prawy przycisk.
3. **Prawy przycisk na pliku → „Pobierz przez…”** pokazuje sposoby dostępne dla Ciebie:

   | pozycja | do czego |
   |---|---|
   | Przeglądarka (HTTPS) | zwykłe pobranie; przerwane można wznowić w menedżerze pobrań przeglądarki |
   | Klient FTP (FTPS) | adres do skopiowania dla FileZilli, WinSCP, Total Commandera (punkt 4) |
   | Kopiuj link bezpośredni | link do pliku; otwiera się tylko po zalogowaniu |
   | Jako ZIP | folder albo zaznaczone pliki w jednym archiwum |
   | Link publiczny… | link bez logowania, z datą ważności (jeśli masz takie prawo) |

   Pozycja wyszarzona ma w opisie powód (np. „Konto bez prawa FTPS”).
4. Folder pobierzesz jako ZIP (prawy przycisk na folderze). Duże foldery pakują się w trakcie pobierania, więc pasek
   postępu przeglądarki może nie znać końcowego rozmiaru.
5. Jeśli masz prawo zapisu: przeciągnij pliki na okno, żeby je wgrać. Przerwane wgrywanie wznawia się od miejsca
   przerwania.

## 4. FileZilla, WinSCP, Total Commander (FTPS)

FTPS to FTP z szyfrowaniem TLS. Działa tylko na kontach z prawem FTPS i tylko w trybie **„FTP z jawnym TLS”**
(explicit). Zwykłe FTP bez szyfrowania jest wyłączone. Dane: host jak w adresie strony (bez `https://`), port FTPS
(domyślnie **2121**, dokładny pokazuje menu „Pobierz przez…” → „Klient FTP”), Twój login i hasło.

| program | ustawienia |
|---|---|
| **FileZilla** | Plik → Menedżer stron → Nowa strona. Protokół: „FTP – File Transfer Protocol”. Szyfrowanie: „Wymagaj jawnego FTP przez TLS”. Host `192.168.0.10`, port `2121`. Tryb logowania: „Normalne”, login i hasło. Karta „Ustawienia transferu”: tryb **pasywny**. Przy pierwszym połączeniu porównaj odcisk SHA-256 w oknie „Nieznany certyfikat” i zaznacz „Zawsze ufaj” |
| **WinSCP** | Nowe połączenie. Protokół pliku: „FTP”. Szyfrowanie: „Jawne szyfrowanie TLS/SSL”. Host, port `2121`, login, hasło. Tryb pasywny jest domyślny („Zaawansowane” → „Połączenie” → „Tryb pasywny”). Odcisk certyfikatu sprawdź w oknie ostrzeżenia |
| **Total Commander** | Sieć → „Połącz z serwerem FTP” → Nowe. Host: `192.168.0.10:2121`, login, hasło. Zaznacz „SSL/TLS” i „Użyj trybu pasywnego”. Przy pierwszym połączeniu Total Commander może poprosić o biblioteki OpenSSL: zgódź się na ich pobranie ze strony ghisler.com |

1. Po zalogowaniu widzisz udziały jako foldery w korzeniu.
2. **Tryb pasywny jest obowiązkowy.** Przez WireGuard i przez router aktywny tryb FTP nie zadziała.
3. Błąd „połączenie odrzucone” albo zawieszenie na „Pobieranie listy”: sprawdź, czy tunel jest aktywny, czy port to
   port FTPS (nie port strony 8443) i czy wybrany jest tryb pasywny.

## 5. Podgląd „Moje pobrania”

Na stronie Pliki w prawym dolnym rogu jest panel **„Moje pobrania”** (można go zwinąć). Każde Twoje pobranie
(z przeglądarki i przez FTPS) ma tam nazwę, sposób (HTTPS albo FTPS), pasek postępu, prędkość, pozostały czas i mały
wykres prędkości z ostatniej minuty. Widzisz tylko swoje pobrania. Właściciel serwera widzi wszystkie pobrania
i w razie potrzeby może je przerwać.

## Najczęstsze problemy

| objaw | co zrobić |
|---|---|
| strona się nie otwiera | tunel WireGuard włączony? adres z `https://` i portem? |
| ostrzeżenie o certyfikacie przy każdym wejściu | normalne przy certyfikacie własnym; zawsze sprawdź odcisk |
| odcisk się nie zgadza | nie wchodź, powiadom właściciela serwera |
| „Zbyt wiele prób logowania” | odczekaj kwadrans, potem wpisz hasło uważnie |
| pobieranie przerwane | ponów z menedżera pobrań przeglądarki; serwer wznawia od miejsca przerwania |
| FTPS nie łączy | „FTP z jawnym TLS”, port FTPS, tryb pasywny, konto z prawem FTPS |
