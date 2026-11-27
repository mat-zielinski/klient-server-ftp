#!/bin/sh
# Włącza haki git projektu (.githooks) na Linuksie; raz na komputer. Na Windows: skrypty\wlacz_haki_git.bat.
cd "$(dirname "$0")/.." || exit 1
chmod +x .githooks/pre-commit .githooks/commit-msg .githooks/pre-push sprawdz.sh
if ! git config core.hooksPath .githooks; then
    echo "Nie udało się włączyć haków git."
    exit 1
fi
echo "Haki git włączone: .githooks"
