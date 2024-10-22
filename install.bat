@echo off
setlocal enabledelayedexpansion

echo CodeSnap Installer
echo ================
echo.

REM Check if Python is installed
python --version > nul 2>&1
if errorlevel 1 (
    echo Python is not installed or not in PATH
    echo Please install Python from https://www.python.org/downloads/
    echo Ensure you check "Add Python to PATH" during installation
    pause
    exit /b 1
)

REM Create installation directory
set "INSTALL_DIR=%USERPROFILE%\CodeSnap"
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

REM Create the Python script
echo Creating CodeSnap script...
set "SCRIPT_PATH=%INSTALL_DIR%\codesnap.py"
copy /y "%~dp0codesnap.py" "%SCRIPT_PATH%" > nul

REM Create launcher batch script
echo Creating launcher...
set "LAUNCHER_PATH=%INSTALL_DIR%\codesnap.bat"
(
    echo @echo off
    echo python "%SCRIPT_PATH%" %%*
) > "%LAUNCHER_PATH%"

REM Install required packages
echo Installing required packages...
python -m pip install pyperclip pyyaml

REM Add to PATH if not already there
echo Updating PATH...
set "PATH_TO_ADD=%INSTALL_DIR%"
for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "CURRENT_PATH=%%b"
if not defined CURRENT_PATH set "CURRENT_PATH="
echo !CURRENT_PATH! | find /i "%PATH_TO_ADD%" > nul
if errorlevel 1 (
    setx PATH "%PATH_TO_ADD%;%CURRENT_PATH%"
    echo Added to PATH successfully
) else (
    echo Already in PATH
)

echo.
echo Installation completed successfully!
echo You can now use 'codesnap' from any directory.
echo Run 'codesnap' in a directory to create a template configuration file.
echo.
echo Press any key to exit...
pause > nul