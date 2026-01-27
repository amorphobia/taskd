# TaskD - 任务守护进程管理工具

TaskD 是一个为 Windows 非管理员用户设计的任务管理工具，可以统一管理和监控用户级别的后台进程，类似于 Windows 服务但无需管理员权限。

## 主要特性

- ✅ 指定可执行文件路径和参数
- ✅ 指定工作目录
- ✅ 环境变量管理（继承或覆盖）
- ✅ 预设标准输入
- ✅ 标准输出和错误重定向
- ✅ 日志轮替
- ✅ TOML 配置文件
- ✅ 命令行任务管理
- ✅ 跨平台支持（Go 语言）

## 快速开始

```bash
# 添加任务
taskd add mytask --exec "python app.py" --workdir "/path/to/app"

# 启动任务
taskd start mytask

# 查看状态
taskd status mytask

# 列出所有任务
taskd list
```

## 技术栈

- **语言**: Go 1.21+
- **配置**: TOML
- **日志**: 结构化日志 + 轮替
- **跨平台**: Windows/Linux/macOS