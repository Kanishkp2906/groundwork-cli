<# 
.SYNOPSIS
    Install GroundWork TUI - Real CVE Training for Developers

.DESCRIPTION
    Downloads and installs the latest GroundWork TUI release from GitHub.

.EXAMPLE
    iwr -useb https://raw.githubusercontent.com/kanishkp/groundwork/main/install.ps1 | iex

.NOTES
    Requires PowerShell 5.1+
    Requires internet access to download from GitHub releases.
#>

[CmdletBinding()]
param(
    [Parameter()]
    [string]$InstallDir = "$env:USERPROFILE\.local\bin",
    
    [Parameter()]
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$REPO = "kanishkp/groundwork"
$BINARY_NAME = "groundwork.exe"
$GITHUB_API = "https://api.github.com/repos/kanishkp/groundwork"

Write-Host "╔══════════════════════════════════════╗" -ForegroundColor Blue
Write-Host "║     GroundWork TUI Installer         ║" -ForegroundColor Blue
Write-Host "║   Real CVE Training for Developers   ║" -ForegroundColor Blue
Write-Host "╚══════════════════════════════════════╝" -ForegroundColor Blue
Write-Host ""

# Detect architecture
$arch = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) { $arch = $env:PROCESSOR_ARCHITEW6432 }

switch ($arch) {
    'AMD64' { $platformArch = 'amd64' }
    'ARM64' { $platformArch = 'arm64' }
    default { Write-Error "Unsupported architecture: $arch"; exit 1 }
}

$platform = "windows_$platformArch"
Write-Host "Detected platform: $platform" -ForegroundColor Green

# Get latest release
Write-Host "Fetching latest release..." -ForegroundColor Blue
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/kanishkp/groundwork/releases/latest" -ErrorAction Stop
$version = $release.tag_name
Write-Host "Latest version: $version" -ForegroundColor Green

$assetName = "groundwork_${version}_windows_amd64.zip"
$downloadUrl = "https://github.com/kanishkp/groundwork/releases/download/$version/$assetName"

Write-Host "Downloading $assetName..." -ForegroundColor Blue

$tempDir = New-TemporaryDirectory
$zipPath = Join-Path $tempDir "groundwork.zip"

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -ErrorAction Stop
} catch {
    Write-Error "Failed to download: $_"
    exit 1
}

Write-Host "Extracting..." -ForegroundColor Blue
$extractDir = Join-Path $env:TEMP "groundwork_extract"
if (Test-Path $extractDir) { Remove-Item $extractDir -Recurse -Force }
New-Item -ItemType Directory -Path $extractDir | Out-Null

Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

$binaryPath = Join-Path $extractDir "groundwork.exe"
if (-not (Test-Path $binaryPath)) {
    $found = Get-ChildItem $extractDir -Filter "groundwork.exe" -Recurse | Select-Object -First 1
    if ($found) { $binaryPath = $found.FullName }
}

if (-not (Test-Path $binaryPath)) {
    Write-Error "Binary not found in archive"
    exit 1
}

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$installPath = Join-Path $InstallDir "groundwork.exe"
Move-Item -Path $binaryPath -Destination $installPath -Force

Write-Host "Installed to $installPath" -ForegroundColor Green

# Check PATH
$pathDirs = $env:Path -split ';'
$installDirInPath = $false
foreach ($dir in $pathDirs) {
    if ($dir -eq $InstallDir) { $installDirInPath = $true; break }
}

if (-not $installDirInPath) {
    Write-Warning "Warning: $InstallDir is not in your PATH."
    Write-Host "Add this to your PowerShell profile:"
    Write-Host '  $env:Path += ";$env:USERPROFILE\.local\bin"'
    Write-Host ""
    Write-Host "Or run this command to add it permanently:"
    Write-Host '  [Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\.local\bin", "User")'
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "Run 'groundwork' to start."
Write-Host "Set GROUNDWORK_SERVER_URL environment variable if needed."