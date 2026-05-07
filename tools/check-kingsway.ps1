param(
  [switch]$SkipE2E,
  [switch]$NoStart
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$FrontendDir = Join-Path $Root "frontend"
$RuntimeDir = Join-Path $Root ".runtime"
$BackendExe = Join-Path $RuntimeDir "backend-api-server.exe"

function Write-Step {
  param([string]$Message)
  Write-Host ""
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Get-RequiredCommand {
  param([string]$Name, [string]$InstallHint)
  $command = Get-Command $Name -ErrorAction SilentlyContinue
  if ($null -eq $command) {
    throw "$Name was not found. $InstallHint"
  }
  return $command
}

function Invoke-CheckedNative {
  param(
    [string]$FilePath,
    [string[]]$Arguments,
    [string]$WorkingDirectory,
    [string]$FailureMessage
  )

  Push-Location $WorkingDirectory
  try {
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw $FailureMessage
    }
  } finally {
    Pop-Location
  }
}

function Test-Port {
  param([int]$Port)
  return $null -ne (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Test-HttpOK {
  param([string]$Uri)
  try {
    $response = Invoke-WebRequest -UseBasicParsing -Uri $Uri -TimeoutSec 15
    return $response.StatusCode -ge 200 -and $response.StatusCode -lt 400
  } catch {
    return $false
  }
}

function Get-BrowserChannel {
  $chromePaths = @(
    "${env:ProgramFiles}\Google\Chrome\Application\chrome.exe",
    "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe",
    "${env:LOCALAPPDATA}\Google\Chrome\Application\chrome.exe"
  )
  foreach ($path in $chromePaths) {
    if (Test-Path $path) {
      return "chrome"
    }
  }

  $edgePaths = @(
    "${env:ProgramFiles}\Microsoft\Edge\Application\msedge.exe",
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
    "${env:LOCALAPPDATA}\Microsoft\Edge\Application\msedge.exe"
  )
  foreach ($path in $edgePaths) {
    if (Test-Path $path) {
      return "msedge"
    }
  }

  throw "Chrome or Microsoft Edge was not found. Install one of them or run Playwright browser install manually."
}

New-Item -ItemType Directory -Path $RuntimeDir -Force | Out-Null

$go = Get-RequiredCommand "go" "Install Go or add it to PATH."
$npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
if ($null -eq $npm) {
  $npm = Get-Command npm -ErrorAction SilentlyContinue
}
if ($null -eq $npm) {
  throw "npm was not found. Install Node.js or add npm to PATH."
}
$git = Get-RequiredCommand "git" "Install Git or add it to PATH."

$gotmp = Join-Path $RuntimeDir "gotmp"
New-Item -ItemType Directory -Path $gotmp -Force | Out-Null
$env:GOTMPDIR = (Resolve-Path $gotmp).Path

Write-Step "Backend tests"
Invoke-CheckedNative $go.Source @("test", "./...") $BackendDir "Backend tests failed."

Write-Step "Backend build"
Invoke-CheckedNative $go.Source @("build", "-o", $BackendExe, ".\cmd\api-server") $BackendDir "Backend build failed."

Write-Step "Frontend lint"
Invoke-CheckedNative $npm.Source @("run", "lint") $FrontendDir "Frontend lint failed."

Write-Step "Frontend build"
Invoke-CheckedNative $npm.Source @("run", "build") $FrontendDir "Frontend build failed."

Write-Step "Whitespace check"
Invoke-CheckedNative $git.Source @("diff", "--check") $Root "git diff --check failed."

if (-not $SkipE2E) {
  Write-Step "Local app health"
  $backendReady = Test-HttpOK "http://127.0.0.1:8080/healthz"
  $frontendReady = Test-HttpOK "http://127.0.0.1:3000/en/login"

  if ((-not $backendReady -or -not $frontendReady) -and -not $NoStart) {
    Write-Host "Local app is not fully ready; starting without Docker..."
    & (Join-Path $PSScriptRoot "start-local-dev.ps1")
    if ($LASTEXITCODE -ne 0) {
      throw "Local app start failed."
    }
    $backendReady = Test-HttpOK "http://127.0.0.1:8080/healthz"
    $frontendReady = Test-HttpOK "http://127.0.0.1:3000/en/login"
  }

  if (-not $backendReady) {
    throw "Backend health check failed: http://127.0.0.1:8080/healthz"
  }
  if (-not $frontendReady) {
    throw "Frontend health check failed: http://127.0.0.1:3000/en/login"
  }

  Write-Step "Owner E2E smoke tests"
  $env:KINGSWAY_E2E_BASE_URL = "http://127.0.0.1:3000"
  $env:KINGSWAY_E2E_BROWSER_CHANNEL = Get-BrowserChannel
  Invoke-CheckedNative $npm.Source @("run", "e2e:smoke") $FrontendDir "Owner E2E smoke tests failed."
} else {
  Write-Host "E2E smoke tests skipped by -SkipE2E."
}

Write-Host ""
Write-Host "Kingsway check passed." -ForegroundColor Green
