package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"taskd/internal/task"
)

var addCmd = &cobra.Command{
	Use:   "add [task-name]",
	Short: "Add a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskName := args[0]
		
		// Get command line arguments
		exec, _ := cmd.Flags().GetString("exec")
		workdir, _ := cmd.Flags().GetString("workdir")
		env, _ := cmd.Flags().GetStringSlice("env")
		inheritEnv, _ := cmd.Flags().GetBool("inherit-env")
		stdin, _ := cmd.Flags().GetString("stdin")
		stdout, _ := cmd.Flags().GetString("stdout")
		stderr, _ := cmd.Flags().GetString("stderr")
		
		// If no working directory specified, use user's home directory
		if workdir == "" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				workdir = homeDir
			}
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
	addCmd.Flags().String("stdout", "", "standard output redirect file")
	addCmd.Flags().String("stderr", "", "standard error redirect file")
	
	addCmd.MarkFlagRequired("exec")
}