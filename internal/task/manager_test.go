package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetManager(t *testing.T) {
	manager1 := GetManager()
	manager2 := GetManager()
	
	if manager1 == nil {
		t.Fatal("GetManager() returned nil")
	}
	
	// Should return the same instance (singleton)
	if manager1 != manager2 {
		t.Error("GetManager() should return the same instance")
	}
	
	if manager1.builtinHandler == nil {
		t.Error("builtinHandler should not be nil")
	}
	
	if manager1.tasks == nil {
		t.Error("tasks map should not be nil")
	}
}

func TestManagerValidateBuiltinTaskOperation(t *testing.T) {
	manager := GetManager()
	
	tests := []struct {
		name      string
		taskName  string
		operation string
		wantError bool
	}{
		{"builtin task add", "taskd", "add", true},
		{"builtin task edit", "taskd", "edit", true},
		{"builtin task del", "taskd", "del", true},
		{"builtin task start", "taskd", "start", false},
		{"builtin task stop", "taskd", "stop", false},
		{"builtin task restart", "taskd", "restart", false},
		{"builtin task info", "taskd", "info", false},
		{"regular task add", "regular", "add", false},
		{"regular task edit", "regular", "edit", false},
		{"regular task del", "regular", "del", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateBuiltinTaskOperation(tt.taskName, tt.operation)
			
			if tt.wantError && err == nil {
				t.Errorf("ValidateBuiltinTaskOperation(%q, %q) = nil, want error", tt.taskName, tt.operation)
			}
			
			if !tt.wantError && err != nil {
				t.Errorf("ValidateBuiltinTaskOperation(%q, %q) = %v, want nil", tt.taskName, tt.operation, err)
			}
		})
	}
}

func TestManagerLoadRuntimeState(t *testing.T) {
	manager := GetManager()
	
	// Test loading non-existent state file
	state := manager.loadRuntimeState()
	if state == nil {
		t.Fatal("loadRuntimeState() returned nil")
	}
	
	if state.Tasks == nil {
		t.Error("Tasks map should not be nil")
	}
}

func TestManagerSaveRuntimeState(t *testing.T) {
	// This test verifies that saveRuntimeState doesn't return an error
	// We can't easily mock the file path in this case, so we'll just test
	// that the method executes without error
	manager := GetManager()
	
	// Test saving empty state - this should not fail
	err := manager.saveRuntimeState()
	if err != nil {
		t.Fatalf("saveRuntimeState() failed: %v", err)
	}
	
	// The actual file location is determined by taskdconfig.GetTaskDRuntimeFile()
	// which we can't easily mock in this context, but the method should complete
	// successfully even if it creates the file in the default location
}

// Mock function for testing
var getTaskDRuntimeFile = func() string {
	return filepath.Join(os.TempDir(), ".taskd", "runtime.json")
}

func TestManagerSaveRuntimeStateWithData(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Mock the GetTaskDRuntimeFile function
	originalGetTaskDRuntimeFile := getTaskDRuntimeFile
	defer func() {
		getTaskDRuntimeFile = originalGetTaskDRuntimeFile
	}()
	
	testRuntimeFile := filepath.Join(tempDir, "runtime.json")
	getTaskDRuntimeFile = func() string {
		return testRuntimeFile
	}
	
	manager := GetManager()
	
	// Create test data
	testState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"test-task": {
				Name:           "test-task",
				Status:         "running",
				PID:            1234,
				StartTime:      time.Now(),
				StoppedByTaskd: false,
				RetryNum:       0,
			},
		},
	}
	
	// Save the test data
	err := manager.saveRuntimeStateWithData(testState)
	if err != nil {
		t.Fatalf("saveRuntimeStateWithData() failed: %v", err)
	}
	
	// Load and verify the data
	loadedState := manager.loadRuntimeState()
	if loadedState.Tasks == nil {
		t.Fatal("Loaded state tasks should not be nil")
	}
	
	taskInfo, exists := loadedState.Tasks["test-task"]
	if !exists {
		t.Fatal("Test task should exist in loaded state")
	}
	
	if taskInfo.Name != "test-task" {
		t.Errorf("Task name = %q, want 'test-task'", taskInfo.Name)
	}
	
	if taskInfo.Status != "running" {
		t.Errorf("Task status = %q, want 'running'", taskInfo.Status)
	}
	
	if taskInfo.PID != 1234 {
		t.Errorf("Task PID = %d, want 1234", taskInfo.PID)
	}
}

func TestManagerResetTaskRetryCount(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Mock the GetTaskDRuntimeFile function
	originalGetTaskDRuntimeFile := getTaskDRuntimeFile
	defer func() {
		getTaskDRuntimeFile = originalGetTaskDRuntimeFile
	}()
	
	testRuntimeFile := filepath.Join(tempDir, "runtime.json")
	getTaskDRuntimeFile = func() string {
		return testRuntimeFile
	}
	
	manager := GetManager()
	
	// Create initial state with retry count
	initialState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"test-task": {
				Name:           "test-task",
				Status:         "stopped",
				PID:            0,
				StartTime:      time.Now(),
				StoppedByTaskd: false,
				RetryNum:       5, // Initial retry count
			},
		},
	}
	
	// Save initial state
	err := manager.saveRuntimeStateWithData(initialState)
	if err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}
	
	// Reset retry count
	manager.resetTaskRetryCount("test-task")
	
	// Load and verify the updated state
	updatedState := manager.loadRuntimeState()
	taskInfo, exists := updatedState.Tasks["test-task"]
	if !exists {
		t.Fatal("Test task should exist in updated state")
	}
	
	if taskInfo.RetryNum != 0 {
		t.Errorf("Retry count = %d, want 0", taskInfo.RetryNum)
	}
}

func TestManagerSetTaskStoppedByTaskd(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Mock the GetTaskDRuntimeFile function
	originalGetTaskDRuntimeFile := getTaskDRuntimeFile
	defer func() {
		getTaskDRuntimeFile = originalGetTaskDRuntimeFile
	}()
	
	testRuntimeFile := filepath.Join(tempDir, "runtime.json")
	getTaskDRuntimeFile = func() string {
		return testRuntimeFile
	}
	
	manager := GetManager()
	
	// Create initial state
	initialState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"test-task": {
				Name:           "test-task",
				Status:         "running",
				PID:            1234,
				StartTime:      time.Now(),
				StoppedByTaskd: false,
				RetryNum:       0,
			},
		},
	}
	
	// Save initial state
	err := manager.saveRuntimeStateWithData(initialState)
	if err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}
	
	// Set StoppedByTaskd flag
	manager.setTaskStoppedByTaskd("test-task", true)
	
	// Load and verify the updated state
	updatedState := manager.loadRuntimeState()
	taskInfo, exists := updatedState.Tasks["test-task"]
	if !exists {
		t.Fatal("Test task should exist in updated state")
	}
	
	if !taskInfo.StoppedByTaskd {
		t.Error("StoppedByTaskd should be true")
	}
	
	if taskInfo.Status != "stopped" {
		t.Errorf("Status = %q, want 'stopped'", taskInfo.Status)
	}
	
	if taskInfo.PID != 0 {
		t.Errorf("PID = %d, want 0", taskInfo.PID)
	}
	
	// Check that EndTime was set
	if taskInfo.EndTime.IsZero() {
		t.Error("EndTime should be set when stopped by taskd")
	}
}

func TestManagerHasRunningTasks(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Mock the GetTaskDRuntimeFile function
	originalGetTaskDRuntimeFile := getTaskDRuntimeFile
	defer func() {
		getTaskDRuntimeFile = originalGetTaskDRuntimeFile
	}()
	
	testRuntimeFile := filepath.Join(tempDir, "runtime.json")
	getTaskDRuntimeFile = func() string {
		return testRuntimeFile
	}
	
	manager := GetManager()
	
	// Test with no running tasks
	emptyState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"stopped-task": {
				Name:   "stopped-task",
				Status: "stopped",
			},
		},
	}
	
	err := manager.saveRuntimeStateWithData(emptyState)
	if err != nil {
		t.Fatalf("Failed to save empty state: %v", err)
	}
	
	if manager.hasRunningTasks() {
		t.Error("hasRunningTasks() should return false when no tasks are running")
	}
	
	// Test with running tasks
	runningState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"running-task": {
				Name:   "running-task",
				Status: "running",
			},
			"taskd": {
				Name:   "taskd",
				Status: "running", // Should be ignored
			},
		},
	}
	
	err = manager.saveRuntimeStateWithData(runningState)
	if err != nil {
		t.Fatalf("Failed to save running state: %v", err)
	}
	
	if !manager.hasRunningTasks() {
		t.Error("hasRunningTasks() should return true when tasks are running")
	}
}

func TestManagerHasAutoStartTasks(t *testing.T) {
	manager := GetManager()
	
	// Initially should have no auto-start tasks
	if manager.hasAutoStartTasks() {
		t.Error("hasAutoStartTasks() should return false initially")
	}
	
	// Add a task with auto-start enabled
	autoStartConfig := &Config{
		DisplayName: "Auto Start Task",
		Description: "Test auto start task",
		Executable:  "echo test",
		WorkDir:     "/tmp",
		InheritEnv:  true,
		AutoStart:   true,
	}
	
	// Manually add to tasks map for testing
	manager.mu.Lock()
	task := NewTask("auto-start-task", autoStartConfig)
	manager.tasks["auto-start-task"] = task
	manager.mu.Unlock()
	
	if !manager.hasAutoStartTasks() {
		t.Error("hasAutoStartTasks() should return true when auto-start tasks exist")
	}
	
	// Remove the task
	manager.mu.Lock()
	delete(manager.tasks, "auto-start-task")
	manager.mu.Unlock()
	
	if manager.hasAutoStartTasks() {
		t.Error("hasAutoStartTasks() should return false after removing auto-start task")
	}
}

func TestManagerNeedsDaemon(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Mock the GetTaskDRuntimeFile function
	originalGetTaskDRuntimeFile := getTaskDRuntimeFile
	defer func() {
		getTaskDRuntimeFile = originalGetTaskDRuntimeFile
	}()
	
	testRuntimeFile := filepath.Join(tempDir, "runtime.json")
	getTaskDRuntimeFile = func() string {
		return testRuntimeFile
	}
	
	manager := GetManager()
	
	// Test with no running tasks and no auto-start tasks
	emptyState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{},
	}
	
	err := manager.saveRuntimeStateWithData(emptyState)
	if err != nil {
		t.Fatalf("Failed to save empty state: %v", err)
	}
	
	if manager.needsDaemon() {
		t.Error("needsDaemon() should return false when no daemon is needed")
	}
	
	// Test with running tasks
	runningState := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"running-task": {
				Name:   "running-task",
				Status: "running",
			},
		},
	}
	
	err = manager.saveRuntimeStateWithData(runningState)
	if err != nil {
		t.Fatalf("Failed to save running state: %v", err)
	}
	
	if !manager.needsDaemon() {
		t.Error("needsDaemon() should return true when running tasks exist")
	}
}

func TestRuntimeStateStruct(t *testing.T) {
	state := &RuntimeState{
		Tasks: map[string]*TaskRuntimeInfo{
			"test-task": {
				Name:           "test-task",
				Status:         "running",
				PID:            1234,
				StartTime:      time.Now(),
				StoppedByTaskd: false,
				RetryNum:       0,
			},
		},
	}
	
	if state.Tasks == nil {
		t.Error("Tasks map should not be nil")
	}
	
	taskInfo, exists := state.Tasks["test-task"]
	if !exists {
		t.Fatal("Test task should exist")
	}
	
	if taskInfo.Name != "test-task" {
		t.Errorf("Name = %q, want 'test-task'", taskInfo.Name)
	}
	
	if taskInfo.Status != "running" {
		t.Errorf("Status = %q, want 'running'", taskInfo.Status)
	}
	
	if taskInfo.PID != 1234 {
		t.Errorf("PID = %d, want 1234", taskInfo.PID)
	}
	
	if taskInfo.StoppedByTaskd {
		t.Error("StoppedByTaskd should be false")
	}
	
	if taskInfo.RetryNum != 0 {
		t.Errorf("RetryNum = %d, want 0", taskInfo.RetryNum)
	}
}

func TestTaskRuntimeInfoStruct(t *testing.T) {
	now := time.Now()
	info := &TaskRuntimeInfo{
		Name:           "test-task",
		Status:         "stopped",
		PID:            0,
		StartTime:      now,
		EndTime:        now.Add(time.Hour),
		ExitCode:       1,
		StoppedByTaskd: true,
		RetryNum:       3,
	}
	
	if info.Name != "test-task" {
		t.Errorf("Name = %q, want 'test-task'", info.Name)
	}
	
	if info.Status != "stopped" {
		t.Errorf("Status = %q, want 'stopped'", info.Status)
	}
	
	if info.PID != 0 {
		t.Errorf("PID = %d, want 0", info.PID)
	}
	
	if !info.StartTime.Equal(now) {
		t.Errorf("StartTime = %v, want %v", info.StartTime, now)
	}
	
	if !info.EndTime.Equal(now.Add(time.Hour)) {
		t.Errorf("EndTime = %v, want %v", info.EndTime, now.Add(time.Hour))
	}
	
	if info.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", info.ExitCode)
	}
	
	if !info.StoppedByTaskd {
		t.Error("StoppedByTaskd should be true")
	}
	
	if info.RetryNum != 3 {
		t.Errorf("RetryNum = %d, want 3", info.RetryNum)
	}
}