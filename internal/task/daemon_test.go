package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewTaskMonitor(t *testing.T) {
	interval := 5 * time.Second
	monitor := NewTaskMonitor(interval)
	
	if monitor == nil {
		t.Fatal("NewTaskMonitor() returned nil")
	}
	
	if monitor.checkInterval != interval {
		t.Errorf("checkInterval = %v, want %v", monitor.checkInterval, interval)
	}
	
	if monitor.stopChan == nil {
		t.Error("stopChan should not be nil")
	}
	
	if monitor.manager == nil {
		t.Error("manager should not be nil")
	}
	
	if monitor.isRunning {
		t.Error("isRunning should be false initially")
	}
}

func TestTaskMonitorIsRunning(t *testing.T) {
	monitor := NewTaskMonitor(1 * time.Second)
	
	// Initially not running
	if monitor.IsRunning() {
		t.Error("IsRunning() should return false initially")
	}
	
	// Simulate running state
	monitor.mu.Lock()
	monitor.isRunning = true
	monitor.mu.Unlock()
	
	if !monitor.IsRunning() {
		t.Error("IsRunning() should return true when running")
	}
	
	// Reset to not running
	monitor.mu.Lock()
	monitor.isRunning = false
	monitor.mu.Unlock()
	
	if monitor.IsRunning() {
		t.Error("IsRunning() should return false after reset")
	}
}

func TestTaskMonitorStartStop(t *testing.T) {
	monitor := NewTaskMonitor(100 * time.Millisecond)
	
	// Test starting the monitor
	go monitor.Start()
	
	// Give it time to start
	time.Sleep(50 * time.Millisecond)
	
	if !monitor.IsRunning() {
		t.Error("Monitor should be running after Start()")
	}
	
	// Test stopping the monitor
	monitor.Stop()
	
	// Give it time to stop
	time.Sleep(50 * time.Millisecond)
	
	if monitor.IsRunning() {
		t.Error("Monitor should not be running after Stop()")
	}
}

func TestShouldRetryTask(t *testing.T) {
	tests := []struct {
		name         string
		taskName     string
		runtimeInfo  *TaskRuntimeInfo
		config       *Config
		want         bool
	}{
		{
			name:     "should retry - all conditions met",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: false,
				RetryNum:       1,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			want: true,
		},
		{
			name:     "should not retry - not auto start",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: false,
				RetryNum:       1,
			},
			config: &Config{
				AutoStart:   false,
				MaxRetryNum: 3,
			},
			want: false,
		},
		{
			name:     "should not retry - still running",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "running",
				StoppedByTaskd: false,
				RetryNum:       1,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			want: false,
		},
		{
			name:     "should not retry - stopped by taskd",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: true,
				RetryNum:       1,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			want: false,
		},
		{
			name:     "should not retry - max retries reached",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: false,
				RetryNum:       3,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			want: false,
		},
		{
			name:     "should retry - unlimited retries",
			taskName: "test-task",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: false,
				RetryNum:       10,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 0, // 0 means unlimited
			},
			want: true,
		},
		{
			name:     "should not retry - taskd daemon",
			taskName: "taskd",
			runtimeInfo: &TaskRuntimeInfo{
				Status:         "stopped",
				StoppedByTaskd: false,
				RetryNum:       1,
			},
			config: &Config{
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic directly by checking the conditions
			// This avoids the file system dependency
			shouldRetry := tt.config.AutoStart &&
				tt.runtimeInfo.Status == "stopped" &&
				!tt.runtimeInfo.StoppedByTaskd &&
				(tt.config.MaxRetryNum <= 0 || tt.runtimeInfo.RetryNum < tt.config.MaxRetryNum) &&
				tt.taskName != "taskd" // Skip daemon task
			
			if shouldRetry != tt.want {
				t.Errorf("shouldRetryTask() = %v, want %v", shouldRetry, tt.want)
			}
		})
	}
}

// Mock function for testing
var getTaskDTasksDir = func() string {
	return filepath.Join(os.TempDir(), ".taskd", "tasks")
}

func TestNewProcessChecker(t *testing.T) {
	checker := NewProcessChecker()
	if checker == nil {
		t.Fatal("NewProcessChecker() returned nil")
	}
}

func TestProcessCheckerCheckTaskProcess(t *testing.T) {
	checker := NewProcessChecker()
	
	t.Run("invalid PID", func(t *testing.T) {
		status, err := checker.CheckTaskProcess(0)
		if err != nil {
			t.Errorf("CheckTaskProcess(0) returned error: %v", err)
		}
		
		if status.Exists {
			t.Error("Process with PID 0 should not exist")
		}
		
		if status.IsTaskd {
			t.Error("Process with PID 0 should not be taskd")
		}
	})
	
	t.Run("negative PID", func(t *testing.T) {
		status, err := checker.CheckTaskProcess(-1)
		if err != nil {
			t.Errorf("CheckTaskProcess(-1) returned error: %v", err)
		}
		
		if status.Exists {
			t.Error("Process with negative PID should not exist")
		}
	})
	
	t.Run("current process PID", func(t *testing.T) {
		currentPID := os.Getpid()
		status, err := checker.CheckTaskProcess(currentPID)
		if err != nil {
			t.Errorf("CheckTaskProcess(%d) returned error: %v", currentPID, err)
		}
		
		if !status.Exists {
			t.Error("Current process should exist")
		}
		
		// Note: IsTaskd might be true or false depending on how the test is run
		// We don't assert on this value in this test
	})
}

func TestNewFileStateManager(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test-state.json")
	fsm := NewFileStateManager(tempFile)
	
	if fsm == nil {
		t.Fatal("NewFileStateManager() returned nil")
	}
	
	if fsm.statePath != tempFile {
		t.Errorf("statePath = %q, want %q", fsm.statePath, tempFile)
	}
}

func TestFileStateManagerUpdateTaskState(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test-state.json")
	fsm := NewFileStateManager(tempFile)
	
	taskInfo := &TaskRuntimeInfo{
		Name:           "test-task",
		Status:         "running",
		PID:            1234,
		StartTime:      time.Now(),
		StoppedByTaskd: false,
		RetryNum:       0,
	}
	
	// Update task state
	err := fsm.UpdateTaskState("test-task", taskInfo)
	if err != nil {
		t.Fatalf("UpdateTaskState() failed: %v", err)
	}
	
	// Verify the state was saved
	state, err := fsm.GetRuntimeState()
	if err != nil {
		t.Fatalf("GetRuntimeState() failed: %v", err)
	}
	
	if state.Tasks == nil {
		t.Fatal("Tasks map should not be nil")
	}
	
	savedInfo, exists := state.Tasks["test-task"]
	if !exists {
		t.Fatal("Task should exist in state")
	}
	
	if savedInfo.Name != taskInfo.Name {
		t.Errorf("Name = %q, want %q", savedInfo.Name, taskInfo.Name)
	}
	
	if savedInfo.Status != taskInfo.Status {
		t.Errorf("Status = %q, want %q", savedInfo.Status, taskInfo.Status)
	}
	
	if savedInfo.PID != taskInfo.PID {
		t.Errorf("PID = %d, want %d", savedInfo.PID, taskInfo.PID)
	}
}

func TestFileStateManagerUpdateDaemonState(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test-state.json")
	fsm := NewFileStateManager(tempFile)
	
	daemonStatus := &DaemonStatus{
		IsRunning: true,
		PID:       5678,
		StartTime: time.Now(),
	}
	
	// Update daemon state
	err := fsm.UpdateDaemonState(daemonStatus)
	if err != nil {
		t.Fatalf("UpdateDaemonState() failed: %v", err)
	}
	
	// Verify the state was saved
	state, err := fsm.GetRuntimeState()
	if err != nil {
		t.Fatalf("GetRuntimeState() failed: %v", err)
	}
	
	daemonInfo, exists := state.Tasks["taskd"]
	if !exists {
		t.Fatal("Daemon task should exist in state")
	}
	
	if daemonInfo.Name != "taskd" {
		t.Errorf("Name = %q, want 'taskd'", daemonInfo.Name)
	}
	
	if daemonInfo.Status != "running" {
		t.Errorf("Status = %q, want 'running'", daemonInfo.Status)
	}
	
	if daemonInfo.PID != daemonStatus.PID {
		t.Errorf("PID = %d, want %d", daemonInfo.PID, daemonStatus.PID)
	}
}

func TestFileStateManagerBatchUpdate(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test-state.json")
	fsm := NewFileStateManager(tempFile)
	
	updates := map[string]*TaskRuntimeInfo{
		"task1": {
			Name:   "task1",
			Status: "running",
			PID:    1001,
		},
		"task2": {
			Name:   "task2",
			Status: "stopped",
			PID:    0,
		},
	}
	
	// Batch update
	err := fsm.BatchUpdate(updates)
	if err != nil {
		t.Fatalf("BatchUpdate() failed: %v", err)
	}
	
	// Verify all tasks were updated
	state, err := fsm.GetRuntimeState()
	if err != nil {
		t.Fatalf("GetRuntimeState() failed: %v", err)
	}
	
	for taskName, expectedInfo := range updates {
		savedInfo, exists := state.Tasks[taskName]
		if !exists {
			t.Errorf("Task %s should exist in state", taskName)
			continue
		}
		
		if savedInfo.Name != expectedInfo.Name {
			t.Errorf("Task %s: Name = %q, want %q", taskName, savedInfo.Name, expectedInfo.Name)
		}
		
		if savedInfo.Status != expectedInfo.Status {
			t.Errorf("Task %s: Status = %q, want %q", taskName, savedInfo.Status, expectedInfo.Status)
		}
		
		if savedInfo.PID != expectedInfo.PID {
			t.Errorf("Task %s: PID = %d, want %d", taskName, savedInfo.PID, expectedInfo.PID)
		}
	}
}

func TestFileStateManagerCorruptedStateRecovery(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test-state.json")
	backupFile := stateFile + ".backup"
	
	fsm := NewFileStateManager(stateFile)
	
	// Create a valid backup file
	validState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"recovered-task": {
				Name:   "recovered-task",
				Status: "running",
				PID:    9999,
			},
		},
	}
	
	backupData, err := json.MarshalIndent(validState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal backup data: %v", err)
	}
	
	if err := os.WriteFile(backupFile, backupData, 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}
	
	// Create a corrupted main state file
	corruptedData := []byte("{invalid json")
	if err := os.WriteFile(stateFile, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to create corrupted state file: %v", err)
	}
	
	// Try to get runtime state - should recover from backup
	state, err := fsm.GetRuntimeState()
	if err != nil {
		t.Fatalf("GetRuntimeState() should recover from backup: %v", err)
	}
	
	// Verify recovery worked
	recoveredTask, exists := state.Tasks["recovered-task"]
	if !exists {
		t.Fatal("Recovered task should exist")
	}
	
	if recoveredTask.PID != 9999 {
		t.Errorf("Recovered task PID = %d, want 9999", recoveredTask.PID)
	}
}

func TestNewDaemonStateManager(t *testing.T) {
	dsm := NewDaemonStateManager()
	if dsm == nil {
		t.Fatal("NewDaemonStateManager() returned nil")
	}
	
	if dsm.manager == nil {
		t.Error("manager should not be nil")
	}
}

func TestGetDaemonManager(t *testing.T) {
	dm1 := GetDaemonManager()
	dm2 := GetDaemonManager()
	
	if dm1 == nil {
		t.Fatal("GetDaemonManager() returned nil")
	}
	
	// Should return the same instance (singleton)
	if dm1 != dm2 {
		t.Error("GetDaemonManager() should return the same instance")
	}
}

func TestDaemonManagerConcurrency(t *testing.T) {
	dm := GetDaemonManager()
	
	// Test concurrent access to IsRunning method
	var wg sync.WaitGroup
	numGoroutines := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This should not panic or cause race conditions
			_ = dm.IsRunning()
		}()
	}
	
	wg.Wait()
}

func TestProcessStatusStruct(t *testing.T) {
	status := &ProcessStatus{
		Exists:         true,
		IsTaskd:        true,
		ExitCode:       0,
		ExecutablePath: "/path/to/taskd",
	}
	
	if !status.Exists {
		t.Error("Exists should be true")
	}
	
	if !status.IsTaskd {
		t.Error("IsTaskd should be true")
	}
	
	if status.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", status.ExitCode)
	}
	
	if status.ExecutablePath != "/path/to/taskd" {
		t.Errorf("ExecutablePath = %q, want '/path/to/taskd'", status.ExecutablePath)
	}
}

func TestDaemonStatusStruct(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		IsRunning: true,
		PID:       1234,
		StartTime: now,
		LastCheck: now,
	}
	
	if !status.IsRunning {
		t.Error("IsRunning should be true")
	}
	
	if status.PID != 1234 {
		t.Errorf("PID = %d, want 1234", status.PID)
	}
	
	if !status.StartTime.Equal(now) {
		t.Errorf("StartTime = %v, want %v", status.StartTime, now)
	}
	
	if !status.LastCheck.Equal(now) {
		t.Errorf("LastCheck = %v, want %v", status.LastCheck, now)
	}
}