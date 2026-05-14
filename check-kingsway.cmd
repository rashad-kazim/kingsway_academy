@echo off
setlocal

cd /d "%~dp0"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\check-kingsway.ps1" %*
exit /b %ERRORLEVEL%
