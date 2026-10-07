# Tests install.ps1 against a release made on the spot, with no network: a real
# mangabind built here, packed the way a release packs it, with its
# checksums.txt. The installer's downloads are answered from a folder by a
# stand-in for Invoke-WebRequest (PowerShell 7 does not read file:// URLs, and
# a local server would need a port). Run it from anywhere:
#
#   pwsh -File scripts/test-installers.ps1        (or powershell.exe -File ...)
#
# It needs Go, to build the program. Where the system is not Windows, the cases
# that need to run the installed program are skipped.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$installer = Join-Path $root "install.ps1"
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("mangabind-installer-test-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $work | Out-Null
$onWindows = ($PSVersionTable.PSVersion.Major -lt 6) -or $IsWindows
$script:failed = $false

function Pass($message) { Write-Host "ok    $message" }
function Fail($message) {
    Write-Host "FAIL  $message"
    $script:failed = $true
}

# Writes checksums.txt for the archives in a folder, as sha256sum would.
function Write-Checksums($folder, [switch]$Upper) {
    $lines = Get-ChildItem -LiteralPath $folder -Filter "*.zip" | ForEach-Object {
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash
        if (-not $Upper) { $hash = $hash.ToLowerInvariant() }
        "$hash  $($_.Name)"
    }
    Set-Content -LiteralPath (Join-Path $folder "checksums.txt") -Value $lines -Encoding ASCII
}

# What the installer asked for, and where the answers come from. This takes the
# place of Invoke-WebRequest for as long as the test runs: the installer, which
# calls it by name, finds this one first.
$script:ReleaseDir = $null
$script:Asked = @()
function Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    $script:Asked += $Uri
    $source = Join-Path $script:ReleaseDir $Uri.Substring($Uri.LastIndexOf("/") + 1)
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) { throw "404 Not Found: $Uri" }
    Copy-Item -LiteralPath $source -Destination $OutFile
}

# Runs the installer with the given environment and returns what happened.
function Invoke-Installer($releaseDir, $installDir, [hashtable]$environment = @{}) {
    $script:ReleaseDir = $releaseDir
    $script:Asked = @()
    $settings = @{ MANGABIND_BASE_URL = "https://mirror.test/release"; MANGABIND_NO_MODIFY_PATH = "1" }
    if ($installDir) { $settings["MANGABIND_INSTALL_DIR"] = $installDir }
    foreach ($key in $environment.Keys) { $settings[$key] = $environment[$key] }
    foreach ($name in "MANGABIND_VERSION", "MANGABIND_INSTALL_DIR", "MANGABIND_BASE_URL", "MANGABIND_NO_MODIFY_PATH") {
        Remove-Item -LiteralPath "Env:\$name" -ErrorAction SilentlyContinue
    }
    foreach ($key in $settings.Keys) {
        if ($null -ne $settings[$key]) { Set-Item -LiteralPath "Env:\$key" -Value $settings[$key] }
    }
    $result = @{ Ok = $true; Message = "" }
    try {
        . $installer | Out-Null
    } catch {
        $result.Ok = $false
        $result.Message = $_.Exception.Message
    }
    foreach ($name in "MANGABIND_VERSION", "MANGABIND_INSTALL_DIR", "MANGABIND_BASE_URL", "MANGABIND_NO_MODIFY_PATH") {
        Remove-Item -LiteralPath "Env:\$name" -ErrorAction SilentlyContinue
    }
    return $result
}

try {
    # A real program, in the archive a release would hold.
    Push-Location $root
    try {
        & go build -trimpath -ldflags "-X main.version=0.0.0-test" -o (Join-Path $work "mangabind.exe") ./cmd/mangabind
        if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    } finally {
        Pop-Location
    }
    $arch = if ([string][System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }
    $archive = "mangabind_windows_$arch.zip"

    $release = Join-Path $work "release"
    New-Item -ItemType Directory -Path $release | Out-Null
    Compress-Archive -LiteralPath (Join-Path $work "mangabind.exe") -DestinationPath (Join-Path $release $archive)
    Write-Checksums $release

    # 1. The release installs, runs, and leaves nothing else in the folder.
    if ($onWindows) {
        $dir = Join-Path $work "i1"
        $r = Invoke-Installer $release $dir
        $files = @(Get-ChildItem -LiteralPath $dir -Force | ForEach-Object { $_.Name })
        $version = (& (Join-Path $dir "mangabind.exe") --version)
        if ($r.Ok -and $version -eq "mangabind 0.0.0-test" -and ($files -join ",") -eq "mangabind.exe") {
            Pass "a release installs, runs, and leaves only the program"
        } else {
            Fail "a release installs, runs, and leaves only the program ($($r.Message); files: $($files -join ', '); version: $version)"
        }
    } else {
        Write-Host "skip  installing a release needs Windows to run the program"
    }

    # 2. A download that is not what checksums.txt says is refused, and what was installed is left alone.
    $damaged = Join-Path $work "damaged"
    New-Item -ItemType Directory -Path $damaged | Out-Null
    Copy-Item (Join-Path $release "checksums.txt") $damaged
    $evilDir = Join-Path $work "evil"
    New-Item -ItemType Directory -Path $evilDir | Out-Null
    Set-Content -LiteralPath (Join-Path $evilDir "mangabind.exe") -Value "not what was published"
    Compress-Archive -LiteralPath (Join-Path $evilDir "mangabind.exe") -DestinationPath (Join-Path $damaged $archive)
    $dir = Join-Path $work "i2"
    New-Item -ItemType Directory -Path $dir | Out-Null
    Set-Content -LiteralPath (Join-Path $dir "mangabind.exe") -Value "old" -NoNewline
    $r = Invoke-Installer $damaged $dir
    $kept = (Get-Content -LiteralPath (Join-Path $dir "mangabind.exe") -Raw)
    $left = @(Get-ChildItem -LiteralPath $dir -Force | ForEach-Object { $_.Name })
    if ((-not $r.Ok) -and $r.Message -match "checksum" -and $r.Message -match "tampered" -and $kept -eq "old" -and ($left -join ",") -eq "mangabind.exe") {
        Pass "a damaged archive is refused and what was installed is left alone"
    } else {
        Fail "a damaged archive is refused and what was installed is left alone ($($r.Message))"
    }

    # 3. A checksums.txt that does not list the archive.
    $nolist = Join-Path $work "nolist"
    New-Item -ItemType Directory -Path $nolist | Out-Null
    Copy-Item (Join-Path $release $archive) $nolist
    Set-Content -LiteralPath (Join-Path $nolist "checksums.txt") -Value ("0" * 64 + "  mangabind_plan9_mips.zip") -Encoding ASCII
    $dir = Join-Path $work "i3"
    $r = Invoke-Installer $nolist $dir
    if ((-not $r.Ok) -and $r.Message -match "does not list" -and -not (Test-Path (Join-Path $dir "mangabind.exe"))) {
        Pass "an archive that checksums.txt does not list is refused"
    } else {
        Fail "an archive that checksums.txt does not list is refused ($($r.Message))"
    }

    # 4. No checksums.txt at all: nothing can be checked, so nothing is installed.
    $nosums = Join-Path $work "nosums"
    New-Item -ItemType Directory -Path $nosums | Out-Null
    Copy-Item (Join-Path $release $archive) $nosums
    $dir = Join-Path $work "i4"
    $r = Invoke-Installer $nosums $dir
    if ((-not $r.Ok) -and $r.Message -match "cannot be checked" -and -not (Test-Path (Join-Path $dir "mangabind.exe"))) {
        Pass "a release with no checksums.txt is refused"
    } else {
        Fail "a release with no checksums.txt is refused ($($r.Message))"
    }

    # 5. No archive for this system.
    $noarchive = Join-Path $work "noarchive"
    New-Item -ItemType Directory -Path $noarchive | Out-Null
    Copy-Item (Join-Path $release "checksums.txt") $noarchive
    $dir = Join-Path $work "i5"
    $r = Invoke-Installer $noarchive $dir
    if ((-not $r.Ok) -and $r.Message -match "could not download" -and -not (Test-Path (Join-Path $dir "mangabind.exe"))) {
        Pass "a missing archive is an error that says so"
    } else {
        Fail "a missing archive is an error that says so ($($r.Message))"
    }

    # 6. An archive that checks out but holds no program.
    $empty = Join-Path $work "empty"
    New-Item -ItemType Directory -Path $empty | Out-Null
    Set-Content -LiteralPath (Join-Path $work "readme.txt") -Value "nothing here"
    Compress-Archive -LiteralPath (Join-Path $work "readme.txt") -DestinationPath (Join-Path $empty $archive)
    Write-Checksums $empty
    $dir = Join-Path $work "i6"
    $r = Invoke-Installer $empty $dir
    if ((-not $r.Ok) -and $r.Message -match "does not hold a mangabind.exe" -and -not (Test-Path (Join-Path $dir "mangabind.exe"))) {
        Pass "an archive without the program is refused"
    } else {
        Fail "an archive without the program is refused ($($r.Message))"
    }

    # 7. A hash written in capitals is the same hash.
    if ($onWindows) {
        $upper = Join-Path $work "upper"
        New-Item -ItemType Directory -Path $upper | Out-Null
        Copy-Item (Join-Path $release $archive) $upper
        Write-Checksums $upper -Upper
        $dir = Join-Path $work "i7"
        $r = Invoke-Installer $upper $dir
        if ($r.Ok -and (Test-Path (Join-Path $dir "mangabind.exe"))) {
            Pass "a checksum in capitals is accepted"
        } else {
            Fail "a checksum in capitals is accepted ($($r.Message))"
        }
    }

    # 8. The address the release is asked for: the installer is run with no
    # mirror, so that the addresses are GitHub's, and fails when asked for them.
    $emptyRelease = Join-Path $work "nothing"
    New-Item -ItemType Directory -Path $emptyRelease | Out-Null
    $base = "https://github.com/gustavommcv/mangabind/releases"
    $none = @{ MANGABIND_BASE_URL = $null }
    $r = Invoke-Installer $emptyRelease (Join-Path $work "i8") $none
    if ($script:Asked[0] -eq "$base/latest/download/$archive") {
        Pass "with no version it asks for the latest release"
    } else {
        Fail "with no version it asks for the latest release (asked for: $($script:Asked[0]))"
    }
    $r = Invoke-Installer $emptyRelease (Join-Path $work "i8") @{ MANGABIND_BASE_URL = $null; MANGABIND_VERSION = "v0.7.0" }
    if ($script:Asked[0] -eq "$base/download/v0.7.0/$archive") {
        Pass "a version is a release by that tag"
    } else {
        Fail "a version is a release by that tag (asked for: $($script:Asked[0]))"
    }
    $r = Invoke-Installer $emptyRelease (Join-Path $work "i8") @{ MANGABIND_BASE_URL = $null; MANGABIND_VERSION = "0.7.0" }
    if ($script:Asked[0] -eq "$base/download/v0.7.0/$archive") {
        Pass "a version without its v gets one"
    } else {
        Fail "a version without its v gets one (asked for: $($script:Asked[0]))"
    }
    if (($script:Asked | Where-Object { $_ -like "*api.github.com*" }).Count -eq 0) {
        Pass "the installer does not ask the API for anything"
    } else {
        Fail "the installer asks the API for a tag"
    }

    # 9. The file parses.
    $errors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($installer, [ref]$null, [ref]$errors)
    if ($errors.Count -eq 0) { Pass "install.ps1 parses" } else { Fail "install.ps1 does not parse: $($errors[0].Message)" }
} finally {
    Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}

if ($script:failed) { exit 1 }
