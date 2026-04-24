@echo off
setlocal
title Pi Mono
cd /d "%~dp0"

echo [Pi Mono] Starting...
where go >nul 2>nul
if errorlevel 1 (
    echo [Pi Mono] go not found. Please install it.
    pause
    exit /b 1
)

go run ./cmd/pi

if errorlevel 1 (
    echo [Pi Mono] Exited with error code %errorlevel%.
    pause
)
endlocal
