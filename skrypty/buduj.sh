#!/bin/sh
# Skrytka: budowanie interfejsu (web/dist) i plików wykonywalnych do wyjscie/ (na Linuksie).
#   wyjscie/skrytka-linux-amd64, wyjscie/skrytka-linux-arm64 (Raspberry Pi 64-bit),
#   wyjscie/skrytka.exe i wyjscie/skrytka_konsola.exe (Windows).
# Go: zmienna GO, inaczej _go-portable/bin/go (jeśli jest), inaczej go z PATH.
set -eu
KORZEN=$(cd "$(dirname "$0")/.." && pwd)
if [ -z "${GO:-}" ]; then
  if [ -x "$KORZEN/_go-portable/bin/go" ]; then GO="$KORZEN/_go-portable/bin/go"; else GO=go; fi
fi
export GOTOOLCHAIN=local CGO_ENABLED=0

echo "[1/5] Interfejs: npm run build"
cd "$KORZEN/web"
[ -d node_modules ] || npm install
npm run build

cd "$KORZEN"
mkdir -p wyjscie
echo "[2/5] Linux amd64"
GOOS=linux GOARCH=amd64 "$GO" build -trimpath -ldflags "-s -w" -o wyjscie/skrytka-linux-amd64 ./cmd/skrytka
echo "[3/5] Linux arm64"
GOOS=linux GOARCH=arm64 "$GO" build -trimpath -ldflags "-s -w" -o wyjscie/skrytka-linux-arm64 ./cmd/skrytka
echo "[4/5] Windows"
GOOS=windows GOARCH=amd64 "$GO" build -trimpath -ldflags "-s -w -H windowsgui" -o wyjscie/skrytka.exe ./cmd/skrytka
echo "[5/5] Windows z konsolą"
GOOS=windows GOARCH=amd64 "$GO" build -trimpath -ldflags "-s -w" -o wyjscie/skrytka_konsola.exe ./cmd/skrytka
echo "Gotowe: pliki w wyjscie/"
