$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$RuntimeDir = Join-Path $Root ".runtime"
$BackendDir = Join-Path $Root "backend"
$BackendExe = Join-Path $RuntimeDir "backend-api-server.exe"
$BackendNextExe = Join-Path $RuntimeDir "backend-api-server.next.exe"
$BackendLog = Join-Path $RuntimeDir "backend.out.log"
$BackendErr = Join-Path $RuntimeDir "backend.err.log"
$BackendPid = Join-Path $RuntimeDir "backend.pid"

New-Item -ItemType Directory -Path $RuntimeDir -Force | Out-Null

function Test-Port {
  param([int]$Port)
  return $null -ne (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Get-PortOwner {
  param([int]$Port)
  $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($null -eq $conn -or $conn.OwningProcess -eq 0) {
    return $null
  }
  return $conn.OwningProcess
}

function Wait-Port {
  param([int]$Port, [int]$Seconds = 30)
  $deadline = (Get-Date).AddSeconds($Seconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-Port $Port) {
      return $true
    }
    Start-Sleep -Milliseconds 500
  }
  return $false
}

function Wait-PortClosed {
  param([int]$Port, [int]$Seconds = 15)
  $deadline = (Get-Date).AddSeconds($Seconds)
  while ((Get-Date) -lt $deadline) {
    if (-not (Test-Port $Port)) {
      return $true
    }
    Start-Sleep -Milliseconds 500
  }
  return $false
}

function Get-BackendSourceStamp {
  $files = Get-ChildItem -LiteralPath $BackendDir -Recurse -File -Include *.go,*.sql,go.mod,go.sum -ErrorAction SilentlyContinue
  $latest = $files | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
  if ($null -eq $latest) {
    return [datetime]::MinValue
  }
  return $latest.LastWriteTimeUtc
}

function Stop-BackendIfOwned {
  $owner = Get-PortOwner 8080
  if ($null -eq $owner) {
    return
  }

  $pidFromFile = $null
  if (Test-Path $BackendPid) {
    $rawPid = (Get-Content -LiteralPath $BackendPid -ErrorAction SilentlyContinue | Select-Object -First 1)
    if (-not [string]::IsNullOrWhiteSpace($rawPid)) {
      $pidFromFile = [int]$rawPid
    }
  }

  $process = Get-Process -Id $owner -ErrorAction SilentlyContinue
  $path = $null
  if ($null -ne $process) {
    try {
      $path = $process.Path
    } catch {
      $path = $null
    }
  }

  $ownedByRuntime = ($null -ne $pidFromFile -and $owner -eq $pidFromFile) -or ($path -eq $BackendExe)
  if (-not $ownedByRuntime) {
    Write-Host "Backend watcher found port 8080 owned by another process ($owner); restart skipped."
    return
  }

  Stop-Process -Id $owner -Force -ErrorAction SilentlyContinue
  if (-not (Wait-PortClosed 8080 15)) {
    throw "Backend process on 8080 did not stop cleanly."
  }
}

function Build-And-Restart-Backend {
  $go = Get-Command go -ErrorAction SilentlyContinue
  if ($null -eq $go) {
    Write-Host "go was not found; backend rebuild skipped."
    return
  }

  Write-Host "$(Get-Date -Format o) Backend source changed; rebuilding..."
  Remove-Item -LiteralPath $BackendNextExe -Force -ErrorAction SilentlyContinue

  Push-Location $BackendDir
  try {
    & $go.Source "build" "-o" $BackendNextExe "./cmd/api-server"
    if ($LASTEXITCODE -ne 0) {
      Write-Host "Backend build failed; old backend remains running."
      return
    }
  } finally {
    Pop-Location
  }

  Stop-BackendIfOwned
  Move-Item -LiteralPath $BackendNextExe -Destination $BackendExe -Force

  Remove-Item -LiteralPath $BackendLog, $BackendErr -Force -ErrorAction SilentlyContinue
  $backend = Start-Process `
    -FilePath $BackendExe `
    -WorkingDirectory $BackendDir `
    -WindowStyle Hidden `
    -RedirectStandardOutput $BackendLog `
    -RedirectStandardError $BackendErr `
    -PassThru
  Set-Content -LiteralPath $BackendPid -Value $backend.Id -Encoding ASCII

  if (-not (Wait-Port 8080 45)) {
    Write-Host "Backend rebuild succeeded, but backend did not start on 127.0.0.1:8080."
    return
  }

  Write-Host "$(Get-Date -Format o) Backend restarted on 127.0.0.1:8080."
}

$lastStamp = Get-BackendSourceStamp
Write-Host "$(Get-Date -Format o) Backend watcher started."

while ($true) {
  Start-Sleep -Seconds 1
  $stamp = Get-BackendSourceStamp
  if ($stamp -gt $lastStamp) {
    Start-Sleep -Milliseconds 700
    $lastStamp = Get-BackendSourceStamp
    Build-And-Restart-Backend
  }
}
