package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"taskd/internal/task"
)

var addCmd = &cobra.Command{
	Use:   "add [task-name]",
	Short: "Add a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		// Validate task name
		if err := validateTaskName(taskName); err != nil {
			return fmt.Errorf("invalid task name: %w", err)
		}
		
		// Get command line arguments
		exec, _ := cmd.Flags().GetString("exec")
		workdir, _ := cmd.Flags().GetString("workdir")
		env, _ := cmd.Flags().GetStringSlice("env")
		inheritEnv, _ := cmd.Flags().GetBool("inherit-env")
		stdin, _ := cmd.Flags().GetString("stdin")
		stdout, _ := cmd.Flags().GetString("stdout")
		stderr, _ := cmd.Flags().GetString("stderr")
		
		// Validate executable
		if err := validateExecutable(exec); err != nil {
			return fmt.Errorf("invalid executable: %w", err)
		}
		
		// Validate working directory
		if workdir != "" {
			if err := validateWorkingDirectory(workdir); err != nil {
				return fmt.Errorf("invalid working directory: %w", err)
			}
		}
		
		// If no working directory specified, use user's home directory
		if workdir == "" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				workdir = homeDir
			}
		}
		
		// Validate environment variables
		if err := validateEnvironmentVariables(env); err != nil {
			return fmt.Errorf("invalid environment variables: %w", err)
		}
		
		// Validate IO redirection paths
		if err := validateIOPaths(stdin, stdout, stderr, workdir); err != nil {
			return fmt.Errorf("invalid IO redirection: %w", err)
		}
		
		taskConfig := &task.Config{
			Name:       taskName,
			Executable: exec,
			WorkDir:    workdir,
			Env:        env,
			InheritEnv: inheritEnv,
			Stdin:      stdin,
			Stdout:     stdout,
			Stderr:     stderr,
		}
		
		if err := task.AddTask(taskConfig); err != nil {
			return fmt.Errorf("failed to add task: %w", err)
		}
		
		fmt.Printf("Task '%s' added successfully\n", taskName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	
	addCmd.Flags().StringP("exec", "e", "", "executable path and arguments (required)")
	addCmd.Flags().StringP("workdir", "w", "", "working directory")
	addCmd.Flags().StringSliceP("env", "E", nil, "environment variables (format: KEY=VALUE)")
	addCmd.Flags().BoolP("inherit-env", "i", true, "inherit system environment variables")
	addCmd.Flags().String("stdin", "", "standard input file")
	addCmd.Flags().String("stdout", "", "standard output redirect file (relative paths resolved from working directory)")
	addCmd.Flags().String("stderr", "", "standard error redirect file (relative paths resolved from working directory)")
	
	addCmd.MarkFlagRequired("exec")
}

// validateTaskName validates the task name
func validateTaskName(name string) error {
	if name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	
	// Check for valid characters (alphanumeric, dash, underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("task name can only contain letters, numbers, dashes, and underscores")
	}
	
	// Check length
	if len(name) > 50 {
		return fmt.Errorf("task name cannot be longer than 50 characters")
	}
	
	return nil
}

// validateExecutable validates the executable command
func validateExecutable(exec string) error {
	if exec == "" {
		return fmt.Errorf("executable cannot be empty")
	}
	
	if strings.TrimSpace(exec) == "" {
		return fmt.Errorf("executable cannot be only whitespace")
	}
	
	return nil
}

// validateWorkingDirectory validates the working directory
func validateWorkingDirectory(workdir string) error {
	if workdir == "" {
		return nil // Empty is allowed, will use default
	}
	
	// Check if directory exists
	info, err := os.Stat(workdir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("working directory does not exist: %s", workdir)
		}
		return fmt.Errorf("cannot access working directory: %w", err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("working directory path is not a directory: %s", workdir)
	}
	
	return nil
}

// validateEnvironmentVariables validates environment variable format
func validateEnvironmentVariables(envVars []string) error {
	for _, env := range envVars {
		if env == "" {
			return fmt.Errorf("environment variable cannot be empty")
		}
		
		if !strings.Contains(env, "=") {
			return fmt.Errorf("environment variable must be in KEY=VALUE format: %s", env)
		}
		
		parts := strings.SplitN(env, "=", 2)
		if parts[0] == "" {
			return fmt.Errorf("environment variable key cannot be empty: %s", env)
		}
		
		// Validate key format (should be valid environment variable name)
		validKey := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		if !validKey.MatchString(parts[0]) {
			return fmt.Errorf("invalid environment variable key format: %s", parts[0])
		}
	}
	
	return nil
}

// validateIOPaths validates input/output redirection paths
func validateIOPaths(stdin, stdout, stderr, workdir string) error {
	pathResolver := task.NewPathResolver()
	
	// Validate stdin path if specified
	if stdin != "" {
		stdinPath, err := pathResolver.ResolvePath(stdin, workdir)
		if err != nil {
			return fmt.Errorf("invalid stdin path: %w", err)
		}
		
		// Check if stdin file exists
		if _, err := os.Stat(stdinPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("stdin file does not exist: %s", stdinPath)
			}
			return fmt.Errorf("cannot access stdin file: %w", err)
		}
	}
	
	// Validate stdout path if specified
	if stdout != "" {
		stdoutPath, err := pathResolver.ResolvePath(stdout, workdir)
		if err != nil {
			return fmt.Errorf("invalid stdout path: %w", err)
		}
		
		// Check if parent directory exists or can be created
		stdoutDir := filepath.Dir(stdoutPath)
		if err := validateOutputDirectory(stdoutDir); err != nil {
			return fmt.Errorf("stdout directory error: %w", err)
		}
	}
	
	// Validate stderr path if specified
	if stderr != "" {
		stderrPath, err := pathResolver.ResolvePath(stderr, workdir)
		if err != nil {
			return fmt.Errorf("invalid stderr path: %w", err)
		}
		
		// Check if parent directory exists or can be created
		stderrDir := filepath.Dir(stderrPath)
		if err := validateOutputDirectory(stderrDir); err != nil {
			return fmt.Errorf("stderr directory error: %w", err)
		}
	}
	
	return nil
}

// validateOutputDirectory validates that output directory exists or can be created
func validateOutputDirectory(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("cannot create directory %s: %w", dir, err)
			}
			return nil
		}
		return fmt.Errorf("cannot access directory %s: %w", dir, err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", dir)
	}
	
	return nil
}