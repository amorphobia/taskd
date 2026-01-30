package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIOManager_CreateTaskIO(t *testing.T) {
	manager := GetIOManager()
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "taskd_io_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test input file
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("test input"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}
	
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "no IO redirection",
			config: &Config{
				Executable: "echo",
				WorkDir:    tempDir,
			},
			wantErr: false,
		},
		{
			name: "stdout redirection only",
			config: &Config{
				Executable: "echo",
				WorkDir:    tempDir,
				Stdout:     "output.log",
			},
			wantErr: false,
		},
		{
			name: "stderr redirection only",
			config: &Config{
				Executable: "echo",
				WorkDir:    tempDir,
				Stderr:     "error.log",
			},
			wantErr: false,
		},
		{
			name: "stdin redirection",
			config: &Config{
				Executable: "cat",
				WorkDir:    tempDir,
				Stdin:      "input.txt",
			},
			wantErr: false,
		},
		{
			name: "all IO redirection",
			config: &Config{
				Executable: "cat",
				WorkDir:    tempDir,
				Stdin:      "input.txt",
				Stdout:     "output.log",
				Stderr:     "error.log",
			},
			wantErr: false,
		},
		{
			name: "stdout and stderr to same file",
			config: &Config{
				Executable: "echo",
				WorkDir:    tempDir,
				Stdout:     "combined.log",
				Stderr:     "combined.log",
			},
			wantErr: false,
		},
		{
			name: "invalid stdin file",
			config: &Config{
				Executable: "cat",
				WorkDir:    tempDir,
				Stdin:      "nonexistent.txt",
			},
			wantErr: true,
		},
		{
			name: "invalid stdout path",
			config: &Config{
				Executable: "echo",
				WorkDir:    tempDir,
				Stdout:     "CON", // Windows reserved name
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskIO, err := manager.CreateTaskIO(tt.config)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTaskIO() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("CreateTaskIO() unexpected error: %v", err)
				return
			}
			
			if taskIO == nil {
				t.Errorf("CreateTaskIO() returned nil TaskIO")
				return
			}
			
			// Verify IO configuration
			if tt.config.Stdin != "" && taskIO.Stdin == nil {
				t.Errorf("Expected stdin to be configured")
			}
			if tt.config.Stdout != "" && taskIO.Stdout == nil {
				t.Errorf("Expected stdout to be configured")
			}
			if tt.config.Stderr != "" && taskIO.Stderr == nil {
				t.Errorf("Expected stderr to be configured")
			}
			
			// Clean up
			if taskIO != nil {
				taskIO.Close()
			}
		})
	}
}
func TestIOManager_Singleton(t *testing.T) {
	// Test that GetIOManager returns the same instance
	manager1 := GetIOManager()
	manager2 := GetIOManager()
	
	if manager1 != manager2 {
		t.Errorf("GetIOManager() should return the same singleton instance")
	}
}

func TestTaskIO_Close(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "taskd_close_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("test input"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}
	
	config := &Config{
		Executable: "cat",
		WorkDir:    tempDir,
		Stdin:      "input.txt",
		Stdout:     "output.log",
		Stderr:     "error.log",
	}
	
	manager := GetIOManager()
	taskIO, err := manager.CreateTaskIO(config)
	if err != nil {
		t.Fatalf("Failed to create TaskIO: %v", err)
	}
	
	// Verify files are open (this is implicit - if CreateTaskIO succeeded, files are open)
	if taskIO.Stdin == nil || taskIO.Stdout == nil || taskIO.Stderr == nil {
		t.Errorf("Expected all IO streams to be configured")
	}
	
	// Close the TaskIO
	err = taskIO.Close()
	if err != nil {
		t.Errorf("TaskIO.Close() returned error: %v", err)
	}
}