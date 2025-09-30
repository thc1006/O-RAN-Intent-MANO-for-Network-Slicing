@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

REM Initialize module if needed
if not exist "go.sum" (
    echo Initializing Go module...
    go mod tidy
    if errorlevel 1 (
        echo Failed to initialize Go module
        exit /b 1
    )
)

REM Run tests
echo Running tests...
go test -v -coverprofile=coverage.out ./...

REM Generate coverage report regardless of test result
if exist "coverage.out" (
    echo.
    echo Coverage Summary:
    go tool cover -func=coverage.out | findstr "total"

    REM Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo HTML coverage report: coverage.html
)

REM Check test result
if errorlevel 1 (
    echo.
    echo Some tests failed. Check output above for details.
    exit /b 1
)

echo.
echo All tests passed successfully!