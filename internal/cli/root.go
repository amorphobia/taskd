package cli

import (
	"github.com/spf13/cobra"
	"taskd/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "taskd",
	Short: "TaskD - Task daemon management tool",
	Long: `TaskD is a task management tool designed for non-administrator users,
providing unified management and monitoring of user-level background processes.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	
	// Global configuration flags
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "config file path (default: ~/.taskd/config.toml)")
	rootCmd.PersistentFlags().BoolVar(&config.Verbose, "verbose", false, "verbose output")
}

func initConfig() {
	config.InitConfig()
}