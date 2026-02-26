# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TaskD is a task daemon management tool for Windows (non-administrator users), providing unified management and monitoring of user-level background processes. It functions similar to Windows services but without requiring admin privileges.

## Commands

```bash
# Set Chinese Go proxy for faster downloads in China
export GOPROXY="https://goproxy.cn"

# Build the application
make build

# Development build with race detection
make dev

# Run tests
go test -v ./...

# Run a single test file
go test -v ./internal/task/

# Build test fixtures (required for some tests)
make test-fixtures

# Format code
make fmt

# Lint code
make lint

# Cross-compile for all platforms
make build-all
```

## Architecture

### Package Structure

- `cmd/taskd/main.go` - Application entry point; handles both CLI and daemon modes via `--daemon` flag
- `internal/cli/` - Cobra-based CLI commands (add, list, start, stop, del, edit, info)
- `internal/task/` - Core task management logic:
  - `config.go` - Task configuration structures (TOML)
  - `manager.go` - Task CRUD operations and state persistence
  - `task.go` - Individual task lifecycle management
  - `daemon.go` - TaskMonitor for auto-restart, ProcessChecker for health checks, DaemonManager for daemon lifecycle
- `internal/config/` - Global configuration using Viper

### Key Concepts

1. **Daemon Mode**: When run with `--daemon` flag, the application starts a background monitor that:
   - Periodically checks task process status
   - Auto-restarts tasks marked with `auto_start = true`
   - Tracks retry counts and respects `max_retry_num`

2. **Task Configuration**: Stored as TOML files in `$TASKD_HOME/tasks/` with fields for executable, args, workdir, env, stdin/stdout/stderr redirection, and auto-start settings.

3. **Runtime State**: Persisted in `$TASKD_HOME/runtime.json` tracking running tasks, PIDs, status, exit codes, and retry counts.

4. **Output Redirection**: Supports relative paths (resolved from task's workdir), append mode, and automatic stdout/stderr merging when pointing to the same file.

5. **Builtin Tasks**: Pre-defined tasks in `builtin.go` that are always available (similar to built-in shell commands).

### Configuration Location

- `TASKD_HOME` environment variable (if set)
- Default: `~/.taskd` (Linux/macOS) or `%USERPROFILE%\.taskd` (Windows)
