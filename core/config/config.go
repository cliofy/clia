package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	ActiveProvider       string                    `yaml:"active_provider"`
	Providers            map[string]ProviderConfig `yaml:"providers"`
	InteractiveCommands  InteractiveConfig         `yaml:"interactive_commands,omitempty"`
	configPath           string                    // Path to config file
}

// InteractiveConfig represents interactive command configuration
type InteractiveConfig struct {
	Always          []string `yaml:"always,omitempty"`           // Commands that are always interactive
	Never           []string `yaml:"never,omitempty"`            // Commands that are never interactive
	Patterns        []string `yaml:"patterns,omitempty"`         // Patterns to match interactive commands
	CaptureLastFrame bool     `yaml:"capture_last_frame,omitempty"` // Capture last frame of TUI programs
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	Type   string                 `yaml:"type"`   // openai, openrouter, ollama
	Config map[string]interface{} `yaml:"config"` // Provider-specific config
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".clia/config.yaml"
	}
	return filepath.Join(home, ".clia", "config.yaml")
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	// Expand environment variables in path
	path = os.ExpandEnv(path)
	
	// Create config directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		cfg := DefaultConfig()
		cfg.configPath = path
		// Try to save the default config
		if saveErr := cfg.Save(); saveErr != nil {
			// If save fails, just return the default config
			return cfg, nil
		}
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.configPath = path

	// Expand environment variables in config values
	cfg.expandEnvVars()

	return &cfg, nil
}

// Save saves configuration to file
func (c *Config) Save() error {
	if c.configPath == "" {
		c.configPath = DefaultConfigPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with secure permissions (readable/writable by owner only)
	if err := os.WriteFile(c.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveToFile saves the configuration to a specific file
func SaveToFile(c *Config, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with secure permissions
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProvider returns the configuration for a specific provider
func (c *Config) GetProvider(name string) (*ProviderConfig, error) {
	if name == "" {
		name = c.ActiveProvider
	}

	provider, ok := c.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return &provider, nil
}

// SetActiveProvider sets the active provider
func (c *Config) SetActiveProvider(name string) error {
	if _, ok := c.Providers[name]; !ok {
		return fmt.Errorf("provider '%s' not found", name)
	}
	c.ActiveProvider = name
	return nil
}

// AddProvider adds or updates a provider configuration
func (c *Config) AddProvider(name string, config ProviderConfig) {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	c.Providers[name] = config
}

// RemoveProvider removes a provider configuration
func (c *Config) RemoveProvider(name string) error {
	if _, ok := c.Providers[name]; !ok {
		return fmt.Errorf("provider '%s' not found", name)
	}
	
	delete(c.Providers, name)
	
	// If this was the active provider, clear it
	if c.ActiveProvider == name {
		c.ActiveProvider = ""
		// Try to set first available provider as active
		for providerName := range c.Providers {
			c.ActiveProvider = providerName
			break
		}
	}
	
	return nil
}

// ListProviders returns a list of configured provider names
func (c *Config) ListProviders() []string {
	providers := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		providers = append(providers, name)
	}
	return providers
}

// ShouldCaptureLastFrame returns whether to capture the last frame of TUI programs
func (c *Config) ShouldCaptureLastFrame() bool {
	return c.InteractiveCommands.CaptureLastFrame
}

// expandEnvVars expands environment variables in config values
func (c *Config) expandEnvVars() {
	for _, provider := range c.Providers {
		for key, value := range provider.Config {
			if strValue, ok := value.(string); ok {
				// Expand environment variables like ${VAR} or $VAR
				expanded := os.ExpandEnv(strValue)
				provider.Config[key] = expanded
			}
		}
	}
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ActiveProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type: "openai",
				Config: map[string]interface{}{
					"api_key":  "${OPENAI_API_KEY}",
					"base_url": "https://api.openai.com/v1",
					"model":    "gpt-3.5-turbo",
				},
			},
			"openrouter": {
				Type: "openrouter",
				Config: map[string]interface{}{
					"api_key":  "${OPENROUTER_API_KEY}",
					"base_url": "https://openrouter.ai/api/v1",
					"model":    "openai/gpt-3.5-turbo",
				},
			},
			"ollama": {
				Type: "ollama",
				Config: map[string]interface{}{
					"base_url": "http://localhost:11434",
					"model":    "llama2",
				},
			},
		},
		InteractiveCommands: InteractiveConfig{
			Always: []string{
				"zellij", "wezterm", "alacritty", "kitty",
				"k9s", "lazydocker", "btm", "bpytop",
				"ncspot", "cmus", "newsboat",
			},
			Never: []string{
				"ls", "grep", "find", "cat", "echo",
				"curl", "wget", "git", "make",
			},
			Patterns: []string{
				"*-tui", "*-repl", "*-cli", 
				"*-interactive", "*repl*",
			},
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ActiveProvider != "" {
		if _, ok := c.Providers[c.ActiveProvider]; !ok {
			return fmt.Errorf("active provider '%s' not found in providers", c.ActiveProvider)
		}
	}

	for name, provider := range c.Providers {
		if provider.Type == "" {
			return fmt.Errorf("provider '%s' has no type specified", name)
		}
		
		// Validate provider type
		validTypes := []string{"openai", "openrouter", "ollama"}
		valid := false
		for _, t := range validTypes {
			if provider.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("provider '%s' has invalid type '%s'", name, provider.Type)
		}

		// Check required fields based on provider type
		switch provider.Type {
		case "openai", "openrouter":
			if provider.Config["api_key"] == "" || provider.Config["api_key"] == "${OPENAI_API_KEY}" || 
			   provider.Config["api_key"] == "${OPENROUTER_API_KEY}" {
				// Check if env var is actually set
				if provider.Type == "openai" && os.Getenv("OPENAI_API_KEY") == "" {
					fmt.Fprintf(os.Stderr, "Warning: OPENAI_API_KEY environment variable not set for provider '%s'\n", name)
				}
				if provider.Type == "openrouter" && os.Getenv("OPENROUTER_API_KEY") == "" {
					fmt.Fprintf(os.Stderr, "Warning: OPENROUTER_API_KEY environment variable not set for provider '%s'\n", name)
				}
			}
		}
	}

	return nil
}

// GetConfigValue safely retrieves a value from provider config
func GetConfigValue[T any](config map[string]interface{}, key string, defaultValue T) T {
	if value, ok := config[key]; ok {
		if typedValue, ok := value.(T); ok {
			return typedValue
		}
	}
	return defaultValue
}

// GetStringValue retrieves a string value from provider config
func GetStringValue(config map[string]interface{}, key string, defaultValue string) string {
	if value, ok := config[key]; ok {
		switch v := value.(type) {
		case string:
			return v
		case fmt.Stringer:
			return v.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return defaultValue
}

// MergeConfig merges two configurations, with the second taking precedence
func MergeConfig(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy base config
	for k, v := range base {
		result[k] = v
	}
	
	// Override with new values
	for k, v := range override {
		result[k] = v
	}
	
	return result
}

// IsInteractiveCommand checks if a command should be interactive based on config
func (c *Config) IsInteractiveCommand(cmd string) (bool, bool) {
	if c == nil {
		return false, false
	}

	cmdBase := strings.Fields(cmd)[0]
	if cmdBase == "" {
		return false, false
	}

	// Check "never" list first (highest priority)
	for _, never := range c.InteractiveCommands.Never {
		if cmdBase == never {
			return false, true // definitive answer
		}
	}

	// Check "always" list
	for _, always := range c.InteractiveCommands.Always {
		if cmdBase == always {
			return true, true // definitive answer
		}
	}

	// Check patterns
	for _, pattern := range c.InteractiveCommands.Patterns {
		if matchesPattern(cmdBase, pattern) {
			return true, true // definitive answer
		}
	}

	return false, false // no definitive answer
}

// matchesPattern implements simple glob-like pattern matching
func matchesPattern(str, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Handle patterns with * at the beginning or end
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// Pattern like "*tui*"
		middle := pattern[1 : len(pattern)-1]
		return strings.Contains(str, middle)
	} else if strings.HasPrefix(pattern, "*") {
		// Pattern like "*tui"
		suffix := pattern[1:]
		return strings.HasSuffix(str, suffix)
	} else if strings.HasSuffix(pattern, "*") {
		// Pattern like "my*"
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(str, prefix)
	}

	// Exact match
	return str == pattern
}