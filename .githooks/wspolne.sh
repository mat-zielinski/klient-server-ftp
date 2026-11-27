# Wspólne dla haków git Skrytki (dołączane przez „. .githooks/wspolne.sh”). Działa w sh na Linuksie i w Git Bash na Windows.
# Repo jest PUBLICZNE: pliki z danymi tego komputera nigdy nie mogą trafić do historii.
export GOTOOLCHAIN=local
unset GOOS

# Ścieżki zakazane w repo (ERE): konta i hashe haseł, linki, konfiguracja, certyfikaty i klucze, logi, kopie,
# konfiguracja WireGuard, kopie awaryjne strażnika mutacji.
ZAKAZANE='^(dane/konta\.json|dane/udostepnienia\.json|dane/konfiguracja\.json|dane/certyfikaty/.*|logi/.*|kopie/.*|.*\.conf|.*\.mutacja_(kopia|tmp|konflikt))$'

# zakazane_w_liscie: z listy ścieżek (stdin) wypisuje zakazane (poza .gitkeep).
zakazane_w_liscie() {
    grep -E "$ZAKAZANE" | grep -vE '(^|/)\.gitkeep$'
}

# go_projektu: przenośne Go projektu albo go z PATH.
go_projektu() {
    for kandydat in _go-portable/bin/go _go-portable/bin/go.exe; do
        if [ -x "$kandydat" ] && "$kandydat" version >/dev/null 2>&1; then
            echo "$kandydat"
            return 0
        fi
    done
    echo go
}
