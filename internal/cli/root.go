package cli

import (
	"fmt"
	"taskd/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	daemonMode bool
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
	PersistentPreRunE: validateDaemonFlag,
	Run: func(cmd *cobra.Command, args []string) {
		// If daemon mode is enabled, this should not be reached
		// because main() should handle daemon mode before calling Execute()
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

// validateDaemonFlag validates that --daemon flag is not used with other arguments
func validateDaemonFlag(cmd *cobra.Command, args []string) error {
	if daemonMode {
		// Check if there are other arguments or flags (excluding daemon itself)
		if len(args) > 0 {
			return fmt.Errorf("--daemon flag cannot be used with other arguments")
		}
		
		// Check if other flags are set (excluding daemon and help flags)
		flagCount := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Changed && flag.Name != "daemon" && flag.Name != "help" {
				flagCount++
			}
		})
		
		if flagCount > 0 {
			return fmt.Errorf("--daemon flag cannot be used with other flags")
		}
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global configuration flags
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "config file path (default: $TASKD_HOME/config.toml or ~/.taskd/config.toml)")
	rootCmd.PersistentFlags().BoolVar(&config.Verbose, "verbose", false, "verbose output")
	
	// Daemon mode flag (hidden from help)
	rootCmd.PersistentFlags().BoolVar(&daemonMode, "daemon", false, "run in daemon mode (internal use only)")
	rootCmd.PersistentFlags().MarkHidden("daemon")
}

func initConfig() {
	config.InitConfig()
}

// IsDaemonMode returns true if running in daemon mode
func IsDaemonMode() bool {
	return daemonMode
}
