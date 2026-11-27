@echo off
REM Wlacza haki git projektu (.githooks: pre-commit = dane komputera + sprawdz szybko; commit-msg = zmiana kodu
REM wymaga zmiany testow; pre-push = dane komputera w repo + testy zmienionych pakietow). Raz na komputer.
REM Haki to skrypty sh: na Windows wykonuje je Git Bash (instalowany razem z Git for Windows).
cd /d "%~dp0.."
git config core.hooksPath .githooks
if errorlevel 1 ( echo Nie udalo sie wlaczyc hakow git. & exit /b 1 )
echo Haki git wlaczone: .githooks
