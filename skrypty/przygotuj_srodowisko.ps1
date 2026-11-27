# Skrytka: przygotowanie narzedzi do budowania na czystym komputerze (Windows, x64).
# Sprawdza Go (_go-portable) i Node (_node-portable albo Node w systemie); brakujace pobiera wylacznie z oficjalnych
# stron (go.dev, nodejs.org) do folderu projektu, bez instalacji w systemie i bez uprawnien administratora.
# Kazde archiwum jest sprawdzane suma SHA-256 opublikowana przez producenta; niezgodna suma = przerwanie.
# Kazdy etap pisze komunikat, pobieranie i rozpakowanie maja wlasny pasek postepu (bez curl i bez Expand-Archive,
# ktore w Windows PowerShell 5.1 rozpakowuje Go kilka minut bez zadnego znaku zycia).
# Wywoluje START.bat. Kod wyjscia: 0 = gotowe, 1 = blad.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Add-Type -AssemblyName System.IO.Compression.FileSystem

$Korzen = Split-Path -Parent $PSScriptRoot
$Tymczasowy = Join-Path $env:TEMP ('skrytka_srodowisko_' + [guid]::NewGuid().ToString('N'))
$LimitCzasuListyMs = 30000

function Pokaz-Krok([string]$tekst) { Write-Host "  $tekst" }

# Pasek w jednej linii: [#########.........]  45%  36,1 / 80,2 MB
function Show-Pasek([string]$opis, [long]$zrobione, [long]$razem, [string]$jednostka) {
    $szer = 30
    if ($razem -gt 0) {
        $ulamek = [math]::Min(1.0, $zrobione / $razem)
        $pelne = [int]($ulamek * $szer)
        $pasek = ('#' * $pelne) + ('.' * ($szer - $pelne))
        $tekst = "{0} [{1}] {2,3}%  {3}" -f $opis, $pasek, [int]($ulamek * 100), $jednostka
    } else {
        $tekst = "{0} {1}" -f $opis, $jednostka
    }
    Write-Host -NoNewline ("`r    " + $tekst.PadRight(78))
}

# Klikniecie w okno konsoli wlacza w Windows „szybkie zaznaczanie” i WSTRZYMUJE program (tytul „Select ...”).
# Wylaczamy je dla tego okna (dotyczy tez dalszej budowy w tym samym oknie). Blad = zostaje jak bylo.
function Disable-SzybkieZaznaczanie {
    try {
        Add-Type -Namespace Skrytka -Name Konsola -MemberDefinition @'
[DllImport("kernel32.dll")] public static extern System.IntPtr GetStdHandle(int n);
[DllImport("kernel32.dll")] public static extern bool GetConsoleMode(System.IntPtr h, out uint tryb);
[DllImport("kernel32.dll")] public static extern bool SetConsoleMode(System.IntPtr h, uint tryb);
'@
        $wejscie = [Skrytka.Konsola]::GetStdHandle(-10)
        $tryb = 0
        if ([Skrytka.Konsola]::GetConsoleMode($wejscie, [ref]$tryb)) {
            # ENABLE_QUICK_EDIT_MODE = 0x40 (zdejmujemy), ENABLE_EXTENDED_FLAGS = 0x80 (wymagane przy zmianie)
            [void][Skrytka.Konsola]::SetConsoleMode($wejscie, (($tryb -band (-bnot 0x40)) -bor 0x80))
        }
    } catch {
        Pokaz-Krok 'Uwaga: nie klikaj w to okno w trakcie pracy (klikniecie je wstrzymuje; Esc wznawia).'
    }
}

# Mala odpowiedz (lista wersji) z limitem czasu, zeby nic nie wisialo bez konca.
function Get-Tekst([string]$adres) {
    $zadanie = [System.Net.HttpWebRequest]::Create($adres)
    $zadanie.Timeout = $LimitCzasuListyMs
    $zadanie.ReadWriteTimeout = $LimitCzasuListyMs
    $odp = $zadanie.GetResponse()
    try {
        $czytnik = New-Object System.IO.StreamReader($odp.GetResponseStream())
        return $czytnik.ReadToEnd()
    } finally { $odp.Close() }
}

# Pobranie strumieniem z paskiem postepu co 0,25 s; 3 proby przy bledzie sieci.
function Get-Plik([string]$adres, [string]$cel) {
    for ($proba = 1; $proba -le 3; $proba++) {
        try {
            $zadanie = [System.Net.HttpWebRequest]::Create($adres)
            $zadanie.ReadWriteTimeout = $LimitCzasuListyMs
            $odp = $zadanie.GetResponse()
            $razem = $odp.ContentLength
            $wej = $odp.GetResponseStream()
            $wyj = [System.IO.File]::Create($cel)
            try {
                $bufor = New-Object byte[] 262144
                $zrobione = 0L
                $zegar = [Diagnostics.Stopwatch]::StartNew()
                $ostatnio = -1000
                while (($n = $wej.Read($bufor, 0, $bufor.Length)) -gt 0) {
                    $wyj.Write($bufor, 0, $n)
                    $zrobione += $n
                    if ($zegar.ElapsedMilliseconds - $ostatnio -ge 250) {
                        $ostatnio = $zegar.ElapsedMilliseconds
                        $mbs = if ($zegar.Elapsed.TotalSeconds -gt 0) { $zrobione / 1MB / $zegar.Elapsed.TotalSeconds } else { 0 }
                        $jedn = '{0:N1} / {1:N1} MB  {2:N1} MB/s' -f ($zrobione / 1MB), ($razem / 1MB), $mbs
                        Show-Pasek 'pobieranie' $zrobione $razem $jedn
                    }
                }
                Show-Pasek 'pobieranie' $zrobione $razem ('{0:N1} MB' -f ($zrobione / 1MB))
                Write-Host ''
                return
            } finally {
                $wyj.Close(); $wej.Close(); $odp.Close()
            }
        } catch {
            Write-Host ''
            if ($proba -eq 3) { throw "Nie udalo sie pobrac $adres ($($_.Exception.Message))." }
            Pokaz-Krok "Blad sieci, ponawiam ($proba/3) ..."
        }
    }
}

function Test-Suma([string]$plik, [string]$oczekiwana) {
    Pokaz-Krok 'Sprawdzam sume SHA-256 ...'
    $suma = (Get-FileHash -Path $plik -Algorithm SHA256).Hash.ToLower()
    if ($suma -ne $oczekiwana.ToLower()) {
        throw "Suma SHA-256 pobranego pliku sie nie zgadza (oczekiwano $oczekiwana, jest $suma). Przerwano."
    }
    Pokaz-Krok 'Suma zgodna.'
}

# Szybkie rozpakowanie (.NET ZipArchive) z paskiem co 500 plikow; ostatni krok to przeniesienie folderu.
function Expand-Do([string]$zip, [string]$cel, [string]$folderWArchiwum) {
    $roboczy = Join-Path $Tymczasowy 'rozpakowane'
    $archiwum = [System.IO.Compression.ZipFile]::OpenRead($zip)
    try {
        $razem = $archiwum.Entries.Count
        $i = 0
        foreach ($wpis in $archiwum.Entries) {
            $i++
            $sciezka = [System.IO.Path]::GetFullPath((Join-Path $roboczy $wpis.FullName))
            if (-not $sciezka.StartsWith($roboczy, [StringComparison]::OrdinalIgnoreCase)) {
                throw "Niebezpieczna sciezka w archiwum: $($wpis.FullName). Przerwano."
            }
            if ($wpis.FullName.EndsWith('/')) {
                [void][System.IO.Directory]::CreateDirectory($sciezka)
            } else {
                [void][System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($sciezka))
                [System.IO.Compression.ZipFileExtensions]::ExtractToFile($wpis, $sciezka, $true)
            }
            if ($i % 500 -eq 0 -or $i -eq $razem) {
                Show-Pasek 'rozpakowanie' $i $razem ("$i / $razem plikow")
            }
        }
        Write-Host ''
    } finally { $archiwum.Dispose() }
    Move-Item -Path (Join-Path $roboczy $folderWArchiwum) -Destination $cel
    Remove-Item -Recurse -Force $roboczy
}

function Install-Go {
    $go = Join-Path $Korzen '_go-portable\bin\go.exe'
    if (Test-Path $go) { Pokaz-Krok 'Go: jest (_go-portable)'; return }
    Pokaz-Krok 'Go: brak. Sprawdzam najnowsza wersje na go.dev ...'
    $wydania = Get-Tekst 'https://go.dev/dl/?mode=json' | ConvertFrom-Json
    $wydanie = $wydania | Where-Object { $_.stable } | Select-Object -First 1
    $plik = $wydanie.files | Where-Object { $_.os -eq 'windows' -and $_.arch -eq 'amd64' -and $_.kind -eq 'archive' }
    Pokaz-Krok "Go: pobieram $($wydanie.version) ($([int]($plik.size / 1MB)) MB) ..."
    $zip = Join-Path $Tymczasowy $plik.filename
    Get-Plik ('https://go.dev/dl/' + $plik.filename) $zip
    Test-Suma $zip $plik.sha256
    Pokaz-Krok 'Rozpakowuje Go ...'
    Expand-Do $zip (Join-Path $Korzen '_go-portable') 'go'
    Pokaz-Krok "Go: zainstalowano $($wydanie.version) w _go-portable"
}

function Install-Node {
    if (Test-Path (Join-Path $Korzen '_node-portable\node.exe')) { Pokaz-Krok 'Node: jest (_node-portable)'; return }
    $systemowy = Get-Command node -ErrorAction SilentlyContinue
    if ($systemowy) { Pokaz-Krok "Node: jest w systemie ($($systemowy.Source))"; return }
    Pokaz-Krok 'Node: brak. Sprawdzam wersje LTS na nodejs.org ...'
    $lista = Get-Tekst 'https://nodejs.org/dist/index.json' | ConvertFrom-Json
    $lts = $lista | Where-Object { $_.lts -and $_.files -contains 'win-x64-zip' } | Select-Object -First 1
    $nazwa = "node-$($lts.version)-win-x64"
    $baza = "https://nodejs.org/dist/$($lts.version)"
    Pokaz-Krok "Node: pobieram $($lts.version) ..."
    $zip = Join-Path $Tymczasowy "$nazwa.zip"
    Get-Plik "$baza/$nazwa.zip" $zip
    $sumy = (Get-Tekst "$baza/SHASUMS256.txt") -split "`n"
    $wiersz = $sumy | Where-Object { $_.Trim().EndsWith("  $nazwa.zip") } | Select-Object -First 1
    if (-not $wiersz) { throw "Brak sumy SHA-256 dla $nazwa.zip w SHASUMS256.txt. Przerwano." }
    Test-Suma $zip ($wiersz.Trim() -split '\s+')[0]
    Pokaz-Krok 'Rozpakowuje Node ...'
    Expand-Do $zip (Join-Path $Korzen '_node-portable') $nazwa
    Pokaz-Krok "Node: zainstalowano $($lts.version) w _node-portable"
}

try {
    Disable-SzybkieZaznaczanie
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
