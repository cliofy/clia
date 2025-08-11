package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourusername/clia/pkg/utils"
)

// Manager handles configuration loading, saving, and management
type Manager struct {
	config     *Config
	configPath string
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	
	configPath := filepath.Join(configDir, "config.yaml")
	
	return &Manager{
		configPath: configPath,
		config:     DefaultConfig(),
	}, nil
}

// Load loads configuration from file or creates default if not exists
func (m *Manager) Load() error {
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults
		return nil
	}
	
	// For now, just use defaults
	// In a full implementation, we would use viper or similar to load YAML
	// This is sufficient for Phase 2 demonstration
	
	return nil
}

// Save saves current configuration to file
func (m *Manager) Save() error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// For now, just create an empty config file
	// In a full implementation, we would marshal the config to YAML
	file, err := os.Create(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	// Write a basic config template
	template := `# clia configuration file
api:
  provider: "openai"  # openai, anthropic, ollama
  key: ""  # Set via environment variable OPENAI_API_KEY, etc.
  model: "gpt-3.5-turbo"
  timeout: 10s
  max_tokens: 1000
  temperature: 0.7

ui:
  theme: "dark"  # dark, light
  language: "en"  # en, zh
  history_size: 100

behavior:
  auto_execute_safe_commands: false
  confirm_dangerous_commands: true
  collect_usage_stats: false

context:
  include_hidden_files: false
  max_files_in_context: 50
  include_env_vars: false
`
	
	_, err = file.WriteString(template)
	return err
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// SetAPIKey sets the API key for the current provider
func (m *Manager) SetAPIKey(key string) {
	m.config.API.Key = key
}

// SetProvider sets the active LLM provider
func (m *Manager) SetProvider(provider string) {
	m.config.API.Provider = provider
}

// GetProviderConfig returns configuration for the specified provider
func (m *Manager) GetProviderConfig(provider string) (*Provider, bool) {
	providerConfig, exists := m.config.API.Providers[provider]
	return &providerConfig, exists
}

// GetActiveProviderConfig returns configuration for the currently active provider
func (m *Manager) GetActiveProviderConfig() (*Provider, bool) {
	return m.GetProviderConfig(m.config.API.Provider)
}

// GetAPIKeyFromEnv gets the API key from environment variables
func (m *Manager) GetAPIKeyFromEnv() string {
	provider := m.config.API.Provider
	
	switch provider {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "claude":
		return os.Getenv("CLAUDE_API_KEY")
	default:
		// Try generic format: PROVIDER_API_KEY
		envVar := fmt.Sprintf("%s_API_KEY", provider)
		return os.Getenv(envVar)
	}
}

// IsProviderConfigured checks if the current provider is properly configured
func (m *Manager) IsProviderConfigured() bool {
	// Check if API key is available
	apiKey := m.config.API.Key
	if apiKey == "" {
		apiKey = m.GetAPIKeyFromEnv()
	}
	
	return apiKey != ""
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// ListProviders returns a list of available providers
func (m *Manager) ListProviders() []string {
	var providers []string
	for provider := range m.config.API.Providers {
		providers = append(providers, provider)
	}
	return providers
}

// ValidateConfig validates the current configuration
func (m *Manager) ValidateConfig() error {
	config := m.config
	
	// Validate API config
	if config.API.Provider == "" {
		return fmt.Errorf("API provider is required")
	}
	
	if config.API.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}
	
	if config.API.Temperature < 0 || config.API.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	
	// Validate UI config
	if config.UI.HistorySize < 0 {
		return fmt.Errorf("history_size cannot be negative")
	}
	
	// Validate Context config
	if config.Context.MaxFilesInContext < 0 {
		return fmt.Errorf("max_files_in_context cannot be negative")
	}
	
	return nil
}

// GetConfigSummary returns a summary of the current configuration
func (m *Manager) GetConfigSummary() map[string]interface{} {
	config := m.config
	
	summary := map[string]interface{}{
		"api": map[string]interface{}{
			"provider":    config.API.Provider,
			"model":       config.API.Model,
			"max_tokens":  config.API.MaxTokens,
			"temperature": config.API.Temperature,
			"configured":  m.IsProviderConfigured(),
		},
		"ui": map[string]interface{}{
			"theme":        config.UI.Theme,
			"language":     config.UI.Language,
			"history_size": config.UI.HistorySize,
		},
		"behavior": map[string]interface{}{
			"auto_execute_safe":        config.Behavior.AutoExecuteSafeCommands,
			"confirm_dangerous":        config.Behavior.ConfirmDangerousCommands,
			"collect_stats":           config.Behavior.CollectUsageStats,
		},
		"context": map[string]interface{}{
			"include_hidden":    config.Context.IncludeHiddenFiles,
			"max_files":        config.Context.MaxFilesInContext,
			"include_env_vars": config.Context.IncludeEnvVars,
		},
	}
	
	return summary
}