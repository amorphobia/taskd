# TaskD 守护进程设计文档

## 概述

本设计文档描述了 TaskD 守护进程机制的技术实现方案。该机制将替换现有的 `onTaskExit` 回调机制，通过一个独立的守护进程来监控任务状态并执行自动重启逻辑，解决命令进程退出时状态更新失效的问题。

## 架构设计

### 系统架构图

```
┌─────────────────┐    启动守护进程    ┌─────────────────┐
│   命令进程      │ ──────────────→   │   守护进程      │
│ (taskd start)   │                   │ (taskd --daemon)│
└─────────────────┘                   └─────────────────┘
         │                                     │
         │ 更新状态                             │ 监控任务
         ↓                                     ↓
┌─────────────────────────────────────────────────────────┐
│                runtime.json                            │
│  - 任务运行状态                                          │
│  - PID 信息                                            │
│  - 重试计数                                             │
│  - 停止标记                                             │
└─────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. 守护进程管理器 (DaemonManager)
- 负责守护进程的启动、停止和状态检查
- 管理守护进程的生命周期
- 处理守护进程的自动启动逻辑

#### 2. 任务监控器 (TaskMonitor)
- 在守护进程中运行，定时检查任务状态
- 执行自动重启逻辑
- 更新运行时状态信息

#### 3. 内置任务处理器 (BuiltinTaskHandler)
- 处理内置任务 `taskd` 的特殊逻辑
- 实现内置任务的操作限制
- 提供虚拟配置信息

## 详细设计

### 1. 数据结构扩展

#### 1.1 任务配置扩展
```go
// Config 任务配置结构扩展
type Config struct {
    // ... 现有字段 ...
    AutoStart    bool `toml:"auto_start"`     // 现有字段
    MaxRetryNum  int  `toml:"max_retry_num"`  // 新增：最大重试次数，默认3
}
```

#### 1.2 运行时状态扩展
```go
// TaskRuntimeInfo 运行时信息扩展
type TaskRuntimeInfo struct {
    // ... 现有字段 ...
    StoppedByTaskd bool `json:"stopped_by_taskd"` // 新增：是否由 taskd stop 停止
    RetryNum       int  `json:"retry_num"`        // 新增：当前重试次数
}
```

#### 1.3 守护进程状态
```go
// DaemonStatus 守护进程状态
type DaemonStatus struct {
    IsRunning   bool      `json:"is_running"`
    PID         int       `json:"pid"`
    StartTime   time.Time `json:"start_time"`
    LastCheck   time.Time `json:"last_check"`
}
```

### 2. 核心组件实现

#### 2.1 守护进程管理器
```go
// DaemonManager 守护进程管理器
type DaemonManager struct {
    mu sync.RWMutex
}

// StartDaemon 启动守护进程
func (dm *DaemonManager) StartDaemon() error {
    // 1. 检查现有守护进程状态
    // 2. 如果需要，启动新的守护进程
    // 3. 更新运行时状态
}

// StopDaemon 停止守护进程
func (dm *DaemonManager) StopDaemon() error {
    // 1. 查找守护进程
    // 2. 终止进程
    // 3. 更新运行时状态
}

// IsRunning 检查守护进程是否运行
func (dm *DaemonManager) IsRunning() bool {
    // 检查 runtime.json 中的守护进程状态
    // 验证 PID 对应的进程是否为 taskd
}

// EnsureDaemonRunning 确保守护进程运行
func (dm *DaemonManager) EnsureDaemonRunning() error {
    // 如果守护进程不存在，自动启动
}
```

#### 2.2 任务监控器
```go
// TaskMonitor 任务监控器（在守护进程中运行）
type TaskMonitor struct {
    checkInterval time.Duration
    stopChan      chan struct{}
}

// Start 启动监控循环
func (tm *TaskMonitor) Start() {
    ticker := time.NewTicker(tm.checkInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            tm.checkAndRestartTasks()
        case <-tm.stopChan:
            return
        }
    }
}

// checkAndRestartTasks 检查并重启任务
func (tm *TaskMonitor) checkAndRestartTasks() {
    // 1. 读取 runtime.json 获取当前状态
    // 2. 检查每个运行中任务的进程是否仍然存在
    // 3. 对于已退出的任务，更新状态信息（退出码、结束时间等）
    // 4. 对符合自动重启条件的任务执行重启
    // 5. 更新重试计数和状态文件
}
```

#### 2.3 内置任务处理器
```go
// BuiltinTaskHandler 内置任务处理器
type BuiltinTaskHandler struct{}

// IsBuiltinTask 检查是否为内置任务
func (bth *BuiltinTaskHandler) IsBuiltinTask(name string) bool {
    return name == "taskd"
}

// GetBuiltinTaskConfig 获取内置任务配置
func (bth *BuiltinTaskHandler) GetBuiltinTaskConfig(name string) *Config {
    if name == "taskd" {
        return &Config{
            DisplayName: "taskd",
            Description: "The daemon task of taskd",
            Executable:  getCurrentExecutablePath() + " --daemon",
            WorkDir:     taskdconfig.GetTaskDHome(),
            InheritEnv:  true,
        }
    }
    return nil
}

// ValidateOperation 验证操作是否允许
func (bth *BuiltinTaskHandler) ValidateOperation(name, operation string) error {
    if name == "taskd" {
        switch operation {
        case "add", "edit", "del":
            return fmt.Errorf("operation '%s' not allowed for builtin task '%s'", operation, name)
        }
    }
    return nil
}
```

### 3. 命令行参数处理

#### 3.1 --daemon 参数处理
```go
// 在 root.go 中添加
var daemonMode bool

func init() {
    rootCmd.PersistentFlags().BoolVar(&daemonMode, "daemon", false, "run in daemon mode (internal use only)")
    
    // 隐藏 daemon 参数，不在帮助中显示
    rootCmd.PersistentFlags().MarkHidden("daemon")
}

// 在 rootCmd.PreRunE 中添加验证
func validateDaemonFlag(cmd *cobra.Command, args []string) error {
    if daemonMode {
        // 检查是否有其他参数
        if len(args) > 0 || cmd.Flags().NFlag() > 1 {
            return fmt.Errorf("--daemon flag cannot be used with other arguments")
        }
    }
    return nil
}
```

#### 3.2 守护进程模式入口
```go
// 在 main.go 中添加
func main() {
    if daemonMode {
        runDaemonMode()
        return
    }
    
    // 正常命令模式
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}

func runDaemonMode() {
    monitor := &TaskMonitor{
        checkInterval: 5 * time.Second,
        stopChan:      make(chan struct{}),
    }
    
    // 设置信号处理
    setupSignalHandling(monitor)
    
    // 启动监控
    monitor.Start()
}
```

### 4. 自动启动机制

#### 4.1 触发条件
守护进程自动启动的时机：
1. 执行任务管理命令时（start, stop, restart, list, info）
2. 启动 auto_start=true 的任务时
3. 系统检测到需要状态监控但守护进程不存在时

#### 4.2 实现方式
```go
// 在每个需要守护进程的命令中添加
func ensureDaemonForCommand() error {
    dm := GetDaemonManager()
    
    // 检查是否有运行中的任务或自动启动任务
    if needsDaemon() {
        return dm.EnsureDaemonRunning()
    }
    return nil
}

func needsDaemon() bool {
    // 检查是否有运行中的任务
    // 检查是否有 auto_start=true 的任务
    return hasRunningTasks() || hasAutoStartTasks()
}
```

### 5. 重试逻辑实现

#### 5.1 重试条件判断
```go
func shouldRetryTask(runtimeInfo *TaskRuntimeInfo, config *Config) bool {
    return config.AutoStart &&
           runtimeInfo.Status == "stopped" &&
           !runtimeInfo.StoppedByTaskd &&
           (config.MaxRetryNum <= 0 || runtimeInfo.RetryNum < config.MaxRetryNum)
}
```

#### 5.2 重试执行
```go
func (tm *TaskMonitor) retryTask(taskName string) error {
    // 1. 启动任务
    manager := task.GetManager()
    if err := manager.StartTask(taskName); err != nil {
        return err
    }
    
    // 2. 更新重试计数
    return tm.incrementRetryCount(taskName)
}
```

### 6. 状态管理

#### 6.1 状态更新时机
- **命令进程**：负责守护进程的启动/停止状态更新，以及用户主动停止任务时设置 `stopped_by_taskd=true`
- **守护进程**：负责所有任务状态的监控和更新，包括任务退出检测、状态变更和自动重启

#### 6.2 状态同步机制
```go
// 状态更新接口
type StateUpdater interface {
    UpdateTaskState(name string, info *TaskRuntimeInfo) error
    UpdateDaemonState(status *DaemonStatus) error
    GetRuntimeState() (*RuntimeState, error)
}

// 文件状态管理器
type FileStateManager struct {
    statePath string
    mu        sync.RWMutex
}

// 进程状态检测器
type ProcessChecker struct{}

// CheckTaskProcess 检查任务进程状态
func (pc *ProcessChecker) CheckTaskProcess(pid int) (*ProcessStatus, error) {
    // 1. 检查进程是否存在
    // 2. 获取进程退出状态（如果已退出）
    // 3. 验证进程是否为预期的任务进程
    // 4. 返回进程状态信息
}
```

## 错误处理策略

### 1. 守护进程启动失败
- 记录详细错误信息
- 提供用户友好的错误提示
- 不影响其他命令的执行

### 2. 任务重启失败
- 记录失败原因
- 继续监控其他任务
- 达到重试上限后停止重试

### 3. 状态文件损坏
- 尝试备份恢复
- 重新初始化状态文件
- 记录警告信息

## 性能考虑

### 1. 监控间隔
- 默认检查间隔：5秒
- 可通过配置文件调整
- 避免过度的 CPU 和 I/O 消耗

### 2. 文件 I/O 优化
- 批量状态更新
- 避免频繁的文件读写
- 使用文件锁防止并发冲突

### 3. 内存使用
- 及时清理不需要的资源
- 避免内存泄漏
- 合理的数据结构设计

## 安全考虑

### 1. 进程权限
- 守护进程以用户权限运行
- 不需要管理员权限
- 限制文件访问权限

### 2. 参数验证
- 严格验证 --daemon 参数使用
- 防止恶意参数注入
- 输入数据校验

## 兼容性

### 1. 向后兼容
- 现有任务配置保持兼容
- 新增字段使用默认值
- 运行时状态格式扩展

### 2. 平台兼容
- 支持 Windows 平台
- 正确的进程管理
- 文件路径处理

## 测试策略

### 1. 单元测试
- 守护进程管理器测试
- 任务监控器测试
- 内置任务处理器测试

### 2. 集成测试
- 完整的守护进程生命周期测试
- 自动重启功能测试
- 错误恢复测试

### 3. 性能测试
- 监控性能影响测试
- 大量任务场景测试
- 长时间运行稳定性测试

## 正确性属性

基于需求文档的验收标准，定义以下正确性属性：

### Property 1: 内置任务保护
**验证需求**: AC1.1, AC1.2, AC1.3, AC1.4, AC1.5
```
对于任何用户操作 op ∈ {add, edit, del} 和内置任务名称 "taskd"：
operation_result(op, "taskd") = Error
```

### Property 2: 守护进程唯一性
**验证需求**: AC4.1, AC4.2, AC4.3, AC4.4
```
在任何时刻，系统中最多只能有一个有效的 taskd 守护进程运行：
∀ time t: count(valid_daemon_processes(t)) ≤ 1
```

### Property 3: 进程分离正确性
**验证需求**: AC3.1, AC3.2
```
当启动守护进程时：
start_daemon() → ∃ daemon_process: 
  daemon_process.executable.contains("--daemon") ∧
  daemon_process.parent ≠ command_process
```

### Property 4: 自动重启条件
**验证需求**: AC8.1, AC8.2, AC8.3, AC8.4
```
对于任务 task，当且仅当满足以下条件时执行自动重启：
should_auto_restart(task) ↔ 
  task.auto_start = true ∧
  task.status = "stopped" ∧
  task.stopped_by_taskd = false ∧
  task.retry_num < task.max_retry_num
```

### Property 5: 状态一致性
**验证需求**: AC6.1, AC6.2, AC6.3
```
运行时状态文件中的信息与实际进程状态保持一致：
∀ task ∈ runtime.json.tasks:
  task.status = "running" → ∃ process: process.pid = task.pid ∧ is_alive(process)
```

### Property 6: 重试计数正确性
**验证需求**: AC7.1, AC7.2, AC9.3, AC9.4
```
重试计数的更新遵循以下规则：
1. 手动启动时重置：manual_start(task) → task.retry_num = 0
2. 自动重启时递增：auto_restart(task) → task.retry_num = task.retry_num + 1
3. 达到上限时停止：task.retry_num ≥ task.max_retry_num → ¬should_auto_restart(task)
```

### Property 7: 守护进程监控持续性
**验证需求**: AC5.5, AC9.1, AC9.5
```
守护进程启动后持续监控，直到被显式停止：
daemon_started(t₀) ∧ ¬explicit_stop(t₀, t) → 
  ∀ t' ∈ [t₀, t]: monitoring_active(t')
```

## 实现计划

### 阶段1：基础架构
1. 实现守护进程管理器
2. 扩展数据结构
3. 添加 --daemon 参数处理

### 阶段2：核心功能
1. 实现任务监控器
2. 实现内置任务处理器
3. 集成自动启动机制

### 阶段3：完善功能
1. 实现自动重启逻辑
2. 完善错误处理
3. 性能优化

### 阶段4：测试和文档
1. 编写测试用例
2. 性能测试
3. 更新文档