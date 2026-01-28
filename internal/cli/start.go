package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"taskd/internal/task"
)

var startCmd = &cobra.Command{
	Use:   "start [task-name]",
	Short: "Start a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		manager := task.GetManager()
		if err := manager.StartTask(taskName); err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}
		
		fmt.Printf("Task '%s' started successfully\n", taskName)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [task-name]",
	Short: "Stop a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		manager := task.GetManager()
		if err := manager.StopTask(taskName); err != nil {
			return fmt.Errorf("failed to stop task: %w", err)
		}
		
		fmt.Printf("Task '%s' stopped successfully\n", taskName)
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [task-name]",
	Short: "Restart a task (stop if running, then start)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		manager := task.GetManager()
		if err := manager.RestartTask(taskName); err != nil {
			return fmt.Errorf("failed to restart task: %w", err)
		}
		
		fmt.Printf("Task '%s' restarted successfully\n", taskName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	// status command has been replaced by info command
}