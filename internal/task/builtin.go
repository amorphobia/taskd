package task

import (
	"fmt"
	"os"
	"path/filepath"
	"taskd/internal/config"
)

// BuiltinTaskHandler handles builtin tasks like the daemon task
type BuiltinTaskHandler struct{}

// NewBuiltinTaskHandler creates a new builtin task handler
func NewBuiltinTaskHandler() *BuiltinTaskHandler {
	return &BuiltinTaskHandler{}
}

// IsBuiltinTask checks if the given task name is a builtin task
func (bth *BuiltinTaskHandler) IsBuiltinTask(name string) bool {
	return name == "taskd"
}

// GetBuiltinTaskConfig returns the configuration for a builtin task
func (bth *BuiltinTaskHandler) GetBuiltinTaskConfig(name string) *Config {
	if name == "taskd" {
		execPath, err := os.Executable()
		if err != nil {
			// Fallback to a reasonable default
			execPath = "taskd"
		} else {
			execPath = filepath.Clean(execPath)
		}
		
		return &Config{
			DisplayName: "taskd",
			Description: "The daemon task of taskd",
			Executable:  execPath + " --daemon",
			WorkDir:     config.GetTaskDHome(),
			InheritEnv:  true,
			AutoStart:   false, // Daemon is started manually or automatically as needed
			MaxRetryNum: 0,     // No automatic retry for daemon
		}
	}
	return nil
}

// ValidateOperation validates if an operation is allowed on a builtin task
func (bth *BuiltinTaskHandler) ValidateOperation(name, operation string) error {
	if name == "taskd" {
		switch operation {
		case "add":
			return fmt.Errorf("cannot add builtin task '%s': task name conflicts with system reserved name", name)
		case "edit":
			return fmt.Errorf("cannot edit builtin task '%s': builtin tasks are not editable", name)
		case "del":
			return fmt.Errorf("cannot delete builtin task '%s': builtin tasks cannot be deleted", name)
		case "start", "stop", "restart", "info":
			// These operations are allowed
			return nil
		default:
			return fmt.Errorf("operation '%s' not supported for builtin task '%s'", operation, name)
		}
	}
	return nil
}