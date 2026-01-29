package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileError_Error(t *testing.T) {
	tests := []struct {
		name     string
		fileErr  *FileError
		contains string // substring that should be in the error message
	}{
		{
			name: "file not found",
			fileErr: &FileError{
				Type:      FileErrorNotFound,
				Path:      "/path/to/file.txt",
				Operation: "open",
			},
			contains: "File not found",
		},
		{
			name: "permission denied",
			fileErr: &FileError{
				Type:      FileErrorPermissionDenied,
				Path:      "/path/to/file.txt",
				Operation: "write",
			},
			contains: "Permission denied",
		},
		{
			name: "invalid path",
			fileErr: &FileError{
				Type:      FileErrorInvalidPath,
				Path:      "CON.txt",
				Operation: "validate",
			},
			contains: "Invalid path",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.fileErr.Error()
			if errMsg == "" {
				t.Errorf("Error() returned empty string")
			}
			if !contains(errMsg, tt.contains) {
				t.Errorf("Error message %q does not contain %q", errMsg, tt.contains)
			}
			// Verify path is included in error message
			if !contains(errMsg, tt.fileErr.Path) {
				t.Errorf("Error message %q does not contain path %q", errMsg, tt.fileErr.Path)
			}
		})
	}
}

func TestValidateFilePermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "taskd_perm_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name      string
		path      string
		operation string
		wantErr   bool
	}{
		{
			name:      "read existing file",
			path:      testFile,
			operation: "read",
			wantErr:   false,
		},
		{
			name:      "write existing file",
			path:      testFile,
			operation: "write",
			wantErr:   false,
		},
		{
			name:      "create new file",
			path:      filepath.Join(tempDir, "new_file.txt"),
			operation: "create",
			wantErr:   false,
		},
		{
			name:      "read nonexistent file",
			path:      filepath.Join(tempDir, "nonexistent.txt"),
			operation: "read",
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePermissions(tt.path, tt.operation)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateFilePermissions() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFilePermissions() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}