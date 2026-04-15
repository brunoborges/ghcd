@echo off
rem ghx-shim: this script redirects gh commands through ghx for caching
rem Routes all gh commands through ghx for caching. Falls back to system gh.

set "SCRIPT_DIR=%~dp0"

rem Delegate to co-located ghx wrapper
if exist "%SCRIPT_DIR%ghx.cmd" (
    "%SCRIPT_DIR%ghx.cmd" %*
    exit /b %ERRORLEVEL%
)

rem Fallback: try ghx on PATH
where ghx >nul 2>&1 && (
    ghx %*
    exit /b %ERRORLEVEL%
)

rem Fallback: try system gh.exe
where gh.exe >nul 2>&1 && (
    gh.exe %*
    exit /b %ERRORLEVEL%
)

echo ghx plugin: no gh binary found 1>&2
exit /b 1
