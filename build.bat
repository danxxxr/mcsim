@echo off
setlocal enabledelayedexpansion
REM Application name
set APP_NAME=mcsim
REM Output directory
set DIST=dist

if not exist "%DIST%" mkdir "%DIST%"

call :build windows amd64 .exe
call :build linux   amd64
call :build darwin  amd64
call :build darwin  arm64

echo.
echo ==================================
echo All builds completed successfully!
echo ==================================
exit /b 0

:build
set GOOS=%1
set GOARCH=%2
set EXT=%3
set OUTDIR=%DIST%\%APP_NAME%-%GOOS%-%GOARCH%
set ARCHIVENAME=%DIST%\%APP_NAME%-%GOOS%-%GOARCH%

echo.
echo =========================
echo Building %GOOS% %GOARCH%...
echo =========================

if not exist "!OUTDIR!" mkdir "!OUTDIR!"

set GOOS=!GOOS!
set GOARCH=!GOARCH!
go build -ldflags="-s -w" -o "!OUTDIR!\%APP_NAME%!EXT!"

if errorlevel 1 (
    echo.
    echo ==================================
    echo Build failed for %GOOS% %GOARCH%
    echo ==================================
    exit /b 1
)

REM --- Archiving ---
if "%GOOS%"=="windows" (
    echo Packing !ARCHIVENAME!.zip ...
    powershell -NoProfile -Command ^
        "Compress-Archive -Path '!OUTDIR!\*' -DestinationPath '!ARCHIVENAME!.zip' -Force"
    if errorlevel 1 (
        echo Archive failed for %GOOS% %GOARCH%
        exit /b 1
    )
    echo Created: !ARCHIVENAME!.zip
) else (
    echo Packing !ARCHIVENAME!.tar.gz ...
    tar -czf "!ARCHIVENAME!.tar.gz" -C "%DIST%" "%APP_NAME%-%GOOS%-%GOARCH%"
    if errorlevel 1 (
        echo Archive failed for %GOOS% %GOARCH%
        exit /b 1
    )
    echo Created: !ARCHIVENAME!.tar.gz
)

exit /b 0