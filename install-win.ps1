# TGPT Installer Script for Windows 10/11

# If you receive "execution of scripts is disabled on this system" error while running this script,
# Open PowerShell as Administrator and type:
#   Set-ExecutionPolicy RemoteSigned

# Once you no longer need to run any PowerShell scripts,
# it's better to set Execution Policy back to 'Restricted' (for better security):
#   Set-ExecutionPolicy Restricted

#Requires -RunAsAdministrator
$ErrorActionPreference = "Stop"

function Check-Command($cmdname){
    return [bool](Get-Command -Name $cmdname -ErrorAction SilentlyContinue)
}

$target_dir = 'C:\Program Files\TGPT' # <-- Needed directory for the executable
$prog_name = "tgpt.exe"               # <-- Needed executable filename

# Check if system is 64-bit or 32-bit
if ( [System.Environment]::Is64BitOperatingSystem ){
    write "Downloading executable for: 64-bit OS"
    Invoke-WebRequest -URI "https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-amd64.exe" -OutFile "$PWD\$prog_name"
}
else {
    write "Downloading executable for: 32-bit OS"
    Invoke-WebRequest -URI "https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-i386.exe" -OutFile "$PWD\$prog_name"
}

# Make sure the 'C:\' exists
if ( !( Test-Path -Path 'C:\' ) ) {
    write "C:\ Doesn't exist"
    exit
} 

write "Installing TGPT in C:\Program Files\"
Sleep 1
# Create 'TGPT' folder in needed directory
if ( !( Test-Path -Path $target_dir ) ) {
    New-Item -ItemType Directory -Path $target_dir
}
# Remove existing TGPT to install the new version
if ( Test-Path -Path $target_dir\$prog_name -PathType Leaf ){
    Remove-Item $target_dir\$prog_name -force
}

# Move executable to 'C:\Program Files\TGPT'
Move-item -Path $PWD\$prog_name -destination $target_dir\$prog_name

# Add TGPT to system PATH
# Read the complete machine PATH from the registry
$machinePath = [System.Environment]::GetEnvironmentVariable('Path','Machine')

# Check if it's already in PATH
if ( $machinePath -notmatch [Regex]::Escape($target_dir) ) {
    write "Adding TGPT to the PATH"
    $newMachinePath = "$machinePath;$target_dir"

    # Add TGPT if it's not
    [System.Environment]::SetEnvironmentVariable(
    'Path',
    $newMachinePath,
    'Machine'
    )
  # No need to start a new pwsh session to be able to use TGPT
  # This command adds it temporaly to PATH, in the current session only
    $Env:PATH += ";$target_dir"
}
else {
    write "$prog_name is already in PATH"
}

write "Complete!`n"
$new = $prog_name.Substring(0, $prog_name.lastIndexOf('.')) # Remove last dot from 'prog_name'
write "To start using it, simply type '$new -h' in PowerShell/CMD.`n"

# Provide the commands to uninstall it:

$remove_path = @'
[Environment]::SetEnvironmentVariable(
  'Path',
  ([Environment]::GetEnvironmentVariable('Path','Machine').Split(';') |
     Where-Object { $_ -ne 'C:\Program Files\TGPT' }
  ) -join ';',
  'Machine'
)
'@

$remove_dir = "Remove-Item -Path '$target_dir' -Recurse -Force"

Sleep 1
write "To uninstall it, open PowerShell as Admin, and run the following commands:"

write "To Delete it:`n$remove_dir"

write "To Remove it from PATH:`n$remove_path"
