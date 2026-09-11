# Downloads and installs the latest mangabind release for Windows.
# Usage: irm https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.ps1 | iex
$ErrorActionPreference = "Stop"

$repo = "gustavommcv/mangabind"

if (-not [Environment]::Is64BitOperatingSystem) {
    Write-Error "mangabind: unsupported architecture (32-bit)"
    exit 1
}
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name
if (-not $version) {
    Write-Error "mangabind: could not determine the latest release"
    exit 1
}

$archive = "mangabind_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$version/$archive"

$installDir = Join-Path $env:LOCALAPPDATA "Programs\mangabind"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$tmpZip = Join-Path $env:TEMP $archive
Write-Host "Downloading mangabind $version for windows/$arch..."
Invoke-WebRequest -Uri $url -OutFile $tmpZip

Expand-Archive -Path $tmpZip -DestinationPath $installDir -Force
Remove-Item $tmpZip

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your PATH. Restart your terminal, then try: mangabind --input <dir>"
} else {
    Write-Host "Done - try: mangabind --input <dir>"
}
