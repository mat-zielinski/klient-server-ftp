@echo off
rem Skrytka: budowanie interfejsu (web\dist) i plikow wykonywalnych do wyjscie\.
rem   wyjscie\skrytka.exe              Windows, bez okna konsoli (zwykle uruchomienie, usluga)
rem   wyjscie\skrytka_konsola.exe      Windows z konsola (diagnoza: komunikaty na ekranie)
rem   wyjscie\skrytka-linux-amd64      Linux x86-64
rem   wyjscie\skrytka-linux-arm64      Linux ARM64 (Raspberry Pi 3/4/5 z systemem 64-bit)
setlocal
chcp 65001 >nul
set "KORZEN=%~dp0.."
set "GO=%KORZEN%\_go-portable\bin\go.exe"
set "GOTOOLCHAIN=local"
set "CGO_ENABLED=0"

if not exist "%GO%" (
  echo Brak Go w _go-portable\bin\go.exe
  exit /b 1
)

echo [1/5] Interfejs: npm run build
pushd "%KORZEN%\web" || exit /b 1
if not exist node_modules (
  call npm install || goto blad_web
)
call npm run build || goto blad_web
popd

pushd "%KORZEN%" || exit /b 1
if not exist wyjscie mkdir wyjscie

echo [2/5] Windows: wyjscie\skrytka.exe
set "GOOS=windows"
set "GOARCH=amd64"
"%GO%" build -trimpath -ldflags "-s -w -H windowsgui" -o wyjscie\skrytka.exe ./cmd/skrytka || goto blad

echo [3/5] Windows z konsola: wyjscie\skrytka_konsola.exe
"%GO%" build -trimpath -ldflags "-s -w" -o wyjscie\skrytka_konsola.exe ./cmd/skrytka || goto blad

echo [4/5] Linux amd64: wyjscie\skrytka-linux-amd64
set "GOOS=linux"
set "GOARCH=amd64"
"%GO%" build -trimpath -ldflags "-s -w" -o wyjscie\skrytka-linux-amd64 ./cmd/skrytka || goto blad

echo [5/5] Linux arm64: wyjscie\skrytka-linux-arm64
set "GOARCH=arm64"
"%GO%" build -trimpath -ldflags "-s -w" -o wyjscie\skrytka-linux-arm64 ./cmd/skrytka || goto blad

popd
echo Gotowe: pliki w wyjscie\
exit /b 0

:blad_web
popd
echo Blad budowania interfejsu.
exit /b 1

:blad
popd
echo Blad budowania programu.
exit /b 1
