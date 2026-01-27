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

var statusCmd = &cobra.Command{
	Use:   "status [task-name]",
	Short: "Show task status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		manager := task.GetManager()
		info, err := manager.GetTaskStatus(taskName)
		if err != nil {
			return fmt.Errorf("failed to get task status: %w", err)
		}
		
		fmt.Printf("Task Name: %s\n", info.Name)
		fmt.Printf("Status: %s\n", info.Status)
		fmt.Printf("Process ID: %d\n", info.PID)
		fmt.Printf("Start Time: %s\n", info.StartTime)
		fmt.Printf("Executable: %s\n", info.Executable)
		if info.ExitCode != 0 {
			fmt.Printf("Exit Code: %d\n", info.ExitCode)
		}
		if info.LastError != "" {
			fmt.Printf("Last Error: %s\n", info.LastError)
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
}