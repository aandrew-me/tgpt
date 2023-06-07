# TGPT installer script for Windows OS

# If receiving error "execution of scripts is disabled on this system", open powershell as admin and type next command:
# Set-ExecutionPolicy RemoteSigned

#Requires -RunAsAdministrator
$ErrorActionPreference = "Stop"

function Check-Command($cmdname){
    return [bool](Get-Command -Name $cmdname -ErrorAction SilentlyContinue)
}

$target_dir = 'C:\Program Files\TGPT' # <-- here we wanna store our executable file
$prog_name = "tgpt.exe" # <-- this is how we wanna call our executable

# Check if system is AMD64 or I386
if ((Get-WmiObject win32_operatingsystem | select osarchitecture).osarchitecture -eq "64-bit"){
    write "Downloading executable for: 64-bit OS"
    Invoke-WebRequest -URI "https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-amd64.exe" -OutFile "$PWD\$prog_name"
}
else{
    write "Downloading executable for: 32-bit OS"
    Invoke-WebRequest -URI "https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-i386.exe" -OutFile "$PWD\$prog_name"
}

# Move executable to 'C:\Program Files\TGPT' folder

# (who knows, maybe someone has installed Windows on another partition)
if (!(Test-Path -Path 'C:\')) {
    write "C:\ Doesn't exist"
    exit
} 

write "Installing TGPT in C:\Program Files\"
Sleep 2
# Create 'TGPT' folder in needed directory
if (!(Test-Path -Path $target_dir)) {
    New-Item -ItemType Directory -Path $target_dir
}
# If tgpt is already installed, will overwrite it
if (Test-Path -Path $target_dir\$prog_name -PathType Leaf){
    Remove-Item $target_dir\$prog_name -force
}
Move-item -Path $PWD\$prog_name -destination $target_dir\$prog_name

# And add it to PATH
write "Adding TGPT to the PATH"
if (Check-Command -cmdname $prog_name){
    write "$prog_name is already in PATH"
}
else {
    setx /M PATH "`$Env:PATH;$target_dir"
}

write-host "Done! Terminal GPT is installed in '$target_dir\'."
#   Remove last dot from variable prog_name
$new = $prog_name.Substring(0, $prog_name.lastIndexOf('.'))
write-host "Now you can open cmd and type '$new -h' for help."
