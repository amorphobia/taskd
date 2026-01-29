# TaskD 使用示例

## 基础使用

### 1. 添加一个简单的任务

```bash
# 添加一个 Python 应用任务
taskd add web-app --exec "python app.py" --workdir "C:\myapp" --stdout "logs\app.log"

# 添加一个 Node.js 服务
taskd add api-server --exec "node server.js" --workdir "C:\api" --env "NODE_ENV=production" --env "PORT=3000"

# 添加一个 Windows 批处理任务
taskd add cleanup --exec "cmd /c cleanup.bat" --workdir "C:\temp"
```

### 2. 管理任务

```bash
# 启动任务
taskd start web-app

# 查看任务详细信息（替代原 status 命令）
taskd info web-app

# 列出所有任务
taskd list

# 只显示运行中的任务
taskd list --running

# 停止任务
taskd stop web-app
```

### 3. 查看帮助

```bash
# 查看主帮助
taskd --help

# 查看特定命令帮助
taskd add --help
taskd list --help
```

## 输出重定向功能

### 基础输出重定向

```bash
# 重定向标准输出到文件
taskd add web-server --exec "python -m http.server" --workdir "C:\web" --stdout "server.log"

# 重定向标准错误到文件
taskd add error-prone --exec "python script.py" --workdir "C:\scripts" --stderr "errors.log"

# 同时重定向标准输出和标准错误
taskd add full-app --exec "python app.py" --workdir "C:\app" --stdout "output.log" --stderr "error.log"
```

### 相对路径使用

```bash
# 相对路径基于工作目录解析
taskd add relative-app --exec "python app.py" --workdir "C:\myapp" --stdout "logs\app.log" --stderr "logs\error.log"

# 使用当前目录
taskd add current-dir --exec "echo test" --workdir "C:\project" --stdout ".\output.log"

# 使用父目录
taskd add parent-dir --exec "python script.py" --workdir "C:\project\src" --stdout "..\logs\output.log"
```

### 输出合并

```bash
# 将标准输出和标准错误重定向到同一文件
taskd add combined-output --exec "python app.py" --workdir "C:\app" --stdout "combined.log" --stderr "combined.log"
```

### 标准输入重定向

```bash
# 从文件读取输入
taskd add batch-processor --exec "python process.py" --workdir "C:\data" --stdin "input.txt" --stdout "output.txt"
```

### 追加模式

所有输出文件都以追加模式打开，多次运行任务不会覆盖之前的日志：

```bash
# 每次启动都会追加到现有日志文件
taskd add logger --exec "python log_generator.py" --workdir "C:\logs" --stdout "app.log"
taskd start logger
# ... 停止后再次启动
taskd start logger  # 新的输出会追加到 app.log
```

## 高级配置示例

### Web 服务器配置

```bash
taskd add nginx-proxy \
  --exec "nginx" \
  --workdir "C:\nginx" \
  --env "NGINX_PORT=80" \
  --stdout "logs\nginx.log" \
  --stderr "logs\nginx.error.log"
```

### 数据库服务

```bash
taskd add mysql-server \
  --exec "mysqld" \
  --workdir "C:\mysql\data" \
  --env "MYSQL_ROOT_PASSWORD=secret" \
  --stdout "logs\mysql.log"
```

### 定时任务

```bash
taskd add backup-job \
  --exec "python backup.py" \
  --workdir "C:\backup" \
  --stdout "logs\backup.log" \
  --stderr "logs\backup.error.log"
```

## 配置文件示例

任务配置会自动保存到 TaskD 配置目录下的 TOML 文件中。配置目录由 `TASKD_HOME` 环境变量决定，默认为 `~/.taskd/tasks/`：

```toml
# $TASKD_HOME/tasks/web-app.toml (默认: ~/.taskd/tasks/web-app.toml)
name = "web-app"
executable = "python"
args = ["app.py"]
workdir = "C:\\myapp"
inherit_env = true
env = ["FLASK_ENV=production"]
stdout = "logs\\app.log"        # 相对路径，基于 workdir 解析
stderr = "logs\\error.log"      # 相对路径，基于 workdir 解析
auto_start = false

[restart]
policy = "always"
max_retry = 3
delay = "5s"

[log]
max_size = 10
max_backups = 5
max_age = 30
compress = true
```

### 自定义配置目录

```bash
# 使用自定义配置目录
export TASKD_HOME="/path/to/my/taskd"
taskd add web-app --exec "python app.py" --workdir "C:\myapp"
# 配置文件将保存到: /path/to/my/taskd/tasks/web-app.toml

# 使用默认配置目录
unset TASKD_HOME
taskd add web-app --exec "python app.py" --workdir "C:\myapp"
# 配置文件将保存到: ~/.taskd/tasks/web-app.toml
```
```

## 常见使用场景

### 1. 开发环境

```bash
# 启动开发服务器
taskd add dev-server --exec "npm run dev" --workdir "C:\project" --stdout "logs\dev.log"
taskd start dev-server

# 启动数据库
taskd add dev-db --exec "docker run -p 5432:5432 postgres" --stdout "logs\db.log"
taskd start dev-db
```

### 2. 生产环境

```bash
# Web 应用
taskd add prod-app --exec "python -m gunicorn app:app" --workdir "C:\app" --env "FLASK_ENV=production" --stdout "logs\app.log" --stderr "logs\error.log"

# 反向代理
taskd add nginx --exec "nginx -g daemon off;" --workdir "C:\nginx" --stdout "logs\access.log" --stderr "logs\error.log"

# 启动所有服务
taskd start prod-app
taskd start nginx
```

### 3. 后台任务

```bash
# 日志清理
taskd add log-cleanup --exec "python cleanup_logs.py" --workdir "C:\scripts" --stdout "logs\cleanup.log"

# 数据同步
taskd add data-sync --exec "rsync -av /source/ /dest/" --workdir "C:\sync" --stdout "logs\sync.log" --stderr "logs\sync_errors.log"
```

## 故障排除

### 常见问题

#### 1. 输出文件未创建

**问题**: 任务运行但输出文件没有创建

**解决方案**:
- 检查工作目录是否存在写入权限
- 确认相对路径是否正确（相对于工作目录）
- 使用 `taskd info <task-name>` 查看解析后的完整路径

```bash
# 查看任务详细信息，包括解析后的路径
taskd info my-task
```

#### 2. 权限错误

**问题**: 无法创建输出文件或目录

**解决方案**:
- 确保 TaskD 有足够权限访问目标目录
- 检查目标目录是否存在，TaskD 会自动创建不存在的目录
- 避免使用系统保留的文件名（如 CON、PRN 等）

#### 3. 路径解析问题

**问题**: 相对路径没有按预期解析

**解决方案**:
- 相对路径总是基于任务的工作目录（`--workdir`）解析
- 使用 `.\` 表示当前工作目录
- 使用 `..\` 表示父目录
- 绝对路径会直接使用，不进行解析

#### 4. 输出内容缺失

**问题**: 某些程序的输出没有写入文件

**解决方案**:
- 某些程序（如 Python）在重定向时使用全缓冲，输出可能延迟
- 程序正常退出后输出会被刷新到文件
- 对于长时间运行的程序，可以在程序中手动调用 flush()

### 最佳实践

1. **使用相对路径**: 相对路径更灵活，便于部署到不同环境
2. **创建日志目录**: 将所有日志文件放在专门的 `logs` 目录中
3. **分离输出和错误**: 使用不同文件记录标准输出和标准错误
4. **定期清理日志**: 设置日志轮替或定期清理旧日志文件
5. **使用 info 命令**: 使用 `taskd info` 而不是 `status` 查看任务详细信息