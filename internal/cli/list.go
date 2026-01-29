package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"taskd/internal/task"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		running, _ := cmd.Flags().GetBool("running")
		stopped, _ := cmd.Flags().GetBool("stopped")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		tasks, err := task.ListTasks()
		if err != nil {
			return fmt.Errorf("failed to get task list: %w", err)
		}
		
		// Filter tasks based on flags
		filteredTasks := filterTasks(tasks, running, stopped)
		
		if len(filteredTasks) == 0 {
			displayNoTasksMessage(running, stopped)
			return nil
		}
		
		// Display tasks
		if verbose {
			displayTasksVerbose(filteredTasks)
		} else {
			displayTasksCompact(filteredTasks)
		}
		
		// Display summary
		displayTasksSummary(tasks, len(filteredTasks))
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	
	listCmd.Flags().BoolP("running", "r", false, "show only running tasks")
	listCmd.Flags().BoolP("stopped", "s", false, "show only stopped tasks")
	listCmd.Flags().BoolP("verbose", "v", false, "show detailed information")
}
// filterTasks filters tasks based on running/stopped flags
func filterTasks(tasks []*task.TaskInfo, running, stopped bool) []*task.TaskInfo {
	if !running && !stopped {
		return tasks // No filter, return all
	}
	
	var filtered []*task.TaskInfo
	for _, t := range tasks {
		if running && t.Status == "running" {
			filtered = append(filtered, t)
		} else if stopped && t.Status != "running" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// displayNoTasksMessage shows appropriate message when no tasks match criteria
func displayNoTasksMessage(running, stopped bool) {
	if running {
		fmt.Printf("No running tasks found.\n")
		fmt.Printf("Use 'taskd list' to see all tasks or 'taskd start <task-name>' to start a task.\n")
	} else if stopped {
		fmt.Printf("No stopped tasks found.\n")
		fmt.Printf("Use 'taskd list' to see all tasks.\n")
	} else {
		fmt.Printf("No tasks configured.\n")
		fmt.Printf("Use 'taskd add <task-name> --exec \"<command>\"' to add your first task.\n")
	}
}

// displayTasksCompact shows tasks in a compact table format
func displayTasksCompact(tasks []*task.TaskInfo) {
	fmt.Printf("Task List\n")
	fmt.Printf("===============================================================\n")
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tPID\tSTART TIME\tEXECUTABLE")
	fmt.Fprintln(w, "----\t------\t---\t----------\t----------")
	
	for _, t := range tasks {
		statusIndicator := getSimpleStatusIndicator(t.Status)
		pidStr := formatPID(t.PID)
		startTime := formatStartTime(t.StartTime)
		executable := truncateString(t.Executable, 30)
		
		fmt.Fprintf(w, "%s\t[%s] %s\t%s\t%s\t%s\n",
			t.Name, statusIndicator, t.Status, pidStr, startTime, executable)
	}
	
	w.Flush()
}

// displayTasksVerbose shows tasks with detailed information
func displayTasksVerbose(tasks []*task.TaskInfo) {
	fmt.Printf("Task List (Detailed)\n")
	fmt.Printf("===============================================================\n")
	
	for i, t := range tasks {
		if i > 0 {
			fmt.Printf("---------------------------------------------------------------\n")
		}
		
		statusIndicator := getSimpleStatusIndicator(t.Status)
		fmt.Printf("Name:       %s\n", t.Name)
		fmt.Printf("Status:     [%s] %s\n", statusIndicator, t.Status)
		
		if t.PID > 0 {
			fmt.Printf("PID:        %d\n", t.PID)
		}
		
		if t.StartTime != "" && t.StartTime != "0001-01-01 00:00:00" {
			fmt.Printf("Started:    %s\n", t.StartTime)
		}
		
		fmt.Printf("Executable: %s\n", t.Executable)
		
		// Try to get additional IO info if available
		if ioInfo, err := getTaskIOInfo(t.Name); err == nil {
			if ioInfo.StdinPath != "" || ioInfo.StdoutPath != "" || ioInfo.StderrPath != "" {
				fmt.Printf("IO Setup:\n")
				if ioInfo.StdinPath != "" {
					fmt.Printf("  stdin:  %s\n", ioInfo.StdinPath)
				}
				if ioInfo.StdoutPath != "" {
					fmt.Printf("  stdout: %s\n", ioInfo.StdoutPath)
				}
				if ioInfo.StderrPath != "" {
					fmt.Printf("  stderr: %s\n", ioInfo.StderrPath)
				}
			}
		}
	}
}

// displayTasksSummary shows a summary of tasks
func displayTasksSummary(allTasks []*task.TaskInfo, displayedCount int) {
	fmt.Printf("===============================================================\n")
	
	runningCount := 0
	stoppedCount := 0
	
	for _, t := range allTasks {
		if t.Status == "running" {
			runningCount++
		} else {
			stoppedCount++
		}
	}
	
	if displayedCount < len(allTasks) {
		fmt.Printf("Showing %d of %d tasks", displayedCount, len(allTasks))
	} else {
		fmt.Printf("Total: %d tasks", len(allTasks))
	}
	
	if runningCount > 0 || stoppedCount > 0 {
		fmt.Printf(" (%d running, %d stopped)", runningCount, stoppedCount)
	}
	fmt.Printf("\n")
	
	// Show helpful commands
	if len(allTasks) > 0 {
		fmt.Printf("Use 'taskd info <task-name>' for detailed information.\n")
	}
}

// Helper functions

func getSimpleStatusIndicator(status string) string {
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

func getTaskStatusIcon(status string) string {
	// Keep this for backward compatibility, but use simple indicators
	return getSimpleStatusIndicator(status)
}

func formatPID(pid int) string {
	if pid <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", pid)
}

func formatStartTime(startTime string) string {
	if startTime == "" || startTime == "0001-01-01 00:00:00" {
		return "-"
	}
	// Try to parse and format the time
	// For now, just return as-is, but could be improved
	return startTime
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getTaskIOInfo tries to get IO information for a task
func getTaskIOInfo(taskName string) (*task.TaskIOInfo, error) {
	// This would need to be implemented in the task package
	// For now, return empty info
	return &task.TaskIOInfo{}, nil
}