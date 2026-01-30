package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	taskdconfig "taskd/internal/config"
)

var (
	taskManager *Manager
	once        sync.Once
)

// Manager task manager
type Manager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// RuntimeState represents the runtime state of tasks
type RuntimeState struct {
	Tasks map[string]*TaskRuntimeInfo `json:"tasks"`
}

// TaskRuntimeInfo represents runtime information for a task
type TaskRuntimeInfo struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
}

// GetManager get task manager singleton
func GetManager() *Manager {
	once.Do(func() {
		taskManager = &Manager{
			tasks: make(map[string]*Task),
		}
		taskManager.loadTasks()
		// Clean up stale runtime state after loading tasks
		taskManager.cleanupRuntimeState()
	})
	return taskManager
}

// AddTask add a task
func AddTask(taskName string, config *Config) error {
	manager := GetManager()
	return manager.addTask(taskName, config)
}

// ListTasks list all tasks
func ListTasks() ([]*TaskInfo, error) {
	manager := GetManager()
	return manager.listTasks()
}

// StartTask start a task
func StartTask(name string) error {
	manager := GetManager()
	return manager.startTask(name)
}

// StopTask stop a task
func StopTask(name string) error {
	manager := GetManager()
	return manager.stopTask(name)
}

// GetTaskStatus get task status
func GetTaskStatus(name string) (*TaskInfo, error) {
	manager := GetManager()
	return manager.getTaskStatus(name)
}

// RemoveTask remove a task
func RemoveTask(name string) error {
	manager := GetManager()
	return manager.removeTask(name)
}

func (m *Manager) addTask(taskName string, config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if task already exists
	if _, exists := m.tasks[taskName]; exists {
		return fmt.Errorf("task '%s' already exists", taskName)
	}

	// Save configuration file
	configPath := filepath.Join(taskdconfig.GetTaskDTasksDir(), taskName+".toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(config); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Create task instance
	task := NewTask(taskName, config)
	// Set exit callback to update runtime state when task exits
	task.SetExitCallback(m.onTaskExit)
	m.tasks[taskName] = task

	return nil
}

func (m *Manager) listTasks() ([]*TaskInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*TaskInfo
	for _, task := range m.tasks {
		info := task.GetInfo()
		tasks = append(tasks, info)
	}

	return tasks, nil
}

// StartTask start a task by name
func (m *Manager) StartTask(name string) error {
	return m.startTask(name)
}

// StopTask stop a task by name
func (m *Manager) StopTask(name string) error {
	return m.stopTask(name)
}

// RestartTask restart a task by name (stop if running, then start)
func (m *Manager) RestartTask(name string) error {
	return m.restartTask(name)
}

// RemoveTask remove a task by name
func (m *Manager) RemoveTask(name string) error {
	return m.removeTask(name)
}

// GetTaskStatus get task status by name
func (m *Manager) GetTaskStatus(name string) (*TaskInfo, error) {
	return m.getTaskStatus(name)
}

func (m *Manager) startTask(name string) error {
	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	err := task.Start()
	if err == nil {
		// Save runtime state after successful start
		m.saveRuntimeState()
	}
	return err
}

func (m *Manager) stopTask(name string) error {
	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	err := task.Stop()
	// Always save runtime state after stop attempt, regardless of success
	// This ensures that even if the task was already stopped, the state is consistent
	m.saveRuntimeState()
	return err
}

func (m *Manager) getTaskStatus(name string) (*TaskInfo, error) {
	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task '%s' does not exist", name)
	}

	return task.GetInfo(), nil
}

func (m *Manager) restartTask(name string) error {
	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	// Stop the task if it's running
	if task.IsRunning() {
		if err := task.Stop(); err != nil {
			return fmt.Errorf("failed to stop task before restart: %w", err)
		}

		// Wait a moment for the process to fully stop
		time.Sleep(100 * time.Millisecond)
	}

	// Start the task
	err := task.Start()
	if err == nil {
		// Save runtime state after successful restart
		m.saveRuntimeState()
	}
	return err
}

func (m *Manager) removeTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[name]
	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	// Stop the task if it's running
	if task.IsRunning() {
		if err := task.Stop(); err != nil {
			return fmt.Errorf("failed to stop task before removal: %w", err)
		}
	}

	// Remove the task from the manager
	delete(m.tasks, name)

	// Save runtime state after removal
	m.saveRuntimeState()

	return nil
}

func (m *Manager) loadTasks() error {
	tasksDir := taskdconfig.GetTaskDTasksDir()
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		return nil // tasks directory doesn't exist, skip
	}

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks directory: %w", err)
	}

	// Load runtime state
	runtimeState := m.loadRuntimeState()

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
			configPath := filepath.Join(tasksDir, entry.Name())
			var config Config

			if _, err := toml.DecodeFile(configPath, &config); err != nil {
				continue // skip invalid config files
			}

			// Extract task name from filename (remove .toml extension)
			taskName := strings.TrimSuffix(entry.Name(), ".toml")

			// Create task instance
			task := NewTask(taskName, &config)
			// Set exit callback to update runtime state when task exits
			task.SetExitCallback(m.onTaskExit)

			// Restore runtime state if available
			if runtimeInfo, exists := runtimeState.Tasks[taskName]; exists {
				task.restoreRuntimeState(runtimeInfo)
			}

			m.tasks[taskName] = task
		}
	}

	return nil
}

// onTaskExit is called when a task exits naturally
func (m *Manager) onTaskExit(taskName string) {
	// Update runtime state when task exits
	go func() {
		// Use a goroutine to avoid potential deadlocks
		time.Sleep(100 * time.Millisecond) // Small delay to ensure task state is updated
		m.saveRuntimeState()
	}()
}

func (m *Manager) loadRuntimeState() *RuntimeState {
	statePath := taskdconfig.GetTaskDRuntimeFile()

	data, err := os.ReadFile(statePath)
	if err != nil {
		return &RuntimeState{Tasks: make(map[string]*TaskRuntimeInfo)}
	}

	var state RuntimeState
	if err := json.Unmarshal(data, &state); err != nil {
		return &RuntimeState{Tasks: make(map[string]*TaskRuntimeInfo)}
	}

	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}

	return &state
}

// cleanupRuntimeState removes stale entries from runtime state
func (m *Manager) cleanupRuntimeState() error {
	statePath := taskdconfig.GetTaskDRuntimeFile()

	// Load current state
	state := m.loadRuntimeState()

	// Update each task's runtime info instead of removing stopped tasks
	updatedTasks := make(map[string]*TaskRuntimeInfo)
	for name, info := range state.Tasks {
		if task, exists := m.tasks[name]; exists {
			// Get current runtime info from the task
			if currentInfo := task.GetRuntimeInfo(); currentInfo != nil {
				updatedTasks[name] = currentInfo
			} else {
				// Keep the old info if we can't get current info
				updatedTasks[name] = info
			}
		} else {
			// Task no longer exists in manager, remove from runtime state
			// This handles the case where tasks are deleted
		}
	}

	// Add any new tasks that aren't in the runtime state yet
	for name, task := range m.tasks {
		if _, exists := updatedTasks[name]; !exists {
			if runtimeInfo := task.GetRuntimeInfo(); runtimeInfo != nil {
				updatedTasks[name] = runtimeInfo
			}
		}
	}

	// Save updated state
	state.Tasks = updatedTasks
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime state: %w", err)
	}

	return os.WriteFile(statePath, data, 0644)
}

func (m *Manager) saveRuntimeState() error {
	statePath := taskdconfig.GetTaskDRuntimeFile()

	state := &RuntimeState{Tasks: make(map[string]*TaskRuntimeInfo)}

	for name, task := range m.tasks {
		info := task.GetRuntimeInfo()
		if info != nil {
			state.Tasks[name] = info
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime state: %w", err)
	}

	return os.WriteFile(statePath, data, 0644)
}

// GetTaskDetailInfo get detailed task information (replaces GetTaskStatus)
func GetTaskDetailInfo(name string) (*TaskDetailInfo, error) {
	manager := GetManager()
	return manager.getTaskDetailInfo(name)
}

// ReloadTask reloads a task configuration from file
func (m *Manager) ReloadTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if task exists
	if _, exists := m.tasks[name]; !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	// Load configuration from file
	configPath := filepath.Join(taskdconfig.GetTaskDTasksDir(), name+".toml")
	var config Config

	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return fmt.Errorf("failed to load task configuration: %w", err)
	}

	// Create new task instance
	newTask := NewTask(name, &config)
	newTask.SetExitCallback(m.onTaskExit)

	// Replace the existing task
	m.tasks[name] = newTask

	return nil
}

func (m *Manager) getTaskDetailInfo(name string) (*TaskDetailInfo, error) {
	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task '%s' does not exist", name)
	}

	// Get basic task info
	basicInfo := task.GetInfo()

	// Get IO info
	ioManager := GetIOManager()
	ioInfo, err := ioManager.GetTaskIOInfo(task.config)
	if err != nil {
		return nil, fmt.Errorf("failed to get IO info: %w", err)
	}

	// Create detailed info
	detailInfo := &TaskDetailInfo{
		Name:        basicInfo.Name,
		Status:      basicInfo.Status,
		PID:         basicInfo.PID,
		StartTime:   basicInfo.StartTime,
		Executable:  basicInfo.Executable,
		ExitCode:    basicInfo.ExitCode,
		LastError:   basicInfo.LastError,
		DisplayName: task.config.DisplayName,
		Description: task.config.Description,
		WorkDir:     task.config.WorkDir,
		Args:        task.config.Args,
		Env:         task.config.Env,
		InheritEnv:  task.config.InheritEnv,
		IOInfo:      ioInfo,
	}

	return detailInfo, nil
}
