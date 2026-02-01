package cli

import (
	"strings"
	"testing"
	"taskd/internal/task"
)

func TestBuiltinTaskProtection(t *testing.T) {
	// Test that builtin task validation works for add, edit, del operations
	manager := task.GetManager()
	
	tests := []struct {
		operation string
		wantError bool
		errorMsg  string
	}{
		{"add", true, "cannot add builtin task 'taskd'"},
		{"edit", true, "cannot edit builtin task 'taskd'"},
		{"del", true, "cannot delete builtin task 'taskd'"},
		{"start", false, ""},
		{"stop", false, ""},
		{"restart", false, ""},
		{"info", false, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := manager.ValidateBuiltinTaskOperation("taskd", tt.operation)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateBuiltinTaskOperation(%q, %q) = nil, want error", "taskd", tt.operation)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateBuiltinTaskOperation(%q, %q) error = %q, want error containing %q", 
						"taskd", tt.operation, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBuiltinTaskOperation(%q, %q) = %v, want nil", "taskd", tt.operation, err)
				}
			}
		})
	}
}

func TestBuiltinTaskProtectionForNonBuiltinTask(t *testing.T) {
	// Test that non-builtin tasks are not affected by builtin task validation
	manager := task.GetManager()
	
	operations := []string{"add", "edit", "del", "start", "stop", "restart", "info"}
	
	for _, op := range operations {
		t.Run(op, func(t *testing.T) {
			err := manager.ValidateBuiltinTaskOperation("regular-task", op)
			if err != nil {
				t.Errorf("ValidateBuiltinTaskOperation(%q, %q) = %v, want nil for non-builtin task", 
					"regular-task", op, err)
			}
		})
	}
}