#!/bin/sh
# Skrytka: usunięcie z Linuksa (odwrotność instaluj-linux.sh).
# Użycie (jako root): sudo sh odinstaluj-linux.sh [--z-danymi] [--tak]
#   bez opcji     zatrzymuje i usuwa usługę, program (/opt/skrytka), ikonę i skrót w menu;
#                 dane (/var/lib/skrytka: konta, konfiguracja, certyfikat, logi) i użytkownik skrytka zostają
#   --z-danymi    dodatkowo usuwa /var/lib/skrytka i użytkownika skrytka (po potwierdzeniu)
#   --tak         potwierdza usunięcie danych bez pytania (tylko razem z --z-danymi)
# Udostępniane foldery i reguły zapory nie są ruszane.
set -eu

KATALOG=/opt/skrytka
DANE=/var/lib/skrytka
UZYTKOWNIK=skrytka
JEDNOSTKA=skrytka.service
PLIK_JEDNOSTKI=/etc/systemd/system/skrytka.service
NAKLADKA_KATALOG=/etc/systemd/system/skrytka.service.d
IKONA=/usr/share/icons/hicolor/256x256/apps/skrytka.png
IKONY_KORZEN=/usr/share/icons/hicolor
SKROT=/usr/share/applications/skrytka.desktop

blad() {
  echo "Błąd: $*" >&2
  exit 1
}

Z_DANYMI=0
BEZ_PYTAN=0
for argument in "$@"; do
  case "$argument" in
    --z-danymi) Z_DANYMI=1 ;;
    --tak|-y) BEZ_PYTAN=1 ;;
    -h|--pomoc|--help)
      echo "Użycie: sudo sh $0 [--z-danymi] [--tak]"
      exit 0
      ;;
    *) blad "Nieznana opcja: $argument (są --z-danymi, --tak i --pomoc)" ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  blad "Uruchom jako root: sudo sh $0"
fi

# Potwierdzenie przed czymkolwiek, żeby odmowa nie zostawiła połowy instalacji.
if [ "$Z_DANYMI" = 1 ] && [ "$BEZ_PYTAN" != 1 ]; then
  [ -t 0 ] || blad "Usunięcie danych wymaga potwierdzenia: uruchom w terminalu albo dodaj --tak."
  printf '%s' "Usunąć NIEODWRACALNIE $DANE (konta, konfiguracja, certyfikat, logi) i użytkownika $UZYTKOWNIK? Wpisz tak: "
  odpowiedz=""
  read -r odpowiedz || odpowiedz=""
  [ "$odpowiedz" = "tak" ] || blad "Przerwano, nic nie zostało usunięte."
fi

echo "[1/3] Usługa systemd"
if command -v systemctl >/dev/null 2>&1; then
  if systemctl list-unit-files "$JEDNOSTKA" >/dev/null 2>&1 || [ -f "$PLIK_JEDNOSTKI" ]; then
    systemctl disable --now "$JEDNOSTKA" >/dev/null 2>&1 || true # celowo: jednostki może już nie być
  fi
fi
rm -f "$PLIK_JEDNOSTKI"
rm -rf "$NAKLADKA_KATALOG"
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  systemctl reset-failed "$JEDNOSTKA" >/dev/null 2>&1 || true # celowo: brak wpisu o błędzie to nie błąd
fi

echo "[2/3] Program, ikona i skrót"
rm -rf "$KATALOG"
rm -f "$IKONA" "$SKROT"
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t -f "$IKONY_KORZEN" || true # celowo: tylko pamięć podręczna ikon
fi
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications || true # celowo: tylko pamięć podręczna menu
fi

echo "[3/3] Dane i użytkownik"
if [ "$Z_DANYMI" = 1 ]; then
  rm -rf "$DANE"
  if id "$UZYTKOWNIK" >/dev/null 2>&1; then
    userdel "$UZYTKOWNIK"
  fi
  echo "      Usunięto $DANE i użytkownika $UZYTKOWNIK."
else
  echo "      Zostawione: $DANE i użytkownik $UZYTKOWNIK (ponowna instalacja je przejmie)."
  echo "      Całkowite usunięcie: ten skrypt z opcją --z-danymi (jest też w paczce tar.gz)."
fi

echo "Gotowe. Reguły zapory (jeśli były dodane) usuń ręcznie, np. sudo ufw delete allow 8443/tcp"
echo "albo sudo firewall-cmd --permanent --remove-port=8443/tcp && sudo firewall-cmd --reload."
