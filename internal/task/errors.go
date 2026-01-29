package task

import (
	"fmt"
	"os"
	"syscall"
)

// FileErrorType represents different types of file errors
type FileErrorType int

const (
	FileErrorUnknown FileErrorType = iota
	FileErrorNotFound
	FileErrorPermissionDenied
	FileErrorAlreadyExists
	FileErrorIsDirectory
	FileErrorNotDirectory
	FileErrorDiskFull
	FileErrorInvalidPath
	FileErrorAccessDenied
)

// FileError represents a structured file operation error
type FileError struct {
	Type      FileErrorType
	Path      string
	Operation string
	Cause     error
}

func (e *FileError) Error() string {
	switch e.Type {
	case FileErrorNotFound:
		return fmt.Sprintf("File not found: '%s'\nPlease check if the file exists and the path is correct.", e.Path)
	case FileErrorPermissionDenied:
		return fmt.Sprintf("Permission denied: Cannot %s '%s'\nPlease check file permissions or run with appropriate privileges.", e.Operation, e.Path)
	case FileErrorAlreadyExists:
		return fmt.Sprintf("File already exists: '%s'\nUse a different filename or remove the existing file.", e.Path)
	case FileErrorIsDirectory:
		return fmt.Sprintf("Path is a directory, expected a file: '%s'\nPlease specify a file path instead of a directory.", e.Path)
	case FileErrorNotDirectory:
		return fmt.Sprintf("Path is not a directory: '%s'\nPlease specify a valid directory path.", e.Path)
	case FileErrorDiskFull:
		return fmt.Sprintf("Disk full: Cannot %s '%s'\nPlease free up disk space and try again.", e.Operation, e.Path)
	case FileErrorInvalidPath:
		return fmt.Sprintf("Invalid path: '%s'\nPlease check the path format and avoid reserved names or invalid characters.", e.Path)
	case FileErrorAccessDenied:
		return fmt.Sprintf("Access denied: Cannot %s '%s'\nPlease check if you have the necessary permissions.", e.Operation, e.Path)
	default:
		if e.Cause != nil {
			return fmt.Sprintf("File operation failed: %s '%s'\nError details: %v\nPlease check the file path and permissions.", e.Operation, e.Path, e.Cause)
		}
		return fmt.Sprintf("File operation failed: %s '%s'\nPlease check the file path and try again.", e.Operation, e.Path)
	}
}

// handleFileError analyzes a file operation error and returns a structured FileError
func handleFileError(err error, path, operation string) *FileError {
	if err == nil {
		return nil
	}

	fileErr := &FileError{
		Type:      FileErrorUnknown,
		Path:      path,
		Operation: operation,
		Cause:     err,
	}

	// Check for specific error types
	if os.IsNotExist(err) {
		fileErr.Type = FileErrorNotFound
		return fileErr
	}

	if os.IsExist(err) {
		fileErr.Type = FileErrorAlreadyExists
		return fileErr
	}

	if os.IsPermission(err) {
		fileErr.Type = FileErrorPermissionDenied
		return fileErr
	}

	// Check for syscall errors (Windows and Unix)
	if pathErr, ok := err.(*os.PathError); ok {
		return handlePathError(pathErr, path, operation)
	}

	// Check for syscall errors directly
	if errno, ok := err.(syscall.Errno); ok {
		return handleSyscallError(errno, path, operation)
	}

	return fileErr
}

// handlePathError handles os.PathError specifically
func handlePathError(pathErr *os.PathError, path, operation string) *FileError {
	fileErr := &FileError{
		Path:      path,
		Operation: operation,
		Cause:     pathErr,
	}

	if pathErr.Err == syscall.ENOENT {
		fileErr.Type = FileErrorNotFound
	} else if pathErr.Err == syscall.EACCES || pathErr.Err == syscall.EPERM {
		fileErr.Type = FileErrorPermissionDenied
	} else if pathErr.Err == syscall.EEXIST {
		fileErr.Type = FileErrorAlreadyExists
	} else if pathErr.Err == syscall.EISDIR {
		fileErr.Type = FileErrorIsDirectory
	} else if pathErr.Err == syscall.ENOTDIR {
		fileErr.Type = FileErrorNotDirectory
	} else if pathErr.Err == syscall.ENOSPC {
		fileErr.Type = FileErrorDiskFull
	} else {
		fileErr.Type = FileErrorUnknown
	}

	return fileErr
}

// handleSyscallError handles syscall.Errno directly
func handleSyscallError(errno syscall.Errno, path, operation string) *FileError {
	fileErr := &FileError{
		Path:      path,
		Operation: operation,
		Cause:     errno,
	}

	switch errno {
	case syscall.ENOENT:
		fileErr.Type = FileErrorNotFound
	case syscall.EACCES, syscall.EPERM:
		fileErr.Type = FileErrorPermissionDenied
	case syscall.EEXIST:
		fileErr.Type = FileErrorAlreadyExists
	case syscall.EISDIR:
		fileErr.Type = FileErrorIsDirectory
	case syscall.ENOTDIR:
		fileErr.Type = FileErrorNotDirectory
	case syscall.ENOSPC:
		fileErr.Type = FileErrorDiskFull
	default:
		fileErr.Type = FileErrorUnknown
	}

	return fileErr
}

// isFileError checks if an error is a FileError
func isFileError(err error) (*FileError, bool) {
	if fileErr, ok := err.(*FileError); ok {
		return fileErr, true
	}
	return nil, false
}

// wrapFileError wraps an existing error as a FileError if it isn't already
func wrapFileError(err error, path, operation string) error {
	if err == nil {
		return nil
	}
	
	if _, ok := isFileError(err); ok {
		return err // Already a FileError
	}
	
	return handleFileError(err, path, operation)
}

// checkDiskSpace checks if there's enough disk space for a file operation
func checkDiskSpace(path string, requiredBytes int64) error {
	// This is a simplified check - in a real implementation, you might want
	// to use platform-specific APIs to check available disk space
	
	// Try to create a temporary file to test write permissions and space
	tempFile, err := os.CreateTemp(path, "taskd_space_check_")
	if err != nil {
		return handleFileError(err, path, "check disk space")
	}
	
	// Try to write some data to test space availability
	testData := make([]byte, min(requiredBytes, 1024)) // Test with up to 1KB
	_, writeErr := tempFile.Write(testData)
	
	// Clean up immediately
	tempFile.Close()
	os.Remove(tempFile.Name())
	
	if writeErr != nil {
		// Check if it's a disk space error
		if fileErr := handleFileError(writeErr, path, "test disk space"); fileErr != nil {
			if fileErr.Type == FileErrorDiskFull {
				return &FileError{
					Type:      FileErrorDiskFull,
					Path:      path,
					Operation: "check disk space",
					Cause:     writeErr,
				}
			}
			return fileErr
		}
		return writeErr
	}
	
	return nil
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// ValidateFilePermissions checks if we have the required permissions for a file operation
func ValidateFilePermissions(path, operation string) error {
	return validateFilePermissions(path, operation)
}

// validateFilePermissions checks if we have the required permissions for a file operation
func validateFilePermissions(path, operation string) error {
	switch operation {
	case "read":
		file, err := os.Open(path)
		if err != nil {
			return wrapFileError(err, path, operation)
		}
		file.Close()
		return nil
		
	case "write", "create":
		// Check if file exists
		if _, err := os.Stat(path); err == nil {
			// File exists, check write permission
			file, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				return wrapFileError(err, path, operation)
			}
			file.Close()
			return nil
		} else if os.IsNotExist(err) {
			// File doesn't exist, check if we can create it
			file, err := os.Create(path)
			if err != nil {
				return wrapFileError(err, path, operation)
			}
			file.Close()
			os.Remove(path) // Clean up test file
			return nil
		} else {
			return wrapFileError(err, path, operation)
		}
		
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}