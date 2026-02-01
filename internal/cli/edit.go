package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	taskdconfig "taskd/internal/config"
	"taskd/internal/task"
)

var editCmd = &cobra.Command{
	Use:   "edit [task-name]",
	Short: "Edit an existing task configuration",
	Long: `Edit an existing task configuration. You can modify any task settings including
executable, working directory, environment variables, and IO redirection.

Examples:
  # Change executable
  taskd edit mytask --exec "python app.py"
  
  # Update working directory
  taskd edit mytask --workdir "/new/path"
  
  # Add environment variables (replaces existing ones)
  taskd edit mytask --env "KEY1=value1" --env "KEY2=value2"
  
  # Clear environment variables
  taskd edit mytask --clear-env
  
  # Update IO redirection
  taskd edit mytask --stdout "new-output.log" --stderr "new-error.log"
  
  # Clear IO redirection
  taskd edit mytask --clear-stdin --clear-stdout --clear-stderr
  
  # Combine multiple changes
  taskd edit mytask --exec "node server.js" --workdir "/app" --stdout "server.log"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		// Check if this is a builtin task
		manager := task.GetManager()
		if err := manager.ValidateBuiltinTaskOperation(taskName, "edit"); err != nil {
			return err
		}
		
		// Get current task configuration
		currentInfo, err := task.GetTaskDetailInfo(taskName)
		if err != nil {
			return fmt.Errorf("task '%s' not found: %w", taskName, err)
		}
		
		// Check if task is running
		if currentInfo.Status == "running" {
			return fmt.Errorf("cannot edit task '%s' while it is running. Please stop the task first", taskName)
		}
		
		// Parse edit flags
		editConfig, err := parseEditFlags(cmd, currentInfo)
		if err != nil {
			return fmt.Errorf("invalid edit parameters: %w", err)
		}
		
		// Check if any changes were specified
		if !hasAnyChanges(editConfig) {
			return fmt.Errorf("no changes specified. Use --help to see available options")
		}
		
		// Validate the new configuration
		if err := validateEditConfig(editConfig, currentInfo); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		
		// Apply the changes
		if err := applyTaskEdit(taskName, editConfig); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}
		
		fmt.Printf("Task '%s' updated successfully\n", taskName)
		return nil
	},
}

// EditConfig represents the configuration changes to apply
type EditConfig struct {
	Name        string
	DisplayName *string   // pointer to distinguish between empty string and not set
	Description *string
	Executable  *string   // pointer to distinguish between empty string and not set
	WorkDir     *string
	Env         []string
	InheritEnv  *bool
	Stdin       *string
	Stdout      *string
	Stderr      *string
	
	// Clear flags
	ClearEnv    bool
	ClearStdin  bool
	ClearStdout bool
	ClearStderr bool
}

func parseEditFlags(cmd *cobra.Command, currentInfo *task.TaskDetailInfo) (*EditConfig, error) {
	config := &EditConfig{
		Name: currentInfo.Name,
	}
	
	// Parse display name
	if cmd.Flags().Changed("display-name") {
		displayName, _ := cmd.Flags().GetString("display-name")
		config.DisplayName = &displayName
	}
	
	// Parse description
	if cmd.Flags().Changed("description") {
		description, _ := cmd.Flags().GetString("description")
		config.Description = &description
	}
	
	// Parse executable
	if cmd.Flags().Changed("exec") {
		exec, _ := cmd.Flags().GetString("exec")
		config.Executable = &exec
	}
	
	// Parse working directory
	if cmd.Flags().Changed("workdir") {
		workdir, _ := cmd.Flags().GetString("workdir")
		config.WorkDir = &workdir
	}
	
	// Parse environment variables
	if cmd.Flags().Changed("env") {
		env, _ := cmd.Flags().GetStringSlice("env")
		config.Env = env
	}
	
	// Parse inherit environment
	if cmd.Flags().Changed("inherit-env") {
		inheritEnv, _ := cmd.Flags().GetBool("inherit-env")
		config.InheritEnv = &inheritEnv
	}
	
	// Parse IO redirection
	if cmd.Flags().Changed("stdin") {
		stdin, _ := cmd.Flags().GetString("stdin")
		config.Stdin = &stdin
	}
	
	if cmd.Flags().Changed("stdout") {
		stdout, _ := cmd.Flags().GetString("stdout")
		config.Stdout = &stdout
	}
	
	if cmd.Flags().Changed("stderr") {
		stderr, _ := cmd.Flags().GetString("stderr")
		config.Stderr = &stderr
	}
	
	// Parse clear flags
	config.ClearEnv, _ = cmd.Flags().GetBool("clear-env")
	config.ClearStdin, _ = cmd.Flags().GetBool("clear-stdin")
	config.ClearStdout, _ = cmd.Flags().GetBool("clear-stdout")
	config.ClearStderr, _ = cmd.Flags().GetBool("clear-stderr")
	
	return config, nil
}

// hasAnyChanges checks if any changes were specified in the edit configuration
func hasAnyChanges(config *EditConfig) bool {
	// Check if any update flags were set
	if config.DisplayName != nil ||
		config.Description != nil ||
		config.Executable != nil ||
		config.WorkDir != nil ||
		len(config.Env) > 0 ||
		config.InheritEnv != nil ||
		config.Stdin != nil ||
		config.Stdout != nil ||
		config.Stderr != nil {
		return true
	}
	
	// Check if any clear flags were set
	if config.ClearEnv ||
		config.ClearStdin ||
		config.ClearStdout ||
		config.ClearStderr {
		return true
	}
	
	return false
}

func validateEditConfig(config *EditConfig, currentInfo *task.TaskDetailInfo) error {
	// Validate required fields cannot be cleared
	if config.Executable != nil && strings.TrimSpace(*config.Executable) == "" {
		return fmt.Errorf("executable cannot be empty (required field)")
	}
	
	// Validate task name format (if somehow changed)
	if err := validateTaskName(config.Name); err != nil {
		return fmt.Errorf("invalid task name: %w", err)
	}
	
	// Validate executable if provided
	if config.Executable != nil {
		if err := validateExecutable(*config.Executable); err != nil {
			return fmt.Errorf("invalid executable: %w", err)
		}
	}
	
	// Validate working directory if provided
	if config.WorkDir != nil && *config.WorkDir != "" {
		if err := validateWorkingDirectory(*config.WorkDir); err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}
	}
	
	// Validate environment variables if provided
	if len(config.Env) > 0 {
		if err := validateEnvironmentVariables(config.Env); err != nil {
			return fmt.Errorf("invalid environment variables: %w", err)
		}
	}
	
	// Validate IO paths if provided
	workdir := currentInfo.WorkDir  // Use current task's working directory
	if config.WorkDir != nil {
		workdir = *config.WorkDir    // Or use the new working directory if being updated
	}
	
	stdin := ""
	if config.Stdin != nil {
		stdin = *config.Stdin
	}
	if config.ClearStdin {
		stdin = ""
	}
	
	stdout := ""
	if config.Stdout != nil {
		stdout = *config.Stdout
	}
	if config.ClearStdout {
		stdout = ""
	}
	
	stderr := ""
	if config.Stderr != nil {
		stderr = *config.Stderr
	}
	if config.ClearStderr {
		stderr = ""
	}
	
	// Only validate IO paths if they are being set (not cleared)
	if stdin != "" || stdout != "" || stderr != "" {
		if err := validateIOPaths(stdin, stdout, stderr, workdir); err != nil {
			return fmt.Errorf("invalid IO redirection: %w", err)
		}
	}
	
	// Check for configuration conflicts (but skip task name conflict for edit)
	execStr := ""
	if config.Executable != nil {
		execStr = *config.Executable
	}
	
	if err := validateIOConflicts(stdin, stdout, stderr); err != nil {
		return fmt.Errorf("configuration conflict: %w", err)
	}
	
	if err := validateExecutableConflicts(execStr, stdin, stdout, stderr); err != nil {
		return fmt.Errorf("configuration conflict: %w", err)
	}
	
	return nil
}

func applyTaskEdit(taskName string, editConfig *EditConfig) error {
	// Load current task configuration from file
	manager := task.GetManager()
	configPath := filepath.Join(taskdconfig.GetTaskDTasksDir(), taskName+".toml")
	
	// Read current configuration
	var currentConfig task.Config
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("task configuration file not found: %w", err)
	}
	
	// Load current config from file
	if err := loadTaskConfig(configPath, &currentConfig); err != nil {
		return fmt.Errorf("failed to load current configuration: %w", err)
	}
	
	// Apply changes
	newConfig := currentConfig
	
	if editConfig.DisplayName != nil {
		newConfig.DisplayName = *editConfig.DisplayName
	}
	
	if editConfig.Description != nil {
		newConfig.Description = *editConfig.Description
	}
	
	if editConfig.Executable != nil {
		newConfig.Executable = *editConfig.Executable
	}
	
	if editConfig.WorkDir != nil {
		newConfig.WorkDir = *editConfig.WorkDir
	}
	
	if editConfig.InheritEnv != nil {
		newConfig.InheritEnv = *editConfig.InheritEnv
	}
	
	// Handle environment variables
	if editConfig.ClearEnv {
		newConfig.Env = []string{}
	} else if len(editConfig.Env) > 0 {
		newConfig.Env = editConfig.Env
	}
	
	// Handle IO redirection
	if editConfig.ClearStdin {
		newConfig.Stdin = ""
	} else if editConfig.Stdin != nil {
		newConfig.Stdin = *editConfig.Stdin
	}
	
	if editConfig.ClearStdout {
		newConfig.Stdout = ""
	} else if editConfig.Stdout != nil {
		newConfig.Stdout = *editConfig.Stdout
	}
	
	if editConfig.ClearStderr {
		newConfig.Stderr = ""
	} else if editConfig.Stderr != nil {
		newConfig.Stderr = *editConfig.Stderr
	}
	
	// Save updated configuration
	if err := saveTaskConfig(configPath, &newConfig); err != nil {
		return fmt.Errorf("failed to save updated configuration: %w", err)
	}
	
	// Update in-memory task instance
	if err := manager.ReloadTask(taskName); err != nil {
		return fmt.Errorf("failed to reload task in memory: %w", err)
	}
	
	return nil
}

// Helper functions
func loadTaskConfig(configPath string, config *task.Config) error {
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return fmt.Errorf("failed to decode TOML file: %w", err)
	}
	return nil
}

func saveTaskConfig(configPath string, config *task.Config) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create/overwrite the file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	// Encode to TOML
	if err := toml.NewEncoder(file).Encode(config); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}
	
	return nil
}

func init() {
	rootCmd.AddCommand(editCmd)
	
	// Configuration flags
	editCmd.Flags().String("display-name", "", "update display name for the task")
	editCmd.Flags().String("description", "", "update description of the task")
	editCmd.Flags().StringP("exec", "e", "", "update executable path and arguments")
	editCmd.Flags().StringP("workdir", "w", "", "update working directory")
	editCmd.Flags().StringSliceP("env", "E", nil, "update environment variables (format: KEY=VALUE, replaces all existing)")
	editCmd.Flags().BoolP("inherit-env", "i", false, "update inherit system environment variables setting")
	
	// IO redirection flags
	editCmd.Flags().String("stdin", "", "update standard input file")
	editCmd.Flags().String("stdout", "", "update standard output redirect file")
	editCmd.Flags().String("stderr", "", "update standard error redirect file")
	
	// Clear flags
	editCmd.Flags().Bool("clear-env", false, "clear all environment variables")
	editCmd.Flags().Bool("clear-stdin", false, "clear standard input redirection")
	editCmd.Flags().Bool("clear-stdout", false, "clear standard output redirection")
	editCmd.Flags().Bool("clear-stderr", false, "clear standard error redirection")
}