@echo off
REM ======================================================================
REM  SPRAWDZ - komplet kontroli jakosci Skrytki (to samo na Linuksie: sprawdz.sh)
REM    1) go vet dla Windows i Linuksa (GOOS=windows, GOOS=linux)
REM    2) straznik: architektura, system, jakosc, testy, styl web (narzedzia_dev\straznik)
REM    3) testy Go         (go test ./...)
REM    4) interfejs        (web: npm run build)
REM    5) mutacje blokad   (narzedzia_dev\mutacje: testy padaja bez blokady)
REM  Uzycie (z dowolnego miejsca):  sprawdz.bat            wszystko
REM                                 sprawdz.bat szybko     bez testow, budowy web i mutacji (1-2)
REM  Haki git (te same kontrole przy commit/push): skrypty\wlacz_haki_git.bat, raz na komputer.
REM  Kod wyjscia: 0 = OK, 1 = ktorykolwiek krok nieudany, 9009 = brak Go.
REM ======================================================================
setlocal EnableExtensions
cd /d "%~dp0"
set "GOTOOLCHAIN=local"
set "GOOS="
set "GO=%CD%\_go-portable\bin\go.exe"
if not exist "%GO%" set "GO=go"
"%GO%" version >nul 2>nul
if errorlevel 1 ( echo Nie znaleziono Go: _go-portable\bin\go.exe ani go w PATH & exit /b 9009 )
set "HAKI="
for /f "delims=" %%H in ('git config core.hooksPath 2^>nul') do set "HAKI=%%H"
if not defined HAKI echo UWAGA: haki git wylaczone (brak core.hooksPath). Wlacz raz na komputer: skrypty\wlacz_haki_git.bat

set "BLEDY=0"
set "NIEUDANE="
echo [1/5] go vet (windows, linux)
set "GOOS=windows"
"%GO%" vet ./...
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% vet-windows" )
set "GOOS=linux"
"%GO%" vet ./...
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% vet-linux" )
set "GOOS="
echo [2/5] straznik (architektura, system, jakosc, testy, styl web)
"%GO%" run ./narzedzia_dev/straznik
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% straznik" )
if /i "%~1"=="szybko" goto koniec
echo [3/5] testy Go
"%GO%" test ./...
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% testy" )
echo [4/5] interfejs (npm run build)
if not exist "web\package.json" ( echo brak web\package.json, pominieto & goto mutacje )
pushd web
call npm run build
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% web" )
popd
:mutacje
echo [5/5] mutacje blokad bezpieczenstwa
"%GO%" run ./narzedzia_dev/mutacje
if errorlevel 1 ( set /a BLEDY+=1 & set "NIEUDANE=%NIEUDANE% mutacje" )
:koniec
if %BLEDY% GTR 0 ( echo WYNIK: NIEUDANE kroki %BLEDY%:%NIEUDANE% & exit /b 1 )
echo WYNIK: WSZYSTKO OK
exit /b 0
