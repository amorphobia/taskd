package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	ConfigFile string
	Verbose    bool
)

// GlobalConfig global configuration
type GlobalConfig struct {
	LogLevel    string `mapstructure:"log_level"`
	LogFile     string `mapstructure:"log_file"`
	PidFile     string `mapstructure:"pid_file"`
	AutoStart   bool   `mapstructure:"auto_start"`
	MaxTasks    int    `mapstructure:"max_tasks"`
}

// InitConfig initialize configuration
func InitConfig() {
	if ConfigFile != "" {
		viper.SetConfigFile(ConfigFile)
	} else {
		// Default config file path
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".taskd")
		
		// Ensure config directory exists
		os.MkdirAll(configDir, 0755)
		
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}
	
	// Set default values
	setDefaults()
	
	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default config
			createDefaultConfig()
		}
	}
}

// GetGlobalConfig get global configuration
func GetGlobalConfig() *GlobalConfig {
	var config GlobalConfig
	viper.Unmarshal(&config)
	return &config
}

func setDefaults() {
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_file", "")
	viper.SetDefault("pid_file", "")
	viper.SetDefault("auto_start", false)
	viper.SetDefault("max_tasks", 100)
}

func createDefaultConfig() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".taskd", "config.toml")
	
	defaultConfig := `# TaskD Global Configuration File

# Log level: debug, info, warn, error
log_level = "info"

# Log file path (empty means output to console)
log_file = ""

# PID file path
pid_file = ""

# Auto start all tasks
auto_start = false

# Maximum number of tasks
max_tasks = 100
`
	
	os.WriteFile(configPath, []byte(defaultConfig), 0644)
}