#!/bin/sh
# Skrytka: instalacja na Linuksie z systemd (uruchom jako root: sudo sh skrypty/instaluj-linux.sh [plik]).
#   1. kopiuje program do /opt/skrytka/skrytka (argument; domyślnie ./skrytka obok skryptu,
#      a gdy go brak, wyjscie/skrytka-linux-<architektura>),
#   2. zakłada użytkownika systemowego skrytka (bez logowania) i oddaje mu /opt/skrytka,
#   3. zapisuje usługę systemd (skrytka --usluga instaluj) i ją uruchamia.
# Dane: /opt/skrytka/dane, dziennik: /opt/skrytka/logi, komunikaty: journalctl -u skrytka.
# Udostępniane foldery muszą być czytelne (a do wgrywania zapisywalne) dla użytkownika skrytka.
set -eu
KATALOG=/opt/skrytka
UZYTKOWNIK=skrytka
KORZEN=$(cd "$(dirname "$0")/.." && pwd)

if [ "$(id -u)" -ne 0 ]; then
  echo "Uruchom jako root: sudo sh $0" >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Nieobsługiwana architektura: $(uname -m) (są amd64 i arm64)" >&2; exit 1 ;;
esac
# Domyślnie binarka "skrytka" obok skryptu (paczka tar.gz), a gdy jej brak, wynik budowania w wyjscie/.
SKRYPT=$(cd "$(dirname "$0")" && pwd)
ZRODLO=${1:-"$SKRYPT/skrytka"}
if [ $# -eq 0 ] && [ ! -f "$ZRODLO" ] && [ -f "$KORZEN/wyjscie/skrytka-linux-$ARCH" ]; then
  ZRODLO="$KORZEN/wyjscie/skrytka-linux-$ARCH"
fi
if [ ! -f "$ZRODLO" ]; then
  echo "Brak pliku $ZRODLO (zbuduj: skrypty/buduj.sh albo skrypty\buduj.bat)" >&2
  exit 1
fi

if ! id "$UZYTKOWNIK" >/dev/null 2>&1; then
  useradd --system --home-dir "$KATALOG" --shell /usr/sbin/nologin "$UZYTKOWNIK"
fi

if systemctl is-active --quiet skrytka.service; then
  systemctl stop skrytka.service
fi

mkdir -p "$KATALOG"
install -m 0755 "$ZRODLO" "$KATALOG/skrytka"
chown -R "$UZYTKOWNIK:$UZYTKOWNIK" "$KATALOG"
chmod 0750 "$KATALOG"

"$KATALOG/skrytka" --usluga instaluj
systemctl start skrytka.service
echo "Gotowe. Stan: systemctl status skrytka, komunikaty: journalctl -u skrytka -f"
echo "Pierwsze uruchomienie: otwórz https://<adres-serwera>:8443 i przejdź kreator."
