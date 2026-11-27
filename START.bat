@echo off
rem Skrytka: uruchomienie jednym kliknieciem.
rem   Jest wyjscie\skrytka.exe -> uruchamia.
rem   Nie ma -> sprawdza Go i Node, brakujace pobiera z oficjalnych stron (go.dev, nodejs.org) do folderu projektu
rem   (skrypty\przygotuj_srodowisko.ps1, sumy SHA-256), buduje program (skrypty\buduj.bat) i uruchamia.
rem   START.bat --tylko-buduj  = przygotowanie i budowa bez uruchamiania.
setlocal
chcp 65001 >nul
set "KORZEN=%~dp0"
if exist "%KORZEN%wyjscie\skrytka.exe" goto uruchom

echo Pierwsze uruchomienie: przygotowanie programu (jednorazowo, kilka minut).
powershell -NoProfile -ExecutionPolicy Bypass -File "%KORZEN%skrypty\przygotuj_srodowisko.ps1"
if errorlevel 1 goto blad
if exist "%KORZEN%_node-portable\node.exe" set "PATH=%KORZEN%_node-portable;%PATH%"
call "%KORZEN%skrypty\buduj.bat"
if errorlevel 1 goto blad
if not exist "%KORZEN%wyjscie\skrytka.exe" goto blad

:uruchom
if /i "%~1"=="--tylko-buduj" exit /b 0
start "" "%KORZEN%wyjscie\skrytka.exe" %*
exit /b 0

:blad
echo.
echo Nie udalo sie przygotowac programu. Gotowy program do pobrania:
echo https://github.com/mat-zielinski/klient-server-ftp/releases/latest
pause
exit /b 1
