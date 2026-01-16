# Structyl CLI Installer for Windows
# https://get.structyl.akinshin.dev
#
# Usage:
#   irm https://get.structyl.akinshin.dev/install.ps1 | iex
#   irm https://get.structyl.akinshin.dev/install.ps1 -OutFile install.ps1; .\install.ps1 -Version 0.1.0

[CmdletBinding()]
param(
    [string]$Version,
    [string]$InstallDir
)

$ErrorActionPreference = "Stop"

# Configuration
$GitHubRepo = "akinshin/structyl"
$DefaultInstallDir = "$env:USERPROFILE\.structyl"
$InstallDir = if ($InstallDir) { $InstallDir } elseif ($env:STRUCTYL_INSTALL_DIR) { $env:STRUCTYL_INSTALL_DIR } else { $DefaultInstallDir }
$BinDir = "$InstallDir\bin"
$VersionsDir = "$InstallDir\versions"

function Write-Info {
    param([string]$Message)
    Write-Host "info: " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "success: " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warn {
    param([string]$Message)
    Write-Host "warn: " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Error-Exit {
    param([string]$Message)
    Write-Host "error: " -ForegroundColor Red -NoNewline
    Write-Host $Message
    exit 1
}

# Get version from .structyl/version file
function Get-ProjectVersion {
    $dir = Get-Location
    while ($dir) {
        $versionFile = Join-Path $dir ".structyl" "version"
        if (Test-Path $versionFile) {
            $version = (Get-Content $versionFile -Raw).Trim()
            Write-Info "Found .structyl/version: $version"
            return $version
        }
        $parent = Split-Path $dir -Parent
        if ($parent -eq $dir) { break }
        $dir = $parent
    }
    return $null
}

# Get latest version from GitHub API
function Get-LatestVersion {
    $url = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $url -UseBasicParsing
        return $response.tag_name -replace "^v", ""
    }
    catch {
        Write-Error-Exit "Failed to get latest version from GitHub: $_"
    }
}

# Download file with progress
function Get-File {
    param(
        [string]$Url,
        [string]$OutFile
    )
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing
    }
    catch {
        Write-Error-Exit "Failed to download $Url`: $_"
    }
}

# Verify SHA256 checksum
function Test-Checksum {
    param(
        [string]$File,
        [string]$ExpectedHash
    )
    $actualHash = (Get-FileHash -Path $File -Algorithm SHA256).Hash.ToLower()
    return $actualHash -eq $ExpectedHash.ToLower()
}

# Create shim CMD script
function New-Shim {
    $shimPath = "$BinDir\structyl.cmd"
    $shimContent = @'
@echo off
setlocal EnableDelayedExpansion

set "STRUCTYL_DIR=%USERPROFILE%\.structyl"
if defined STRUCTYL_INSTALL_DIR set "STRUCTYL_DIR=%STRUCTYL_INSTALL_DIR%"

:: Version resolution order:
:: 1. STRUCTYL_VERSION environment variable
:: 2. .structyl\version file (searches current dir up to root)
:: 3. %STRUCTYL_DIR%\default-version file
:: 4. Latest installed version

set "VERSION="

:: 1. Check environment variable
if defined STRUCTYL_VERSION (
    set "VERSION=%STRUCTYL_VERSION%"
    goto :found_version
)

:: 2. Search for .structyl\version file
set "SEARCH_DIR=%CD%"
:search_loop
if exist "%SEARCH_DIR%\.structyl\version" (
    set /p VERSION=<"%SEARCH_DIR%\.structyl\version"
    goto :found_version
)
for %%I in ("%SEARCH_DIR%\..") do set "PARENT=%%~fI"
if "%PARENT%"=="%SEARCH_DIR%" goto :check_default
set "SEARCH_DIR=%PARENT%"
goto :search_loop

:check_default
:: 3. Check default-version file
if exist "%STRUCTYL_DIR%\default-version" (
    set /p VERSION=<"%STRUCTYL_DIR%\default-version"
    goto :found_version
)

:: 4. Find latest installed version
set "LATEST="
for /d %%D in ("%STRUCTYL_DIR%\versions\*") do set "LATEST=%%~nxD"
if defined LATEST (
    set "VERSION=%LATEST%"
    goto :found_version
)

echo error: No structyl version installed >&2
echo Install with: irm https://get.structyl.akinshin.dev/install.ps1 ^| iex >&2
exit /b 1

:found_version
:: Trim whitespace from version
for /f "tokens=*" %%a in ("%VERSION%") do set "VERSION=%%a"

set "BINARY=%STRUCTYL_DIR%\versions\%VERSION%\structyl.exe"

if not exist "%BINARY%" (
    echo error: structyl %VERSION% is not installed >&2
    echo Install with: irm https://get.structyl.akinshin.dev/install.ps1 ^| iex >&2
    exit /b 1
)

"%BINARY%" %*
exit /b %ERRORLEVEL%
'@
    Set-Content -Path $shimPath -Value $shimContent -Encoding ASCII
    Write-Info "Created shim at $shimPath"
}

# Add to user PATH
function Add-ToPath {
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$BinDir*") {
        $newPath = "$BinDir;$userPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$BinDir;$env:PATH"
        Write-Info "Added $BinDir to user PATH"
        Write-Warn "You may need to restart your terminal for PATH changes to take effect"
    }
}

function Main {
    Write-Info "Structyl CLI Installer for Windows"

    # Detect architecture
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    Write-Info "Detected architecture: windows/$arch"

    # Determine version to install
    if (-not $Version) {
        $Version = $env:STRUCTYL_VERSION
    }
    if (-not $Version) {
        $Version = Get-ProjectVersion
    }
    if (-not $Version) {
        Write-Info "Fetching latest version..."
        $Version = Get-LatestVersion
    }
    Write-Info "Installing version: $Version"

    # Determine if this is a nightly build
    $isNightly = $Version -eq "nightly"

    # Check if already installed (skip for nightly - always update)
    $versionDir = "$VersionsDir\$Version"
    $binaryPath = "$versionDir\structyl.exe"

    if (-not $isNightly -and (Test-Path $binaryPath)) {
        Write-Info "Version $Version is already installed"
    }
    else {
        # Create directories
        New-Item -ItemType Directory -Force -Path $versionDir | Out-Null
        New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

        # Determine archive name and URLs based on version type
        if ($isNightly) {
            $archiveName = "structyl_nightly_windows_${arch}.zip"
            $downloadUrl = "https://github.com/$GitHubRepo/releases/download/nightly/$archiveName"
            $checksumsUrl = "https://github.com/$GitHubRepo/releases/download/nightly/checksums.txt"
        }
        else {
            $archiveName = "structyl_${Version}_windows_${arch}.zip"
            $downloadUrl = "https://github.com/$GitHubRepo/releases/download/v${Version}/$archiveName"
            $checksumsUrl = "https://github.com/$GitHubRepo/releases/download/v${Version}/checksums.txt"
        }

        $tempDir = Join-Path $env:TEMP "structyl-install-$(Get-Random)"
        New-Item -ItemType Directory -Force -Path $tempDir | Out-Null

        try {
            $archivePath = Join-Path $tempDir $archiveName
            $checksumsPath = Join-Path $tempDir "checksums.txt"

            Write-Info "Downloading $archiveName..."
            Get-File -Url $downloadUrl -OutFile $archivePath

            Write-Info "Downloading checksums..."
            Get-File -Url $checksumsUrl -OutFile $checksumsPath

            # Verify checksum
            Write-Info "Verifying checksum..."
            $checksums = Get-Content $checksumsPath
            $expectedHash = ($checksums | Where-Object { $_ -like "*$archiveName*" }) -split "\s+" | Select-Object -First 1

            if (-not (Test-Checksum -File $archivePath -ExpectedHash $expectedHash)) {
                Write-Error-Exit "Checksum verification failed!"
            }

            # Extract
            Write-Info "Extracting..."
            Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

            # Install binary
            Move-Item -Path (Join-Path $tempDir "structyl.exe") -Destination $binaryPath -Force

            Write-Success "Installed structyl $Version to $versionDir"
        }
        finally {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    # Create shim
    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    New-Shim

    # Set as default version (unless installing nightly alongside stable)
    if (-not $isNightly) {
        Set-Content -Path "$InstallDir\default-version" -Value $Version
        Write-Info "Set $Version as default version"
    }
    else {
        # Only set nightly as default if no other version is installed
        if (-not (Test-Path "$InstallDir\default-version")) {
            Set-Content -Path "$InstallDir\default-version" -Value $Version
            Write-Info "Set $Version as default version"
        }
        else {
            Write-Info "Keeping existing default version (use 'echo nightly > ~/.structyl/default-version' to change)"
        }
    }

    # Add to PATH
    Add-ToPath

    # Verify installation
    if (Test-Path $binaryPath) {
        Write-Host ""
        Write-Success "Structyl $Version installed successfully!"
        Write-Host ""
        Write-Host "To get started, run:"
        Write-Host "  structyl --help"
        Write-Host ""
        Write-Host "To pin a project to this version, create .structyl\version:"
        Write-Host "  mkdir -p .structyl; echo '$Version' > .structyl\version"
    }
}

Main
