package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathResolver path resolver interface
type PathResolver interface {
	// ResolvePath resolves path (relative paths based on working directory)
	ResolvePath(path string, workDir string) (string, error)
	
	// ValidatePath validates if path is valid
	ValidatePath(path string) error
	
	// EnsureDir ensures directory exists
	EnsureDir(path string) error
}

// DefaultPathResolver default path resolver
type DefaultPathResolver struct{}

// NewPathResolver creates a new path resolver
func NewPathResolver() PathResolver {
	return &DefaultPathResolver{}
}

// ResolvePath resolves path
func (r *DefaultPathResolver) ResolvePath(path string, workDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	
	// If it's an absolute path, return directly
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	
	// Relative path based on working directory
	if workDir == "" {
		return "", fmt.Errorf("workDir cannot be empty for relative path")
	}
	
	resolved := filepath.Join(workDir, path)
	return filepath.Clean(resolved), nil
}

// ValidatePath validates if path is valid
func (r *DefaultPathResolver) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	// Check if path contains invalid characters (cross-platform consideration)
	cleaned := filepath.Clean(path)
	if cleaned != path && cleaned != filepath.FromSlash(path) {
		return &FileError{
			Type:      FileErrorInvalidPath,
			Path:      path,
			Operation: "validate path",
		}
	}
	
	// Check for reserved names on Windows
	if err := r.validateWindowsReservedNames(path); err != nil {
		return err
	}
	
	// Check path length limits
	if len(path) > 260 { // Windows MAX_PATH limit
		return &FileError{
			Type:      FileErrorInvalidPath,
			Path:      path,
			Operation: "validate path length",
		}
	}
	
	return nil
}

// validateWindowsReservedNames checks for Windows reserved file names
func (r *DefaultPathResolver) validateWindowsReservedNames(path string) error {
	// Extract filename from path
	filename := filepath.Base(path)
	
	// Remove extension for checking
	name := strings.ToUpper(filename)
	if idx := strings.LastIndex(name, "."); idx != -1 {
		name = name[:idx]
	}
	
	// Windows reserved names
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	
	for _, reserved := range reservedNames {
		if name == reserved {
			return &FileError{
				Type:      FileErrorInvalidPath,
				Path:      path,
				Operation: "validate reserved name",
			}
		}
	}
	
	return nil
}

// EnsureDir ensures directory exists
func (r *DefaultPathResolver) EnsureDir(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	
	// Check if directory already exists
	if info, err := os.Stat(dirPath); err == nil {
		if !info.IsDir() {
			return &FileError{
				Type:      FileErrorNotDirectory,
				Path:      dirPath,
				Operation: "ensure directory",
			}
		}
		return nil // Directory already exists
	} else if !os.IsNotExist(err) {
		return wrapFileError(err, dirPath, "check directory")
	}
	
	// Check if parent directory is writable before attempting to create
	parentDir := filepath.Dir(dirPath)
	if parentDir != dirPath { // Avoid infinite recursion for root directories
		if err := r.checkDirectoryWritable(parentDir); err != nil {
			return fmt.Errorf("cannot create directory %s: parent directory not writable: %w", dirPath, err)
		}
	}
	
	// Create directory (including parent directories)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		// Provide more specific error messages for common failures
		if os.IsPermission(err) {
			return &FileError{
				Type:      FileErrorPermissionDenied,
				Path:      dirPath,
				Operation: "create directory",
				Cause:     err,
			}
		}
		return wrapFileError(err, dirPath, "create directory")
	}
	
	return nil
}

// checkDirectoryWritable checks if a directory is writable
func (r *DefaultPathResolver) checkDirectoryWritable(dirPath string) error {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, check parent recursively
			parentDir := filepath.Dir(dirPath)
			if parentDir != dirPath {
				return r.checkDirectoryWritable(parentDir)
			}
			return nil // Root directory, assume writable
		}
		return wrapFileError(err, dirPath, "check directory")
	}
	
	if !info.IsDir() {
		return &FileError{
			Type:      FileErrorNotDirectory,
			Path:      dirPath,
			Operation: "check directory writable",
		}
	}
	
	// Try to create a temporary file to test write permissions
	tempFile, err := os.CreateTemp(dirPath, "taskd_write_test_")
	if err != nil {
		return wrapFileError(err, dirPath, "test directory write")
	}
	
	// Clean up immediately
	tempFile.Close()
	os.Remove(tempFile.Name())
	
	return nil
}