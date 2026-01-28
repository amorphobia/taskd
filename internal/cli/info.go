package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"taskd/internal/task"
)

var infoCmd = &cobra.Command{
	Use:   "info [task-name]",
	Short: "Show detailed task information", // Replaces the original status command
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		info, err := task.GetTaskDetailInfo(taskName)
		if err != nil {
			return fmt.Errorf("failed to get task info: %w", err)
		}
		
		// Display complete task information (including all original status information)
		displayTaskInfo(info)
		return nil
	},
}

func displayTaskInfo(info *task.TaskDetailInfo) {
	// Display basic status information (original status command content)
	fmt.Printf("Task Name: %s\n", info.Name)
	fmt.Printf("Status: %s\n", info.Status)
	fmt.Printf("Process ID: %d\n", info.PID)
	fmt.Printf("Start Time: %s\n", info.StartTime)
	fmt.Printf("Executable: %s\n", info.Executable)
	
	// Display exit information
	if info.ExitCode != 0 {
		fmt.Printf("Exit Code: %d\n", info.ExitCode)
	}
	if info.LastError != "" {
		fmt.Printf("Last Error: %s\n", info.LastError)
	}
	
	// Display configuration information
	fmt.Printf("Working Directory: %s\n", info.WorkDir)
	if len(info.Args) > 0 {
		fmt.Printf("Arguments: %v\n", info.Args)
	}
	if len(info.Env) > 0 {
		fmt.Printf("Environment Variables: %v\n", info.Env)
	}
	fmt.Printf("Inherit Environment: %t\n", info.InheritEnv)
	
	// Display IO redirection information
	if info.IOInfo.StdinPath != "" {
		fmt.Printf("Standard Input: %s\n", info.IOInfo.StdinPath)
	}
	if info.IOInfo.StdoutPath != "" {
		fmt.Printf("Standard Output: %s\n", info.IOInfo.StdoutPath)
	}
	if info.IOInfo.StderrPath != "" {
		fmt.Printf("Standard Error: %s\n", info.IOInfo.StderrPath)
	}
	if info.IOInfo.SameOutput {
		fmt.Printf("Note: Standard output and error are redirected to the same file\n")
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)
}