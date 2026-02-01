package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	taskdconfig "taskd/internal/config"
	"taskd/internal/task"
)

var delCmd = &cobra.Command{
	Use:   "del [task-name]",
	Short: "Delete a task",
	Long: `Delete a task and its configuration file.

This command will:
- Stop the task if it's currently running
- Remove the task configuration file
- Remove the task from the task manager

The task cannot be recovered after deletion.`,
	Args: cobra.ExactArgs(1),
	RunE: runDelCommand,
}

func init() {
	rootCmd.AddCommand(delCmd)
}

func runDelCommand(cmd *cobra.Command, args []string) error {
	taskName := args[0]
	
	// Validate task name
	if taskName == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	
	// Check if this is a builtin task
	manager := task.GetManager()
	if err := manager.ValidateBuiltinTaskOperation(taskName, "del"); err != nil {
		return err
	}
	
	// Check if task exists
	taskInfo, err := task.GetTaskStatus(taskName)
	if err != nil {
		return fmt.Errorf("task '%s' not found", taskName)
	}
	
	// Stop the task if it's running
	if taskInfo.Status == "running" {
		fmt.Printf("Task '%s' is currently running. Stopping...\n", taskName)
		if err := task.StopTask(taskName); err != nil {
			return fmt.Errorf("failed to stop task '%s': %w", taskName, err)
		}
		fmt.Printf("Task '%s' stopped.\n", taskName)
	}
	
	// Remove the task from manager
	if err := task.RemoveTask(taskName); err != nil {
		return fmt.Errorf("failed to remove task '%s': %w", taskName, err)
	}
	
	// Remove the configuration file
	configPath := filepath.Join(taskdconfig.GetTaskDTasksDir(), taskName+".toml")
	if err := os.Remove(configPath); err != nil {
		// If file doesn't exist, that's okay - task is still considered deleted
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove configuration file '%s': %v\n", configPath, err)
		}
	}
	
	fmt.Printf("Task '%s' deleted successfully.\n", taskName)
	return nil
}