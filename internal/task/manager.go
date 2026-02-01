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
	tasks          map[string]*Task
	mu             sync.RWMutex
	builtinHandler *BuiltinTaskHandler
}

// RuntimeState represents the runtime state of tasks
type RuntimeState struct {
	Tasks map[string]*TaskRuntimeInfo `json:"tasks"`
}

// TaskRuntimeInfo represents runtime information for a task
type TaskRuntimeInfo struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	PID            int       `json:"pid"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time,omitempty"`
	ExitCode       int       `json:"exit_code,omitempty"`
	StoppedByTaskd bool      `json:"stopped_by_taskd"` // 新增：是否由 taskd stop 停止
	RetryNum       int       `json:"retry_num"`        // 新增：当前重试次数
}

// DaemonStatus represents the status of the daemon process
type DaemonStatus struct {
	IsRunning bool      `json:"is_running"`
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	LastCheck time.Time `json:"last_check"`
}

// ProcessStatus represents the result of process checking
type ProcessStatus struct {
	Exists       bool   `json:"exists"`        // 进程是否存在
	IsTaskd      bool   `json:"is_taskd"`      // 是否为 taskd 进程
	ExitCode     int    `json:"exit_code"`     // 退出码（如果已退出）
	ExecutablePath string `json:"executable_path"` // 可执行文件路径
}

// GetManager get task manager singleton
func GetManager() *Manager {
	once.Do(func() {
		taskManager = &Manager{
			tasks:          make(map[string]*Task),
			builtinHandler: NewBuiltinTaskHandler(),
		}
		taskManager.loadTasks()
		// Clean up stale runtime state after loading tasks
		taskManager.cleanupRuntimeState()
	})
	return taskManager
}

// ValidateBuiltinTaskOperation validates if an operation is allowed on a builtin task
func (m *Manager) ValidateBuiltinTaskOperation(taskName, operation string) error {
	return m.builtinHandler.ValidateOperation(taskName, operation)
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

	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(taskName) {
		return m.builtinHandler.ValidateOperation(taskName, "add")
	}

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
	// 确保守护进程运行（如果需要）
	if err := m.ensureDaemonForCommand(); err != nil {
		fmt.Printf("Warning: Failed to ensure daemon is running: %v\n", err)
		// 继续执行，不因守护进程启动失败而阻止列表显示
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*TaskInfo
	
	// 首先添加守护进程任务（如果存在）
	if daemonInfo, err := m.getBuiltinTaskStatus("taskd"); err == nil {
		tasks = append(tasks, daemonInfo)
	}
	
	// 然后添加其他任务
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
	// 在启动任务前确保守护进程运行（如果需要）
	if err := m.ensureDaemonForCommand(); err != nil {
		fmt.Printf("Warning: Failed to ensure daemon is running: %v\n", err)
		// 继续执行，不因守护进程启动失败而阻止任务启动
	}
	
	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(name) {
		// For builtin tasks, we need to handle them specially
		return m.startBuiltinTask(name)
	}

	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	err := task.Start()
	if err == nil {
		// Reset retry count when manually starting a task
		m.resetTaskRetryCount(name)
		// Save runtime state after successful start
		m.saveRuntimeState()
	}
	return err
}

// startBuiltinTask starts a builtin task
func (m *Manager) startBuiltinTask(name string) error {
	if name == "taskd" {
		// Use DaemonManager to start the daemon
		daemonManager := GetDaemonManager()
		return daemonManager.StartDaemon()
	}
	return fmt.Errorf("unknown builtin task: %s", name)
}

// stopBuiltinTask stops a builtin task
func (m *Manager) stopBuiltinTask(name string) error {
	if name == "taskd" {
		// Use DaemonManager to stop the daemon
		daemonManager := GetDaemonManager()
		return daemonManager.StopDaemon()
	}
	return fmt.Errorf("unknown builtin task: %s", name)
}

// getBuiltinTaskStatus gets the status of a builtin task
func (m *Manager) getBuiltinTaskStatus(name string) (*TaskInfo, error) {
	if name == "taskd" {
		// Get daemon status from runtime state
		state := m.loadRuntimeState()
		daemonInfo, exists := state.Tasks["taskd"]
		
		if !exists {
			// Daemon has never been started
			return &TaskInfo{
				Name:       "taskd",
				Status:     "stopped",
				PID:        0,
				StartTime:  "",
				Executable: "taskd --daemon",
			}, nil
		}
		
		// Check if the daemon is actually running
		daemonManager := GetDaemonManager()
		isRunning := daemonManager.IsRunning()
		
		status := "stopped"
		if isRunning {
			status = "running"
		}
		
		return &TaskInfo{
			Name:       "taskd",
			Status:     status,
			PID:        daemonInfo.PID,
			StartTime:  daemonInfo.StartTime.Format("2006-01-02 15:04:05"),
			Executable: "taskd --daemon",
			ExitCode:   daemonInfo.ExitCode,
		}, nil
	}
	return nil, fmt.Errorf("unknown builtin task: %s", name)
}

// getBuiltinTaskDetailInfo gets the detailed information of a builtin task
func (m *Manager) getBuiltinTaskDetailInfo(name string) (*TaskDetailInfo, error) {
	if name == "taskd" {
		config := m.builtinHandler.GetBuiltinTaskConfig(name)
		if config == nil {
			return nil, fmt.Errorf("failed to get builtin task config")
		}
		
		// Get daemon status from runtime state
		state := m.loadRuntimeState()
		daemonInfo, exists := state.Tasks["taskd"]
		
		status := "stopped"
		pid := 0
		startTime := ""
		
		if exists {
			// Check if the daemon is actually running
			daemonManager := GetDaemonManager()
			isRunning := daemonManager.IsRunning()
			
			if isRunning {
				status = "running"
			}
			pid = daemonInfo.PID
			startTime = daemonInfo.StartTime.Format("2006-01-02 15:04:05")
		}
		
		return &TaskDetailInfo{
			Name:        "taskd",
			Status:      status,
			PID:         pid,
			StartTime:   startTime,
			Executable:  config.Executable,
			DisplayName: config.DisplayName,
			Description: config.Description,
			WorkDir:     config.WorkDir,
			Args:        []string{},
			Env:         []string{},
			InheritEnv:  config.InheritEnv,
			IOInfo:      &TaskIOInfo{}, // No IO redirection for daemon
		}, nil
	}
	return nil, fmt.Errorf("unknown builtin task: %s", name)
}

func (m *Manager) stopTask(name string) error {
	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(name) {
		// For builtin tasks, we need to handle them specially
		return m.stopBuiltinTask(name)
	}

	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' does not exist", name)
	}

	err := task.Stop()
	
	// Set StoppedByTaskd flag when manually stopping a task
	m.setTaskStoppedByTaskd(name, true)
	
	// Always save runtime state after stop attempt, regardless of success
	// This ensures that even if the task was already stopped, the state is consistent
	m.saveRuntimeState()
	return err
}

func (m *Manager) getTaskStatus(name string) (*TaskInfo, error) {
	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(name) {
		// For builtin tasks, we need to get status from runtime state
		return m.getBuiltinTaskStatus(name)
	}

	m.mu.RLock()
	task, exists := m.tasks[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task '%s' does not exist", name)
	}

	return task.GetInfo(), nil
}

func (m *Manager) restartTask(name string) error {
	// 在重启任务前确保守护进程运行（如果需要）
	if err := m.ensureDaemonForCommand(); err != nil {
		fmt.Printf("Warning: Failed to ensure daemon is running: %v\n", err)
		// 继续执行，不因守护进程启动失败而阻止任务重启
	}
	
	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(name) {
		// For builtin tasks, restart means stop then start
		if err := m.stopBuiltinTask(name); err != nil {
			// If stop fails, still try to start (maybe it wasn't running)
			fmt.Printf("Warning: Failed to stop builtin task %s: %v\n", name, err)
		}
		return m.startBuiltinTask(name)
	}
	
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
		// Reset retry count when manually restarting a task
		m.resetTaskRetryCount(name)
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
		} else if m.builtinHandler.IsBuiltinTask(name) {
			// Keep builtin tasks in runtime state even if they're not in m.tasks
			// Builtin tasks don't have .toml files and are managed differently
			updatedTasks[name] = info
		} else {
			// Task no longer exists in manager and is not builtin, remove from runtime state
			// This handles the case where regular tasks are deleted
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

// saveRuntimeStateWithData saves the given runtime state data
func (m *Manager) saveRuntimeStateWithData(state *RuntimeState) error {
	statePath := taskdconfig.GetTaskDRuntimeFile()

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
	// 确保守护进程运行（如果需要）
	if err := m.ensureDaemonForCommand(); err != nil {
		fmt.Printf("Warning: Failed to ensure daemon is running: %v\n", err)
		// 继续执行，不因守护进程启动失败而阻止信息显示
	}
	
	// Check if this is a builtin task
	if m.builtinHandler.IsBuiltinTask(name) {
		return m.getBuiltinTaskDetailInfo(name)
	}

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

// resetTaskRetryCount 重置任务的重试计数
func (m *Manager) resetTaskRetryCount(taskName string) {
	state := m.loadRuntimeState()
	if state.Tasks == nil {
		return
	}
	
	runtimeInfo, exists := state.Tasks[taskName]
	if !exists {
		return
	}
	
	// 重置重试计数
	runtimeInfo.RetryNum = 0
	
	// 保存更新后的状态
	if err := m.saveRuntimeStateWithData(state); err != nil {
		fmt.Printf("Warning: Failed to reset retry count for task %s: %v\n", taskName, err)
	}
}

// setTaskStoppedByTaskd 设置任务的 StoppedByTaskd 标记
func (m *Manager) setTaskStoppedByTaskd(taskName string, stoppedByTaskd bool) {
	state := m.loadRuntimeState()
	if state.Tasks == nil {
		return
	}
	
	runtimeInfo, exists := state.Tasks[taskName]
	if !exists {
		return
	}
	
	// 设置 StoppedByTaskd 标记
	runtimeInfo.StoppedByTaskd = stoppedByTaskd
	
	// 如果是手动停止，同时更新结束时间
	if stoppedByTaskd {
		runtimeInfo.EndTime = time.Now()
		runtimeInfo.Status = "stopped"
		runtimeInfo.PID = 0
	}
	
	// 保存更新后的状态
	if err := m.saveRuntimeStateWithData(state); err != nil {
		fmt.Printf("Warning: Failed to set StoppedByTaskd flag for task %s: %v\n", taskName, err)
	}
}

// ensureDaemonForCommand 确保守护进程在需要时运行
func (m *Manager) ensureDaemonForCommand() error {
	// 检查是否需要守护进程
	if m.needsDaemon() {
		daemonManager := GetDaemonManager()
		return daemonManager.EnsureDaemonRunning()
	}
	return nil
}

// needsDaemon 检查是否需要守护进程
func (m *Manager) needsDaemon() bool {
	// 检查是否有运行中的任务或自动启动任务
	return m.hasRunningTasks() || m.hasAutoStartTasks()
}

// hasRunningTasks 检查是否有运行中的任务
func (m *Manager) hasRunningTasks() bool {
	state := m.loadRuntimeState()
	if state.Tasks == nil {
		return false
	}
	
	for taskName, runtimeInfo := range state.Tasks {
		// 跳过守护进程本身
		if taskName == "taskd" {
			continue
		}
		
		if runtimeInfo.Status == "running" {
			return true
		}
	}
	
	return false
}

// hasAutoStartTasks 检查是否有自动启动任务
func (m *Manager) hasAutoStartTasks() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, task := range m.tasks {
		if task.config.AutoStart {
			return true
		}
	}
	
	return false
}
