package task

import (
	"fmt"
	"os"
	"path/filepath"
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
		return fmt.Errorf("path contains invalid characters: %s", path)
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
			return fmt.Errorf("path exists but is not a directory: %s", dirPath)
		}
		return nil // Directory already exists
	}
	
	// Create directory (including parent directories)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}
	
	return nil
}