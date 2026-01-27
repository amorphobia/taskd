@echo off
REM TaskD Windows Build Script

echo Starting TaskD build...

REM Check Go environment
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Go environment not found, please install Go 1.21+
    exit /b 1
)

REM Create build directory
if not exist build mkdir build

REM Download dependencies
echo Downloading dependencies...
go mod tidy
go mod download

REM Build executable
echo Building executable...
go build -o build\taskd.exe cmd\taskd\main.go

if %errorlevel% equ 0 (
    echo Build successful! Executable located at: build\taskd.exe
    echo.
    echo Usage:
    echo   build\taskd.exe --help
) else (
    echo Build failed!
    exit /b 1
)

pause