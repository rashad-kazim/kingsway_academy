param(
  [string]$MigrationsDir = ""
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($MigrationsDir)) {
  $MigrationsDir = Join-Path $Root "backend\migrations"
}

if (-not (Test-Path $MigrationsDir)) {
  throw "Migration directory was not found: $MigrationsDir"
}

$files = @(Get-ChildItem -Path $MigrationsDir -File -Filter "*.sql" | Sort-Object Name)
if ($files.Count -eq 0) {
  throw "No SQL migration files found in $MigrationsDir"
}

$seenVersions = New-Object 'System.Collections.Generic.HashSet[string]'
$previousVersion = ""

foreach ($file in $files) {
  if ($file.Name -notmatch '^(\d{12})_[a-z0-9_]+\.sql$') {
    throw "Invalid migration filename '$($file.Name)'. Use YYYYMMDDNNNN_snake_case.sql."
  }

  $version = $Matches[1]
  if (-not $seenVersions.Add($version)) {
    throw "Duplicate migration version '$version'."
  }
  if ($previousVersion -ne "" -and [string]::CompareOrdinal($previousVersion, $version) -ge 0) {
    throw "Migration versions must be strictly increasing: $previousVersion then $version."
  }
  $previousVersion = $version

  $body = Get-Content -Raw -LiteralPath $file.FullName
  $upIndex = $body.IndexOf("-- +goose Up", [System.StringComparison]::Ordinal)
  $downIndex = $body.IndexOf("-- +goose Down", [System.StringComparison]::Ordinal)
  if ($upIndex -lt 0) {
    throw "$($file.Name) is missing -- +goose Up marker."
  }
  if ($downIndex -lt 0) {
    throw "$($file.Name) is missing -- +goose Down marker."
  }
  if ($downIndex -le $upIndex) {
    throw "$($file.Name) has -- +goose Down before -- +goose Up."
  }

  $upSQL = $body.Substring($upIndex + "-- +goose Up".Length, $downIndex - ($upIndex + "-- +goose Up".Length)).Trim()
  if ([string]::IsNullOrWhiteSpace($upSQL)) {
    throw "$($file.Name) has an empty Up migration."
  }
}

$snapshotPath = Join-Path $Root "backend\docs\schema-snapshot-2026-05-13.md"
if (Test-Path $snapshotPath) {
  $snapshot = Get-Content -Raw -LiteralPath $snapshotPath
  if ($snapshot -notmatch [regex]::Escape($previousVersion)) {
    throw "Schema snapshot does not mention latest migration version $previousVersion. Update $snapshotPath."
  }
}

Write-Host "DB migration check passed. Files: $($files.Count). Latest version: $previousVersion."
