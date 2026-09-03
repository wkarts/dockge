@echo off
setlocal
chcp 65001 >nul

fltmc >nul 2>&1
if errorlevel 1 (
  echo Solicitando privilegios de Administrador...
  powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
  exit /b
)

powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0install-service.ps1"
set EXITCODE=%ERRORLEVEL%
if not "%EXITCODE%"=="0" (
  echo.
  echo O instalador terminou com codigo %EXITCODE%.
  pause
)
exit /b %EXITCODE%
