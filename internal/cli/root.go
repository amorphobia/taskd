package cli

import (
	"taskd/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "taskd",
	Short: "TaskD - Task daemon management tool",
	Long: `TaskD is a task management tool designed for non-administrator users,
providing unified management and monitoring of user-level background processes.

Features:
- Output redirection with support for stdin, stdout, and stderr
- Relative paths are resolved based on the task's working directory
- Automatic file creation in append mode for output files
- Unified task information display with the 'info' command`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global configuration flags
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "config file path (default: $TASKD_HOME/config.toml or ~/.taskd/config.toml)")
	rootCmd.PersistentFlags().BoolVar(&config.Verbose, "verbose", false, "verbose output")
}

func initConfig() {
	config.InitConfig()
}
