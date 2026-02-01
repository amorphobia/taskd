package task

// Config task configuration structure
type Config struct {
	DisplayName  string            `toml:"display_name,omitempty"`
	Description  string            `toml:"description,omitempty"`
	Executable   string            `toml:"executable"`
	Args         []string          `toml:"args,omitempty"`
	WorkDir      string            `toml:"workdir,omitempty"`
	Env          []string          `toml:"env,omitempty"`
	InheritEnv   bool              `toml:"inherit_env"`
	Stdin        string            `toml:"stdin,omitempty"`
	Stdout       string            `toml:"stdout,omitempty"`
	Stderr       string            `toml:"stderr,omitempty"`
	AutoStart    bool              `toml:"auto_start"`
	MaxRetryNum  int               `toml:"max_retry_num"`  // Maximum retry count, default is 3
	Restart      RestartPolicy     `toml:"restart,omitempty"`
	Log          LogConfig         `toml:"log,omitempty"`
}

// RestartPolicy restart policy configuration
type RestartPolicy struct {
	Policy    string `toml:"policy"`    // always, on-failure, never
	MaxRetry  int    `toml:"max_retry"`
	Delay     string `toml:"delay"`     // restart delay, e.g. "5s", "1m"
}

// LogConfig log configuration
type LogConfig struct {
	MaxSize    int  `toml:"max_size"`    // MB
	MaxBackups int  `toml:"max_backups"`
	MaxAge     int  `toml:"max_age"`     // days
	Compress   bool `toml:"compress"`
}

// TaskInfo task runtime information
type TaskInfo struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`     // running, stopped, failed
	PID        int       `json:"pid"`
	StartTime  string    `json:"start_time"`
	Executable string    `json:"executable"`
	ExitCode   int       `json:"exit_code,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}
// TaskDetailInfo detailed task information (merges all fields from original TaskInfo)
type TaskDetailInfo struct {
	// Basic status information (information displayed by original status command)
	Name       string `json:"name"`
	Status     string `json:"status"`
	PID        int    `json:"pid"`
	StartTime  string `json:"start_time"`
	Executable string `json:"executable"`
	ExitCode   int    `json:"exit_code,omitempty"`
	LastError  string `json:"last_error,omitempty"`
	
	// Extended configuration information
	DisplayName string   `json:"display_name,omitempty"`
	Description string   `json:"description,omitempty"`
	WorkDir     string   `json:"work_dir"`
	Args        []string `json:"args,omitempty"`
	Env         []string `json:"env,omitempty"`
	InheritEnv  bool     `json:"inherit_env"`
	
	// IO redirection information
	IOInfo     *TaskIOInfo `json:"io_info"`
}