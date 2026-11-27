#!/bin/sh
# Skrytka: instalacja i aktualizacja na Linuksie z systemd.
# Użycie (jako root): sudo sh instaluj-linux.sh [--tak] [plik_programu]
#   --tak          bez pytań (porty w zaporze NIE są wtedy otwierane, bo domyślna odpowiedź to „nie”)
#   plik_programu  domyślnie ./skrytka obok skryptu (paczka tar.gz), a gdy go brak,
#                  wyjscie/skrytka-linux-<architektura> (repozytorium po skrypty/buduj.sh)
# Co robi:
#   1. kopiuje program do /opt/skrytka/skrytka (właściciel root, usługa nie może go podmienić),
#   2. zakłada użytkownika systemowego skrytka (bez logowania), dane w /var/lib/skrytka
#      (dane/ i logi/; zmienna SKRYTKA_KATALOG w nakładce systemd skrytka.service.d/katalog.conf),
#   3. zapisuje usługę systemd (skrytka --usluga instaluj), uruchamia ją i włącza start z systemem,
#   4. instaluje ikonę i skrót w menu (otwiera panel w przeglądarce),
#   5. na życzenie otwiera porty w ufw albo firewalld.
# Ponowne uruchomienie = aktualizacja programu; dane, konta i certyfikat zostają.
# Usuwanie: sudo sh /opt/skrytka/odinstaluj-linux.sh [--z-danymi]
set -eu

KATALOG=/opt/skrytka
DANE=/var/lib/skrytka
UZYTKOWNIK=skrytka
JEDNOSTKA=skrytka.service
NAKLADKA_KATALOG=/etc/systemd/system/skrytka.service.d
NAKLADKA="$NAKLADKA_KATALOG/katalog.conf"
IKONA_CEL=/usr/share/icons/hicolor/256x256/apps/skrytka.png
IKONY_KORZEN=/usr/share/icons/hicolor
SKROT_CEL=/usr/share/applications/skrytka.desktop
# Domyślne porty programu (wewn/konfiguracja/konfiguracja.go); zmienione w Ustawieniach trzeba otworzyć ręcznie.
PORT_HTTPS=8443
PORT_FTPS=2121
PASYWNE_OD=50000
PASYWNE_DO=50100

blad() {
  echo "Błąd: $*" >&2
  exit 1
}

# pytaj "pytanie": 0 = tak, 1 = nie. Domyślnie nie; z --tak albo bez terminala też nie.
pytaj() {
  if [ "$BEZ_PYTAN" = 1 ] || [ ! -t 0 ]; then
    return 1
  fi
  printf '%s [t/N] ' "$1"
  odpowiedz=""
  read -r odpowiedz || return 1
  case "$odpowiedz" in
    t|T|tak|TAK|Tak|y|Y|yes) return 0 ;;
    *) return 1 ;;
  esac
}

BEZ_PYTAN=0
ZRODLO=""
for argument in "$@"; do
  case "$argument" in
    --tak|-y) BEZ_PYTAN=1 ;;
    -h|--pomoc|--help)
      echo "Użycie: sudo sh $0 [--tak] [plik_programu]"
      exit 0
      ;;
    -*) blad "Nieznana opcja: $argument (są --tak i --pomoc)" ;;
    *)
      [ -z "$ZRODLO" ] || blad "Podaj najwyżej jeden plik programu."
      ZRODLO=$argument
      ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  blad "Uruchom jako root: sudo sh $0"
fi
command -v systemctl >/dev/null 2>&1 || blad "Brak systemctl: skrypt wymaga systemd."

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) blad "Nieobsługiwana architektura: $(uname -m) (są amd64 i arm64)." ;;
esac

SKRYPT=$(cd "$(dirname "$0")" && pwd)
KORZEN=$(cd "$SKRYPT/.." && pwd)
if [ -z "$ZRODLO" ]; then
  ZRODLO="$SKRYPT/skrytka"
  if [ ! -f "$ZRODLO" ] && [ -f "$KORZEN/wyjscie/skrytka-linux-$ARCH" ]; then
    ZRODLO="$KORZEN/wyjscie/skrytka-linux-$ARCH"
  fi
fi
[ -f "$ZRODLO" ] || blad "Brak pliku $ZRODLO (pobierz paczkę skrytka-<wersja>-linux-$ARCH.tar.gz albo zbuduj: sh skrypty/buduj.sh)."

# Ikona i skrót: w paczce assets/linux obok skryptu, w repozytorium ../assets/linux.
ZASOBY=""
for kandydat in "$SKRYPT/assets/linux" "$KORZEN/assets/linux"; do
  if [ -f "$kandydat/skrytka.desktop" ]; then
    ZASOBY=$kandydat
    break
  fi
done

echo "[1/6] Użytkownik systemowy $UZYTKOWNIK"
if ! id "$UZYTKOWNIK" >/dev/null 2>&1; then
  POWLOKA=/usr/sbin/nologin
  [ -x "$POWLOKA" ] || POWLOKA=/sbin/nologin
  [ -x "$POWLOKA" ] || POWLOKA=/bin/false
  useradd --system --user-group --home-dir "$DANE" --no-create-home --shell "$POWLOKA" "$UZYTKOWNIK"
fi

echo "[2/6] Program w $KATALOG"
if systemctl is-active --quiet "$JEDNOSTKA"; then
  systemctl stop "$JEDNOSTKA"
fi
mkdir -p "$KATALOG"
chown root:root "$KATALOG"
chmod 0755 "$KATALOG"
# Kopia do pliku tymczasowego i podmiana: działający proces nie zostaje z uciętym plikiem.
install -m 0755 -o root -g root "$ZRODLO" "$KATALOG/skrytka.nowy"
mv -f "$KATALOG/skrytka.nowy" "$KATALOG/skrytka"
if [ -f "$SKRYPT/odinstaluj-linux.sh" ]; then
  install -m 0755 -o root -g root "$SKRYPT/odinstaluj-linux.sh" "$KATALOG/odinstaluj-linux.sh"
fi

echo "[3/6] Dane w $DANE"
mkdir -p "$DANE"
# Starsze instalacje trzymały dane w /opt/skrytka: przenosimy je raz, jeśli w nowym miejscu ich nie ma.
for nazwa in dane logi; do
  if [ -d "$KATALOG/$nazwa" ] && [ ! -e "$DANE/$nazwa" ]; then
    echo "      Przenoszę $KATALOG/$nazwa do $DANE/$nazwa"
    mv "$KATALOG/$nazwa" "$DANE/$nazwa"
  fi
done
chown -R "$UZYTKOWNIK:$UZYTKOWNIK" "$DANE"
chmod 0750 "$DANE"

echo "[4/6] Usługa systemd"
mkdir -p "$NAKLADKA_KATALOG"
cat > "$NAKLADKA" <<KONIEC
# Zapisane przez instaluj-linux.sh: dane i logi Skrytki poza folderem programu.
[Service]
Environment=SKRYTKA_KATALOG=$DANE
KONIEC
chmod 0644 "$NAKLADKA"
# Zapisuje /etc/systemd/system/skrytka.service (User=skrytka), daemon-reload i enable.
"$KATALOG/skrytka" --usluga instaluj
systemctl daemon-reload
systemctl start "$JEDNOSTKA"

echo "[5/6] Ikona i skrót w menu"
if [ -n "$ZASOBY" ]; then
  install -D -m 0644 "$ZASOBY/skrytka.png" "$IKONA_CEL"
  install -D -m 0644 "$ZASOBY/skrytka.desktop" "$SKROT_CEL"
  if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f "$IKONY_KORZEN" || echo "      Uwaga: gtk-update-icon-cache nie powiódł się (ikona pojawi się po ponownym zalogowaniu)."
  fi
  if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true # celowo: tylko pamięć podręczna menu
  fi
else
  echo "      Pominięte: brak assets/linux (skrytka.png, skrytka.desktop) obok skryptu."
fi

echo "[6/6] Zapora"
ZAPORA=""
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ZAPORA=ufw
elif command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
  ZAPORA=firewalld
fi
if [ -z "$ZAPORA" ]; then
  echo "      Nie wykryto włączonej zapory ufw ani firewalld: nic do zrobienia."
elif pytaj "Otworzyć w $ZAPORA porty $PORT_HTTPS/tcp (HTTPS), $PORT_FTPS/tcp (FTPS) i $PASYWNE_OD-$PASYWNE_DO/tcp (pasywne FTPS)?"; then
  if [ "$ZAPORA" = ufw ]; then
    ufw allow "$PORT_HTTPS/tcp"
    ufw allow "$PORT_FTPS/tcp"
    ufw allow "$PASYWNE_OD:$PASYWNE_DO/tcp"
  else
    firewall-cmd --permanent --add-port="$PORT_HTTPS/tcp"
    firewall-cmd --permanent --add-port="$PORT_FTPS/tcp"
    firewall-cmd --permanent --add-port="$PASYWNE_OD-$PASYWNE_DO/tcp"
    firewall-cmd --reload
  fi
else
  echo "      Porty nie zostały otwarte. Ręcznie ($ZAPORA):"
  if [ "$ZAPORA" = ufw ]; then
    echo "        sudo ufw allow $PORT_HTTPS/tcp && sudo ufw allow $PORT_FTPS/tcp && sudo ufw allow $PASYWNE_OD:$PASYWNE_DO/tcp"
  else
    echo "        sudo firewall-cmd --permanent --add-port=$PORT_HTTPS/tcp --add-port=$PORT_FTPS/tcp --add-port=$PASYWNE_OD-$PASYWNE_DO/tcp && sudo firewall-cmd --reload"
  fi
fi

echo
if systemctl is-active --quiet "$JEDNOSTKA"; then
  echo "Gotowe: usługa Skrytka działa (start z systemem włączony)."
else
  echo "Uwaga: usługa nie działa. Sprawdź: journalctl -u skrytka -n 50" >&2
fi
echo "Panel na tym komputerze: https://127.0.0.1:$PORT_HTTPS (pierwsze wejście = kreator)."
ADRESY=""
if command -v ip >/dev/null 2>&1; then
  ADRESY=$(ip -4 -o addr show scope global 2>/dev/null | awk '{ split($4, a, "/"); print a[1] }')
elif command -v hostname >/dev/null 2>&1; then
  ADRESY=$(hostname -I 2>/dev/null | tr ' ' '\n' | grep -F . || true)
fi
if [ -n "$ADRESY" ]; then
  echo "Adresy dla innych urządzeń (dostępność zależy od trybu sieci w kreatorze):"
  for adres in $ADRESY; do
    echo "  https://$adres:$PORT_HTTPS    ftps://$adres:$PORT_FTPS"
  done
fi
echo "Dane i logi: $DANE (dane/, logi/). Udostępniane foldery muszą być czytelne dla użytkownika $UZYTKOWNIK."
echo "Stan: systemctl status skrytka    Komunikaty: journalctl -u skrytka -f"
echo "Usuwanie: sudo sh $KATALOG/odinstaluj-linux.sh [--z-danymi]"
