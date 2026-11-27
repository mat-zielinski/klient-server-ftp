# Skrytka: przygotowanie narzedzi do budowania na czystym komputerze (Windows, x64).
# Sprawdza Go (_go-portable) i Node (_node-portable albo Node w systemie); brakujace pobiera wylacznie z oficjalnych
# stron (go.dev, nodejs.org) do folderu projektu, bez instalacji w systemie i bez uprawnien administratora.
# Kazde archiwum jest sprawdzane suma SHA-256 opublikowana przez producenta; niezgodna suma = przerwanie.
# Wywoluje START.bat. Kod wyjscia: 0 = gotowe, 1 = blad.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Korzen = Split-Path -Parent $PSScriptRoot
$Tymczasowy = Join-Path $env:TEMP ('skrytka_srodowisko_' + [guid]::NewGuid().ToString('N'))

function Pokaz-Krok([string]$tekst) { Write-Host "  $tekst" }

function Test-Suma([string]$plik, [string]$oczekiwana) {
    $suma = (Get-FileHash -Path $plik -Algorithm SHA256).Hash.ToLower()
    if ($suma -ne $oczekiwana.ToLower()) {
        throw "Suma SHA-256 pobranego pliku sie nie zgadza (oczekiwano $oczekiwana, jest $suma). Przerwano."
    }
}

function Expand-Do([string]$zip, [string]$cel, [string]$folderWArchiwum) {
    $roboczy = Join-Path $Tymczasowy 'rozpakowane'
    Expand-Archive -Path $zip -DestinationPath $roboczy -Force
    Move-Item -Path (Join-Path $roboczy $folderWArchiwum) -Destination $cel
    Remove-Item -Recurse -Force $roboczy
}

function Install-Go {
    $go = Join-Path $Korzen '_go-portable\bin\go.exe'
    if (Test-Path $go) { Pokaz-Krok 'Go: jest (_go-portable)'; return }
    Pokaz-Krok 'Go: brak, pobieram z go.dev (ok. 80 MB) ...'
    $wydania = Invoke-RestMethod 'https://go.dev/dl/?mode=json'
    $wydanie = $wydania | Where-Object { $_.stable } | Select-Object -First 1
    $plik = $wydanie.files | Where-Object { $_.os -eq 'windows' -and $_.arch -eq 'amd64' -and $_.kind -eq 'archive' }
    $zip = Join-Path $Tymczasowy $plik.filename
    Invoke-WebRequest -Uri ('https://go.dev/dl/' + $plik.filename) -OutFile $zip -UseBasicParsing
    Test-Suma $zip $plik.sha256
    Expand-Do $zip (Join-Path $Korzen '_go-portable') 'go'
    Pokaz-Krok "Go: zainstalowano $($wydanie.version) w _go-portable"
}

function Install-Node {
    if (Test-Path (Join-Path $Korzen '_node-portable\node.exe')) { Pokaz-Krok 'Node: jest (_node-portable)'; return }
    $systemowy = Get-Command node -ErrorAction SilentlyContinue
    if ($systemowy) { Pokaz-Krok "Node: jest w systemie ($($systemowy.Source))"; return }
    Pokaz-Krok 'Node: brak, pobieram wersje LTS z nodejs.org (ok. 30 MB) ...'
    $lista = Invoke-RestMethod 'https://nodejs.org/dist/index.json'
    $lts = $lista | Where-Object { $_.lts -and $_.files -contains 'win-x64-zip' } | Select-Object -First 1
    $nazwa = "node-$($lts.version)-win-x64"
    $baza = "https://nodejs.org/dist/$($lts.version)"
    $zip = Join-Path $Tymczasowy "$nazwa.zip"
    Invoke-WebRequest -Uri "$baza/$nazwa.zip" -OutFile $zip -UseBasicParsing
    $sumy = (Invoke-WebRequest -Uri "$baza/SHASUMS256.txt" -UseBasicParsing).Content -split "`n"
    $wiersz = $sumy | Where-Object { $_.Trim().EndsWith("  $nazwa.zip") } | Select-Object -First 1
    if (-not $wiersz) { throw "Brak sumy SHA-256 dla $nazwa.zip w SHASUMS256.txt. Przerwano." }
    Test-Suma $zip ($wiersz.Trim() -split '\s+')[0]
    Expand-Do $zip (Join-Path $Korzen '_node-portable') $nazwa
    Pokaz-Krok "Node: zainstalowano $($lts.version) w _node-portable"
}

try {
    New-Item -ItemType Directory -Force -Path $Tymczasowy | Out-Null
    Write-Host 'Sprawdzanie narzedzi do budowania Skrytki:'
    Install-Go
    Install-Node
    exit 0
} catch {
    Write-Host ''
    Write-Host "BLAD: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host 'Sprawdz polaczenie z internetem albo pobierz gotowy program:'
    Write-Host 'https://github.com/mat-zielinski/klient-server-ftp/releases/latest'
    exit 1
} finally {
    if (Test-Path $Tymczasowy) { Remove-Item -Recurse -Force $Tymczasowy -ErrorAction SilentlyContinue }
}
