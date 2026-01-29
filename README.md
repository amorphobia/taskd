# TaskD - Task Daemon Management Tool

TaskD is a task management tool designed for Windows non-administrator users, providing unified management and monitoring of user-level background processes, similar to Windows services but without requiring administrator privileges.

## Key Features

- ✅ Specify executable file path and arguments
- ✅ Specify working directory
- ✅ Environment variable management (inherit or override)
- ✅ Standard input redirection
- ✅ Standard output and error redirection with relative path support
- ✅ Automatic file creation in append mode
- ✅ Output merging when stdout and stderr point to the same file
- ✅ TOML configuration files
- ✅ Command-line task management
- ✅ Cross-platform support (Go language)

## Quick Start

```bash
# Add a task with output redirection
taskd add mytask --exec "python app.py" --workdir "/path/to/app" --stdout "logs/output.log" --stderr "logs/error.log"

# Start task
taskd start mytask

# View detailed task information (replaces status command)
taskd info mytask

# List all tasks
taskd list

# Stop task
taskd stop mytask
```

## Output Redirection

TaskD supports comprehensive output redirection:

- **Relative paths**: Resolved based on the task's working directory
- **Append mode**: All output files are opened in append mode
- **Output merging**: When stdout and stderr point to the same file, TaskD automatically handles merging
- **Optional output**: If no output files are specified, output is discarded

### Examples

```bash
# Redirect to relative paths (resolved from working directory)
taskd add mytask --exec "python app.py" --workdir "/app" --stdout "logs/out.log" --stderr "logs/err.log"

# Redirect to absolute paths
taskd add mytask --exec "python app.py" --stdout "/var/log/app.log" --stderr "/var/log/app.log"

# Redirect only stdout, discard stderr
taskd add mytask --exec "python app.py" --stdout "output.log"
```

## Configuration

### TaskD Home Directory

TaskD stores its configuration files, task definitions, and runtime state in a dedicated directory. You can customize this location using the `TASKD_HOME` environment variable:

```bash
# Use custom directory
export TASKD_HOME="/path/to/my/taskd"
taskd add mytask --exec "python app.py"

# Use default directory (~/.taskd)
unset TASKD_HOME
taskd add mytask --exec "python app.py"
```

**Default locations:**
- **Linux/macOS**: `~/.taskd/`
- **Windows**: `%USERPROFILE%\.taskd\`

**Directory structure:**
```
$TASKD_HOME/
├── config.toml          # Global configuration
├── tasks/               # Task configuration files
│   ├── mytask.toml
│   └── anothertask.toml
└── runtime.json         # Runtime state
```

### Environment Variable Priority

1. `TASKD_HOME` environment variable (if set)
2. Default: `$HOME/.taskd` (Linux/macOS) or `%USERPROFILE%\.taskd` (Windows)

If the specified `TASKD_HOME` directory doesn't exist, TaskD will create it automatically.

## Technology Stack

- **Language**: Go 1.21+
- **Configuration**: TOML
- **Cross-platform**: Windows/Linux/macOS