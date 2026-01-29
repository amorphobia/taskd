# Test Fixtures

This directory contains standalone programs used as test subjects for TaskD.

## Programs

- **callback-test.go**: Tests task exit callbacks (uses `//go:build ignore`)
- **debug-task.go**: General debugging utility (uses `//go:build ignore`)
- **long-running/**: Program that runs for extended periods
- **quick-exit/**: Program that exits quickly

Note: The `.go` files in this directory use `//go:build ignore` to prevent them from being compiled during normal `go test` runs, since they contain `main` functions and are meant to be built as standalone executables.

## Building

Use the Makefile to build all fixtures at once:

```bash
# Build all test fixtures
make test-fixtures

# Clean test fixtures
make clean-fixtures
```

Or build individually:

```bash
# Build individual fixtures
go build -o test/fixtures/bin/callback-test.exe test/fixtures/callback-test.go
go build -o test/fixtures/bin/debug-task.exe test/fixtures/debug-task.go
go build -o test/fixtures/bin/long-running.exe test/fixtures/long-running/main.go
go build -o test/fixtures/bin/quick-exit.exe test/fixtures/quick-exit/main.go
```

## Usage

These programs are designed to be managed by TaskD for testing various scenarios:
- Process lifecycle management
- Output redirection
- Callback functionality
- Long-running vs short-lived tasks