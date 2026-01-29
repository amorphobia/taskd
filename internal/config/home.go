package config

import (
	"os"
	"path/filepath"
)

// GetTaskDHome returns the TaskD home directory
// It checks the TASKD_HOME environment variable first,
// if not set, defaults to $HOME/.taskd
func GetTaskDHome() string {
	// Check TASKD_HOME environment variable
	if taskdHome := os.Getenv("TASKD_HOME"); taskdHome != "" {
		// Ensure the directory exists
		if err := os.MkdirAll(taskdHome, 0755); err != nil {
			// If we can't create the custom directory, fall back to default
			return getDefaultTaskDHome()
		}
		return taskdHome
	}
	
	return getDefaultTaskDHome()
}

// getDefaultTaskDHome returns the default TaskD home directory ($HOME/.taskd)
func getDefaultTaskDHome() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home directory
		return ".taskd"
	}
	
	taskdHome := filepath.Join(homeDir, ".taskd")
	
	// Ensure the directory exists
	os.MkdirAll(taskdHome, 0755)
	
	return taskdHome
}

// GetTaskDConfigDir returns the configuration directory within TaskD home
func GetTaskDConfigDir() string {
	return GetTaskDHome()
}

// GetTaskDTasksDir returns the tasks directory within TaskD home
func GetTaskDTasksDir() string {
	return filepath.Join(GetTaskDHome(), "tasks")
}

// GetTaskDRuntimeFile returns the runtime state file path
func GetTaskDRuntimeFile() string {
	return filepath.Join(GetTaskDHome(), "runtime.json")
}