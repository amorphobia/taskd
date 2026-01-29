package task

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultPathResolver_ResolvePath(t *testing.T) {
	resolver := NewPathResolver()
	
	tests := []struct {
		name     string
		path     string
		workDir  string
		expected string
		wantErr  bool
	}{
		{
			name:     "absolute path on Windows",
			path:     "C:\\temp\\output.log",
			workDir:  "C:\\project",
			expected: "C:\\temp\\output.log",
			wantErr:  false,
		},
		{
			name:     "absolute path on Unix",
			path:     "/tmp/output.log",
			workDir:  "/home/user/project",
			expected: "/tmp/output.log",
			wantErr:  false,
		},
		{
			name:     "relative path",
			path:     "logs/output.log",
			workDir:  "/home/user/project",
			expected: "/home/user/project/logs/output.log",
			wantErr:  false,
		},
		{
			name:     "relative path with dot",
			path:     "./logs/output.log",
			workDir:  "/home/user/project",
			expected: "/home/user/project/logs/output.log",
			wantErr:  false,
		},
		{
			name:     "relative path with parent directory",
			path:     "../logs/output.log",
			workDir:  "/home/user/project",
			expected: "/home/user/logs/output.log",
			wantErr:  false,
		},
		{
			name:    "empty path",
			path:    "",
			workDir: "/home/user/project",
			wantErr: true,
		},
		{
			name:    "empty workDir for relative path",
			path:    "logs/output.log",
			workDir: "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip platform-specific tests
			if runtime.GOOS == "windows" && len(tt.path) > 0 && tt.path[0] == '/' {
				t.Skip("Skipping Unix path test on Windows")
			}
			if runtime.GOOS != "windows" && len(tt.path) > 1 && tt.path[1] == ':' {
				t.Skip("Skipping Windows path test on Unix")
			}
			
			result, err := resolver.ResolvePath(tt.path, tt.workDir)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolvePath() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ResolvePath() unexpected error: %v", err)
				return
			}
			
			// Normalize expected path for comparison
			expected := filepath.Clean(tt.expected)
			if result != expected {
				t.Errorf("ResolvePath() = %v, want %v", result, expected)
			}
		})
	}
}

func TestDefaultPathResolver_ValidatePath(t *testing.T) {
	resolver := NewPathResolver()
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid relative path",
			path:    "logs/output.log",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			path:    "/tmp/output.log",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Windows reserved name - CON",
			path:    "CON.txt",
			wantErr: true,
		},
		{
			name:    "Windows reserved name - PRN",
			path:    "PRN",
			wantErr: true,
		},
		{
			name:    "Windows reserved name - COM1",
			path:    "COM1.log",
			wantErr: true,
		},
		{
			name:    "Windows reserved name - LPT1",
			path:    "LPT1.txt",
			wantErr: true,
		},
		{
			name:    "valid name similar to reserved",
			path:    "CONSOLE.txt",
			wantErr: false,
		},
		{
			name:    "path too long",
			path:    string(make([]byte, 300)), // 300 characters
			wantErr: true,
		},
		{
			name:    "valid normal path",
			path:    "normal/path/file.txt",
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.ValidatePath(tt.path)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error for path %q, got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error for path %q: %v", tt.path, err)
				}
			}
		})
	}
}
func TestDefaultPathResolver_EnsureDir(t *testing.T) {
	resolver := NewPathResolver()
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "taskd_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tests := []struct {
		name    string
		dirPath string
		setup   func() error
		wantErr bool
	}{
		{
			name:    "create new directory",
			dirPath: filepath.Join(tempDir, "new_dir"),
			setup:   func() error { return nil },
			wantErr: false,
		},
		{
			name:    "create nested directory",
			dirPath: filepath.Join(tempDir, "nested", "deep", "dir"),
			setup:   func() error { return nil },
			wantErr: false,
		},
		{
			name:    "directory already exists",
			dirPath: filepath.Join(tempDir, "existing_dir"),
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "existing_dir"), 0755)
			},
			wantErr: false,
		},
		{
			name:    "path is a file",
			dirPath: filepath.Join(tempDir, "file_not_dir"),
			setup: func() error {
				file, err := os.Create(filepath.Join(tempDir, "file_not_dir"))
				if err != nil {
					return err
				}
				return file.Close()
			},
			wantErr: true,
		},
		{
			name:    "empty path",
			dirPath: "",
			setup:   func() error { return nil },
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			if err := tt.setup(); err != nil {
				t.Fatalf("Test setup failed: %v", err)
			}
			
			err := resolver.EnsureDir(tt.dirPath)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("EnsureDir() expected error for path %q, got nil", tt.dirPath)
				}
				return
			}
			
			if err != nil {
				t.Errorf("EnsureDir() unexpected error for path %q: %v", tt.dirPath, err)
				return
			}
			
			// Verify directory was created
			if info, err := os.Stat(tt.dirPath); err != nil {
				t.Errorf("Directory was not created: %v", err)
			} else if !info.IsDir() {
				t.Errorf("Path exists but is not a directory")
			}
		})
	}
}

func TestDefaultPathResolver_CrossPlatform(t *testing.T) {
	resolver := NewPathResolver()
	
	// Test platform-specific behavior
	if runtime.GOOS == "windows" {
		t.Run("Windows path handling", func(t *testing.T) {
			// Test Windows-style paths
			result, err := resolver.ResolvePath("logs\\output.log", "C:\\project")
			if err != nil {
				t.Errorf("Windows path resolution failed: %v", err)
			}
			expected := "C:\\project\\logs\\output.log"
			if result != expected {
				t.Errorf("Windows path resolution = %v, want %v", result, expected)
			}
		})
		
		t.Run("Windows reserved names", func(t *testing.T) {
			reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "LPT1"}
			for _, name := range reservedNames {
				err := resolver.ValidatePath(name)
				if err == nil {
					t.Errorf("Expected error for Windows reserved name %q", name)
				}
			}
		})
	} else {
		t.Run("Unix path handling", func(t *testing.T) {
			// Test Unix-style paths
			result, err := resolver.ResolvePath("logs/output.log", "/home/user/project")
			if err != nil {
				t.Errorf("Unix path resolution failed: %v", err)
			}
			expected := "/home/user/project/logs/output.log"
			if result != expected {
				t.Errorf("Unix path resolution = %v, want %v", result, expected)
			}
		})
	}
}

func TestDefaultPathResolver_EdgeCases(t *testing.T) {
	resolver := NewPathResolver()
	
	t.Run("path with spaces", func(t *testing.T) {
		result, err := resolver.ResolvePath("my logs/output file.log", "/project")
		if err != nil {
			t.Errorf("Path with spaces failed: %v", err)
		}
		expected := filepath.Join("/project", "my logs", "output file.log")
		if result != expected {
			t.Errorf("Path with spaces = %v, want %v", result, expected)
		}
	})
	
	t.Run("path with special characters", func(t *testing.T) {
		result, err := resolver.ResolvePath("logs/output-file_v1.2.log", "/project")
		if err != nil {
			t.Errorf("Path with special characters failed: %v", err)
		}
		expected := filepath.Join("/project", "logs", "output-file_v1.2.log")
		if result != expected {
			t.Errorf("Path with special characters = %v, want %v", result, expected)
		}
	})
	
	t.Run("path normalization", func(t *testing.T) {
		result, err := resolver.ResolvePath("./logs/../logs/./output.log", "/project")
		if err != nil {
			t.Errorf("Path normalization failed: %v", err)
		}
		expected := filepath.Join("/project", "logs", "output.log")
		if result != expected {
			t.Errorf("Path normalization = %v, want %v", result, expected)
		}
	})
}