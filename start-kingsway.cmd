@echo off
setlocal

cd /d "%~dp0"

where docker >nul 2>nul
if errorlevel 1 (
  echo Docker bulunamadi. Docker Desktop'i kurup calistirdiktan sonra tekrar deneyin.
  pause
  exit /b 1
)

echo Kingsway full stack baslatiliyor...
docker compose up -d --build
if errorlevel 1 (
  echo.
  echo Baslatma basarisiz oldu. Detay icin: docker compose logs
  pause
  exit /b 1
)

echo.
echo Kingsway calisiyor:
echo   Frontend: http://127.0.0.1:3000/en/login
echo   API:      http://127.0.0.1:8080/healthz
echo.
pause
