# 输出重定向功能设计文档

## 设计概述

本设计文档描述了 TaskD 输出重定向功能的详细实现方案，包括架构设计、接口定义、数据流程和实现细节。重点关注基础的输出重定向功能，不包含自动日志创建和日志轮替功能。

## 架构设计

### 组件架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Layer     │    │  Config Layer   │    │   IO Layer      │
│                 │    │                 │    │                 │
│ • add command   │───▶│ • Task Config   │───▶│ • IO Manager    │
│ • info command  │    │ • Path Resolver │    │ • File Writer   │
│ • flags parsing │    │ • Validation    │    │ • Stream Merger │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Task Layer    │    │  Manager Layer  │    │  Storage Layer  │
│                 │    │                 │    │                 │
│ • Task Runner   │───▶│ • Task Manager  │───▶│ • Output Files  │
│ • IO Setup      │    │ • State Mgmt    │    │ • Config Files  │
│ • Process Mgmt  │    │ • Lifecycle     │    │ • Runtime State │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 数据流程

1. **任务创建**: CLI → Config → Path Resolver → Task Manager → Storage
2. **任务启动**: Task Manager → Task Runner → IO Setup → File Writers
3. **输出写入**: Process → File Writers → Output Files
4. **信息查看**: CLI → Task Manager → Task Info → Output

## 详细设计

### 1. 配置结构简化

#### 1.1 任务配置结构（保持现有结构）

```go
// Config 任务配置结构（现有结构，无需修改）
type Config struct {
    Name       string   `toml:"name"`
    Executable string   `toml:"executable"`
    Args       []string `toml:"args,omitempty"`
    WorkDir    string   `toml:"workdir,omitempty"`
    Env        []string `toml:"env,omitempty"`
    InheritEnv bool     `toml:"inherit_env"`
    Stdin      string   `toml:"stdin,omitempty"`
    Stdout     string   `toml:"stdout,omitempty"`
    Stderr     string   `toml:"stderr,omitempty"`
    AutoStart  bool     `toml:"auto_start"`
    // ... 其他现有字段
}
```

### 2. 路径处理器

#### 2.1 路径解析器接口

```go
// PathResolver 路径解析器接口
type PathResolver interface {
    // 解析路径（相对路径基于工作目录）
    ResolvePath(path string, workDir string) (string, error)
    
    // 验证路径是否有效
    ValidatePath(path string) error
    
    // 确保目录存在
    EnsureDir(path string) error
}

// DefaultPathResolver 默认路径解析器
type DefaultPathResolver struct{}

// ResolvePath 解析路径
func (r *DefaultPathResolver) ResolvePath(path string, workDir string) (string, error) {
    if filepath.IsAbs(path) {
        return path, nil
    }
    
    // 相对路径基于工作目录
    return filepath.Join(workDir, path), nil
}
```

### 3. IO 管理器

#### 3.1 IO 管理器接口

```go
// IOManager IO 管理器接口
type IOManager interface {
    // 创建任务 IO 设置
    CreateTaskIO(config *Config) (*TaskIO, error)
    
    // 获取任务 IO 信息
    GetTaskIOInfo(config *Config) (*TaskIOInfo, error)
}

// TaskIO 任务 IO 设置
type TaskIO struct {
    Stdin  io.Reader
    Stdout io.Writer
    Stderr io.Writer
    
    // 文件句柄管理
    files []io.Closer
}

// TaskIOInfo 任务 IO 信息
type TaskIOInfo struct {
    StdinPath  string `json:"stdin_path,omitempty"`
    StdoutPath string `json:"stdout_path,omitempty"`
    StderrPath string `json:"stderr_path,omitempty"`
    SameOutput bool   `json:"same_output"` // stdout 和 stderr 是否指向同一文件
}
```

#### 3.2 输出合并处理

```go
// MultiWriter 多写入器，处理 stdout 和 stderr 指向同一文件的情况
type MultiWriter struct {
    writers []io.Writer
    mutex   sync.Mutex
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
    return &MultiWriter{
        writers: writers,
    }
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
    mw.mutex.Lock()
    defer mw.mutex.Unlock()
    
    for _, w := range mw.writers {
        if n, err = w.Write(p); err != nil {
            return
        }
    }
    return len(p), nil
}
```

### 4. CLI 命令扩展

#### 4.1 info 命令实现（合并 status 功能）

```go
var infoCmd = &cobra.Command{
    Use:   "info [task-name]",
    Short: "Show detailed task information", // 替代原 status 命令
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        taskName := args[0]
        
        manager := task.GetManager()
        info, err := manager.GetTaskDetailInfo(taskName) // 扩展的信息获取方法
        if err != nil {
            return fmt.Errorf("failed to get task info: %w", err)
        }
        
        // 显示完整的任务信息（包含原 status 的所有信息）
        displayTaskInfo(info)
        return nil
    },
}

// TaskDetailInfo 详细任务信息（合并原 TaskInfo 的所有字段）
type TaskDetailInfo struct {
    // 基础状态信息（原 status 命令显示的信息）
    Name       string `json:"name"`
    Status     string `json:"status"`
    PID        int    `json:"pid"`
    StartTime  string `json:"start_time"`
    Executable string `json:"executable"`
    ExitCode   int    `json:"exit_code,omitempty"`
    LastError  string `json:"last_error,omitempty"`
    
    // 扩展配置信息
    WorkDir    string   `json:"work_dir"`
    Args       []string `json:"args,omitempty"`
    Env        []string `json:"env,omitempty"`
    InheritEnv bool     `json:"inherit_env"`
    
    // IO 重定向信息
    IOInfo     *TaskIOInfo `json:"io_info"`
}

func displayTaskInfo(info *TaskDetailInfo) {
    // 显示基础状态信息（原 status 命令的内容）
    fmt.Printf("Task Name: %s\n", info.Name)
    fmt.Printf("Status: %s\n", info.Status)
    fmt.Printf("Process ID: %d\n", info.PID)
    fmt.Printf("Start Time: %s\n", info.StartTime)
    fmt.Printf("Executable: %s\n", info.Executable)
    
    // 显示退出信息
    if info.ExitCode != 0 {
        fmt.Printf("Exit Code: %d\n", info.ExitCode)
    }
    if info.LastError != "" {
        fmt.Printf("Last Error: %s\n", info.LastError)
    }
    
    // 显示配置信息
    fmt.Printf("Working Directory: %s\n", info.WorkDir)
    if len(info.Args) > 0 {
        fmt.Printf("Arguments: %v\n", info.Args)
    }
    if len(info.Env) > 0 {
        fmt.Printf("Environment Variables: %v\n", info.Env)
    }
    fmt.Printf("Inherit Environment: %t\n", info.InheritEnv)
    
    // 显示 IO 重定向信息
    if info.IOInfo.StdinPath != "" {
        fmt.Printf("Standard Input: %s\n", info.IOInfo.StdinPath)
    }
    if info.IOInfo.StdoutPath != "" {
        fmt.Printf("Standard Output: %s\n", info.IOInfo.StdoutPath)
    }
    if info.IOInfo.StderrPath != "" {
        fmt.Printf("Standard Error: %s\n", info.IOInfo.StderrPath)
    }
    if info.IOInfo.SameOutput {
        fmt.Printf("Note: Standard output and error are redirected to the same file\n")
    }
}
```

### 5. 实现细节

#### 5.1 增强的 IO 设置

```go
// setupIO 设置任务的输入输出重定向（增强现有方法）
func (t *Task) setupIO(cmd *exec.Cmd) error {
    ioManager := GetIOManager()
    taskIO, err := ioManager.CreateTaskIO(t.config)
    if err != nil {
        return fmt.Errorf("failed to create task IO: %w", err)
    }
    
    // 保存 IO 引用用于清理
    t.taskIO = taskIO
    
    // 设置标准输入输出
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

// CreateTaskIO 创建任务 IO 设置
func (m *DefaultIOManager) CreateTaskIO(config *Config) (*TaskIO, error) {
    taskIO := &TaskIO{}
    pathResolver := &DefaultPathResolver{}
    
    // 处理标准输入
    if config.Stdin != "" {
        stdinPath, err := pathResolver.ResolvePath(config.Stdin, config.WorkDir)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve stdin path: %w", err)
        }
        
        file, err := os.Open(stdinPath)
        if err != nil {
            return nil, fmt.Errorf("failed to open stdin file %s: %w", stdinPath, err)
        }
        
        taskIO.Stdin = file
        taskIO.files = append(taskIO.files, file)
    }
    
    // 处理标准输出和错误
    var stdoutWriter, stderrWriter io.Writer
    
    if config.Stdout != "" {
        stdoutPath, err := pathResolver.ResolvePath(config.Stdout, config.WorkDir)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve stdout path: %w", err)
        }
        
        if err := pathResolver.EnsureDir(filepath.Dir(stdoutPath)); err != nil {
            return nil, fmt.Errorf("failed to create stdout directory: %w", err)
        }
        
        file, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return nil, fmt.Errorf("failed to open stdout file %s: %w", stdoutPath, err)
        }
        
        stdoutWriter = file
        taskIO.files = append(taskIO.files, file)
    }
    
    if config.Stderr != "" {
        stderrPath, err := pathResolver.ResolvePath(config.Stderr, config.WorkDir)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve stderr path: %w", err)
        }
        
        // 检查是否与 stdout 相同
        if config.Stdout != "" {
            stdoutPath, _ := pathResolver.ResolvePath(config.Stdout, config.WorkDir)
            if stdoutPath == stderrPath {
                // 使用同一个写入器
                stderrWriter = stdoutWriter
            }
        }
        
        if stderrWriter == nil {
            if err := pathResolver.EnsureDir(filepath.Dir(stderrPath)); err != nil {
                return nil, fmt.Errorf("failed to create stderr directory: %w", err)
            }
            
            file, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
            if err != nil {
                return nil, fmt.Errorf("failed to open stderr file %s: %w", stderrPath, err)
            }
            
            stderrWriter = file
            taskIO.files = append(taskIO.files, file)
        }
    }
    
    taskIO.Stdout = stdoutWriter
    taskIO.Stderr = stderrWriter
    
    return taskIO, nil
}
```

### 6. 错误处理策略

#### 6.1 文件操作错误

```go
// 文件创建失败处理
func handleFileError(path string, err error) error {
    if os.IsPermission(err) {
        return fmt.Errorf("permission denied: cannot access file %s", path)
    }
    if os.IsNotExist(err) {
        return fmt.Errorf("file or directory does not exist: %s", path)
    }
    return fmt.Errorf("file operation failed for %s: %w", path, err)
}
```

## 实现计划

### 阶段 1: 路径处理和 IO 管理（1 天）
1. 实现路径解析器
2. 实现 IO 管理器
3. 处理相对路径解析

### 阶段 2: 输出合并和文件处理（1 天）
1. 实现输出合并逻辑
2. 增强文件操作
3. 添加错误处理

### 阶段 3: CLI 命令扩展（1 天）
1. 实现 info 命令
2. 更新任务信息显示
3. 集成到现有系统

### 阶段 4: 测试和优化（1 天）
1. 单元测试
2. 集成测试
3. 跨平台测试

## 测试策略

### 单元测试覆盖
- [ ] 路径解析测试
- [ ] IO 管理器测试
- [ ] 文件操作测试
- [ ] 错误处理测试

### 集成测试覆盖
- [ ] 端到端输出重定向测试
- [ ] 输出合并测试
- [ ] 跨平台兼容性测试

这个简化的设计专注于核心的输出重定向功能，移除了复杂的日志轮替和自动日志创建功能，更符合实际需求。