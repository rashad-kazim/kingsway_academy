@echo off
setlocal

cd /d "%~dp0"

where docker >nul 2>nul
if errorlevel 1 (
  echo Docker bulunamadi.
  pause
  exit /b 1
)

echo Kingsway full stack durduruluyor...
docker compose down
if errorlevel 1 (
  echo.
  echo Durdurma basarisiz oldu. Detay icin: docker compose logs
  pause
  exit /b 1
)

echo.
echo Kingsway durduruldu. PostgreSQL/Redis/MinIO verileri silinmedi.
echo Verileri de silmek icin elle calistirin: docker compose down -v
echo.
pause
