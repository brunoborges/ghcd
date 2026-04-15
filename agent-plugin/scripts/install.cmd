@echo off
rem Install ghx and ghxd binaries into the given directory.
rem Usage: install.cmd [INSTALL_DIR]
setlocal enabledelayedexpansion

set "REPO=brunoborges/ghx"

if not "%~1"=="" (
    set "INSTALL_DIR=%~1"
) else if defined CLAUDE_PLUGIN_DATA (
    set "INSTALL_DIR=%CLAUDE_PLUGIN_DATA%\bin"
) else (
    set "INSTALL_DIR=%USERPROFILE%\.ghx-plugin\bin"
)

rem Skip if already installed
if exist "%INSTALL_DIR%\ghx.exe" if exist "%INSTALL_DIR%\ghxd.exe" exit /b 0

rem Detect architecture
set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

rem Determine version to install
if defined GHCD_VERSION (
    set "VERSION=%GHCD_VERSION%"
) else (
    set "VERSION="
    for /f "tokens=*" %%i in ('powershell -NoProfile -Command "(Invoke-RestMethod -Uri 'https://api.github.com/repos/%REPO%/releases' -UseBasicParsing) | Where-Object { $_.tag_name -notmatch '^plugin-' } | Select-Object -First 1 -ExpandProperty tag_name"') do set "VERSION=%%i"
)

if "%VERSION%"=="" (
    echo ghxd-install: could not determine version to install 1>&2
    exit /b 1
)

rem Download and extract
set "ZIPNAME=ghx-windows-%ARCH%.zip"
set "URL=https://github.com/%REPO%/releases/download/%VERSION%/%ZIPNAME%"

set "TMPDIR=%TEMP%\ghx-install-%RANDOM%"
mkdir "%TMPDIR%" 2>nul

echo ghxd-install: downloading %ZIPNAME% (%VERSION%)... 1>&2
powershell -NoProfile -Command "Invoke-WebRequest -Uri '%URL%' -OutFile '%TMPDIR%\%ZIPNAME%' -UseBasicParsing"
if %ERRORLEVEL% neq 0 (
    echo ghxd-install: download failed 1>&2
    rmdir /s /q "%TMPDIR%" 2>nul
    exit /b 1
)

powershell -NoProfile -Command "Expand-Archive -Path '%TMPDIR%\%ZIPNAME%' -DestinationPath '%TMPDIR%' -Force"

rem Install
mkdir "%INSTALL_DIR%" 2>nul
copy /y "%TMPDIR%\ghx.exe" "%INSTALL_DIR%\" >nul
copy /y "%TMPDIR%\ghxd.exe" "%INSTALL_DIR%\" >nul

rem Record installed version
echo %VERSION%> "%INSTALL_DIR%\.ghx-version"

echo ghxd-install: installed ghx and ghxd %VERSION% to %INSTALL_DIR% 1>&2

rem Cleanup
rmdir /s /q "%TMPDIR%" 2>nul
exit /b 0
