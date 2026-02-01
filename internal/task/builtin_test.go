package task

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBuiltinTaskHandler(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	if handler == nil {
		t.Fatal("NewBuiltinTaskHandler() returned nil")
	}
}

func TestIsBuiltinTask(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	
	tests := []struct {
		name     string
		taskName string
		want     bool
	}{
		{"builtin taskd", "taskd", true},
		{"regular task", "mytask", false},
		{"empty name", "", false},
		{"similar name", "taskd-test", false},
		{"case sensitive", "TASKD", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.IsBuiltinTask(tt.taskName)
			if got != tt.want {
				t.Errorf("IsBuiltinTask(%q) = %v, want %v", tt.taskName, got, tt.want)
			}
		})
	}
}

func TestGetBuiltinTaskConfig(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	
	t.Run("taskd builtin task", func(t *testing.T) {
		config := handler.GetBuiltinTaskConfig("taskd")
		if config == nil {
			t.Fatal("GetBuiltinTaskConfig('taskd') returned nil")
		}
		
		// Check required fields
		if config.DisplayName != "taskd" {
			t.Errorf("DisplayName = %q, want 'taskd'", config.DisplayName)
		}
		
		if config.Description != "The daemon task of taskd" {
			t.Errorf("Description = %q, want 'The daemon task of taskd'", config.Description)
		}
		
		if !strings.Contains(config.Executable, "--daemon") {
			t.Errorf("Executable = %q, should contain '--daemon'", config.Executable)
		}
		
		expectedWorkDir := config.WorkDir // The config should have the correct WorkDir set
		if expectedWorkDir == "" {
			t.Error("WorkDir should not be empty for daemon task")
		}
		
		if !config.InheritEnv {
			t.Error("InheritEnv should be true for daemon task")
		}
		
		if config.AutoStart {
			t.Error("AutoStart should be false for daemon task")
		}
		
		if config.MaxRetryNum != 0 {
			t.Errorf("MaxRetryNum = %d, want 0", config.MaxRetryNum)
		}
	})
	
	t.Run("non-builtin task", func(t *testing.T) {
		config := handler.GetBuiltinTaskConfig("regular-task")
		if config != nil {
			t.Errorf("GetBuiltinTaskConfig('regular-task') = %v, want nil", config)
		}
	})
	
	t.Run("empty task name", func(t *testing.T) {
		config := handler.GetBuiltinTaskConfig("")
		if config != nil {
			t.Errorf("GetBuiltinTaskConfig('') = %v, want nil", config)
		}
	})
}

func TestGetBuiltinTaskConfigExecutablePath(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	config := handler.GetBuiltinTaskConfig("taskd")
	
	if config == nil {
		t.Fatal("GetBuiltinTaskConfig('taskd') returned nil")
	}
	
	// The executable should be a clean path with --daemon
	execParts := strings.Split(config.Executable, " ")
	if len(execParts) < 2 {
		t.Fatalf("Executable should contain at least executable path and --daemon flag, got: %q", config.Executable)
	}
	
	execPath := execParts[0]
	daemonFlag := execParts[1]
	
	if daemonFlag != "--daemon" {
		t.Errorf("Expected --daemon flag, got: %q", daemonFlag)
	}
	
	// Check that the path is clean (no double slashes, etc.)
	if execPath != filepath.Clean(execPath) {
		t.Errorf("Executable path should be clean, got: %q", execPath)
	}
}

func TestValidateOperation(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	
	tests := []struct {
		name      string
		taskName  string
		operation string
		wantError bool
		errorMsg  string
	}{
		// Builtin task operations
		{"taskd add forbidden", "taskd", "add", true, "cannot add builtin task 'taskd'"},
		{"taskd edit forbidden", "taskd", "edit", true, "cannot edit builtin task 'taskd'"},
		{"taskd del forbidden", "taskd", "del", true, "cannot delete builtin task 'taskd'"},
		{"taskd start allowed", "taskd", "start", false, ""},
		{"taskd stop allowed", "taskd", "stop", false, ""},
		{"taskd restart allowed", "taskd", "restart", false, ""},
		{"taskd info allowed", "taskd", "info", false, ""},
		{"taskd unsupported operation", "taskd", "unknown", true, "operation 'unknown' not supported for builtin task 'taskd'"},
		
		// Non-builtin task operations (should all be allowed)
		{"regular task add", "mytask", "add", false, ""},
		{"regular task edit", "mytask", "edit", false, ""},
		{"regular task del", "mytask", "del", false, ""},
		{"regular task start", "mytask", "start", false, ""},
		{"regular task stop", "mytask", "stop", false, ""},
		{"regular task restart", "mytask", "restart", false, ""},
		{"regular task info", "mytask", "info", false, ""},
		{"regular task unknown", "mytask", "unknown", false, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateOperation(tt.taskName, tt.operation)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateOperation(%q, %q) = nil, want error", tt.taskName, tt.operation)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateOperation(%q, %q) error = %q, want error containing %q", 
						tt.taskName, tt.operation, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateOperation(%q, %q) = %v, want nil", tt.taskName, tt.operation, err)
				}
			}
		})
	}
}

func TestValidateOperationEdgeCases(t *testing.T) {
	handler := NewBuiltinTaskHandler()
	
	t.Run("empty task name", func(t *testing.T) {
		err := handler.ValidateOperation("", "add")
		if err != nil {
			t.Errorf("ValidateOperation('', 'add') = %v, want nil for empty task name", err)
		}
	})
	
	t.Run("empty operation", func(t *testing.T) {
		err := handler.ValidateOperation("taskd", "")
		if err == nil {
			t.Error("ValidateOperation('taskd', '') = nil, want error for empty operation")
		}
		if !strings.Contains(err.Error(), "operation '' not supported") {
			t.Errorf("Expected error about unsupported operation, got: %v", err)
		}
	})
	
	t.Run("case sensitivity", func(t *testing.T) {
		// Task names are case sensitive, so "TASKD" is not a builtin task
		err := handler.ValidateOperation("TASKD", "add")
		if err != nil {
			t.Errorf("ValidateOperation('TASKD', 'add') = %v, want nil for case-sensitive non-builtin", err)
		}
	})
}