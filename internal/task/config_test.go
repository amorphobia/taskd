package task

import (
	"testing"
)

func TestConfigStruct(t *testing.T) {
	config := &Config{
		DisplayName: "Test Task",
		Description: "A test task for unit testing",
		Executable:  "echo hello",
		Args:        []string{"arg1", "arg2"},
		WorkDir:     "/tmp",
		Env:         []string{"KEY1=value1", "KEY2=value2"},
		InheritEnv:  true,
		Stdin:       "input.txt",
		Stdout:      "output.log",
		Stderr:      "error.log",
		AutoStart:   true,
		MaxRetryNum: 5,
	}
	
	// Test basic fields
	if config.DisplayName != "Test Task" {
		t.Errorf("DisplayName = %q, want 'Test Task'", config.DisplayName)
	}
	
	if config.Description != "A test task for unit testing" {
		t.Errorf("Description = %q, want 'A test task for unit testing'", config.Description)
	}
	
	if config.Executable != "echo hello" {
		t.Errorf("Executable = %q, want 'echo hello'", config.Executable)
	}
	
	// Test args
	if len(config.Args) != 2 {
		t.Errorf("Args length = %d, want 2", len(config.Args))
	} else {
		if config.Args[0] != "arg1" {
			t.Errorf("Args[0] = %q, want 'arg1'", config.Args[0])
		}
		if config.Args[1] != "arg2" {
			t.Errorf("Args[1] = %q, want 'arg2'", config.Args[1])
		}
	}
	
	// Test work directory
	if config.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want '/tmp'", config.WorkDir)
	}
	
	// Test environment variables
	if len(config.Env) != 2 {
		t.Errorf("Env length = %d, want 2", len(config.Env))
	} else {
		if config.Env[0] != "KEY1=value1" {
			t.Errorf("Env[0] = %q, want 'KEY1=value1'", config.Env[0])
		}
		if config.Env[1] != "KEY2=value2" {
			t.Errorf("Env[1] = %q, want 'KEY2=value2'", config.Env[1])
		}
	}
	
	// Test boolean fields
	if !config.InheritEnv {
		t.Error("InheritEnv should be true")
	}
	
	if !config.AutoStart {
		t.Error("AutoStart should be true")
	}
	
	// Test IO redirection
	if config.Stdin != "input.txt" {
		t.Errorf("Stdin = %q, want 'input.txt'", config.Stdin)
	}
	
	if config.Stdout != "output.log" {
		t.Errorf("Stdout = %q, want 'output.log'", config.Stdout)
	}
	
	if config.Stderr != "error.log" {
		t.Errorf("Stderr = %q, want 'error.log'", config.Stderr)
	}
	
	// Test new daemon-related fields
	if config.MaxRetryNum != 5 {
		t.Errorf("MaxRetryNum = %d, want 5", config.MaxRetryNum)
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &Config{
		Executable: "test",
	}
	
	// Test default values
	if config.DisplayName != "" {
		t.Errorf("DisplayName default = %q, want empty string", config.DisplayName)
	}
	
	if config.Description != "" {
		t.Errorf("Description default = %q, want empty string", config.Description)
	}
	
	if config.WorkDir != "" {
		t.Errorf("WorkDir default = %q, want empty string", config.WorkDir)
	}
	
	if config.InheritEnv {
		t.Error("InheritEnv default should be false")
	}
	
	if config.AutoStart {
		t.Error("AutoStart default should be false")
	}
	
	if config.MaxRetryNum != 0 {
		t.Errorf("MaxRetryNum default = %d, want 0", config.MaxRetryNum)
	}
	
	if len(config.Args) != 0 {
		t.Errorf("Args default length = %d, want 0", len(config.Args))
	}
	
	if len(config.Env) != 0 {
		t.Errorf("Env default length = %d, want 0", len(config.Env))
	}
}

func TestRestartPolicyStruct(t *testing.T) {
	policy := &RestartPolicy{
		Policy:   "on-failure",
		MaxRetry: 3,
		Delay:    "5s",
	}
	
	if policy.Policy != "on-failure" {
		t.Errorf("Policy = %q, want 'on-failure'", policy.Policy)
	}
	
	if policy.MaxRetry != 3 {
		t.Errorf("MaxRetry = %d, want 3", policy.MaxRetry)
	}
	
	if policy.Delay != "5s" {
		t.Errorf("Delay = %q, want '5s'", policy.Delay)
	}
}

func TestRestartPolicyDefaults(t *testing.T) {
	policy := &RestartPolicy{}
	
	if policy.Policy != "" {
		t.Errorf("Policy default = %q, want empty string", policy.Policy)
	}
	
	if policy.MaxRetry != 0 {
		t.Errorf("MaxRetry default = %d, want 0", policy.MaxRetry)
	}
	
	if policy.Delay != "" {
		t.Errorf("Delay default = %q, want empty string", policy.Delay)
	}
}

func TestLogConfigStruct(t *testing.T) {
	logConfig := &LogConfig{
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	
	if logConfig.MaxSize != 100 {
		t.Errorf("MaxSize = %d, want 100", logConfig.MaxSize)
	}
	
	if logConfig.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d, want 5", logConfig.MaxBackups)
	}
	
	if logConfig.MaxAge != 30 {
		t.Errorf("MaxAge = %d, want 30", logConfig.MaxAge)
	}
	
	if !logConfig.Compress {
		t.Error("Compress should be true")
	}
}

func TestLogConfigDefaults(t *testing.T) {
	logConfig := &LogConfig{}
	
	if logConfig.MaxSize != 0 {
		t.Errorf("MaxSize default = %d, want 0", logConfig.MaxSize)
	}
	
	if logConfig.MaxBackups != 0 {
		t.Errorf("MaxBackups default = %d, want 0", logConfig.MaxBackups)
	}
	
	if logConfig.MaxAge != 0 {
		t.Errorf("MaxAge default = %d, want 0", logConfig.MaxAge)
	}
	
	if logConfig.Compress {
		t.Error("Compress default should be false")
	}
}

func TestTaskInfoStruct(t *testing.T) {
	taskInfo := &TaskInfo{
		Name:       "test-task",
		Status:     "running",
		PID:        1234,
		StartTime:  "2024-01-01 12:00:00",
		Executable: "echo hello",
		ExitCode:   0,
		LastError:  "",
	}
	
	if taskInfo.Name != "test-task" {
		t.Errorf("Name = %q, want 'test-task'", taskInfo.Name)
	}
	
	if taskInfo.Status != "running" {
		t.Errorf("Status = %q, want 'running'", taskInfo.Status)
	}
	
	if taskInfo.PID != 1234 {
		t.Errorf("PID = %d, want 1234", taskInfo.PID)
	}
	
	if taskInfo.StartTime != "2024-01-01 12:00:00" {
		t.Errorf("StartTime = %q, want '2024-01-01 12:00:00'", taskInfo.StartTime)
	}
	
	if taskInfo.Executable != "echo hello" {
		t.Errorf("Executable = %q, want 'echo hello'", taskInfo.Executable)
	}
	
	if taskInfo.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", taskInfo.ExitCode)
	}
	
	if taskInfo.LastError != "" {
		t.Errorf("LastError = %q, want empty string", taskInfo.LastError)
	}
}

func TestTaskDetailInfoStruct(t *testing.T) {
	ioInfo := &TaskIOInfo{
		StdinPath:  "input.txt",
		StdoutPath: "output.log",
		StderrPath: "error.log",
	}
	
	detailInfo := &TaskDetailInfo{
		Name:        "test-task",
		Status:      "stopped",
		PID:         0,
		StartTime:   "2024-01-01 12:00:00",
		Executable:  "echo hello",
		ExitCode:    1,
		LastError:   "Process failed",
		DisplayName: "Test Task",
		Description: "A test task",
		WorkDir:     "/tmp",
		Args:        []string{"arg1", "arg2"},
		Env:         []string{"KEY=value"},
		InheritEnv:  true,
		IOInfo:      ioInfo,
	}
	
	// Test basic fields
	if detailInfo.Name != "test-task" {
		t.Errorf("Name = %q, want 'test-task'", detailInfo.Name)
	}
	
	if detailInfo.Status != "stopped" {
		t.Errorf("Status = %q, want 'stopped'", detailInfo.Status)
	}
	
	if detailInfo.PID != 0 {
		t.Errorf("PID = %d, want 0", detailInfo.PID)
	}
	
	if detailInfo.StartTime != "2024-01-01 12:00:00" {
		t.Errorf("StartTime = %q, want '2024-01-01 12:00:00'", detailInfo.StartTime)
	}
	
	if detailInfo.Executable != "echo hello" {
		t.Errorf("Executable = %q, want 'echo hello'", detailInfo.Executable)
	}
	
	if detailInfo.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", detailInfo.ExitCode)
	}
	
	if detailInfo.LastError != "Process failed" {
		t.Errorf("LastError = %q, want 'Process failed'", detailInfo.LastError)
	}
	
	// Test extended fields
	if detailInfo.DisplayName != "Test Task" {
		t.Errorf("DisplayName = %q, want 'Test Task'", detailInfo.DisplayName)
	}
	
	if detailInfo.Description != "A test task" {
		t.Errorf("Description = %q, want 'A test task'", detailInfo.Description)
	}
	
	if detailInfo.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want '/tmp'", detailInfo.WorkDir)
	}
	
	if len(detailInfo.Args) != 2 {
		t.Errorf("Args length = %d, want 2", len(detailInfo.Args))
	}
	
	if len(detailInfo.Env) != 1 {
		t.Errorf("Env length = %d, want 1", len(detailInfo.Env))
	}
	
	if !detailInfo.InheritEnv {
		t.Error("InheritEnv should be true")
	}
	
	// Test IO info
	if detailInfo.IOInfo == nil {
		t.Fatal("IOInfo should not be nil")
	}
	
	if detailInfo.IOInfo.StdinPath != "input.txt" {
		t.Errorf("IOInfo.StdinPath = %q, want 'input.txt'", detailInfo.IOInfo.StdinPath)
	}
	
	if detailInfo.IOInfo.StdoutPath != "output.log" {
		t.Errorf("IOInfo.StdoutPath = %q, want 'output.log'", detailInfo.IOInfo.StdoutPath)
	}
	
	if detailInfo.IOInfo.StderrPath != "error.log" {
		t.Errorf("IOInfo.StderrPath = %q, want 'error.log'", detailInfo.IOInfo.StderrPath)
	}
}

func TestConfigWithComplexRestartPolicy(t *testing.T) {
	config := &Config{
		Executable: "test-app",
		AutoStart:  true,
		Restart: RestartPolicy{
			Policy:   "always",
			MaxRetry: 10,
			Delay:    "30s",
		},
		Log: LogConfig{
			MaxSize:    50,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}
	
	if config.Restart.Policy != "always" {
		t.Errorf("Restart.Policy = %q, want 'always'", config.Restart.Policy)
	}
	
	if config.Restart.MaxRetry != 10 {
		t.Errorf("Restart.MaxRetry = %d, want 10", config.Restart.MaxRetry)
	}
	
	if config.Restart.Delay != "30s" {
		t.Errorf("Restart.Delay = %q, want '30s'", config.Restart.Delay)
	}
	
	if config.Log.MaxSize != 50 {
		t.Errorf("Log.MaxSize = %d, want 50", config.Log.MaxSize)
	}
	
	if config.Log.MaxBackups != 3 {
		t.Errorf("Log.MaxBackups = %d, want 3", config.Log.MaxBackups)
	}
	
	if config.Log.MaxAge != 7 {
		t.Errorf("Log.MaxAge = %d, want 7", config.Log.MaxAge)
	}
	
	if !config.Log.Compress {
		t.Error("Log.Compress should be true")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "valid minimal config",
			config: &Config{
				Executable: "echo hello",
			},
			valid: true,
		},
		{
			name: "invalid empty executable",
			config: &Config{
				Executable: "",
			},
			valid: false,
		},
		{
			name: "valid config with all fields",
			config: &Config{
				DisplayName: "Test",
				Description: "Test task",
				Executable:  "echo hello",
				Args:        []string{"world"},
				WorkDir:     "/tmp",
				Env:         []string{"TEST=1"},
				InheritEnv:  true,
				Stdin:       "input.txt",
				Stdout:      "output.log",
				Stderr:      "error.log",
				AutoStart:   true,
				MaxRetryNum: 3,
			},
			valid: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation: executable must not be empty
			isValid := tt.config.Executable != ""
			
			if isValid != tt.valid {
				t.Errorf("Config validation = %v, want %v", isValid, tt.valid)
			}
		})
	}
}