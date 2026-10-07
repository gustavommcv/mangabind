# Downloads, checks and installs a mangabind release for Windows.
#
#   irm https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.ps1 | iex
#
# With no version it installs the latest release. To choose, set variables
# before running it (a script piped into iex cannot take parameters):
#
#   $env:MANGABIND_VERSION = "v0.7.0"      # a release to install instead of the latest (or 0.7.0)
#   $env:MANGABIND_INSTALL_DIR = "D:\Tools" # where to put it (default: %LOCALAPPDATA%\Programs\mangabind)
#   $env:MANGABIND_BASE_URL = "..."         # a folder or URL holding the release files, for a mirror or a test
#   $env:MANGABIND_NO_MODIFY_PATH = "1"     # do not add the folder to your user PATH
#
# The archive is only installed if its SHA-256 is the one in the release's
# checksums.txt. Everything is inside Install-Mangabind, which is called on the
# last line, so a download that is cut short does not run half a script.
$ErrorActionPreference = "Stop"

function Install-Mangabind {
    $repo = "gustavommcv/mangabind"
    $version = $env:MANGABIND_VERSION
    $installDir = $env:MANGABIND_INSTALL_DIR
    if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA "Programs\mangabind" }

    $architecture = [string][System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($architecture) {
        "X64" { $arch = "amd64" }
        "Arm64" { $arch = "arm64" }
        default { throw "mangabind: unsupported architecture: $architecture" }
    }
    $archive = "mangabind_windows_${arch}.zip"

    # Where the release files are. GitHub serves "the latest release" at a fixed
    # address, so nothing needs to ask the API for a tag.
    if ($env:MANGABIND_BASE_URL) {
        $base = $env:MANGABIND_BASE_URL.TrimEnd("/")
        $label = "from $base"
    } elseif ($version) {
        if (-not $version.StartsWith("v")) { $version = "v$version" }
        $base = "https://github.com/$repo/releases/download/$version"
        $label = $version
    } else {
        $base = "https://github.com/$repo/releases/latest/download"
        $label = "the latest release"
    }

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("mangabind-" + [System.Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    $previousProgress = $ProgressPreference
    # Windows PowerShell 5.1 draws a progress bar that makes a download many times slower.
    $ProgressPreference = "SilentlyContinue"
    try {
        $zipPath = Join-Path $tmp $archive
        $sumsPath = Join-Path $tmp "checksums.txt"

        Write-Host "Downloading mangabind ($label) for windows/$arch..."
        try {
            Invoke-WebRequest -Uri "$base/$archive" -OutFile $zipPath -UseBasicParsing
        } catch {
            throw "mangabind: could not download $base/$archive (is the version right? the releases are listed at https://github.com/$repo/releases)"
        }
        try {
            Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumsPath -UseBasicParsing
        } catch {
            throw "mangabind: could not download $base/checksums.txt, so the download cannot be checked; nothing was installed"
        }

        # The line of checksums.txt for this archive: "<sha256>  <name>".
        $expected = $null
        foreach ($line in Get-Content -LiteralPath $sumsPath) {
            $parts = $line.Trim() -split "\s+", 2
            if ($parts.Count -eq 2 -and $parts[1].TrimStart("*") -eq $archive) {
                $expected = $parts[0].ToLowerInvariant()
                break
            }
        }
        if (-not $expected) { throw "mangabind: checksums.txt does not list $archive; nothing was installed" }
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "mangabind: the checksum of $archive is $actual, but checksums.txt says $expected; the download is damaged or has been tampered with, and nothing was installed"
        }

        $unpacked = Join-Path $tmp "unpacked"
        Expand-Archive -LiteralPath $zipPath -DestinationPath $unpacked
        $exe = Join-Path $unpacked "mangabind.exe"
        if (-not (Test-Path -LiteralPath $exe)) { throw "mangabind: $archive does not hold a mangabind.exe" }

        # Moved into place in two steps, the last one a rename inside the folder,
        # so that a failure never leaves half a program.
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
        $target = Join-Path $installDir "mangabind.exe"
        $staged = Join-Path $installDir "mangabind.exe.new"
        Copy-Item -LiteralPath $exe -Destination $staged -Force
        try {
            Move-Item -LiteralPath $staged -Destination $target -Force
        } catch {
            Remove-Item -LiteralPath $staged -Force -ErrorAction SilentlyContinue
            throw "mangabind: could not replace $target (is mangabind running? close it and run this again)"
        }

        Write-Host "Installed to $target"
        & $target --version
        if ($LASTEXITCODE -ne 0) { throw "mangabind: the installed program does not run on this system" }

        if ($env:MANGABIND_NO_MODIFY_PATH) {
            Write-Host "Done - add $installDir to your PATH, then try: mangabind --input <dir>"
        } else {
            $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
            $entries = @()
            if ($userPath) { $entries = @($userPath -split ";" | Where-Object { $_ }) }
            if ($entries -notcontains $installDir) {
                [Environment]::SetEnvironmentVariable("Path", ((@($entries) + $installDir) -join ";"), "User")
                Write-Host "Added $installDir to your PATH. Restart your terminal, then try: mangabind --input <dir>"
            } else {
                Write-Host "Done - try: mangabind --input <dir>"
            }
        }
    } finally {
        $ProgressPreference = $previousProgress
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Install-Mangabind
