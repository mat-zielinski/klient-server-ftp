#!/bin/sh
# SPRAWDZ - komplet kontroli jakości Skrytki (Linux; na Windows to samo robi sprawdz.bat, ten skrypt działa też w Git Bash)
#   1) go vet dla Windows i Linuksa (GOOS=windows, GOOS=linux)
#   2) strażnik: architektura, system, jakość, testy, styl web (narzedzia_dev/straznik)
#   3) testy Go         (go test ./...)
#   4) interfejs        (web: npm run build)
#   5) mutacje blokad   (narzedzia_dev/mutacje: testy padają bez blokady)
# Użycie:  ./sprawdz.sh           wszystko
#          ./sprawdz.sh szybko    bez testów, budowy web i mutacji (1-2)
# Haki git: skrypty/wlacz_haki_git.sh (Linux) albo skrypty\wlacz_haki_git.bat (Windows), raz na komputer.
# Kod wyjścia: 0 = OK, 1 = któryś krok nieudany, 127 = brak Go.
cd "$(dirname "$0")" || exit 1
export GOTOOLCHAIN=local
unset GOOS
GO=go
for kandydat in _go-portable/bin/go _go-portable/bin/go.exe; do
    if [ -x "$kandydat" ] && "$kandydat" version >/dev/null 2>&1; then GO=$kandydat; break; fi
done
if ! "$GO" version >/dev/null 2>&1; then
    echo "Nie znaleziono Go: _go-portable/bin/go ani go w PATH"
    exit 127
fi
if [ -z "$(git config core.hooksPath 2>/dev/null)" ]; then
    echo "UWAGA: haki git wyłączone (brak core.hooksPath). Włącz raz na komputer: skrypty/wlacz_haki_git.sh"
fi

BLEDY=0
NIEUDANE=""
nieudany() { BLEDY=$((BLEDY + 1)); NIEUDANE="$NIEUDANE $1"; }

echo "[1/5] go vet (windows, linux)"
GOOS=windows "$GO" vet ./... || nieudany vet-windows
GOOS=linux "$GO" vet ./... || nieudany vet-linux
echo "[2/5] strażnik (architektura, system, jakość, testy, styl web)"
"$GO" run ./narzedzia_dev/straznik || nieudany straznik
if [ "$1" != "szybko" ]; then
    echo "[3/5] testy Go"
    "$GO" test ./... || nieudany testy
    echo "[4/5] interfejs (npm run build)"
    if [ -f web/package.json ]; then
        (cd web && npm run build) || nieudany web
    else
        echo "brak web/package.json, pominięto"
    fi
    echo "[5/5] mutacje blokad bezpieczeństwa"
    "$GO" run ./narzedzia_dev/mutacje || nieudany mutacje
fi
if [ "$BLEDY" -gt 0 ]; then
    echo "WYNIK: NIEUDANE kroki $BLEDY:$NIEUDANE"
    exit 1
fi
echo "WYNIK: WSZYSTKO OK"
exit 0
