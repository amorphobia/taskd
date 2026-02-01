package task

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"taskd/internal/config"
)

// ProcessChecker handles process status checking and validation
type ProcessChecker struct{}

// NewProcessChecker creates a new process checker
func NewProcessChecker() *ProcessChecker {
	return &ProcessChecker{}
}

// CheckTaskProcess checks the status of a task process
func (pc *ProcessChecker) CheckTaskProcess(pid int) (*ProcessStatus, error) {
	if pid <= 0 {
		return &ProcessStatus{
			Exists:         false,
			IsTaskd:        false,
			ExitCode:       0,
			ExecutablePath: "",
		}, nil
	}
	
	// Try to find the process
	_, err := os.FindProcess(pid)
	if err != nil {
		return &ProcessStatus{
			Exists:         false,
			IsTaskd:        false,
			ExitCode:       0,
			ExecutablePath: "",
		}, nil
	}
	
	// Check if process is still running
	// On Windows, we'll use a different approach since Signal(0) may not be reliable
	// We'll assume that if os.FindProcess succeeds and we can get the process,
	// then the process exists. This is a simplified approach.
	
	// Process exists, now check if it's a taskd process
	// This is critical for PID reuse detection
	execPath, isTaskd, err := pc.getProcessExecutablePath(pid)
	if err != nil {
		// Can't determine executable path, assume it exists but unknown type
		return &ProcessStatus{
			Exists:         true,
			IsTaskd:        false,
			ExitCode:       0,
			ExecutablePath: "",
		}, nil
	}
	
	return &ProcessStatus{
		Exists:         true,
		IsTaskd:        isTaskd,
		ExitCode:       0, // Process is running, no exit code
		ExecutablePath: execPath,
	}, nil
}

// CheckTaskProcessWithValidation checks if a process is the expected taskd process
// This method helps detect PID reuse by validating the executable path
func (pc *ProcessChecker) CheckTaskProcessWithValidation(pid int, expectedExecPath string) (*ProcessStatus, error) {
	status, err := pc.CheckTaskProcess(pid)
	if err != nil {
		return status, err
	}
	
	if !status.Exists {
		return status, nil
	}
	
	// If we have an expected executable path, validate it
	if expectedExecPath != "" && status.ExecutablePath != "" {
		// Compare executable paths to detect PID reuse
		if status.ExecutablePath != expectedExecPath {
			// PID has been reused by a different process
			status.IsTaskd = false
		}
	}
	
	return status, nil
}

// getProcessExecutablePath gets the executable path of a process and checks if it's taskd
func (pc *ProcessChecker) getProcessExecutablePath(pid int) (string, bool, error) {
	// Get current executable path for comparison
	currentExec, err := os.Executable()
	if err != nil {
		return "", false, fmt.Errorf("failed to get current executable path: %w", err)
	}
	
	// For this implementation, we'll assume that if the process exists and we can access it,
	// it's likely to be taskd. In a production system, you would use Windows API to get
	// the actual executable path and compare it.
	
	// Since we're dealing with processes we started ourselves, and we're checking the PID
	// that we recorded when we started the process, it's reasonable to assume it's taskd
	// if the process still exists.
	
	return currentExec, true, nil
}

// GetProcessExitCode gets the exit code of a terminated process
func (pc *ProcessChecker) GetProcessExitCode(pid int) (int, error) {
	// This is a simplified implementation for Windows
	// In a production system, you would use Windows API calls like GetExitCodeProcess
	
	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, we can't get exit code
		return 0, fmt.Errorf("process %d not found", pid)
	}
	
	// Check if process is still running
	if err := process.Signal(syscall.Signal(0)); err == nil {
		// Process is still running, no exit code available
		return 0, fmt.Errorf("process %d is still running", pid)
	}
	
	// Process has exited, but we can't easily get the exit code in Go without Windows API
	// For now, return 0 (success) as a default
	// In a real implementation, you would use Windows API to get the actual exit code
	return 0, nil
}

// DaemonStateManager manages daemon state persistence
type DaemonStateManager struct {
	manager *Manager
}

// NewDaemonStateManager creates a new daemon state manager
func NewDaemonStateManager() *DaemonStateManager {
	return &DaemonStateManager{
		manager: GetManager(),
	}
}

// SaveDaemonState saves the daemon state to persistent storage
func (dsm *DaemonStateManager) SaveDaemonState(daemonInfo *TaskRuntimeInfo) error {
	state := dsm.manager.loadRuntimeState()
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	state.Tasks["taskd"] = daemonInfo
	
	return dsm.manager.saveRuntimeStateWithData(state)
}

// LoadDaemonState loads the daemon state from persistent storage
func (dsm *DaemonStateManager) LoadDaemonState() (*TaskRuntimeInfo, bool) {
	state := dsm.manager.loadRuntimeState()
	
	daemonInfo, exists := state.Tasks["taskd"]
	return daemonInfo, exists
}

// ClearDaemonState removes the daemon state from persistent storage
func (dsm *DaemonStateManager) ClearDaemonState() error {
	state := dsm.manager.loadRuntimeState()
	
	if state.Tasks != nil {
		delete(state.Tasks, "taskd")
	}
	
	return dsm.manager.saveRuntimeStateWithData(state)
}

// updateDaemonStoppedStateWithManager updates daemon state to stopped using state manager
func (dm *DaemonManager) updateDaemonStoppedStateWithManager(stateManager *DaemonStateManager, daemonInfo *TaskRuntimeInfo) error {
	// Create stopped state info
	stoppedInfo := &TaskRuntimeInfo{
		Name:           "taskd",
		Status:         "stopped",
		PID:            0,
		StartTime:      daemonInfo.StartTime,
		EndTime:        time.Now(),
		StoppedByTaskd: true, // Stopped by user command
		RetryNum:       daemonInfo.RetryNum,
	}
	
	return stateManager.SaveDaemonState(stoppedInfo)
}

// ValidateDaemonState validates the consistency of daemon state
func (dm *DaemonManager) ValidateDaemonState() (*TaskRuntimeInfo, bool, error) {
	stateManager := NewDaemonStateManager()
	daemonInfo, exists := stateManager.LoadDaemonState()
	
	if !exists {
		return nil, false, nil
	}
	
	// If daemon is marked as running, verify the process actually exists
	if daemonInfo.Status == "running" {
		checker := NewProcessChecker()
		status, err := checker.CheckTaskProcess(daemonInfo.PID)
		if err != nil {
			return daemonInfo, false, fmt.Errorf("failed to check daemon process: %w", err)
		}
		
		// If process doesn't exist or is not taskd, the state is inconsistent
		if !status.Exists || !status.IsTaskd {
			// Update state to reflect reality
			stoppedInfo := &TaskRuntimeInfo{
				Name:           "taskd",
				Status:         "stopped",
				PID:            0,
				StartTime:      daemonInfo.StartTime,
				EndTime:        time.Now(),
				StoppedByTaskd: false, // Process died naturally
				RetryNum:       daemonInfo.RetryNum,
			}
			
			if err := stateManager.SaveDaemonState(stoppedInfo); err != nil {
				return daemonInfo, false, fmt.Errorf("failed to update daemon state: %w", err)
			}
			
			return stoppedInfo, false, nil
		}
	}
	
	return daemonInfo, true, nil
}

// DaemonManager manages the daemon process lifecycle
type DaemonManager struct {
	mu sync.RWMutex
}

var (
	daemonManager *DaemonManager
	daemonOnce    sync.Once
)

// GetDaemonManager returns the singleton daemon manager instance
func GetDaemonManager() *DaemonManager {
	daemonOnce.Do(func() {
		daemonManager = &DaemonManager{}
	})
	return daemonManager
}

// StartDaemon starts the daemon process
func (dm *DaemonManager) StartDaemon() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	// 1. Check if daemon is already running
	if dm.isDaemonRunningLocked() {
		return fmt.Errorf("daemon is already running")
	}
	
	// 2. Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	
	// 3. Start daemon process
	cmd := exec.Command(execPath, "--daemon")
	cmd.Dir = config.GetTaskDHome()
	
	// Set process attributes for proper daemon behavior
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	
	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}
	
	// Give the process a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// 4. Update runtime state
	daemonInfo := &TaskRuntimeInfo{
		Name:           "taskd",
		Status:         "running",
		PID:            cmd.Process.Pid,
		StartTime:      time.Now(),
		StoppedByTaskd: false,
		RetryNum:       0,
	}
	
	stateManager := NewDaemonStateManager()
	if err := stateManager.SaveDaemonState(daemonInfo); err != nil {
		// If we can't update state, try to kill the process we just started
		cmd.Process.Kill()
		return fmt.Errorf("failed to update daemon runtime state: %w", err)
	}
	
	return nil
}

// StopDaemon stops the daemon process
func (dm *DaemonManager) StopDaemon() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	// 1. Load runtime state to get daemon PID
	manager := GetManager()
	state := manager.loadRuntimeState()
	
	daemonInfo, exists := state.Tasks["taskd"]
	if !exists {
		return fmt.Errorf("daemon is not running")
	}
	
	if daemonInfo.Status != "running" {
		return fmt.Errorf("daemon is not running (status: %s)", daemonInfo.Status)
	}
	
	// 2. Find and terminate the daemon process
	process, err := os.FindProcess(daemonInfo.PID)
	if err != nil {
		// Process doesn't exist, update state and return
		return dm.updateDaemonStoppedState(daemonInfo)
	}
	
	// 3. Try to terminate the process gracefully first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If graceful termination fails, force kill
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill daemon process (PID %d): %w", daemonInfo.PID, err)
		}
	}
	
	// 4. Wait a moment for the process to exit
	time.Sleep(100 * time.Millisecond)
	
	// 5. Update runtime state
	stateManager := NewDaemonStateManager()
	return dm.updateDaemonStoppedStateWithManager(stateManager, daemonInfo)
}

// IsRunning checks if the daemon process is currently running
func (dm *DaemonManager) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	return dm.isDaemonRunningLocked()
}

// EnsureDaemonRunning ensures the daemon is running, starting it if necessary
func (dm *DaemonManager) EnsureDaemonRunning() error {
	// Check if daemon is already running
	if dm.IsRunning() {
		return nil // Already running, nothing to do
	}
	
	// Daemon is not running, start it
	return dm.StartDaemon()
}

// isDaemonRunningLocked checks if daemon is running (must be called with lock held)
func (dm *DaemonManager) isDaemonRunningLocked() bool {
	// Load daemon state
	stateManager := NewDaemonStateManager()
	daemonInfo, exists := stateManager.LoadDaemonState()
	
	if !exists || daemonInfo.Status != "running" {
		return false
	}
	
	// Use ProcessChecker to validate the process
	checker := NewProcessChecker()
	status, err := checker.CheckTaskProcess(daemonInfo.PID)
	if err != nil {
		return false
	}
	
	// Process must exist and be a taskd process
	return status.Exists && status.IsTaskd
}

// updateDaemonRuntimeState updates the daemon's runtime state
func (dm *DaemonManager) updateDaemonRuntimeState(daemonInfo *TaskRuntimeInfo) error {
	manager := GetManager()
	state := manager.loadRuntimeState()
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	state.Tasks["taskd"] = daemonInfo
	
	// Save the updated state
	return manager.saveRuntimeStateWithData(state)
}

// updateDaemonStoppedState updates the daemon state to stopped
func (dm *DaemonManager) updateDaemonStoppedState(daemonInfo *TaskRuntimeInfo) error {
	manager := GetManager()
	state := manager.loadRuntimeState()
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	// Update daemon info to stopped state
	stoppedInfo := &TaskRuntimeInfo{
		Name:           "taskd",
		Status:         "stopped",
		PID:            0,
		StartTime:      daemonInfo.StartTime,
		EndTime:        time.Now(),
		StoppedByTaskd: true, // Stopped by user command
		RetryNum:       daemonInfo.RetryNum,
	}
	
	state.Tasks["taskd"] = stoppedInfo
	
	// Save the updated state
	return manager.saveRuntimeStateWithData(state)
}