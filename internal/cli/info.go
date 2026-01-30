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
	// Display header
	fmt.Printf("===============================================================\n")
	fmt.Printf("                    TASK INFORMATION                          \n")
	fmt.Printf("===============================================================\n")
	
	// Display basic status information
	fmt.Printf("Task Name:        %s\n", info.Name)
	
	// Display name and description if available
	if info.DisplayName != "" {
		fmt.Printf("Display Name:     %s\n", info.DisplayName)
	}
	
	if info.Description != "" {
		fmt.Printf("Description:      %s\n", info.Description)
	}
	
	// Status with simple indicators
	statusIndicator := getStatusIndicator(info.Status)
	fmt.Printf("Status:           [%s] %s\n", statusIndicator, info.Status)
	
	if info.PID > 0 {
		fmt.Printf("Process ID:       %d\n", info.PID)
	}
	
	if info.StartTime != "" && info.StartTime != "0001-01-01 00:00:00" {
		fmt.Printf("Start Time:       %s\n", info.StartTime)
	}
	
	fmt.Printf("Executable:       %s\n", info.Executable)
	
	// Display exit information
	if info.ExitCode != 0 {
		fmt.Printf("Exit Code:        %d\n", info.ExitCode)
	}
	if info.LastError != "" {
		fmt.Printf("Last Error:       %s\n", info.LastError)
	}
	
	fmt.Printf("\n")
	fmt.Printf("---------------------------------------------------------------\n")
	fmt.Printf("                    CONFIGURATION                             \n")
	fmt.Printf("---------------------------------------------------------------\n")
	
	// Display configuration information
	fmt.Printf("Working Directory: %s\n", info.WorkDir)
	
	if len(info.Args) > 0 {
		fmt.Printf("Arguments:         ")
		for i, arg := range info.Args {
			if i > 0 {
				fmt.Printf(" ")
			}
			fmt.Printf("\"%s\"", arg)
		}
		fmt.Printf("\n")
	}
	
	if len(info.Env) > 0 {
		fmt.Printf("Environment:       \n")
		for _, env := range info.Env {
			fmt.Printf("                   %s\n", env)
		}
	}
	
	fmt.Printf("Inherit Env:       %s\n", getBoolIndicator(info.InheritEnv))
	
	// Display IO redirection information
	if info.IOInfo.StdinPath != "" || info.IOInfo.StdoutPath != "" || info.IOInfo.StderrPath != "" {
		fmt.Printf("\n")
		fmt.Printf("---------------------------------------------------------------\n")
		fmt.Printf("                  IO REDIRECTION                              \n")
		fmt.Printf("---------------------------------------------------------------\n")
		
		if info.IOInfo.StdinPath != "" {
			fmt.Printf("Standard Input:    %s\n", info.IOInfo.StdinPath)
		}
		if info.IOInfo.StdoutPath != "" {
			fmt.Printf("Standard Output:   %s\n", info.IOInfo.StdoutPath)
		}
		if info.IOInfo.StderrPath != "" {
			fmt.Printf("Standard Error:    %s\n", info.IOInfo.StderrPath)
		}
		if info.IOInfo.SameOutput {
			fmt.Printf("Note:              Standard output and error are redirected to the same file\n")
		}
	}
	
	fmt.Printf("===============================================================\n")
}

// getStatusIndicator returns a simple ASCII indicator for the task status
func getStatusIndicator(status string) string {
	switch status {
	case "running":
		return "RUN"
	case "stopped":
		return "STOP"
	case "starting":
		return "START"
	case "stopping":
		return "STOP"
	default:
		return "UNKN"
	}
}

// getBoolIndicator returns a simple ASCII display for boolean values
func getBoolIndicator(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func init() {
	rootCmd.AddCommand(infoCmd)
}