$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$RuntimeDir = Join-Path $Root ".runtime"
$FrontendDir = Join-Path $Root "frontend"
$BackendDir = Join-Path $Root "backend"
$FrontendLog = Join-Path $RuntimeDir "frontend.out.log"
$FrontendErr = Join-Path $RuntimeDir "frontend.err.log"
$BackendLog = Join-Path $RuntimeDir "backend.out.log"
$BackendErr = Join-Path $RuntimeDir "backend.err.log"
$BackendWatchLog = Join-Path $RuntimeDir "backend-watch.out.log"
$BackendWatchErr = Join-Path $RuntimeDir "backend-watch.err.log"
$PostgresLog = Join-Path $RuntimeDir "postgres.log"
$RedisLog = Join-Path $RuntimeDir "redis.out.log"
$RedisErr = Join-Path $RuntimeDir "redis.err.log"
$RabbitLog = Join-Path $RuntimeDir "rabbitmq.out.log"
$RabbitErr = Join-Path $RuntimeDir "rabbitmq.err.log"
$MinioLog = Join-Path $RuntimeDir "minio.out.log"
$MinioErr = Join-Path $RuntimeDir "minio.err.log"
$FrontendPid = Join-Path $RuntimeDir "frontend.pid"
$BackendPid = Join-Path $RuntimeDir "backend.pid"
$BackendWatchPid = Join-Path $RuntimeDir "backend-watch.pid"
$BackendExe = Join-Path $RuntimeDir "backend-api-server.exe"
$FrontendCache = Join-Path $FrontendDir ".next"
$PostgresData = Join-Path $RuntimeDir "postgres-data"
$RedisData = Join-Path $RuntimeDir "redis-data"
$RabbitBase = Join-Path $RuntimeDir "rabbitmq"
$MinioData = Join-Path $RuntimeDir "minio-data"
$RedisPid = Join-Path $RuntimeDir "redis.pid"
$RabbitPid = Join-Path $RuntimeDir "rabbitmq.pid"
$MinioPid = Join-Path $RuntimeDir "minio.pid"

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
    throw "Backend source changed, but port 8080 is owned by another process ($owner). Stop that process manually, then run start-kingsway.cmd again."
  }

  Write-Host "Restarting backend because source files changed..."
  Stop-Process -Id $owner -Force -ErrorAction SilentlyContinue
  if (-not (Wait-PortClosed 8080 15)) {
    throw "Backend process on 8080 did not stop cleanly."
  }
}

function Stop-FrontendIfOwned {
  param([string]$Reason)

  $owner = Get-PortOwner 3000
  if ($null -eq $owner) {
    return
  }

  $pidFromFile = $null
  if (Test-Path $FrontendPid) {
    $rawPid = (Get-Content -LiteralPath $FrontendPid -ErrorAction SilentlyContinue | Select-Object -First 1)
    if (-not [string]::IsNullOrWhiteSpace($rawPid)) {
      $pidFromFile = [int]$rawPid
    }
  }

  $processInfo = Get-CimInstance Win32_Process -Filter "ProcessId=$owner" -ErrorAction SilentlyContinue
  $commandLine = if ($null -ne $processInfo) { $processInfo.CommandLine } else { "" }
  $frontendNextPath = Join-Path $FrontendDir "node_modules\next"
  $ownedByRuntime = ($null -ne $pidFromFile -and $owner -eq $pidFromFile) -or ($commandLine -like "*$frontendNextPath*")

  if (-not $ownedByRuntime) {
    throw "$Reason, but port 3000 is owned by another process ($owner). Stop that process manually, then run start-kingsway.cmd again."
  }

  Write-Host "Restarting frontend because $Reason..."
  Stop-Process -Id $owner -Force -ErrorAction SilentlyContinue
  if (-not (Wait-PortClosed 3000 15)) {
    throw "Frontend process on 3000 did not stop cleanly."
  }
}

function Test-FrontendReady {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:3000/en/login" -TimeoutSec 15
    return $response.StatusCode -ge 200 -and $response.StatusCode -lt 400
  } catch {
    return $false
  }
}

function Clear-FrontendCache {
  if (Test-Path $FrontendCache) {
    Write-Host "Clearing stale Next.js dev cache..."
    Remove-Item -LiteralPath $FrontendCache -Recurse -Force -ErrorAction SilentlyContinue
  }
}

function Start-BackendWatcher {
  $pidFromFile = $null
  if (Test-Path $BackendWatchPid) {
    $rawPid = (Get-Content -LiteralPath $BackendWatchPid -ErrorAction SilentlyContinue | Select-Object -First 1)
    if (-not [string]::IsNullOrWhiteSpace($rawPid)) {
      $pidFromFile = [int]$rawPid
    }
  }
  if ($null -ne $pidFromFile) {
    $existing = Get-Process -Id $pidFromFile -ErrorAction SilentlyContinue
    if ($null -ne $existing) {
      Write-Host "Backend watcher already running."
      return
    }
  }

  $powershell = Get-Command powershell.exe -ErrorAction SilentlyContinue
  if ($null -eq $powershell) {
    $powershell = Get-Command powershell -ErrorAction SilentlyContinue
  }
  if ($null -eq $powershell) {
    throw "powershell was not found; backend watcher cannot start."
  }

  Remove-Item -LiteralPath $BackendWatchLog, $BackendWatchErr -Force -ErrorAction SilentlyContinue
  $watcher = Start-Process `
    -FilePath $powershell.Source `
    -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", (Join-Path $PSScriptRoot "watch-backend.ps1")) `
    -WorkingDirectory $Root `
    -WindowStyle Hidden `
    -RedirectStandardOutput $BackendWatchLog `
    -RedirectStandardError $BackendWatchErr `
    -PassThru
  Set-Content -LiteralPath $BackendWatchPid -Value $watcher.Id -Encoding ASCII
  Write-Host "Backend watcher running."
}

function Ensure-File {
  param([string]$Path, [string]$Content)
  if (-not (Test-Path $Path)) {
    Set-Content -LiteralPath $Path -Value $Content -Encoding UTF8
  }
}

function Get-RequiredCommand {
  param([string]$Name, [string]$InstallHint)
  $command = Get-Command $Name -ErrorAction SilentlyContinue
  if ($null -eq $command) {
    throw "$Name was not found. $InstallHint"
  }
  return $command
}

function Get-ScoopPrefix {
  param([string]$App)
  $scoop = Get-Command scoop -ErrorAction SilentlyContinue
  if ($null -eq $scoop) {
    return $null
  }
  try {
    $prefix = (& $scoop.Source prefix $App 2>$null | Select-Object -First 1)
    if (-not [string]::IsNullOrWhiteSpace($prefix)) {
      return $prefix.Trim()
    }
  } catch {
    return $null
  }
  return $null
}

function Invoke-Checked {
  param([scriptblock]$Command, [string]$FailureMessage)
  & $Command
  if ($LASTEXITCODE -ne 0) {
    throw $FailureMessage
  }
}

function Invoke-NativeQuiet {
  param([string]$FilePath, [string[]]$Arguments)
  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    & $FilePath @Arguments 1>$null 2>$null
    return $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $oldErrorActionPreference
  }
}

function Start-Postgres {
  if (Test-Port 5432) {
    Write-Host "PostgreSQL already listening on 5432."
    return
  }

  $initdb = Get-RequiredCommand "initdb.exe" "Install PostgreSQL 16, for example: scoop install versions/postgresql16"
  $pgCtl = Get-RequiredCommand "pg_ctl.exe" "Install PostgreSQL 16, for example: scoop install versions/postgresql16"
  $createdb = Get-RequiredCommand "createdb.exe" "Install PostgreSQL 16, for example: scoop install versions/postgresql16"

  New-Item -ItemType Directory -Path $PostgresData -Force | Out-Null
  if (-not (Test-Path (Join-Path $PostgresData "PG_VERSION"))) {
    Write-Host "Initializing local PostgreSQL data directory..."
    Invoke-Checked { & $initdb.Source "-D" $PostgresData "-U" "kingsway" "-A" "trust" "-E" "UTF8" } "PostgreSQL initdb failed."
  }

  Write-Host "Starting local PostgreSQL..."
  Invoke-Checked { & $pgCtl.Source "-D" $PostgresData "-l" $PostgresLog "-o" "-h 127.0.0.1 -p 5432" "start" } "PostgreSQL failed to start."
  if (-not (Wait-Port 5432 45)) {
    throw "PostgreSQL did not start on 127.0.0.1:5432."
  }

  Invoke-NativeQuiet $createdb.Source @("-h", "127.0.0.1", "-p", "5432", "-U", "kingsway", "kingsway") | Out-Null
  Write-Host "PostgreSQL ready on 5432."
}

function Start-Redis {
  if (Test-Port 6379) {
    Write-Host "Redis already listening on 6379."
    return
  }

  $redis = Get-RequiredCommand "redis-server.exe" "Install Redis, for example: scoop install redis"
  New-Item -ItemType Directory -Path $RedisData -Force | Out-Null
  Remove-Item -LiteralPath $RedisLog, $RedisErr -Force -ErrorAction SilentlyContinue

  Write-Host "Starting local Redis..."
  $process = Start-Process `
    -FilePath $redis.Source `
    -ArgumentList @("--bind", "127.0.0.1", "--port", "6379", "--dir", $RedisData, "--appendonly", "yes") `
    -WorkingDirectory $RedisData `
    -WindowStyle Hidden `
    -RedirectStandardOutput $RedisLog `
    -RedirectStandardError $RedisErr `
    -PassThru
  Set-Content -LiteralPath $RedisPid -Value $process.Id -Encoding ASCII
  if (-not (Wait-Port 6379 30)) {
    throw "Redis did not start on 127.0.0.1:6379. Logs: $RedisLog $RedisErr"
  }
  Write-Host "Redis ready on 6379."
}

function Start-Minio {
  if (Test-Port 9000) {
    Write-Host "MinIO already listening on 9000."
    return
  }

  $minio = Get-RequiredCommand "minio.exe" "Install MinIO, for example: scoop install minio"
  New-Item -ItemType Directory -Path $MinioData -Force | Out-Null
  Remove-Item -LiteralPath $MinioLog, $MinioErr -Force -ErrorAction SilentlyContinue

  Write-Host "Starting local MinIO..."
  $oldRootUser = $env:MINIO_ROOT_USER
  $oldRootPassword = $env:MINIO_ROOT_PASSWORD
  $env:MINIO_ROOT_USER = "kingsway"
  $env:MINIO_ROOT_PASSWORD = "kingsway-secret"
  try {
    $process = Start-Process `
      -FilePath $minio.Source `
      -ArgumentList @("server", $MinioData, "--address", "127.0.0.1:9000", "--console-address", "127.0.0.1:9001") `
      -WorkingDirectory $RuntimeDir `
      -WindowStyle Hidden `
      -RedirectStandardOutput $MinioLog `
      -RedirectStandardError $MinioErr `
      -PassThru
  } finally {
    $env:MINIO_ROOT_USER = $oldRootUser
    $env:MINIO_ROOT_PASSWORD = $oldRootPassword
  }
  Set-Content -LiteralPath $MinioPid -Value $process.Id -Encoding ASCII
  if (-not (Wait-Port 9000 30)) {
    throw "MinIO did not start on 127.0.0.1:9000. Logs: $MinioLog $MinioErr"
  }
  Write-Host "MinIO ready on 9000."
}

function Start-RabbitMQ {
  $erlangHome = Get-ScoopPrefix "erlang"
  if ([string]::IsNullOrWhiteSpace($erlangHome)) {
    throw "Erlang was not found. Install it with: scoop install erlang"
  }
  $rabbitServer = Get-RequiredCommand "rabbitmq-server.cmd" "Install RabbitMQ, for example: scoop bucket add extras; scoop install rabbitmq"
  $rabbitCtl = Get-RequiredCommand "rabbitmqctl.cmd" "Install RabbitMQ, for example: scoop bucket add extras; scoop install rabbitmq"

  New-Item -ItemType Directory -Path $RabbitBase -Force | Out-Null

  $oldErlangHome = $env:ERLANG_HOME
  $oldRabbitBase = $env:RABBITMQ_BASE
  $env:ERLANG_HOME = $erlangHome
  $env:RABBITMQ_BASE = $RabbitBase

  try {
    if (-not (Test-Port 5672)) {
      Remove-Item -LiteralPath $RabbitLog, $RabbitErr -Force -ErrorAction SilentlyContinue
      Write-Host "Starting local RabbitMQ..."
      $process = Start-Process `
        -FilePath $rabbitServer.Source `
        -WorkingDirectory $RuntimeDir `
        -WindowStyle Hidden `
        -RedirectStandardOutput $RabbitLog `
        -RedirectStandardError $RabbitErr `
        -PassThru
      Set-Content -LiteralPath $RabbitPid -Value $process.Id -Encoding ASCII
      if (-not (Wait-Port 5672 90)) {
        throw "RabbitMQ did not start on 127.0.0.1:5672. Logs: $RabbitLog $RabbitErr"
      }
    } else {
      Write-Host "RabbitMQ already listening on 5672."
    }

    $rabbitReady = $false
    for ($i = 0; $i -lt 30; $i++) {
      $statusCode = Invoke-NativeQuiet $rabbitCtl.Source @("status")
      if ($statusCode -eq 0) {
        $rabbitReady = $true
        break
      }
      Start-Sleep -Seconds 1
    }
    if (-not $rabbitReady) {
      throw "RabbitMQ opened port 5672, but rabbitmqctl could not connect. Logs: $RabbitLog $RabbitErr"
    }

    Invoke-NativeQuiet $rabbitCtl.Source @("add_user", "kingsway", "kingsway") | Out-Null
    $permissionsCode = Invoke-NativeQuiet $rabbitCtl.Source @("set_permissions", "-p", "/", "kingsway", ".*", ".*", ".*")
    $tagsCode = Invoke-NativeQuiet $rabbitCtl.Source @("set_user_tags", "kingsway", "administrator")
    if ($permissionsCode -ne 0 -or $tagsCode -ne 0) {
      throw "RabbitMQ user setup failed for kingsway."
    }
  } finally {
    $env:ERLANG_HOME = $oldErlangHome
    $env:RABBITMQ_BASE = $oldRabbitBase
  }

  Write-Host "RabbitMQ ready on 5672."
}

function Ensure-DefaultOwner {
  $body = @{
    email = "owner@kingsway.local"
    password = "Kingsway123!"
    first_name = "Kingsway"
    last_name = "Owner"
  } | ConvertTo-Json

  try {
    $response = Invoke-WebRequest `
      -UseBasicParsing `
      -Uri "http://127.0.0.1:8080/v1/auth/bootstrap-owner" `
      -Method Post `
      -ContentType "application/json" `
      -Body $body `
      -TimeoutSec 15
    if ($response.StatusCode -eq 201) {
      Write-Host "Default local owner created: owner@kingsway.local / Kingsway123!"
    }
  } catch {
    $statusCode = $null
    if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
      $statusCode = [int]$_.Exception.Response.StatusCode
    }
    if ($statusCode -eq 409) {
      Write-Host "Default local owner skipped; owner already exists."
      return
    }
    throw "Default local owner bootstrap failed. $($_.Exception.Message)"
  }
}

Ensure-File `
  -Path (Join-Path $FrontendDir ".env.local") `
  -Content "KINGSWAY_API_BASE_URL=http://127.0.0.1:8080`nKINGSWAY_AUTH_COOKIE_SECURE=false`n"

Ensure-File `
  -Path (Join-Path $BackendDir ".env") `
  -Content @"
APP_ENV=local
HTTP_ADDR=127.0.0.1:8080
JWT_SECRET=dev-only-change-me
DATA_STORE=postgres
RUN_MIGRATIONS=true
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_MINUTE=600

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=kingsway
POSTGRES_USER=kingsway
POSTGRES_PASSWORD=kingsway

REDIS_ADDR=localhost:6379
RABBITMQ_URL=amqp://kingsway:kingsway@localhost:5672/

MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=kingsway
MINIO_SECRET_KEY=kingsway-secret
MINIO_BUCKET_STANDARD=kingsway-standard
MINIO_BUCKET_SPECIAL=kingsway-special
MINIO_USE_SSL=false

RETENTION_WORKER_ENABLED=true
RETENTION_WORKER_INTERVAL_SECONDS=3600
RETENTION_WORKER_LIMIT=100

OUTBOX_DISPATCHER_ENABLED=true
OUTBOX_DISPATCHER_INTERVAL_SECONDS=5
OUTBOX_DISPATCHER_BATCH=50
OUTBOX_MAX_ATTEMPTS=10

NOTIFICATION_WORKERS_ENABLED=true
PAYMENT_REMINDER_INTERVAL_SECONDS=21600
PAYMENT_REMINDER_HORIZON_DAYS=7
FILE_RETENTION_NOTICE_INTERVAL_SECONDS=21600
FILE_RETENTION_NOTICE_HORIZON_DAYS=7
"@

$npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
if ($null -eq $npm) {
  $npm = Get-Command npm -ErrorAction SilentlyContinue
}
if ($null -eq $npm) {
  throw "npm was not found. Install Node.js or add npm to PATH."
}

Start-Postgres
Start-Redis
Start-Minio
Start-RabbitMQ

$backendSourceStamp = Get-BackendSourceStamp
$backendExeStamp = if (Test-Path $BackendExe) { (Get-Item -LiteralPath $BackendExe).LastWriteTimeUtc } else { [datetime]::MinValue }
$backendNeedsStart = -not (Test-Port 8080)
$backendNeedsRebuild = $backendSourceStamp -gt $backendExeStamp

if ((Test-Port 8080) -and $backendNeedsRebuild) {
  Stop-BackendIfOwned
  $backendNeedsStart = $true
}

if ($backendNeedsStart) {
  $go = Get-Command go -ErrorAction SilentlyContinue
  if ($null -eq $go) {
    throw "go was not found. Install Go or add it to PATH."
  }

  Write-Host "Building backend..."
  Push-Location $BackendDir
  try {
    Invoke-Checked { & $go.Source "build" "-o" $BackendExe "./cmd/api-server" } "Backend build failed."
  } finally {
    Pop-Location
  }

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
    Write-Host "Backend did not start on 127.0.0.1:8080." -ForegroundColor Red
    Write-Host "Backend logs:"
    Write-Host "  $BackendLog"
    Write-Host "  $BackendErr"
    exit 1
  }
} else {
  $owner = Get-PortOwner 8080
  if ($null -ne $owner) {
    Set-Content -LiteralPath $BackendPid -Value $owner -Encoding ASCII
  }
}

Ensure-DefaultOwner
Start-BackendWatcher

$frontendNeedsStart = -not (Test-Port 3000)
if (-not $frontendNeedsStart -and -not (Test-FrontendReady)) {
  Stop-FrontendIfOwned "the current frontend dev server is unhealthy"
  Clear-FrontendCache
  $frontendNeedsStart = $true
}

if ($frontendNeedsStart) {
  Remove-Item -LiteralPath $FrontendLog, $FrontendErr -Force -ErrorAction SilentlyContinue
  $frontend = Start-Process `
    -FilePath $npm.Source `
    -ArgumentList @("run", "dev", "--", "--hostname", "127.0.0.1", "--port", "3000") `
    -WorkingDirectory $FrontendDir `
    -WindowStyle Hidden `
    -RedirectStandardOutput $FrontendLog `
    -RedirectStandardError $FrontendErr `
    -PassThru
  Set-Content -LiteralPath $FrontendPid -Value $frontend.Id -Encoding ASCII

  if (-not (Wait-Port 3000 45)) {
    Write-Host "Frontend did not start on 127.0.0.1:3000." -ForegroundColor Red
    Write-Host "Frontend logs:"
    Write-Host "  $FrontendLog"
    Write-Host "  $FrontendErr"
    exit 1
  }

  if (-not (Test-FrontendReady)) {
    Write-Host "Frontend started on 127.0.0.1:3000, but /en/login is not healthy." -ForegroundColor Red
    Write-Host "Frontend logs:"
    Write-Host "  $FrontendLog"
    Write-Host "  $FrontendErr"
    exit 1
  }
} else {
  $owner = Get-PortOwner 3000
  if ($null -ne $owner) {
    Set-Content -LiteralPath $FrontendPid -Value $owner -Encoding ASCII
  }
}

Write-Host "Kingsway local dev is running without Docker:" -ForegroundColor Green
Write-Host "  Frontend: http://127.0.0.1:3000/en/login"
Write-Host "  Backend:  http://127.0.0.1:8080/healthz"
Write-Host ""
Write-Host "Logs:"
Write-Host "  $FrontendLog"
Write-Host "  $FrontendErr"
Write-Host "  $BackendLog"
Write-Host "  $BackendErr"
Write-Host "  $BackendWatchLog"
Write-Host "  $BackendWatchErr"
