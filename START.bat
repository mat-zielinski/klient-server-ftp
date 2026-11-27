@echo off
rem Skrytka: uruchomienie (gdy brak programu, najpierw budowanie).
setlocal
set "KORZEN=%~dp0"
if not exist "%KORZEN%wyjscie\skrytka.exe" (
  call "%KORZEN%skrypty\buduj.bat" || exit /b 1
)
start "" "%KORZEN%wyjscie\skrytka.exe" %*
