package task

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Task task instance
type Task struct {
	config    *Config
	process   *os.Process
	status    string
	startTime time.Time
	exitCode  int
	lastError string
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	onExit    func(taskName string) // Callback when task exits
	taskIO    *TaskIO              // IO manager
}

// NewTask create a new task
func NewTask(config *Config) *Task {
	ctx, cancel := context.WithCancel(context.Background())
	return &Task{
		config: config,
		status: "stopped",
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetExitCallback sets the callback function to be called when task exits
func (t *Task) SetExitCallback(callback func(taskName string)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onExit = callback
}

// Start start the task
func (t *Task) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.status == "running" {
		return fmt.Errorf("task is already running")
	}
	
	// Reset context if it was cancelled
	if t.ctx.Err() != nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}
	
	// Parse executable and arguments
	executable, args := t.parseExecutable()
	
	// Create command
	cmd := exec.CommandContext(t.ctx, executable, args...)
	
	// Set working directory
	// Always set working directory - use config value or default to user home
	workDir := t.config.WorkDir
	if workDir == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			workDir = homeDir
		}
	}
	if workDir != "" {
		cmd.Dir = workDir
	}
	
	// Set environment variables
	if t.config.InheritEnv {
		cmd.Env = os.Environ()
	}
	for _, env := range t.config.Env {
		cmd.Env = append(cmd.Env, env)
	}
	
	// Setup standard input/output
	if err := t.setupIO(cmd); err != nil {
		return fmt.Errorf("failed to setup IO: %w", err)
	}
	
	// Start process
	if err := cmd.Start(); err != nil {
		t.status = "failed"
		t.lastError = err.Error()
		return fmt.Errorf("failed to start process '%s': %w", executable, err)
	}
	
	t.process = cmd.Process
	t.status = "running"
	t.startTime = time.Now()
	t.lastError = ""
	t.exitCode = 0
	
	// Wait for process to exit asynchronously
	go t.waitForExit(cmd)
	
	return nil
}

// parseExecutable parses the executable string into command and arguments
func (t *Task) parseExecutable() (string, []string) {
	// If Args is already set, use Executable as command and Args as arguments
	if len(t.config.Args) > 0 {
		return t.config.Executable, t.config.Args
	}
	
	// Otherwise, parse the Executable string
	// This is a simple implementation - for more complex parsing, we might need a proper shell parser
	parts := strings.Fields(t.config.Executable)
	if len(parts) == 0 {
		return "", nil
	}
	
	if len(parts) == 1 {
		return parts[0], nil
	}
	
	return parts[0], parts[1:]
}

// Stop stop the task
func (t *Task) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.status != "running" {
		return fmt.Errorf("task is not running")
	}
	
	// Cancel context first
	t.cancel()
	
	// Send interrupt signal to process
	if t.process != nil {
		// Try to kill the process
		if err := t.process.Kill(); err != nil {
			// If kill fails, check if process is already dead
			if t.process.Signal(os.Kill) != nil {
				// Process is already dead, update status
				t.status = "stopped"
				t.process = nil
				t.exitCode = 0
				t.lastError = ""
				// Clean up IO resources
				if t.taskIO != nil {
					t.taskIO.Close()
					t.taskIO = nil
				}
				return nil
			}
			return fmt.Errorf("failed to terminate process: %w", err)
		}
		
		// Update status immediately since we've terminated the process
		t.status = "stopped"
		t.process = nil
		t.exitCode = -1 // Indicates forced termination
		t.lastError = "Process terminated by user"
	}
	
	// Clean up IO resources
	if t.taskIO != nil {
		if err := t.taskIO.Close(); err != nil {
			// Log the error but don't fail the stop operation
			fmt.Printf("Warning: failed to close IO resources: %v\n", err)
		}
		t.taskIO = nil
	}
	
	return nil
}


// GetInfo get task information
func (t *Task) GetInfo() *TaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var pid int
	if t.process != nil {
		pid = t.process.Pid
	}
	
	return &TaskInfo{
		Name:       t.config.Name,
		Status:     t.status,
		PID:        pid,
		StartTime:  t.startTime.Format("2006-01-02 15:04:05"),
		Executable: t.config.Executable,
		ExitCode:   t.exitCode,
		LastError:  t.lastError,
	}
}

// IsRunning check if task is currently running
func (t *Task) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status == "running"
}

// GetRuntimeInfo get runtime information for persistence
func (t *Task) GetRuntimeInfo() *TaskRuntimeInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Only persist running tasks
	if t.status != "running" || t.process == nil {
		return nil
	}
	
	return &TaskRuntimeInfo{
		Name:      t.config.Name,
		Status:    t.status,
		PID:       t.process.Pid,
		StartTime: t.startTime,
	}
}

// restoreRuntimeState restore task state from persistence
func (t *Task) restoreRuntimeState(info *TaskRuntimeInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Check if the process is still running
	if info.PID > 0 {
		if process, err := os.FindProcess(info.PID); err == nil {
			// On Windows, FindProcess always succeeds even for non-existent PIDs
			// We need to try to do something with the process to check if it's real
			// For now, we'll assume the process might still be running
			t.status = info.Status
			t.startTime = info.StartTime
			t.process = process
			
			// Start monitoring the process
			go t.monitorExistingProcess(process)
			return
		}
	}
	
	// Process is not running, keep default stopped state
	t.status = "stopped"
}

// monitorExistingProcess monitors an existing process
func (t *Task) monitorExistingProcess(process *os.Process) {
	// Wait for the process to exit
	state, err := process.Wait()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.status = "stopped"
	t.process = nil
	
	if err != nil {
		t.lastError = err.Error()
		t.exitCode = -1
	} else {
		t.exitCode = state.ExitCode()
		t.lastError = ""
	}
	
	// Notify manager to update runtime state when task exits
	if t.onExit != nil {
		t.onExit(t.config.Name)
	}
}

func (t *Task) setupIO(cmd *exec.Cmd) error {
	// Create task IO configuration
	ioManager := GetIOManager()
	
	// Perform runtime validation before creating IO
	if err := t.validateRuntimeIO(); err != nil {
		return fmt.Errorf("runtime IO validation failed: %w", err)
	}
	
	taskIO, err := ioManager.CreateTaskIO(t.config)
	if err != nil {
		return fmt.Errorf("failed to create task IO: %w", err)
	}
	
	// Save IO reference for cleanup
	t.taskIO = taskIO
	
	// Set standard input/output
	if taskIO.Stdin != nil {
		cmd.Stdin = taskIO.Stdin
	}
	if taskIO.Stdout != nil {
		cmd.Stdout = taskIO.Stdout
	}
	if taskIO.Stderr != nil {
		cmd.Stderr = taskIO.Stderr
	}
	
	return nil
}

// validateRuntimeIO performs runtime validation of IO configuration
func (t *Task) validateRuntimeIO() error {
	pathResolver := NewPathResolver()
	
	// Validate stdin file exists at runtime
	if t.config.Stdin != "" {
		stdinPath, err := pathResolver.ResolvePath(t.config.Stdin, t.config.WorkDir)
		if err != nil {
			return fmt.Errorf("failed to resolve stdin path: %w", err)
		}
		
		// Check if file exists
		info, err := os.Stat(stdinPath)
		if err != nil {
			if os.IsNotExist(err) {
				return &FileError{
					Type:      FileErrorNotFound,
					Path:      stdinPath,
					Operation: "validate stdin file",
				}
			}
			return wrapFileError(err, stdinPath, "validate stdin file")
		}
		
		// Check if it's actually a file (not a directory)
		if info.IsDir() {
			return &FileError{
				Type:      FileErrorIsDirectory,
				Path:      stdinPath,
				Operation: "validate stdin file",
			}
		}
		
		// Check read permissions
		if err := ValidateFilePermissions(stdinPath, "read"); err != nil {
			return fmt.Errorf("stdin file permission error: %w", err)
		}
	}
	
	// Validate output directories exist and are writable
	if t.config.Stdout != "" {
		stdoutPath, err := pathResolver.ResolvePath(t.config.Stdout, t.config.WorkDir)
		if err != nil {
			return fmt.Errorf("failed to resolve stdout path: %w", err)
		}
		
		// Ensure parent directory exists and is writable
		if err := pathResolver.EnsureDir(filepath.Dir(stdoutPath)); err != nil {
			return fmt.Errorf("stdout directory validation failed: %w", err)
		}
	}
	
	if t.config.Stderr != "" {
		stderrPath, err := pathResolver.ResolvePath(t.config.Stderr, t.config.WorkDir)
		if err != nil {
			return fmt.Errorf("failed to resolve stderr path: %w", err)
		}
		
		// Skip directory check if stderr is the same as stdout (already checked)
		if t.config.Stdout == "" || stderrPath != t.config.Stdout {
			// Ensure parent directory exists and is writable
			if err := pathResolver.EnsureDir(filepath.Dir(stderrPath)); err != nil {
				return fmt.Errorf("stderr directory validation failed: %w", err)
			}
		}
	}
	
	return nil
}

func (t *Task) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.status = "stopped"
	t.process = nil // Clear the process reference
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.exitCode = exitError.ExitCode()
		} else {
			t.exitCode = -1
		}
		t.lastError = err.Error()
	} else {
		t.exitCode = 0
		t.lastError = ""
	}
	
	// Clean up IO resources
	if t.taskIO != nil {
		if closeErr := t.taskIO.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close IO resources: %v\n", closeErr)
		}
		t.taskIO = nil
	}
	
	// Notify manager to update runtime state when task exits
	// We need a way to callback to the manager to update the runtime state
	if t.onExit != nil {
		t.onExit(t.config.Name)
	}
}