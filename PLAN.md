# Serwer plików: plan działania (wersja 0.1, 2026-11-27)

Nazwa robocza: **Skrytka** (do zmiany). Wzór działania: FileBrowser Quantum. Wygląd: interfejs Ubiquiti (UniFi).
Zasady pracy przejęte z `Programator_Czarek` (CLAUDE.md + skill `programator-totalny`), przełożone na nowy język.

## A. Co budujemy

1. Jeden plik `skrytka.exe` na Twoim komputerze. Udostępnia wskazane foldery (np. `D:\Udostepnione`) tylko do
   odczytu albo z prawem wgrywania, zależnie od użytkownika.
2. Klient nic nie instaluje: otwiera przeglądarkę (telefon, laptop) i pobiera pliki. Dodatkowo, dla programów typu
   Total Commander / FileZilla, ten sam serwer mówi **FTPS** (FTP szyfrowane) na osobnym porcie.
3. Dwa tryby sieci, wybierane w konfiguracji (jedno pole `tryb_sieci`):
   1. **VPN (WireGuard)** — jak na Twoim zrzucie: klient ma tunel z `Dozwolone adresy IP: 192.168.0.0/24,
      10.10.10.0/24`, więc widzi Twój komputer pod adresem LAN (`192.168.0.x`). Serwer słucha tylko na adresie LAN /
      tunelu, z internetu niewidoczny. To tryb zalecany.
   2. **Publiczny (IP zewnętrzne + otwarty port)** — przekierowanie portu na routerze na Twój komputer. Wtedy serwer
      WYMUSZA HTTPS (certyfikat Let's Encrypt albo własny), blokadę po złych hasłach i limit prób; zwykłe FTP bez
      szyfrowania jest w tym trybie wyłączone.
4. Serwer VPN zostaje tam, gdzie jest (router / UniFi z WireGuard). Aplikacja go nie zastępuje, tylko podpowiada:
   pokazuje adresy, pod którymi jest osiągalna, i sprawdza, czy port jest otwarty.

## B. Język i technologia (zalecenie)

| element | wybór | dlaczego |
|---|---|---|
| serwer | **Go 1.2x** | jeden `.exe` bez instalacji i bez Pythona, start < 0,1 s, pobieranie plików z prędkością dysku / sieci (strumień + `Range` = wznawianie), wiele połączeń naraz bez wątków ręcznie; FileBrowser Quantum też jest w Go |
| FTPS | biblioteka `ftpserverlib` | dojrzała, obsługa TLS, ten sam magazyn plików co WWW |
| interfejs | **Svelte 5 + Vite**, wkompilowany w `.exe` (`go:embed`) | mały i szybki pakiet (~100 KB), brak osobnego serwera stron |
| hasła | argon2id, sesje w ciasteczku `HttpOnly` | standard |
| konfiguracja | `dane/konfiguracja.json` + kreator przy pierwszym starcie | jak w stacji |
| narzędzia | `go-portable\` i `node` w folderze projektu (jak `python-portable`) | Go nie jest zainstalowany na tym komputerze |

Odrzucone: Python (wolniejszy, wymaga paczki interpretera), Rust (szybki, ale wolniej się w nim buduje i zmienia),
Node na serwerze (wymaga runtime u Ciebie).

## C. Zasady z `Programator_Czarek` w nowym projekcie

1. Wszystko po polsku: nazwy (bez ogonków w kodzie: `sciezki`, `uzytkownik`), komentarze, komunikaty, dokumenty.
   Raporty A/B/C/D, wewnątrz 1/2/3, zawsze lista „czego NIE zrobiono”.
2. Warstwy, import tylko w dół (pilnuje test architektury w Go, odpowiednik `sprawdz_architekture.py`):

   | warstwa | katalog | zasada |
   |---|---|---|
   | konfiguracja | `wewn/konfiguracja/` | WSZYSTKIE stałe, ścieżki, wersja (`wersja.go`); żadnych liczb i ścieżek na sztywno gdzie indziej |
   | rdzen | `wewn/rdzen/` | czysta logika: piaskownica ścieżek (zakaz wyjścia poza udział), uprawnienia, listowanie, sortowanie, wyszukiwanie |
   | magazyn | `wewn/magazyn/` | jedyny dostęp do dysku |
   | uslugi | `wewn/uslugi/` | przebiegi bez HTTP/FTP: logowanie, udziały, linki, dziennik, bramka (`Bramka.CzyWolno`) |
   | widoki | `wewn/http/`, `wewn/ftp/` | tylko tłumaczą protokół na wywołania usług |
   | interfejs | `web/` | wygląd WYŁĄCZNIE z tokenów (`web/src/styl/tokeny.css`), ikony tylko SVG z paczki, bez emoji, motyw ciemny i jasny |
   | start | `cmd/skrytka/main.go` | składa całość |
3. Limity jakości: plik ≤ 600 linii, funkcja ≤ 80 linii, złożoność ≤ 15, ≤ 8 argumentów, bez cichego połykania
   błędów (wyjątek `// celowo: <powód>`), bez `fmt.Println` (tylko dziennik). Pilnują `golangci-lint` + własny strażnik.
4. Każda operacja na plikach i każde logowanie przechodzi przez bramkę uprawnień; każda nowa blokada
   bezpieczeństwa ma test mutacyjny (test pada, gdy blokadę zdjąć).
5. Zmiana logiki = testy w tym samym commicie (hak `commit-msg`, wyjątek `[bez testów: <powód>]`).
6. Dziennik serwera `logi/dziennik_RRRR-MM.jsonl` (kto, co pobrał, skąd, ile, czas) + strona „Dziennik” w UI; nigdy
   hasło ani token.
7. Wersja w tytule okna i w stopce UI, podbijana przy każdej zmianie, wpis w `zmiany.md`; `do-zrobienia.md`; mapa
   dokumentów w `CLAUDE.md`; własny skill projektu w `.claude/skills/skrytka/`.
8. Przed testami zawsze pytanie oknem: częściowe czy pełne; `sprawdz.bat` = pełne, `sprawdz.bat szybko` = bez testów.
9. Responsywność bez przeskakiwania (zasada 4a stacji): pasek i tabela plików zwężają się, nic nie znika bez zamiennika.

## D. Wygląd w stylu UniFi

1. Ciemne tło grafitowe, karty o lekkim kontraście, jeden niebieski akcent, cienkie ikony liniowe 24×24, zwarte tabele
   z jasnym nagłówkiem, pigułki statusu (zielona / szara / czerwona), wąski pasek boczny z ikonami po lewej.
2. Strony: **Pliki** (tabela / kafelki, ścieżka okruszkowa, podgląd obrazów, PDF, wideo, tekstu), **Udostępnienia**
   (linki z datą wygaśnięcia i hasłem), **Użytkownicy**, **Dziennik**, **Ustawienia** (foldery, tryb sieci, porty,
   certyfikat, test dostępności), **Pulpit** (adresy serwera, aktywne pobrania, przepustowość).
3. Odwzorowuję charakter (układ, gęstość, kolory), nie kopiuję logo ani grafik Ubiquiti.
4. (decyzja usera 2026-11-27: „tak fajnie ostylowane jak programator_czarek”) **Tokeny wprost z motywu stacji**
   (`programator/qt/motyw.py`), przeniesione do `web/src/styl/tokeny.css` pod tymi samymi nazwami: tło okna `#0e1420`,
   karty `#151d2b`, pasek boczny `#0b111b`, ramki `#1f2a3b`, akcent `#60a5fa` / przycisk główny `#2f6fe4`, zielony,
   czerwony, żółty jak w stacji, oraz motyw jasny z `_KOLORY_JASNE`. Ta sama skala odstępów, promieni i czcionek
   (`ODSTEP`, `PROMIEN`, `CZCIONKA`), te same ikony liniowe z `programator/styl/ikony_svg.py` (w Twoim stylu 24×24),
   pigułki stanów i kafle jak w stacji. Paleta stacji jest już granatowa w duchu UniFi, więc oba wzory się łączą.
5. Sprawdzenie wyglądu: zrzuty każdej strony w obu motywach, obok zrzutu stacji, w sprincie 2 i 6.

## E. Sprinty (praca wieloagentowa)

Zasada równoległości ze stacji: każdy agent ma ROZŁĄCZNE pliki, sam poprawia dokumenty swojej zmiany, nie robi
`git add/commit`; commituje sesja główna po przeglądzie. Sprint 0 robi jeden agent (fundament musi być spójny),
dalej 3–4 agentów naraz.

| sprint | co | agenci równolegle | czas agenta (zegar) |
|---|---|---|---|
| **0. Fundament** | `go-portable`, szkielet warstw, `konfiguracja`, CLAUDE.md, skill, haki git, `sprawdz.bat`, strażnik architektury i jakości, tokeny CSS + paczka ikon | 1 (sesja główna) | 1–1,5 h |
| **1. Rdzeń + pobieranie** | A: piaskownica ścieżek, listowanie, wyszukiwanie + testy; B: magazyn, pobieranie z `Range`, folder jako ZIP strumieniem; C: użytkownicy, argon2id, sesje, bramka uprawnień, blokada po złych hasłach; D: dziennik JSONL | 4 | 1,5–2 h |
| **2. Interfejs UniFi** | A: powłoka (pasek boczny, nagłówek, motywy); B: strona Pliki (tabela, kafelki, okruszki, podgląd); C: logowanie, Użytkownicy, Ustawienia; D: API HTTP pod te strony | 4 | 2–3 h |
| **3. Wgrywanie i udostępnianie** | A: wgrywanie w kawałkach z wznawianiem, przeciągnij i upuść; B: linki publiczne (wygasanie, hasło, limit pobrań); C: operacje: nowy folder, zmiana nazwy, usuwanie do kosza aplikacji | 3 | 1,5–2 h |
| **4. Sieć i FTPS** | A: serwer FTPS na tym samym magazynie i bramce; B: tryby VPN / publiczny, wybór interfejsu, HTTPS (Let's Encrypt + certyfikat własny); C: Pulpit: adresy, test portu, reguła Zapory Windows po zgodzie | 3 | 2 h |
| **5. Wydanie** | A: kreator pierwszego uruchomienia; B: usługa Windows + ikona w zasobniku (start z systemem); C: `przygotuj_pakiet.bat`, instrukcja dla klienta (WWW, FTPS, WireGuard) | 3 | 1–1,5 h |
| **6. Twardnienie** | A: testy obciążenia (wiele pobrań, plik 10+ GB); B: przegląd bezpieczeństwa (`/code-review`, ścieżki, sesje); C: testy przeglądarki na zrzutach obu motywów | 3 | 1,5 h |

**Razem: ok. 11–14 h pracy agentów (zegar), czyli 2–3 dni robocze** z Twoimi przeglądami między sprintami.
Pojedynczy agent bez równoległości: ok. 25–30 h. Pierwsza używalna wersja (logowanie, przeglądanie, pobieranie przez
VPN) jest po sprincie 2, ok. 5–6 h od startu.

## F. Czego plan NIE obejmuje i co niesprawdzone

1. Nie konfiguruje serwera WireGuard ani routera / UniFi (port forwarding robisz Ty, aplikacja tylko sprawdza).
2. Brak zwykłego FTP bez szyfrowania w trybie publicznym (świadomie: hasła szłyby otwartym tekstem).
3. Brak edycji plików online, OnlyOffice, miniatur wideo (FileBrowser Quantum to ma; można dodać w sprincie 7).
4. Szacunki czasu to ocena, nie pomiar; zależą od liczby poprawek po Twoich przeglądach.
5. Go nie jest zainstalowany: sprint 0 pobiera oficjalne Go do `go-portable\` (potrzebna Twoja zgoda na pobranie).

## G. Decyzje (2026-11-27)

1. Język: **Go**.
2. Klient: **przeglądarka + FTPS**.
3. **Git** (zmiana decyzji 2026-11-27): repo https://github.com/mat-zielinski/klient-server-ftp (PUBLICZNE),
   gałąź `main`. Agenci nie używają gita, commit i push robi sesja główna. Do repo nigdy: konta, hasła,
   certyfikaty, logi, konfiguracja serwera, pliki WireGuard.
4. Pobranie oficjalnego Go do `go-portable\`: **zgoda**.
5. Do ustalenia: nazwa aplikacji (robocza „Skrytka”).

## H. Standard profesjonalny (decyzja usera 2026-11-27: „jak aplikacja bigtech, prosta ale nowoczesna”)

| obszar | co dodajemy | agent (fala 2) |
|---|---|---|
| **Windows + Linux** | jeden kod, pliki `*_windows.go` / `*_linux.go`; binarki `skrytka.exe`, `skrytka-linux-amd64`, `skrytka-linux-arm64` (Raspberry Pi); usługa Windows i systemd; `instaluj-linux.sh` | aplikacja, straznicy, ci |
| **Monitor serwera** | CPU, RAM, dysk (zajętość i odczyt/zapis MB/s), sieć na interfejs (w tym WireGuard), połączenia, gorutyny, czas pracy; historia 10 min co 1 s i 24 h co 1 min; strona „Monitor” z wykresami na żywo | monitor, web-monitor |
| **Analityka pobrań** | mapa ciepła (dzień tygodnia × godzina), zajętość udziałów wg rodzaju plików, trend tygodniowy | indeks, web-monitor |
| **Szybkie wyszukiwanie** | indeks plików w tle (jak FileBrowser Quantum), odświeżany przy zmianach, wyniki w milisekundach | indeks |
| **Miniatury** | obrazy (jpg, png, gif, webp) skalowane w tle, pamięć podręczna na dysku, kafelki i galeria | miniatury |
| **UX klasy premium** | paleta poleceń Ctrl+K, skróty klawiszowe z pomocą „?”, wirtualizacja list (10 000+ plików bez przycięć), galeria ze strzałkami, kod QR linku (telefon), PWA (instalacja na telefonie), motyw automatyczny wg systemu, puste stany i szkielety ładowania | web-pro |
| **Bezpieczeństwo** | lista zalogowanych urządzeń z wylogowaniem, limit szybkości API, suma SHA-256 pliku do sprawdzenia pobrania (2FA usunięte decyzją usera 2026-11-28) | konta, http |
| **Limity** | limit przepustowości na użytkownika i globalny (np. żeby pobieranie nie zapchało łącza) | transfery |
| **Jakość i wydania** | GitHub Actions: budowa i testy na Windows i Linux przy każdym pushu, wydanie z binarkami przy tagu `v*`; Dependabot | ci |
| **Operacje** | `/api/zdrowie`, rotacja logów, kopia ustawień do pliku, sprawdzanie nowej wersji w GitHub Releases | http, aplikacja |
