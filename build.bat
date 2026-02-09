@echo off
REM TaskD Build Script for Windows

setlocal EnableDelayedExpansion

REM Variable definitions
set BINARY_NAME=taskd
set BUILD_DIR=build
set MAIN_PATH=cmd/taskd/main.go

REM Check if command line argument is provided
if "%1"=="" (
    call :build
    goto :eof
)

REM Route to appropriate function based on argument
if /i "%1"=="build" call :build
if /i "%1"=="dev" call :dev
if /i "%1"=="clean" call :clean
if /i "%1"=="test" call :test
if /i "%1"=="test-fixtures" call :test-fixtures
if /i "%1"=="clean-fixtures" call :clean-fixtures
if /i "%1"=="fmt" call :fmt
if /i "%1"=="lint" call :lint
if /i "%1"=="deps" call :deps
if /i "%1"=="build-all" call :build-all
if /i "%1"=="run" call :run
if /i "%1"=="help" call :help
if /i "%1"=="all" call :build

goto :eof

:build
echo Building %BINARY_NAME%...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
set /p VERSION=<VERSION
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%BINARY_NAME%.exe %MAIN_PATH%
goto :eof

:dev
echo Development build...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
go build -race -o %BUILD_DIR%\%BINARY_NAME%-dev.exe %MAIN_PATH%
goto :eof

:clean
echo Cleaning build files...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist test\fixtures\bin rmdir /s /q test\fixtures\bin
go clean
goto :eof

:test
echo Running tests...
go test -v ./...
goto :eof

:test-fixtures
echo Building test fixtures...
if not exist test\fixtures\bin mkdir test\fixtures\bin
go build -o test\fixtures\bin\callback-test.exe test\fixtures\callback-test.go
go build -o test\fixtures\bin\debug-task.exe test\fixtures\debug-task.go
go build -o test\fixtures\bin\long-running.exe test\fixtures\long-running\main.go
go build -o test\fixtures\bin\quick-exit.exe test\fixtures\quick-exit\main.go
goto :eof

:clean-fixtures
echo Cleaning test fixtures...
if exist test\fixtures\bin rmdir /s /q test\fixtures\bin
goto :eof

:fmt
echo Formatting code...
go fmt ./...
goto :eof

:lint
echo Linting code...
golangci-lint run
goto :eof

:deps
echo Installing dependencies...
go mod tidy
go mod download
goto :eof

:build-all
echo Cross compiling...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
set /p VERSION=<VERSION
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%BINARY_NAME%-windows-amd64.exe %MAIN_PATH%
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%BINARY_NAME%-linux-amd64 %MAIN_PATH%
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%BINARY_NAME%-darwin-amd64 %MAIN_PATH%
goto :eof

:run
go run %MAIN_PATH%
goto :eof

:help
echo Available commands:
echo   build         - Build executable
echo   dev           - Development build
echo   clean         - Clean build files
echo   test          - Run tests
echo   test-fixtures - Build test fixture programs
echo   clean-fixtures- Clean test fixtures
echo   fmt           - Format code
echo   lint          - Lint code
echo   deps          - Install dependencies
echo   build-all     - Cross compile
echo   run           - Run program
echo   help          - Show this help
goto :eof