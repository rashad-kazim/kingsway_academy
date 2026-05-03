$ErrorActionPreference = "Continue"

$Root = Split-Path -Parent $PSScriptRoot
$RuntimeDir = Join-Path $Root ".runtime"
$FrontendPid = Join-Path $RuntimeDir "frontend.pid"
$BackendPid = Join-Path $RuntimeDir "backend.pid"
$BackendWatchPid = Join-Path $RuntimeDir "backend-watch.pid"
$RedisPid = Join-Path $RuntimeDir "redis.pid"
$MinioPid = Join-Path $RuntimeDir "minio.pid"
$RabbitPid = Join-Path $RuntimeDir "rabbitmq.pid"
$PostgresData = Join-Path $RuntimeDir "postgres-data"
$RabbitBase = Join-Path $RuntimeDir "rabbitmq"

function Stop-PidFile {
  param([string]$Path, [string]$Name)
  if (-not (Test-Path $Path)) {
    Write-Host "$Name PID file not found; skipping."
    return
  }

  $pidValue = (Get-Content -Raw -LiteralPath $Path).Trim()
  if ($pidValue -match '^\d+$') {
    $process = Get-Process -Id ([int]$pidValue) -ErrorAction SilentlyContinue
    if ($null -ne $process) {
      Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
      Write-Host "Stopped $Name process $pidValue."
    } else {
      Write-Host "$Name process $pidValue is not running."
    }
  }
  Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
}

Stop-PidFile -Path $FrontendPid -Name "frontend"
Stop-PidFile -Path $BackendWatchPid -Name "backend watcher"
Stop-PidFile -Path $BackendPid -Name "backend"
Stop-PidFile -Path $RedisPid -Name "redis"
Stop-PidFile -Path $MinioPid -Name "minio"

if (Test-Path $RabbitBase) {
  $scoop = Get-Command scoop -ErrorAction SilentlyContinue
  $rabbitCtl = Get-Command rabbitmqctl.cmd -ErrorAction SilentlyContinue
  if ($null -ne $scoop -and $null -ne $rabbitCtl) {
    $erlangHome = (& $scoop.Source prefix erlang 2>$null | Select-Object -First 1)
    if (-not [string]::IsNullOrWhiteSpace($erlangHome)) {
      $oldErlangHome = $env:ERLANG_HOME
      $oldRabbitBase = $env:RABBITMQ_BASE
      $env:ERLANG_HOME = $erlangHome.Trim()
      $env:RABBITMQ_BASE = $RabbitBase
      & $rabbitCtl.Source stop *> $null
      if ($LASTEXITCODE -eq 0) {
        Write-Host "Stopped rabbitmq."
      } else {
        Write-Host "RabbitMQ was not running for this project."
      }
      $env:ERLANG_HOME = $oldErlangHome
      $env:RABBITMQ_BASE = $oldRabbitBase
    }
  }
}
Remove-Item -LiteralPath $RabbitPid -Force -ErrorAction SilentlyContinue

if (Test-Path (Join-Path $PostgresData "PG_VERSION")) {
  $pgCtl = Get-Command pg_ctl.exe -ErrorAction SilentlyContinue
  if ($null -ne $pgCtl) {
    & $pgCtl.Source -D $PostgresData stop -m fast *> $null
    if ($LASTEXITCODE -eq 0) {
      Write-Host "Stopped postgresql."
    } else {
      Write-Host "PostgreSQL was not running for this project."
    }
  }
}

Write-Host "Local Kingsway dev processes stopped where possible."
