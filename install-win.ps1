<#
.SYNOPSIS
  Per-user installer/uninstaller for tgpt on Windows.

.DESCRIPTION
  Downloads and installs the latest tgpt binary to %LOCALAPPDATA%\tgpt, adding it to the current user's PATH.
  Or, when run with -Uninstall, removes the binary and folder, and cleans up the user PATH entry.

.PARAMETER Uninstall
  Switch to remove tgpt instead of installing it.

.EXAMPLE
  # Install tgpt (per-user, no admin required)
  .\tgpt-installer.ps1

  # Uninstall tgpt
  .\tgpt-installer.ps1 -Uninstall
#>

[CmdletBinding(DefaultParameterSetName = 'Install')]
param(
    [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'

# Configuration
$installDir = Join-Path -Path $env:LOCALAPPDATA -ChildPath 'tgpt'
$exeName    = 'tgpt.exe'
$tempPath   = Join-Path -Path $PWD -ChildPath $exeName

function Install-tgpt {
    Write-Host "=== Installing tgpt (per-user) ===`n"

    # Determine the correct binary URL
    if ([Environment]::Is64BitOperatingSystem) {
        Write-Host "Detected 64-bit OS; downloading tgpt-amd64.exe..."
        $url = 'https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-amd64.exe'
    } else {
        Write-Host "Detected 32-bit OS; downloading tgpt-i386.exe..."
        $url = 'https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-i386.exe'
    }

    # Download the executable
    Write-Host "Downloading from $url"
    Invoke-WebRequest -Uri $url `
                      -OutFile $tempPath `
                      -UseBasicParsing `
                      -TimeoutSec 30
    Write-Host "Download complete.`n"

    # Prepare install directory under LOCALAPPDATA
    if (Test-Path $installDir) {
        Write-Host "Cleaning existing installation in $installDir..."
        Remove-Item -Path (Join-Path $installDir $exeName) -Force -ErrorAction SilentlyContinue
    } else {
        Write-Host "Creating install directory at $installDir"
        New-Item -ItemType Directory -Path $installDir | Out-Null
    }

    # Move the binary into place
    Move-Item -Path $tempPath -Destination (Join-Path $installDir $exeName) -Force
    Write-Host "tgpt executable installed to $installDir`n"

    # Update user PATH
    $currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not ($currentPath -split ';' | Where-Object { $_ -eq $installDir })) {
        Write-Host "Adding tgpt folder to user PATH..."
        $newPath = if ($currentPath) { "$currentPath;$installDir" } else { $installDir }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        # Also update current session
        $Env:PATH += ";$installDir"
        Write-Host "User PATH updated."
    } else {
        Write-Host "tgpt folder is already in your user PATH."
    }

    Write-Host "`nInstallation complete! Close and reopen your terminal, or run 'refreshenv' if you have it.`n"
}

function Uninstall-tgpt {
    Write-Host "=== Uninstalling tgpt (per-user) ===`n"

    # Remove executable and folder
    if (Test-Path $installDir) {
        Write-Host "Removing installation directory at $installDir..."
        Remove-Item -Path $installDir -Recurse -Force
        Write-Host "Deleted $installDir"
    } else {
        Write-Host "Install directory not found; skipping removal."
    }

    # Clean up user PATH
    $currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $paths       = $currentPath -split ';' | Where-Object { $_ -ne $installDir -and $_ -ne '' }

    if ($paths.Count -lt ($currentPath -split ';').Count) {
        Write-Host "Removing tgpt entry from your user PATH..."
        $newPath = ($paths -join ';')
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        # Also update current session
        $Env:PATH = ($Env:PATH -split ';' | Where-Object { $_ -ne $installDir -and $_ -ne '' }) -join ';'
        Write-Host "User PATH cleaned."
    } else {
        Write-Host "No tgpt entry found in your PATH; skipping."
    }

    Write-Host "`nUninstallation complete.`n"
}

# Main
if ($Uninstall) {
    Uninstall-tgpt
} else {
    Install-tgpt
}
