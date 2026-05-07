param(
  [ValidateSet("quick", "standard", "full")]
  [string]$Mode = "quick",
  [switch]$SkipE2E,
  [switch]$NoStart,
  [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$FrontendDir = Join-Path $Root "frontend"
$RuntimeDir = Join-Path $Root ".runtime"
$BackendExe = Join-Path $RuntimeDir "backend-api-server.exe"
$LogFile = Join-Path $RuntimeDir "check-kingsway.log"
$script:CurrentStepLog = $null

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

  if (-not $script:CurrentStepLog) {
    throw "Internal check script error: CurrentStepLog was not initialized."
  }

  Push-Location $WorkingDirectory
  try {
    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & $FilePath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($output) {
      $output | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      if ($VerboseOutput) {
        $output | Write-Host
      }
    }
    if ($exitCode -ne 0) {
      throw $FailureMessage
    }
  } finally {
    if ($null -ne $oldErrorActionPreference) {
      $ErrorActionPreference = $oldErrorActionPreference
    }
    Pop-Location
  }
}

function Invoke-CheckedScript {
  param(
    [string]$ScriptPath,
    [string]$FailureMessage
  )

  if (-not $script:CurrentStepLog) {
    throw "Internal check script error: CurrentStepLog was not initialized."
  }

  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    $output = & powershell -NoProfile -ExecutionPolicy Bypass -File $ScriptPath 2>&1
    $exitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $oldErrorActionPreference
  }
  if ($output) {
    $output | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
    if ($VerboseOutput) {
      $output | Write-Host
    }
  }
  if ($exitCode -ne 0) {
    throw $FailureMessage
  }
}

function Invoke-GoTests {
  param(
    [string]$GoPath,
    [string]$PackagePattern,
    [string]$WorkingDirectory,
    [string]$FailureMessage
  )

  if (-not $script:CurrentStepLog) {
    throw "Internal check script error: CurrentStepLog was not initialized."
  }

  Push-Location $WorkingDirectory
  try {
    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & $GoPath test $PackagePattern 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($output) {
      $output | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      if ($VerboseOutput) {
        $output | Write-Host
      }
    }
    if ($exitCode -eq 0) {
      return
    }

    $outputText = ($output | Out-String)
    if ($outputText -notmatch "Application Control policy" -or $outputText -notmatch "backend/internal/httpapi") {
      throw $FailureMessage
    }

    "Windows Application Control blocked Go's temp httpapi test binary; running stable fallback." | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
    $fallbackExe = Join-Path $RuntimeDir "httpapi.test.exe"

    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $compileOutput = & $GoPath test -c -o $fallbackExe ".\internal\httpapi" 2>&1
    $compileExitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($compileOutput) {
      $compileOutput | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      if ($VerboseOutput) {
        $compileOutput | Write-Host
      }
    }
    if ($compileExitCode -ne 0) {
      throw $FailureMessage
    }

    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $runOutput = & $fallbackExe "-test.v" 2>&1
    $runExitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($runOutput) {
      $runOutput | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      if ($VerboseOutput) {
        $runOutput | Write-Host
      }
    }
    if ($runExitCode -ne 0) {
      throw $FailureMessage
    }

    if ($PackagePattern -ne "./...") {
      return
    }

    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $packageOutput = & $GoPath list -f '{{.ImportPath}}|{{len .TestGoFiles}}|{{len .XTestGoFiles}}' "./..." 2>&1
    $packageExitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($packageExitCode -ne 0) {
      $packageOutput | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      throw $FailureMessage
    }
    foreach ($line in $packageOutput) {
      if ($line -notmatch '\|') {
        continue
      }
      $parts = $line -split '\|'
      $package = $parts[0]
      $testCount = [int]$parts[1] + [int]$parts[2]
      if ($testCount -eq 0 -or $package -eq "kingsway/backend/internal/httpapi") {
        continue
      }
      $oldErrorActionPreference = $ErrorActionPreference
      $ErrorActionPreference = "Continue"
      $singleOutput = & $GoPath test $package 2>&1
      $singleExitCode = $LASTEXITCODE
      $ErrorActionPreference = $oldErrorActionPreference
      if ($singleOutput) {
        $singleOutput | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
        if ($VerboseOutput) {
          $singleOutput | Write-Host
        }
      }
      if ($singleExitCode -ne 0) {
        throw $FailureMessage
      }
    }
  } finally {
    if ($null -ne $oldErrorActionPreference) {
      $ErrorActionPreference = $oldErrorActionPreference
    }
    Pop-Location
  }
}

function Invoke-CheckStep {
  param(
    [string]$Name,
    [scriptblock]$Action
  )

  $safeName = ($Name -replace '[^a-zA-Z0-9.-]+', '-').Trim('-').ToLowerInvariant()
  $script:CurrentStepLog = Join-Path $RuntimeDir "check-$safeName.log"
  Set-Content -Path $script:CurrentStepLog -Value "## $Name`nStarted: $(Get-Date -Format o)`n" -Encoding utf8

  Write-Host ("[RUN ] {0}" -f $Name) -ForegroundColor Cyan
  try {
    & $Action
    Add-Content -Path $script:CurrentStepLog -Value "`nCompleted: $(Get-Date -Format o)`n"
    Add-Content -Path $LogFile -Value "`n===== PASS: $Name =====`n"
    Get-Content -Path $script:CurrentStepLog | Add-Content -Path $LogFile
    Write-Host ("[PASS] {0}" -f $Name) -ForegroundColor Green
  } catch {
    Add-Content -Path $script:CurrentStepLog -Value "`nFAILED: $($_.Exception.Message)`n"
    Add-Content -Path $LogFile -Value "`n===== FAIL: $Name =====`n"
    Get-Content -Path $script:CurrentStepLog | Add-Content -Path $LogFile
    Write-Host ("[FAIL] {0}" -f $Name) -ForegroundColor Red
    Write-Host ("Reason: {0}" -f $_.Exception.Message) -ForegroundColor Red
    Write-Host ("Full log: {0}" -f $LogFile) -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Failed step log tail:" -ForegroundColor Yellow
    Get-Content -Path $script:CurrentStepLog -Tail 120
    exit 1
  } finally {
    $script:CurrentStepLog = $null
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

function Get-ChangedFiles {
  param([string]$GitPath)

  $changed = @()
  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    $tracked = & $GitPath diff --name-only HEAD 2>$null
    if ($tracked) {
      $changed += $tracked
    }
    $untracked = & $GitPath ls-files --others --exclude-standard 2>$null
    if ($untracked) {
      $changed += $untracked
    }
  } finally {
    $ErrorActionPreference = $oldErrorActionPreference
  }

  return $changed | Where-Object { $_ } | Sort-Object -Unique
}

function Test-ChangedUnder {
  param([string[]]$Files, [string]$Prefix)
  foreach ($file in $Files) {
    if ($file -like "$Prefix/*") {
      return $true
    }
  }
  return $false
}

function Get-ChangedBackendPackages {
  param([string[]]$Files)

  $packages = New-Object System.Collections.Generic.HashSet[string]
  foreach ($file in $Files) {
    if ($file -notlike "backend/*.go" -and $file -notlike "backend/**/*.go") {
      continue
    }
    $directory = Split-Path $file -Parent
    if ([string]::IsNullOrWhiteSpace($directory)) {
      continue
    }
    $relative = $directory.Substring("backend".Length).TrimStart("\", "/").Replace("/", "\")
    if ([string]::IsNullOrWhiteSpace($relative)) {
      [void]$packages.Add(".")
    } else {
      [void]$packages.Add(".\$relative")
    }
  }

  return @($packages | Sort-Object)
}

function Resolve-GoImportPaths {
  param(
    [string]$GoPath,
    [string]$WorkingDirectory,
    [string[]]$PackagePatterns
  )

  $paths = @()
  if ($PackagePatterns.Count -eq 0) {
    return $paths
  }

  Push-Location $WorkingDirectory
  try {
    foreach ($package in $PackagePatterns) {
      $oldErrorActionPreference = $ErrorActionPreference
      $ErrorActionPreference = "Continue"
      $output = & $GoPath list $package 2>&1
      $exitCode = $LASTEXITCODE
      $ErrorActionPreference = $oldErrorActionPreference
      if ($output -and $script:CurrentStepLog) {
        $output | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      }
      if ($exitCode -ne 0) {
        throw "Failed to resolve Go package path for $package."
      }
      foreach ($line in $output) {
        if ($line -and $line -notmatch "\s") {
          $paths += $line
        }
      }
    }
  } finally {
    if ($null -ne $oldErrorActionPreference) {
      $ErrorActionPreference = $oldErrorActionPreference
    }
    Pop-Location
  }

  return @($paths | Sort-Object -Unique)
}

function Get-GoTestPackages {
  param(
    [string]$GoPath,
    [string]$WorkingDirectory
  )

  Push-Location $WorkingDirectory
  try {
    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & $GoPath list -f '{{.ImportPath}}|{{len .TestGoFiles}}|{{len .XTestGoFiles}}' "./..." 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldErrorActionPreference
    if ($output -and $script:CurrentStepLog) {
      $output | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
    }
    if ($exitCode -ne 0) {
      throw "Failed to list Go test packages."
    }

    $packages = @()
    foreach ($line in $output) {
      if ($line -notmatch '\|') {
        continue
      }
      $parts = $line -split '\|'
      $testCount = [int]$parts[1] + [int]$parts[2]
      if ($testCount -gt 0) {
        $packages += $parts[0]
      }
    }

    return @($packages | Sort-Object -Unique)
  } finally {
    if ($null -ne $oldErrorActionPreference) {
      $ErrorActionPreference = $oldErrorActionPreference
    }
    Pop-Location
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
Set-Content -Path $LogFile -Value "Kingsway check log`nStarted: $(Get-Date -Format o)`nRoot: $Root`n" -Encoding utf8

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
$changedFiles = @(Get-ChangedFiles $git.Source)
$backendChanged = Test-ChangedUnder $changedFiles "backend"
$frontendChanged = Test-ChangedUnder $changedFiles "frontend"
$toolingChanged = Test-ChangedUnder $changedFiles "tools"
$rootCheckChanged = $false
foreach ($file in $changedFiles) {
  if ($file -in @("check-kingsway.cmd", "README.md")) {
    $rootCheckChanged = $true
  }
}

Add-Content -Path $LogFile -Value "Mode: $Mode`nChanged files:`n$($changedFiles -join "`n")`n"

$script:WhitespaceDone = $false
$script:BackendBuildDone = $false
$script:FrontendLintDone = $false
$script:QuickBackendPackages = @()
$script:QuickBackendImportPaths = @()

function Invoke-QuickPhase {
  Write-Host "Kingsway check phase: quick (changed files only; no build/E2E unless needed)." -ForegroundColor Cyan
  Invoke-CheckStep "Whitespace check" {
    Invoke-CheckedNative $git.Source @("-c", "core.autocrlf=false", "diff", "--check") $Root "git diff --check failed."
  }
  $script:WhitespaceDone = $true

  if ($backendChanged) {
    $script:QuickBackendPackages = @(Get-ChangedBackendPackages $changedFiles)
    if ($script:QuickBackendPackages.Count -gt 0) {
      Invoke-CheckStep "Backend changed package tests" {
        $script:QuickBackendImportPaths = @(Resolve-GoImportPaths $go.Source $BackendDir $script:QuickBackendPackages)
        foreach ($package in $script:QuickBackendPackages) {
          Invoke-GoTests $go.Source $package $BackendDir "Backend changed package tests failed."
        }
      }
    }

    Invoke-CheckStep "Backend quick build" {
      Invoke-CheckedNative $go.Source @("build", "-o", $BackendExe, ".\cmd\api-server") $BackendDir "Backend quick build failed."
    }
    $script:BackendBuildDone = $true
  } else {
    Write-Host "[SKIP] Backend unchanged." -ForegroundColor Yellow
  }

  if ($frontendChanged) {
    Invoke-CheckStep "Frontend lint" {
      Invoke-CheckedNative $npm.Source @("run", "lint") $FrontendDir "Frontend lint failed."
    }
    $script:FrontendLintDone = $true
  } else {
    Write-Host "[SKIP] Frontend unchanged." -ForegroundColor Yellow
  }

  if (-not $backendChanged -and -not $frontendChanged -and -not $toolingChanged -and -not $rootCheckChanged) {
    Write-Host "[INFO] Only non-app files changed; quick mode kept checks minimal." -ForegroundColor Yellow
  }
}

function Invoke-StandardAdditions {
  Write-Host "Kingsway check phase: standard additions." -ForegroundColor Cyan

  Invoke-CheckStep "Backend remaining unit tests" {
    $previousIntegration = $env:KINGSWAY_INTEGRATION
    $env:KINGSWAY_INTEGRATION = ""
    try {
      $skipPackages = New-Object 'System.Collections.Generic.HashSet[string]'
      foreach ($package in $script:QuickBackendImportPaths) {
        [void]$skipPackages.Add($package)
      }

      $allPackages = @(Get-GoTestPackages $go.Source $BackendDir)
      $remainingPackages = @()
      foreach ($package in $allPackages) {
        if (-not $skipPackages.Contains($package)) {
          $remainingPackages += $package
        }
      }

      if ($remainingPackages.Count -eq 0) {
        "No remaining backend unit test packages after quick phase." | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
      }

      foreach ($package in $remainingPackages) {
        Invoke-GoTests $go.Source $package $BackendDir "Backend remaining unit tests failed."
      }
    } finally {
      $env:KINGSWAY_INTEGRATION = $previousIntegration
    }
  }

  if (-not $script:BackendBuildDone) {
    Invoke-CheckStep "Backend build" {
      Invoke-CheckedNative $go.Source @("build", "-o", $BackendExe, ".\cmd\api-server") $BackendDir "Backend build failed."
    }
    $script:BackendBuildDone = $true
  } else {
    Write-Host "[SKIP] Backend build already passed in quick phase." -ForegroundColor Yellow
  }

  if (-not $script:FrontendLintDone) {
    Invoke-CheckStep "Frontend lint" {
      Invoke-CheckedNative $npm.Source @("run", "lint") $FrontendDir "Frontend lint failed."
    }
    $script:FrontendLintDone = $true
  } else {
    Write-Host "[SKIP] Frontend lint already passed in quick phase." -ForegroundColor Yellow
  }
}

function Invoke-FullAdditions {
  Write-Host "Kingsway check phase: full additions." -ForegroundColor Cyan

  Invoke-CheckStep "Frontend build" {
    Invoke-CheckedNative $npm.Source @("run", "build") $FrontendDir "Frontend build failed."
  }

  if (-not $SkipE2E) {
    Invoke-CheckStep "Local app health" {
      $backendReady = Test-HttpOK "http://127.0.0.1:8080/healthz"
      $frontendReady = Test-HttpOK "http://127.0.0.1:3000/en/login"

      if ((-not $backendReady -or -not $frontendReady) -and -not $NoStart) {
        "Local app is not fully ready; starting without Docker..." | Out-File -FilePath $script:CurrentStepLog -Append -Encoding utf8
        Invoke-CheckedScript (Join-Path $PSScriptRoot "start-local-dev.ps1") "Local app start failed."
        $backendReady = Test-HttpOK "http://127.0.0.1:8080/healthz"
        $frontendReady = Test-HttpOK "http://127.0.0.1:3000/en/login"
      }

      if (-not $backendReady) {
        throw "Backend health check failed: http://127.0.0.1:8080/healthz"
      }
      if (-not $frontendReady) {
        throw "Frontend health check failed: http://127.0.0.1:3000/en/login"
      }
    }

    Invoke-CheckStep "Backend PostgreSQL integration tests" {
      $previousIntegration = $env:KINGSWAY_INTEGRATION
      $env:KINGSWAY_INTEGRATION = "1"
      try {
        Invoke-GoTests $go.Source "./internal/integration" $BackendDir "Backend PostgreSQL integration tests failed."
      } finally {
        $env:KINGSWAY_INTEGRATION = $previousIntegration
      }
    }

    Invoke-CheckStep "Owner E2E smoke tests" {
      $env:KINGSWAY_E2E_BASE_URL = "http://127.0.0.1:3000"
      $env:KINGSWAY_E2E_BACKEND_URL = "http://127.0.0.1:8080"
      $env:KINGSWAY_E2E_BROWSER_CHANNEL = Get-BrowserChannel
      Invoke-CheckedNative $npm.Source @("run", "e2e:smoke") $FrontendDir "Owner E2E smoke tests failed."
    }
  } else {
    Write-Host "[SKIP] E2E smoke tests skipped by -SkipE2E." -ForegroundColor Yellow
  }
}

Invoke-QuickPhase

if ($Mode -eq "quick") {
  Write-Host ""
  Write-Host "Kingsway quick check passed. Full log: $LogFile" -ForegroundColor Green
  Write-Host "For deeper checks: .\check-kingsway.cmd -Mode standard or .\check-kingsway.cmd -Mode full" -ForegroundColor Yellow
  exit 0
}

Invoke-StandardAdditions

if ($Mode -eq "full") {
  Invoke-FullAdditions
} else {
  Write-Host "[SKIP] Frontend build, integration, and E2E skipped in standard mode." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Kingsway $Mode check passed. Full log: $LogFile" -ForegroundColor Green
