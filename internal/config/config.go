package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	API      APIConfig      `yaml:"api" mapstructure:"api"`
	UI       UIConfig       `yaml:"ui" mapstructure:"ui"`
	Behavior BehaviorConfig `yaml:"behavior" mapstructure:"behavior"`
	Context  ContextConfig  `yaml:"context" mapstructure:"context"`
}

// APIConfig contains LLM API configuration
type APIConfig struct {
	Provider  string            `yaml:"provider" mapstructure:"provider"`
	Key       string            `yaml:"key" mapstructure:"key"`
	Model     string            `yaml:"model" mapstructure:"model"`
	Endpoint  string            `yaml:"endpoint" mapstructure:"endpoint"`
	Timeout   time.Duration     `yaml:"timeout" mapstructure:"timeout"`
	Providers map[string]Provider `yaml:"providers" mapstructure:"providers"`
}

// Provider represents individual LLM provider configuration
type Provider struct {
	Key      string `yaml:"key" mapstructure:"key"`
	Model    string `yaml:"model" mapstructure:"model"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
}

// UIConfig contains user interface configuration
type UIConfig struct {
	Theme       string `yaml:"theme" mapstructure:"theme"`
	Language    string `yaml:"language" mapstructure:"language"`
	HistorySize int    `yaml:"history_size" mapstructure:"history_size"`
}

// BehaviorConfig contains application behavior settings
type BehaviorConfig struct {
	AutoExecuteSafeCommands   bool `yaml:"auto_execute_safe_commands" mapstructure:"auto_execute_safe_commands"`
	ConfirmDangerousCommands  bool `yaml:"confirm_dangerous_commands" mapstructure:"confirm_dangerous_commands"`
	CollectUsageStats         bool `yaml:"collect_usage_stats" mapstructure:"collect_usage_stats"`
}

// ContextConfig contains context collection settings
type ContextConfig struct {
	IncludeHiddenFiles bool `yaml:"include_hidden_files" mapstructure:"include_hidden_files"`
	MaxFilesInContext  int  `yaml:"max_files_in_context" mapstructure:"max_files_in_context"`
	IncludeEnvVars     bool `yaml:"include_env_vars" mapstructure:"include_env_vars"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		API: APIConfig{
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			Endpoint: "https://api.openai.com/v1",
			Timeout:  10 * time.Second,
		},
		UI: UIConfig{
			Theme:       "dark",
			Language:    "en",
			HistorySize: 100,
		},
		Behavior: BehaviorConfig{
			AutoExecuteSafeCommands:  false,
			ConfirmDangerousCommands: true,
			CollectUsageStats:        false,
		},
		Context: ContextConfig{
			IncludeHiddenFiles: false,
			MaxFilesInContext:  50,
			IncludeEnvVars:     false,
		},
	}
}