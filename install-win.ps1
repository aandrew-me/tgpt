# TGPT Installer Script for Windows 10/11

# If you receive "execution of scripts is disabled on this system" error while running this script,
# In PowerShell (as Administrator) type:
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

# Check if system is AMD64 or I386
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
# Remove existing tgpt to install the new version
if ( Test-Path -Path $target_dir\$prog_name -PathType Leaf ){
    Remove-Item $target_dir\$prog_name -force
}

# Move executable to 'C:\Program Files\TGPT'
Move-item -Path $PWD\$prog_name -destination $target_dir\$prog_name

# Add TGPT to PATH
write "Adding TGPT to the PATH"
if ( Check-Command -cmdname $prog_name ){
    write "$prog_name is already in PATH"
}
else {


}

# Read the complete machine PATH from the registry
$machinePath = [System.Environment]::GetEnvironmentVariable('Path','Machine')

# Check if it's already in PATH
if ( $machinePath -notmatch [Regex]::Escape($target_dir) ) {
    $newMachinePath = "$machinePath;$target_dir"

    # Add TGPT
    [System.Environment]::SetEnvironmentVariable(
    'Path',
    $newMachinePath,
    'Machine'
    )
  # No need to start a new cmd/pwsh session to be able to use TGPT
    $Env:PATH += ";$target_dir"
}

write "Complete!"
$new = $prog_name.Substring(0, $prog_name.lastIndexOf('.')) # Remove last dot from 'prog_name'
write "To start using it, simply type '$new -h'."
