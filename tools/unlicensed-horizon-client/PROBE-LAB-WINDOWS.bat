@echo off
setlocal
cd /d "%~dp0"
echo.
echo HORIZON INTEGRATION PROBE - Experimental only
 echo Runs with SCRATCH copies of open-content lab files, separate from your saves.
 echo The tested laboratory is not a complete, playable game.
echo.
"%~dp0unlicensed-horizon-client.exe" lab-probe --timeout 20 --frames 2
if errorlevel 1 (
    echo.
    echo Probe setup rejected. Use the launcher to install/select the Horizon engine.
    echo Create the original content lab first with OPEN-LAB-WINDOWS.bat.
)
echo.
echo Reports are under %%APPDATA%%\unlicensed-horizon-client\probe-reports\
pause
