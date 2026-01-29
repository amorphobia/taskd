package task

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	ioManager IOManager
	ioOnce    sync.Once
)

// IOManager IO manager interface
type IOManager interface {
	// CreateTaskIO creates task IO configuration
	CreateTaskIO(config *Config) (*TaskIO, error)
	
	// GetTaskIOInfo gets task IO information
	GetTaskIOInfo(config *Config) (*TaskIOInfo, error)
}

// TaskIO task IO configuration
type TaskIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	
	// File handle management
	files []io.Closer
}

// TaskIOInfo task IO information
type TaskIOInfo struct {
	StdinPath  string `json:"stdin_path,omitempty"`
	StdoutPath string `json:"stdout_path,omitempty"`
	StderrPath string `json:"stderr_path,omitempty"`
	SameOutput bool   `json:"same_output"` // whether stdout and stderr point to the same file
}

// Close closes all file handles
func (tio *TaskIO) Close() error {
	var lastErr error
	for _, file := range tio.files {
		if err := file.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// DefaultIOManager default IO manager
type DefaultIOManager struct {
	pathResolver PathResolver
}

// GetIOManager gets IO manager singleton
func GetIOManager() IOManager {
	ioOnce.Do(func() {
		ioManager = &DefaultIOManager{
			pathResolver: NewPathResolver(),
		}
	})
	return ioManager
}

// CreateTaskIO creates task IO configuration
func (m *DefaultIOManager) CreateTaskIO(config *Config) (*TaskIO, error) {
	taskIO := &TaskIO{}
	
	// Handle standard input
	if config.Stdin != "" {
		stdinPath, err := m.pathResolver.ResolvePath(config.Stdin, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stdin path: %w", err)
		}
		
		if err := m.pathResolver.ValidatePath(stdinPath); err != nil {
			return nil, fmt.Errorf("invalid stdin path: %w", err)
		}
		
		// Check file permissions before opening
		if err := validateFilePermissions(stdinPath, "read"); err != nil {
			return nil, fmt.Errorf("stdin file permission check failed: %w", err)
		}
		
		file, err := os.Open(stdinPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open stdin file: %w", wrapFileError(err, stdinPath, "open"))
		}
		
		taskIO.Stdin = file
		taskIO.files = append(taskIO.files, file)
	}
	
	// Handle standard output and error
	var stdoutWriter, stderrWriter io.Writer
	var stdoutPath, stderrPath string
	
	// Handle standard output
	if config.Stdout != "" {
		var err error
		stdoutPath, err = m.pathResolver.ResolvePath(config.Stdout, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stdout path: %w", err)
		}
		
		if err := m.pathResolver.ValidatePath(stdoutPath); err != nil {
			return nil, fmt.Errorf("invalid stdout path: %w", err)
		}
		
		if err := m.pathResolver.EnsureDir(filepath.Dir(stdoutPath)); err != nil {
			return nil, fmt.Errorf("failed to create stdout directory: %w", err)
		}
		
		// Check disk space before creating file
		if err := checkDiskSpace(filepath.Dir(stdoutPath), 1024); err != nil {
			return nil, fmt.Errorf("stdout disk space check failed: %w", err)
		}
		
		file, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open stdout file: %w", wrapFileError(err, stdoutPath, "create"))
		}
		
		stdoutWriter = file
		taskIO.files = append(taskIO.files, file)
	}
	
	// Handle standard error
	if config.Stderr != "" {
		var err error
		stderrPath, err = m.pathResolver.ResolvePath(config.Stderr, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stderr path: %w", err)
		}
		
		if err := m.pathResolver.ValidatePath(stderrPath); err != nil {
			return nil, fmt.Errorf("invalid stderr path: %w", err)
		}
		
		// Check if it's the same as stdout
		if stdoutPath != "" && stdoutPath == stderrPath {
			// Use the same writer
			stderrWriter = stdoutWriter
		} else {
			if err := m.pathResolver.EnsureDir(filepath.Dir(stderrPath)); err != nil {
				return nil, fmt.Errorf("failed to create stderr directory: %w", err)
			}
			
			// Check disk space before creating file
			if err := checkDiskSpace(filepath.Dir(stderrPath), 1024); err != nil {
				return nil, fmt.Errorf("stderr disk space check failed: %w", err)
			}
			
			file, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open stderr file: %w", wrapFileError(err, stderrPath, "create"))
			}
			
			stderrWriter = file
			taskIO.files = append(taskIO.files, file)
		}
	}
	
	taskIO.Stdout = stdoutWriter
	taskIO.Stderr = stderrWriter
	
	return taskIO, nil
}

// GetTaskIOInfo gets task IO information
func (m *DefaultIOManager) GetTaskIOInfo(config *Config) (*TaskIOInfo, error) {
	info := &TaskIOInfo{}
	
	// Parse path information
	if config.Stdin != "" {
		stdinPath, err := m.pathResolver.ResolvePath(config.Stdin, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stdin path: %w", err)
		}
		
		// Validate that stdin file exists at runtime
		if _, err := os.Stat(stdinPath); err != nil {
			if os.IsNotExist(err) {
				return nil, wrapFileError(err, stdinPath, "check stdin file")
			}
			return nil, wrapFileError(err, stdinPath, "access stdin file")
		}
		
		info.StdinPath = stdinPath
	}
	
	if config.Stdout != "" {
		stdoutPath, err := m.pathResolver.ResolvePath(config.Stdout, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stdout path: %w", err)
		}
		info.StdoutPath = stdoutPath
	}
	
	if config.Stderr != "" {
		stderrPath, err := m.pathResolver.ResolvePath(config.Stderr, config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stderr path: %w", err)
		}
		info.StderrPath = stderrPath
		
		// Check if it's the same as stdout
		if info.StdoutPath != "" && info.StdoutPath == stderrPath {
			info.SameOutput = true
		}
	}
	
	return info, nil
}