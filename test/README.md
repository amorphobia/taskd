# Test Directory

This directory contains test fixtures, integration tests, and test results for the TaskD project.

## Directory Structure

```
test/
├── fixtures/           # Test fixture programs and utilities
│   ├── callback-test.go    # Program to test task callback functionality
│   ├── debug-task.go       # Debug utility for task testing
│   ├── long-running/       # Long-running test program
│   │   └── main.go
│   └── quick-exit/         # Quick-exit test program
│       └── main.go
└── results/            # Test results and reports
    └── TEST_RESULTS.md     # Manual test results documentation
```

## Test Fixtures

The `fixtures/` directory contains standalone Go programs that serve as test subjects for TaskD:

- **callback-test.go**: Tests the task exit callback functionality
- **debug-task.go**: General debugging utility for task behavior
- **long-running/**: A program that runs for an extended period, useful for testing task lifecycle management
- **quick-exit/**: A program that exits quickly, useful for testing rapid task completion

## Usage

### Building Test Fixtures

To build the test fixture programs:

```bash
# Use Makefile (recommended)
make test-fixtures

# Or build individually
go build -o test/fixtures/bin/callback-test.exe test/fixtures/callback-test.go
go build -o test/fixtures/bin/debug-task.exe test/fixtures/debug-task.go
go build -o test/fixtures/bin/long-running.exe test/fixtures/long-running/main.go
go build -o test/fixtures/bin/quick-exit.exe test/fixtures/quick-exit/main.go
```

Note: Some fixture files use `//go:build ignore` to prevent compilation during `go test` runs.

### Using with TaskD

These fixtures can be used as tasks in TaskD for testing:

```bash
# Add a long-running task
taskd add long-test --exec "test/fixtures/long-running.exe" --workdir "."

# Add a quick-exit task
taskd add quick-test --exec "test/fixtures/quick-exit.exe" --workdir "."

# Test with output redirection
taskd add debug-test --exec "go run test/fixtures/debug-task.go" --stdout "debug.log"
```

## Integration Tests

For automated integration tests, see the unit tests in the respective package directories:
- `internal/task/*_test.go` - Unit tests for task management
- `internal/cli/*_test.go` - Unit tests for CLI commands

## Test Results

The `results/` directory contains:
- Manual test results and reports
- Performance benchmarks
- Integration test outputs

## Notes

- Test fixtures are separate from Go's standard `*_test.go` unit tests
- These programs provide real executables for testing TaskD's process management
- Compiled executables (`.exe` files) are not committed to version control