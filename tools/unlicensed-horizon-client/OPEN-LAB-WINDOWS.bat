@echo off
setlocal
cd /d "%~dp0"
echo Creating original experimental resource lab...
"%~dp0unlicensed-horizon-client.exe" lab-init
if errorlevel 1 (
  echo The lab may already exist. It will never be overwritten automatically.
)
echo.
echo Inspect the bundled example: examples\original-resource-lab-v0.6
start "" "%~dp0examples\original-resource-lab-v0.6"
echo.
echo IMPORTANT: This experimental content does not start Horizon yet.
pause
