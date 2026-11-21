@echo off
setlocal
cd /d "%~dp0"

set "EXE=%~dp0game-performance-advisor.exe"
if not exist "%EXE%" (
    echo Building Game Performance Advisor...
    go build -o "%EXE%" ./cmd/game-performance-advisor
    if errorlevel 1 (
        echo.
        echo Build failed. Install Go 1.26 or newer and try again.
        pause
        exit /b 1
    )
)

:menu
cls
echo ==================================================
echo          Game Performance Advisor
echo ==================================================
echo.
echo 1. Scan this Windows system
echo 2. Monitor for 15 seconds
echo 3. Analyze an imported CSV or JSON file
echo 4. Exit
echo.
choice /C 1234 /N /M "Choose an option:"
if %errorlevel%==4 goto end
if %errorlevel%==3 goto import
if %errorlevel%==2 goto monitor
goto scan

:scan
"%EXE%" scan --html scan.html --report scan.md
echo.
pause
goto menu

:monitor
"%EXE%" monitor --duration 15 --interval 2 --html monitor.html --report monitor.md
echo.
pause
goto menu

:import
echo.
set /p "INPUT_FILE=Enter CSV or JSON path: "
if not defined INPUT_FILE goto menu
"%EXE%" analyze "%INPUT_FILE%" --target-fps 60 --report frame-analysis.md
echo.
pause
goto menu

:end
