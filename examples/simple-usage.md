# TaskD 使用示例

## 基础使用

### 1. 添加一个简单的任务

```bash
# 添加一个 Python 应用任务
taskd add web-app --exec "python app.py" --workdir "C:\myapp" --stdout "C:\logs\app.log"

# 添加一个 Node.js 服务
taskd add api-server --exec "node server.js" --workdir "C:\api" --env "NODE_ENV=production" --env "PORT=3000"

# 添加一个 Windows 批处理任务
taskd add cleanup --exec "cmd /c cleanup.bat" --workdir "C:\temp"
```

### 2. 管理任务

```bash
# 启动任务
taskd start web-app

# 查看任务状态
taskd status web-app

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

## 高级配置示例

### Web 服务器配置

```bash
taskd add nginx-proxy \
  --exec "nginx" \
  --workdir "C:\nginx" \
  --env "NGINX_PORT=80" \
  --stdout "C:\logs\nginx.log" \
  --stderr "C:\logs\nginx.error.log"
```

### 数据库服务

```bash
taskd add mysql-server \
  --exec "mysqld" \
  --workdir "C:\mysql\data" \
  --env "MYSQL_ROOT_PASSWORD=secret" \
  --stdout "C:\logs\mysql.log"
```

### 定时任务

```bash
taskd add backup-job \
  --exec "python backup.py" \
  --workdir "C:\backup" \
  --stdout "C:\logs\backup.log" \
  --stderr "C:\logs\backup.error.log"
```

## 配置文件示例

任务配置会自动保存到 `~/.taskd/tasks/` 目录下的 TOML 文件中：

```toml
# ~/.taskd/tasks/web-app.toml
name = "web-app"
executable = "python"
args = ["app.py"]
workdir = "C:\\myapp"
inherit_env = true
env = ["FLASK_ENV=production"]
stdout = "C:\\logs\\app.log"
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

## 常见使用场景

### 1. 开发环境

```bash
# 启动开发服务器
taskd add dev-server --exec "npm run dev" --workdir "C:\project"
taskd start dev-server

# 启动数据库
taskd add dev-db --exec "docker run -p 5432:5432 postgres" 
taskd start dev-db
```

### 2. 生产环境

```bash
# Web 应用
taskd add prod-app --exec "python -m gunicorn app:app" --workdir "C:\app" --env "FLASK_ENV=production"

# 反向代理
taskd add nginx --exec "nginx -g daemon off;" --workdir "C:\nginx"

# 启动所有服务
taskd start prod-app
taskd start nginx
```

### 3. 后台任务

```bash
# 日志清理
taskd add log-cleanup --exec "python cleanup_logs.py" --workdir "C:\scripts"

# 数据同步
taskd add data-sync --exec "rsync -av /source/ /dest/" --workdir "C:\sync"
```