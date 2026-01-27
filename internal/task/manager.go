package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	taskManager *Manager
	once        sync.Once
)

// Manager task manager
type Manager struct {
	configDir string
	tasks     map[string]*Task
	mu        sync.RWMutex
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
}

// GetManager get task manager singleton
func GetManager() *Manager {
	once.Do(func() {
		configDir := getConfigDir()
		taskManager = &Manager{
			configDir: configDir,
			tasks:     make(map[string]*Task),
		}
		taskManager.loadTasks()
		// Clean up stale runtime state after loading tasks
		taskManager.cleanupRuntimeState()
	})
	return taskManager
}

// AddTask add a task
func AddTask(config *Config) error {
	manager := GetManager()
	return manager.addTask(config)
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

func (m *Manager) addTask(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if task already exists
	if _, exists := m.tasks[config.Name]; exists {
		return fmt.Errorf("task '%s' already exists", config.Name)
	}
	
	// Save configuration file
	configPath := filepath.Join(m.configDir, "tasks", config.Name+".toml")
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
	task := NewTask(config)
	// Set exit callback to update runtime state when task exits
	task.SetExitCallback(m.onTaskExit)
	m.tasks[config.Name] = task
	
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
	if err == nil {
		// Save runtime state after successful stop
		m.saveRuntimeState()
	}
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

func (m *Manager) loadTasks() error {
	tasksDir := filepath.Join(m.configDir, "tasks")
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
			
			// Create task instance
			task := NewTask(&config)
			// Set exit callback to update runtime state when task exits
			task.SetExitCallback(m.onTaskExit)
			
			// Restore runtime state if available
			if runtimeInfo, exists := runtimeState.Tasks[config.Name]; exists {
				task.restoreRuntimeState(runtimeInfo)
			}
			
			m.tasks[config.Name] = task
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
	statePath := filepath.Join(m.configDir, "runtime.json")
	
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
	statePath := filepath.Join(m.configDir, "runtime.json")
	
	// Load current state
	state := m.loadRuntimeState()
	
	// Check each task and remove if not actually running
	cleanedTasks := make(map[string]*TaskRuntimeInfo)
	for name, info := range state.Tasks {
		if task, exists := m.tasks[name]; exists {
			// Check if task is actually running
			if task.IsRunning() && task.GetRuntimeInfo() != nil {
				cleanedTasks[name] = info
			}
		}
	}
	
	// Save cleaned state
	state.Tasks = cleanedTasks
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime state: %w", err)
	}
	
	return os.WriteFile(statePath, data, 0644)
}

func (m *Manager) saveRuntimeState() error {
	statePath := filepath.Join(m.configDir, "runtime.json")
	
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

func getConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".taskd")
}