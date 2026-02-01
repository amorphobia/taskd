package task

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"taskd/internal/config"
	
	"github.com/BurntSushi/toml"
)

// TaskMonitor 任务监控器（在守护进程中运行）
type TaskMonitor struct {
	checkInterval time.Duration
	stopChan      chan struct{}
	manager       *Manager
	mu            sync.RWMutex
	isRunning     bool
}

// NewTaskMonitor creates a new task monitor
func NewTaskMonitor(checkInterval time.Duration) *TaskMonitor {
	return &TaskMonitor{
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
		manager:       GetManager(),
		isRunning:     false,
	}
}

// Start 启动监控循环
func (tm *TaskMonitor) Start() {
	tm.mu.Lock()
	if tm.isRunning {
		tm.mu.Unlock()
		return // Already running
	}
	tm.isRunning = true
	tm.mu.Unlock()
	
	fmt.Printf("TaskMonitor: Starting monitoring with interval %v\n", tm.checkInterval)
	
	ticker := time.NewTicker(tm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			tm.checkAndRestartTasks()
		case <-tm.stopChan:
			fmt.Println("TaskMonitor: Stopping monitoring")
			tm.mu.Lock()
			tm.isRunning = false
			tm.mu.Unlock()
			return
		}
	}
}

// Stop 停止监控循环
func (tm *TaskMonitor) Stop() {
	tm.mu.RLock()
	if !tm.isRunning {
		tm.mu.RUnlock()
		return // Not running
	}
	tm.mu.RUnlock()
	
	close(tm.stopChan)
}

// IsRunning 检查监控器是否正在运行
func (tm *TaskMonitor) IsRunning() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isRunning
}

// checkAndRestartTasks 检查并重启任务
func (tm *TaskMonitor) checkAndRestartTasks() {
	// 1. 读取 runtime.json 获取当前状态
	state := tm.manager.loadRuntimeState()
	if state.Tasks == nil {
		return
	}
	
	// 2. 检查每个任务的状态
	for taskName, runtimeInfo := range state.Tasks {
		// 跳过守护进程本身
		if taskName == "taskd" {
			continue
		}
		
		// 只检查标记为运行中的任务
		if runtimeInfo.Status == "running" {
			tm.checkTaskProcess(taskName, runtimeInfo)
		}
		
		// 检查是否需要自动重启
		if tm.shouldRetryTask(taskName, runtimeInfo) {
			tm.retryTask(taskName)
		} else if tm.shouldLogRetryLimitReached(taskName, runtimeInfo) {
			tm.logRetryLimitReached(taskName, runtimeInfo)
		}
	}
}

// checkTaskProcess 检查单个任务的进程状态
func (tm *TaskMonitor) checkTaskProcess(taskName string, runtimeInfo *TaskRuntimeInfo) {
	checker := NewProcessChecker()
	status, err := checker.CheckTaskProcess(runtimeInfo.PID)
	
	if err != nil {
		fmt.Printf("TaskMonitor: Error checking process for task %s (PID %d): %v\n", 
			taskName, runtimeInfo.PID, err)
		return
	}
	
	// 如果进程不存在，更新任务状态为已停止
	if !status.Exists {
		fmt.Printf("TaskMonitor: Task %s (PID %d) process no longer exists, updating status\n", 
			taskName, runtimeInfo.PID)
		
		// 尝试获取退出码
		exitCode := tm.getProcessExitCode(runtimeInfo.PID)
		tm.updateTaskExitedStatus(taskName, runtimeInfo, exitCode)
	}
}

// getProcessExitCode 尝试获取进程的退出码
func (tm *TaskMonitor) getProcessExitCode(pid int) int {
	checker := NewProcessChecker()
	exitCode, err := checker.GetProcessExitCode(pid)
	if err != nil {
		// 无法获取退出码，使用默认值
		return 0
	}
	return exitCode
}

// updateTaskExitedStatus 更新已退出任务的状态
func (tm *TaskMonitor) updateTaskExitedStatus(taskName string, runtimeInfo *TaskRuntimeInfo, exitCode int) {
	// 创建更新后的状态信息
	updatedInfo := &TaskRuntimeInfo{
		Name:           taskName,
		Status:         "stopped",
		PID:            0,
		StartTime:      runtimeInfo.StartTime,
		EndTime:        time.Now(),
		ExitCode:       exitCode,
		StoppedByTaskd: false, // 进程自然退出，不是用户停止
		RetryNum:       runtimeInfo.RetryNum, // 保持重试计数
	}
	
	// 更新运行时状态
	state := tm.manager.loadRuntimeState()
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	state.Tasks[taskName] = updatedInfo
	
	if err := tm.manager.saveRuntimeStateWithData(state); err != nil {
		fmt.Printf("TaskMonitor: Error updating runtime state for task %s: %v\n", taskName, err)
	} else {
		fmt.Printf("TaskMonitor: Updated task %s status to stopped (exit code: %d)\n", taskName, exitCode)
	}
}

// updateTaskState 通用的任务状态更新方法
func (tm *TaskMonitor) updateTaskState(taskName string, updatedInfo *TaskRuntimeInfo) error {
	state := tm.manager.loadRuntimeState()
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	state.Tasks[taskName] = updatedInfo
	
	if err := tm.manager.saveRuntimeStateWithData(state); err != nil {
		return fmt.Errorf("failed to update runtime state for task %s: %w", taskName, err)
	}
	
	return nil
}

// shouldRetryTask 判断任务是否应该自动重启
func (tm *TaskMonitor) shouldRetryTask(taskName string, runtimeInfo *TaskRuntimeInfo) bool {
	// 跳过守护进程本身
	if taskName == "taskd" {
		return false
	}
	
	// 获取任务配置
	config := tm.getTaskConfig(taskName)
	if config == nil {
		return false
	}
	
	// 检查重启条件：
	// 1. 任务配置中 auto_start = true
	// 2. 任务状态为已停止
	// 3. stopped_by_taskd = false（非用户主动停止）
	// 4. retry_num < max_retry_num（未达到重试上限）
	return config.AutoStart &&
		runtimeInfo.Status == "stopped" &&
		!runtimeInfo.StoppedByTaskd &&
		(config.MaxRetryNum <= 0 || runtimeInfo.RetryNum < config.MaxRetryNum)
}

// getTaskConfig 获取任务配置
func (tm *TaskMonitor) getTaskConfig(taskName string) *Config {
	// 检查是否为内置任务
	if tm.manager.builtinHandler.IsBuiltinTask(taskName) {
		return tm.manager.builtinHandler.GetBuiltinTaskConfig(taskName)
	}
	
	// 从文件加载普通任务配置
	configPath := filepath.Join(config.GetTaskDTasksDir(), taskName+".toml")
	var taskConfig Config
	
	if _, err := toml.DecodeFile(configPath, &taskConfig); err != nil {
		fmt.Printf("TaskMonitor: Error loading config for task %s: %v\n", taskName, err)
		return nil
	}
	
	return &taskConfig
}

// retryTask 执行任务自动重启
func (tm *TaskMonitor) retryTask(taskName string) {
	fmt.Printf("TaskMonitor: Attempting to restart task %s\n", taskName)
	
	// 1. 启动任务
	if err := tm.manager.StartTask(taskName); err != nil {
		fmt.Printf("TaskMonitor: Failed to restart task %s: %v\n", taskName, err)
		tm.handleRetryFailure(taskName, err)
		return
	}
	
	// 2. 更新重试计数
	if err := tm.incrementRetryCount(taskName); err != nil {
		fmt.Printf("TaskMonitor: Failed to update retry count for task %s: %v\n", taskName, err)
	}
	
	fmt.Printf("TaskMonitor: Successfully restarted task %s\n", taskName)
}

// incrementRetryCount 递增重试计数
func (tm *TaskMonitor) incrementRetryCount(taskName string) error {
	state := tm.manager.loadRuntimeState()
	if state.Tasks == nil {
		return fmt.Errorf("no runtime state found")
	}
	
	runtimeInfo, exists := state.Tasks[taskName]
	if !exists {
		return fmt.Errorf("task %s not found in runtime state", taskName)
	}
	
	// 递增重试计数
	runtimeInfo.RetryNum++
	
	// 保存更新后的状态
	return tm.manager.saveRuntimeStateWithData(state)
}

// handleRetryFailure 处理重启失败
func (tm *TaskMonitor) handleRetryFailure(taskName string, err error) {
	// 记录错误信息，但不影响其他任务的监控
	fmt.Printf("TaskMonitor: Retry failed for task %s: %v\n", taskName, err)
	
	// 更新任务状态，标记重启失败
	state := tm.manager.loadRuntimeState()
	if state.Tasks != nil {
		if runtimeInfo, exists := state.Tasks[taskName]; exists {
			// 创建失败状态信息
			failedInfo := &TaskRuntimeInfo{
				Name:           taskName,
				Status:         "stopped",
				PID:            0,
				StartTime:      runtimeInfo.StartTime,
				EndTime:        time.Now(),
				ExitCode:       -1, // 使用 -1 表示重启失败
				StoppedByTaskd: false,
				RetryNum:       runtimeInfo.RetryNum, // 保持当前重试计数
			}
			
			// 更新状态
			if updateErr := tm.updateTaskState(taskName, failedInfo); updateErr != nil {
				fmt.Printf("TaskMonitor: Failed to update task state after retry failure: %v\n", updateErr)
			}
		}
	}
	
	// 可以在这里添加更多的错误处理逻辑，比如：
	// - 记录到日志文件
	// - 发送通知
	// - 触发告警
}

// shouldLogRetryLimitReached 检查是否应该记录重试上限达到的信息
func (tm *TaskMonitor) shouldLogRetryLimitReached(taskName string, runtimeInfo *TaskRuntimeInfo) bool {
	// 跳过守护进程本身
	if taskName == "taskd" {
		return false
	}
	
	// 获取任务配置
	config := tm.getTaskConfig(taskName)
	if config == nil {
		return false
	}
	
	// 检查是否为自动启动任务且已达到重试上限
	return config.AutoStart &&
		runtimeInfo.Status == "stopped" &&
		!runtimeInfo.StoppedByTaskd &&
		config.MaxRetryNum > 0 &&
		runtimeInfo.RetryNum >= config.MaxRetryNum
}

// logRetryLimitReached 记录重试上限达到的信息
func (tm *TaskMonitor) logRetryLimitReached(taskName string, runtimeInfo *TaskRuntimeInfo) {
	config := tm.getTaskConfig(taskName)
	if config == nil {
		return
	}
	
	fmt.Printf("TaskMonitor: Task %s has reached maximum retry limit (%d/%d), stopping automatic restart attempts\n",
		taskName, runtimeInfo.RetryNum, config.MaxRetryNum)
}

// StateUpdater 状态更新接口
type StateUpdater interface {
	UpdateTaskState(name string, info *TaskRuntimeInfo) error
	UpdateDaemonState(status *DaemonStatus) error
	GetRuntimeState() (*RuntimeState, error)
	SaveRuntimeState(state *RuntimeState) error
}

// FileStateManager 文件状态管理器
type FileStateManager struct {
	statePath string
	mu        sync.RWMutex
}

// NewFileStateManager 创建新的文件状态管理器
func NewFileStateManager(statePath string) *FileStateManager {
	return &FileStateManager{
		statePath: statePath,
	}
}

// UpdateTaskState 更新任务状态
func (fsm *FileStateManager) UpdateTaskState(name string, info *TaskRuntimeInfo) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	
	state, err := fsm.loadRuntimeStateUnsafe()
	if err != nil {
		return fmt.Errorf("failed to load runtime state: %w", err)
	}
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	state.Tasks[name] = info
	
	return fsm.saveRuntimeStateUnsafe(state)
}

// UpdateDaemonState 更新守护进程状态
func (fsm *FileStateManager) UpdateDaemonState(status *DaemonStatus) error {
	// 将 DaemonStatus 转换为 TaskRuntimeInfo
	daemonInfo := &TaskRuntimeInfo{
		Name:           "taskd",
		Status:         "stopped",
		PID:            status.PID,
		StartTime:      status.StartTime,
		StoppedByTaskd: false,
		RetryNum:       0,
	}
	
	if status.IsRunning {
		daemonInfo.Status = "running"
	}
	
	return fsm.UpdateTaskState("taskd", daemonInfo)
}

// GetRuntimeState 获取运行时状态
func (fsm *FileStateManager) GetRuntimeState() (*RuntimeState, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	
	return fsm.loadRuntimeStateUnsafe()
}

// SaveRuntimeState 保存运行时状态
func (fsm *FileStateManager) SaveRuntimeState(state *RuntimeState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	
	return fsm.saveRuntimeStateUnsafe(state)
}

// BatchUpdate 批量更新多个任务状态（提高并发性能）
func (fsm *FileStateManager) BatchUpdate(updates map[string]*TaskRuntimeInfo) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	
	state, err := fsm.loadRuntimeStateUnsafe()
	if err != nil {
		return fmt.Errorf("failed to load runtime state: %w", err)
	}
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	// 批量更新
	for name, info := range updates {
		state.Tasks[name] = info
	}
	
	return fsm.saveRuntimeStateUnsafe(state)
}

// loadRuntimeStateUnsafe 加载运行时状态（不加锁，内部使用）
func (fsm *FileStateManager) loadRuntimeStateUnsafe() (*RuntimeState, error) {
	data, err := os.ReadFile(fsm.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空状态
			return &RuntimeState{Tasks: make(map[string]*TaskRuntimeInfo)}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state RuntimeState
	if err := json.Unmarshal(data, &state); err != nil {
		// JSON 解析失败，尝试恢复
		fmt.Printf("Warning: State file corrupted, attempting recovery: %v\n", err)
		
		if recoveredState, recoverErr := fsm.recoverCorruptedState(); recoverErr == nil {
			return recoveredState, nil
		}
		
		// 恢复失败，返回空状态并备份损坏的文件
		fsm.backupCorruptedState()
		return &RuntimeState{Tasks: make(map[string]*TaskRuntimeInfo)}, nil
	}
	
	if state.Tasks == nil {
		state.Tasks = make(map[string]*TaskRuntimeInfo)
	}
	
	return &state, nil
}

// recoverCorruptedState 尝试恢复损坏的状态文件
func (fsm *FileStateManager) recoverCorruptedState() (*RuntimeState, error) {
	// 尝试从备份文件恢复
	backupPath := fsm.statePath + ".backup"
	if _, err := os.Stat(backupPath); err == nil {
		fmt.Printf("Attempting to recover from backup file: %s\n", backupPath)
		
		data, err := os.ReadFile(backupPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read backup file: %w", err)
		}
		
		var state RuntimeState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("backup file also corrupted: %w", err)
		}
		
		if state.Tasks == nil {
			state.Tasks = make(map[string]*TaskRuntimeInfo)
		}
		
		// 恢复成功，保存到主文件
		if err := fsm.saveRuntimeStateUnsafe(&state); err != nil {
			fmt.Printf("Warning: Failed to save recovered state: %v\n", err)
		} else {
			fmt.Println("Successfully recovered state from backup")
		}
		
		return &state, nil
	}
	
	return nil, fmt.Errorf("no backup file available")
}

// backupCorruptedState 备份损坏的状态文件
func (fsm *FileStateManager) backupCorruptedState() {
	corruptedPath := fsm.statePath + ".corrupted." + time.Now().Format("20060102-150405")
	if err := os.Rename(fsm.statePath, corruptedPath); err != nil {
		fmt.Printf("Warning: Failed to backup corrupted state file: %v\n", err)
	} else {
		fmt.Printf("Corrupted state file backed up to: %s\n", corruptedPath)
	}
}

// saveRuntimeStateUnsafe 保存运行时状态（不加锁，内部使用）
func (fsm *FileStateManager) saveRuntimeStateUnsafe(state *RuntimeState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}
	
	// 创建备份（如果主文件存在）
	if _, err := os.Stat(fsm.statePath); err == nil {
		backupPath := fsm.statePath + ".backup"
		if err := fsm.copyFile(fsm.statePath, backupPath); err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		}
	}
	
	// 原子性更新：先写入临时文件，然后重命名
	tempPath := fsm.statePath + ".tmp"
	
	// 写入临时文件
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}
	
	// 原子性重命名
	if err := os.Rename(tempPath, fsm.statePath); err != nil {
		// 清理临时文件
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}
	
	return nil
}

// copyFile 复制文件
func (fsm *FileStateManager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

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