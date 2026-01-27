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
		
		tasks, err := task.ListTasks()
		if err != nil {
			return fmt.Errorf("failed to get task list: %w", err)
		}
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tPID\tSTART_TIME\tEXECUTABLE")
		
		for _, t := range tasks {
			// Filter based on conditions
			if running && t.Status != "running" {
				continue
			}
			if stopped && t.Status == "running" {
				continue
			}
			
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				t.Name, t.Status, t.PID, t.StartTime, t.Executable)
		}
		
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	
	listCmd.Flags().BoolP("running", "r", false, "show only running tasks")
	listCmd.Flags().BoolP("stopped", "s", false, "show only stopped tasks")
}