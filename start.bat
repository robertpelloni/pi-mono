@echo off
setlocal enabledelayedexpansion

:: ═══════════════════════════════════════════════════════════════
::  pi-go - AI Coding Agent (Go Port)
::  Build, test, and run script for Windows
:: ═══════════════════════════════════════════════════════════════

:: Change to script directory
cd /d "%~dp0"

:: ─── Color setup ───
for /f %%a in ('echo prompt $E ^| cmd') do set "ESC=%%a"
set "GREEN=!ESC![92m"
set "RED=!ESC![91m"
set "YELLOW=!ESC![93m"
set "CYAN=!ESC![96m"
set "BOLD=!ESC![1m"
set "RESET=!ESC![0m"

:: ─── Parse arguments ───
set "ACTION=run"
set "EXTRA_ARGS="

:parse_args
if "%~1"=="" goto :done_parse
if /i "%~1"=="build" set "ACTION=build" & shift & goto :parse_args
if /i "%~1"=="test" set "ACTION=test" & shift & goto :parse_args
if /i "%~1"=="clean" set "ACTION=clean" & shift & goto :parse_args
if /i "%~1"=="lint" set "ACTION=lint" & shift & goto :parse_args
if /i "%~1"=="run" set "ACTION=run" & shift & goto :parse_args
if /i "%~1"=="install" set "ACTION=install" & shift & goto :parse_args
if /i "%~1"=="stats" set "ACTION=stats" & shift & goto :parse_args
if /i "%~1"=="help" goto :show_help
if /i "%~1"=="/?" goto :show_help
if /i "%~1"=="--help" goto :show_help
set "EXTRA_ARGS=!EXTRA_ARGS! %~1"
shift
goto :parse_args
:done_parse

:: ─── Check Go installation ───
where go >nul 2>&1
if errorlevel 1 (
    echo !RED!!BOLD!Error: Go is not installed or not on PATH!RESET!
    echo.
    echo Install Go from: https://go.dev/dl/
    echo After installing, restart your terminal.
    exit /b 1
)

for /f "tokens=*" %%v in ('go version') do set "GO_VERSION=%%v"
echo !CYAN!!BOLD!pi-go!RESET! !GO_VERSION!
echo.

:: ─── Execute action ───
if "%ACTION%"=="build" goto :do_build
if "%ACTION%"=="test" goto :do_test
if "%ACTION%"=="clean" goto :do_clean
if "%ACTION%"=="lint" goto :do_lint
if "%ACTION%"=="install" goto :do_install
if "%ACTION%"=="stats" goto :do_stats
if "%ACTION%"=="run" goto :do_run
goto :show_help

:do_build
echo !GREEN!!BOLD!Building pi-go...!RESET!
echo.

echo [1/3] Downloading dependencies...
go mod download
if errorlevel 1 (
    echo !RED!Failed to download dependencies!RESET!
    exit /b 1
)

echo [2/3] Compiling...
set "GIT_VERSION=dev"
for /f "tokens=*" %%v in ('git describe --tags --always --dirty 2^>nul') do set "GIT_VERSION=%%v"
go build -ldflags="-s -w" -o pi-go.exe ./cmd/pi/main.go
if errorlevel 1 (
    echo !RED!Build failed!RESET!
    exit /b 1
)

echo [3/3] Verifying...
for %%f in (pi-go.exe) do set "BINARY_SIZE=%%~zf"
echo.
echo !GREEN!!BOLD!Build successful!RESET!
echo   Binary:     pi-go.exe
echo   Size:       !BINARY_SIZE! bytes
echo   Version:    !GIT_VERSION!
echo.
goto :end

:do_test
echo !GREEN!!BOLD!Running tests...!RESET!
echo.

go test -v -count=1 -timeout 120s ./pkg/...
if errorlevel 1 (
    echo.
    echo !RED!!BOLD!Tests failed!RESET!
    exit /b 1
)

echo.
echo !GREEN!!BOLD!All tests passed!RESET!
echo.
goto :end

:do_clean
echo !GREEN!!BOLD!Cleaning build artifacts...!RESET!
echo.

if exist pi-go.exe (
    del pi-go.exe
    echo   Removed pi-go.exe
)

go clean -cache -testcache 2>nul
echo   Cleaned Go build cache
echo.
echo !GREEN!Clean complete!RESET!
echo.
goto :end

:do_lint
echo !GREEN!!BOLD!Running linter...!RESET!
echo.

go vet ./pkg/... ./cmd/...
if errorlevel 1 (
    echo !RED!go vet found issues!RESET!
    exit /b 1
)

echo   go vet: OK
echo.
echo Checking formatting...
gofmt -l ./pkg/ ./cmd/ 2>nul
if errorlevel 1 (
    echo !YELLOW!Some files need formatting. Run: gofmt -w ./pkg/ ./cmd/!RESET!
) else (
    echo   All files formatted correctly.
)

echo.
echo !GREEN!Lint complete!RESET!
echo.
goto :end

:do_install
echo !GREEN!!BOLD!Installing pi-go...!RESET!
echo.

go install ./cmd/pi/main.go
if errorlevel 1 (
    echo !RED!Install failed!RESET!
    exit /b 1
)

echo.
echo !GREEN!!BOLD!Installed successfully!RESET!
echo   Run with: pi-go
echo.
goto :end

:do_stats
echo !GREEN!!BOLD!Codebase Statistics!RESET!
echo.
echo   Run: go test ./pkg/... for test results
echo   Run: go list ./pkg/... for package list
echo.
goto :end

:do_run
if not exist pi-go.exe (
    echo !YELLOW!Binary not found, building first...!RESET!
    echo.
    call :do_build
    if errorlevel 1 exit /b 1
    echo.
)

echo !GREEN!!BOLD!Starting pi-go...!RESET!
echo.
echo ┌──────────────────────────────────────────────────────────┐
echo │  pi-go - AI Coding Agent (Go Port)                      │
echo │                                                          │
echo │  Keys:                                                   │
echo │    Enter       Send message                              │
echo │    Ctrl+C      Abort current operation                   │
echo │    Ctrl+D      Exit                                      │
echo │    /help       Show slash commands                       │
echo │    /model      Switch model                              │
echo │    /compact    Compact context                           │
echo │    /new        New session                               │
echo │                                                          │
echo │  CLI flags:                                              │
echo │    --model       Set model (e.g. gpt-4o)                │
echo │    --provider    Set provider (openai/anthropic/google)  │
echo │    --api-key     Set API key                             │
echo │    --thinking    Set thinking level (low/medium/high)    │
echo │    --offline     Run offline                             │
echo │    --continue    Continue last session                   │
echo │    --message     Single message mode                     │
echo │    --list-models="" List available models                │
echo └──────────────────────────────────────────────────────────┘
echo.

pi-go.exe !EXTRA_ARGS!
set "EXIT_CODE=!errorlevel!"

if !EXIT_CODE! neq 0 (
    echo.
    if !EXIT_CODE! equ 2 (
        echo !YELLOW!Process was interrupted.!RESET!
    ) else (
        echo !RED!pi-go exited with code !EXIT_CODE!!RESET!
    )
)

goto :end

:show_help
echo.
echo !BOLD!pi-go - AI Coding Agent (Go Port)!RESET!
echo.
echo !BOLD!Usage:!RESET!
echo   start.bat [action] [options]
echo.
echo !BOLD!Actions:!RESET!
echo   build     Build the pi-go binary
echo   run       Build (if needed) and run pi-go interactively (default)
echo   test      Run all Go test suites
echo   lint      Run go vet and format checks
echo   clean     Remove build artifacts and caches
echo   install   Install pi-go to GOPATH/bin
echo   stats     Show codebase statistics
echo   help      Show this help message
echo.
echo !BOLD!Run Options (after action):!RESET!
echo   Any flags are passed to pi-go when using 'run':
echo     start.bat run --model gpt-4o
echo     start.bat run --provider anthropic --api-key sk-xxx
echo     start.bat run --offline
echo     start.bat run --continue
echo     start.bat run --message "explain this codebase"
echo     start.bat run --list-models=""
echo.
echo !BOLD!Environment Variables:!RESET!
echo   OPENAI_API_KEY       OpenAI API key
echo   ANTHROPIC_API_KEY    Anthropic API key
echo   GEMINI_API_KEY       Google Gemini API key
echo   PI_OFFLINE           Set to "1" for offline mode
echo.
echo !BOLD!Examples:!RESET!
echo   start.bat                          Build and run interactively
echo   start.bat build                    Build only
echo   start.bat test                     Run tests
echo   start.bat run --model gpt-4o      Run with specific model
echo   start.bat run --offline            Run in offline mode
echo   start.bat run --list-models=""     List available models
echo.
goto :end

:end
endlocal
