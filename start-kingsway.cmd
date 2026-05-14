@echo off
setlocal

cd /d "%~dp0"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\start-local-dev.ps1"
if errorlevel 1 (
  echo.
  echo Local start failed. Check the message above.
  pause
  exit /b 1
)

echo.
pause
