$ErrorActionPreference = "Stop"

$LogPath = "C:\Users\rasha\Desktop\Kingsway\docker-install.log"
$InstallerDir = "D:\Installers"
$DockerInstallDir = "D:\Docker\Docker"
$DockerDataRoot = "D:\DockerData"
$InstallerPath = Join-Path $InstallerDir "Docker Desktop Installer.exe"
$InstallerUrl = "https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe"

function Write-Log {
    param([string]$Message)
    $line = "{0} {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Message
    $line | Tee-Object -FilePath $LogPath -Append
}

function Run-Step {
    param(
        [string]$Name,
        [string]$FilePath,
        [string[]]$Arguments
    )
    Write-Log "START $Name"
    $process = Start-Process -FilePath $FilePath -ArgumentList $Arguments -Wait -PassThru
    Write-Log "END $Name exit=$($process.ExitCode)"
    return $process.ExitCode
}

New-Item -ItemType Directory -Path $InstallerDir, $DockerInstallDir, $DockerDataRoot -Force | Out-Null
Write-Log "Docker install script started"
Write-Log "InstallerDir=$InstallerDir"
Write-Log "DockerInstallDir=$DockerInstallDir"
Write-Log "DockerDataRoot=$DockerDataRoot"

$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
Write-Log "IsAdmin=$isAdmin"
if (-not $isAdmin) {
    throw "This script must run as Administrator."
}

$wslFeatureExit = Run-Step -Name "Enable WSL feature" -FilePath "dism.exe" -Arguments @("/online", "/enable-feature", "/featurename:Microsoft-Windows-Subsystem-Linux", "/all", "/norestart")
$vmpFeatureExit = Run-Step -Name "Enable VirtualMachinePlatform feature" -FilePath "dism.exe" -Arguments @("/online", "/enable-feature", "/featurename:VirtualMachinePlatform", "/all", "/norestart")

if ($wslFeatureExit -eq 3010 -or $vmpFeatureExit -eq 3010) {
    Write-Log "REBOOT_REQUIRED before Docker Desktop install"
    exit 3010
}

try {
    Write-Log "START WSL update"
    wsl --update *>> $LogPath
    Write-Log "END WSL update exit=$LASTEXITCODE"
} catch {
    Write-Log "WSL update failed: $($_.Exception.Message)"
}

if (-not (Test-Path $InstallerPath)) {
    Write-Log "START download Docker Desktop installer"
    Invoke-WebRequest -Uri $InstallerUrl -OutFile $InstallerPath
    Write-Log "END download Docker Desktop installer"
} else {
    Write-Log "Docker Desktop installer already exists"
}

$dockerExit = Run-Step -Name "Install Docker Desktop" -FilePath $InstallerPath -Arguments @(
    "install",
    "--accept-license",
    "--backend=wsl-2",
    "--installation-dir=$DockerInstallDir",
    "--wsl-default-data-root=$DockerDataRoot"
)

if ($dockerExit -eq 3010) {
    Write-Log "REBOOT_REQUIRED after Docker Desktop install"
    exit 3010
}
if ($dockerExit -ne 0) {
    throw "Docker Desktop installer failed with exit code $dockerExit"
}

Write-Log "Docker install script completed"
