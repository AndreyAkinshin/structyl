# Structyl Bootstrap Script
# Downloads and installs the pinned version of structyl for this project.
#
# Usage: .structyl\setup.ps1

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$VersionFile = Join-Path $ScriptDir "version"

if (Test-Path $VersionFile) {
    $Version = (Get-Content $VersionFile -Raw).Trim()
    Write-Host "Installing structyl $Version..."
    irm https://get.structyl.akinshin.dev/install.ps1 -OutFile "$env:TEMP\install.ps1"
    & "$env:TEMP\install.ps1" -Version $Version
} else {
    Write-Host "Installing latest structyl..."
    irm https://get.structyl.akinshin.dev/install.ps1 | iex
}
