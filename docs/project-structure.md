# TaskD 项目结构

```
taskd/
├── cmd/                          # 应用程序入口
│   └── taskd/
│       └── main.go              # 主程序入口
├── internal/                     # 内部包，不对外暴露
│   ├── cli/                     # 命令行接口
│   │   ├── root.go              # 根命令和全局配置
│   │   ├── add.go               # 添加任务命令
│   │   ├── list.go              # 列出任务命令
│   │   └── start.go             # 启动/停止/状态命令
│   ├── task/                    # 任务管理核心
│   │   ├── config.go            # 任务配置结构
│   │   ├── manager.go           # 任务管理器
│   │   └── task.go              # 任务实例
│   └── config/                  # 全局配置管理
│       └── config.go            # 配置文件处理
├── docs/                        # 文档
│   ├── development-plan.md      # 开发计划
│   ├── project-structure.md     # 项目结构说明
│   └── config-example.toml      # 配置文件示例
├── examples/                    # 使用示例
│   └── simple-usage.md          # 基础使用示例
├── scripts/                     # 构建和部署脚本
│   └── build.bat               # Windows 构建脚本
├── build/                       # 构建输出目录 (gitignore)
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖版本锁定
├── Makefile                     # 构建配置
├── README.md                    # 项目说明
└── .gitignore                   # Git 忽略文件
```

## 目录说明

### `/cmd`
应用程序的入口点。每个子目录代表一个可执行程序。

- `cmd/taskd/main.go`: TaskD 主程序入口

### `/internal`
私有应用程序代码，这些代码不希望被其他应用程序或库导入。

#### `/internal/cli`
命令行接口实现，使用 Cobra 框架：
- `root.go`: 根命令定义和全局配置
- `add.go`: 任务添加命令
- `list.go`: 任务列表命令  
- `start.go`: 任务启动、停止、状态查询命令

#### `/internal/task`
任务管理的核心逻辑：
- `config.go`: 任务配置数据结构
- `manager.go`: 任务管理器，负责任务的 CRUD 操作
- `task.go`: 单个任务的生命周期管理

#### `/internal/config`
全局配置管理：
- `config.go`: 全局配置文件处理，使用 Viper

### `/docs`
项目文档：
- `development-plan.md`: 详细的开发计划和里程碑
- `project-structure.md`: 项目结构说明
- `config-example.toml`: 配置文件示例

### `/examples`
使用示例和教程：
- `simple-usage.md`: 基础使用方法和常见场景

### `/scripts`
构建、安装、分析等脚本：
- `build.bat`: Windows 平台构建脚本

## 设计原则

### 1. 清晰的分层架构
- **CLI 层**: 处理用户输入和输出
- **业务逻辑层**: 任务管理核心功能
- **配置层**: 配置文件管理

### 2. 模块化设计
- 每个包都有明确的职责
- 包之间的依赖关系清晰
- 便于测试和维护

### 3. 跨平台兼容
- 使用 Go 标准库的跨平台 API
- 平台特定代码通过构建标签隔离
- 配置文件使用通用格式 (TOML)

### 4. 可扩展性
- 插件化的命令结构
- 可配置的任务属性
- 支持自定义重启策略

## 配置文件位置

TaskD 使用 `TASKD_HOME` 环境变量来确定配置文件的存储位置：

### 目录选择优先级
1. `TASKD_HOME` 环境变量（如果设置）
2. 默认位置：
   - Windows: `%USERPROFILE%\.taskd`
   - Linux/macOS: `~/.taskd`

### 目录结构
```
$TASKD_HOME/
├── config.toml          # 全局配置文件
├── tasks/               # 任务配置目录
│   ├── task1.toml      # 任务配置文件
│   └── task2.toml
└── runtime.json         # 运行时状态文件
```

### 全局配置
- 默认: `$TASKD_HOME/config.toml`
- 可通过 `--config` 参数指定自定义路径

### 任务配置
- 目录: `$TASKD_HOME/tasks/`
- 格式: `任务名.toml`

### 运行时状态
- 文件: `$TASKD_HOME/runtime.json`
- 存储当前运行任务的状态信息

### 使用示例
```bash
# 使用自定义目录
export TASKD_HOME="/path/to/my/taskd"
taskd add mytask --exec "python app.py"

# 使用默认目录
unset TASKD_HOME
taskd add mytask --exec "python app.py"
```

## 构建和部署

### 开发环境
```bash
# 安装依赖
go mod tidy

# 运行程序
go run cmd/taskd/main.go

# 运行测试
go test ./...
```

### 生产构建
```bash
# 使用 Makefile
make build

# 或直接使用 go build
go build -o taskd.exe cmd/taskd/main.go
```

### 交叉编译
```bash
# 构建所有平台版本
make build-all
```