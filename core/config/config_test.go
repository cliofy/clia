package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Case 1: Configuration loading
func TestConfig_Load(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string // Returns config path
		cleanup     func(t *testing.T, path string)
		validate    func(t *testing.T, cfg *Config)
		expectError bool
	}{
		{
			name: "load default config when file doesn't exist",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "config.yaml")
			},
			cleanup: func(t *testing.T, path string) {
				// Cleanup is handled by t.TempDir()
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg)
				assert.NotEmpty(t, cfg.Providers)
				assert.Equal(t, "openai", cfg.ActiveProvider)
				// Should have default providers
				assert.Contains(t, cfg.Providers, "openai")
				assert.Contains(t, cfg.Providers, "openrouter")
				assert.Contains(t, cfg.Providers, "ollama")
			},
			expectError: false,
		},
		{
			name: "load existing config file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				
				// Create a test config
				testConfig := `active_provider: custom
providers:
  custom:
    type: openai
    config:
      api_key: test-key
      model: gpt-4`
				
				err := os.WriteFile(configPath, []byte(testConfig), 0600)
				require.NoError(t, err)
				
				return configPath
			},
			cleanup: func(t *testing.T, path string) {
				// Cleanup is handled by t.TempDir()
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg)
				assert.Equal(t, "custom", cfg.ActiveProvider)
				assert.Contains(t, cfg.Providers, "custom")
				
				provider := cfg.Providers["custom"]
				assert.Equal(t, "openai", provider.Type)
				assert.Equal(t, "test-key", provider.Config["api_key"])
				assert.Equal(t, "gpt-4", provider.Config["model"])
			},
			expectError: false,
		},
		{
			name: "expand environment variables",
			setup: func(t *testing.T) string {
				// Set test environment variable
				os.Setenv("TEST_API_KEY", "secret-key-123")
				t.Cleanup(func() { os.Unsetenv("TEST_API_KEY") })
				
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				
				testConfig := `active_provider: test
providers:
  test:
    type: openai
    config:
      api_key: ${TEST_API_KEY}
      base_url: https://api.test.com`
				
				err := os.WriteFile(configPath, []byte(testConfig), 0600)
				require.NoError(t, err)
				
				return configPath
			},
			cleanup: func(t *testing.T, path string) {
				os.Unsetenv("TEST_API_KEY")
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg)
				provider := cfg.Providers["test"]
				// Environment variable should be expanded
				assert.Equal(t, "secret-key-123", provider.Config["api_key"])
			},
			expectError: false,
		},
		{
			name: "handle invalid YAML",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				
				// Write invalid YAML
				err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0600)
				require.NoError(t, err)
				
				return configPath
			},
			cleanup: func(t *testing.T, path string) {},
			validate: func(t *testing.T, cfg *Config) {
				// Should not reach here
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup(t)
			defer tt.cleanup(t, configPath)
			
			cfg, err := Load(configPath)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

// Test Case 2: Configuration saving
func TestConfig_Save(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		setup       func(t *testing.T) string
		validate    func(t *testing.T, path string)
		expectError bool
	}{
		{
			name: "save new config file",
			config: &Config{
				ActiveProvider: "test-provider",
				Providers: map[string]ProviderConfig{
					"test-provider": {
						Type: "openai",
						Config: map[string]interface{}{
							"api_key": "test-key",
							"model":   "test-model",
						},
					},
				},
			},
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "config.yaml")
			},
			validate: func(t *testing.T, path string) {
				// Check file exists
				info, err := os.Stat(path)
				require.NoError(t, err)
				
				// Check file permissions (should be 0600)
				mode := info.Mode().Perm()
				assert.Equal(t, os.FileMode(0600), mode, "Config file should have 0600 permissions")
				
				// Load and verify content
				cfg, err := Load(path)
				require.NoError(t, err)
				assert.Equal(t, "test-provider", cfg.ActiveProvider)
			},
			expectError: false,
		},
		{
			name: "overwrite existing config",
			config: &Config{
				ActiveProvider: "new-provider",
				Providers: map[string]ProviderConfig{
					"new-provider": {
						Type: "ollama",
						Config: map[string]interface{}{
							"base_url": "http://localhost:11434",
						},
					},
				},
			},
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				
				// Create existing config
				oldConfig := `active_provider: old
providers:
  old:
    type: openai`
				err := os.WriteFile(configPath, []byte(oldConfig), 0600)
				require.NoError(t, err)
				
				return configPath
			},
			validate: func(t *testing.T, path string) {
				cfg, err := Load(path)
				require.NoError(t, err)
				assert.Equal(t, "new-provider", cfg.ActiveProvider)
				assert.Contains(t, cfg.Providers, "new-provider")
				assert.NotContains(t, cfg.Providers, "old")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup(t)
			tt.config.configPath = configPath
			
			err := tt.config.Save()
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			tt.validate(t, configPath)
		})
	}
}

// Test Case 3: Provider switching and management
func TestConfig_ProviderManagement(t *testing.T) {
	t.Run("switch active provider", func(t *testing.T) {
		cfg := &Config{
			ActiveProvider: "openai",
			Providers: map[string]ProviderConfig{
				"openai": {Type: "openai"},
				"ollama": {Type: "ollama"},
			},
		}
		
		// Switch to existing provider
		err := cfg.SetActiveProvider("ollama")
		assert.NoError(t, err)
		assert.Equal(t, "ollama", cfg.ActiveProvider)
		
		// Try to switch to non-existent provider
		err = cfg.SetActiveProvider("nonexistent")
		assert.Error(t, err)
		assert.Equal(t, "ollama", cfg.ActiveProvider) // Should remain unchanged
	})

	t.Run("add and remove providers", func(t *testing.T) {
		cfg := &Config{
			ActiveProvider: "openai",
			Providers: map[string]ProviderConfig{
				"openai": {Type: "openai"},
			},
		}
		
		// Add new provider
		cfg.AddProvider("custom", ProviderConfig{
			Type: "openrouter",
			Config: map[string]interface{}{
				"api_key": "test",
			},
		})
		
		assert.Contains(t, cfg.Providers, "custom")
		assert.Len(t, cfg.Providers, 2)
		
		// Remove provider
		err := cfg.RemoveProvider("custom")
		assert.NoError(t, err)
		assert.NotContains(t, cfg.Providers, "custom")
		assert.Len(t, cfg.Providers, 1)
		
		// Try to remove non-existent provider
		err = cfg.RemoveProvider("nonexistent")
		assert.Error(t, err)
		
		// Remove active provider
		err = cfg.RemoveProvider("openai")
		assert.NoError(t, err)
		assert.Empty(t, cfg.ActiveProvider)
	})

	t.Run("get provider config", func(t *testing.T) {
		cfg := &Config{
			ActiveProvider: "openai",
			Providers: map[string]ProviderConfig{
				"openai": {
					Type: "openai",
					Config: map[string]interface{}{
						"model": "gpt-4",
					},
				},
			},
		}
		
		// Get active provider
		provider, err := cfg.GetProvider("")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Type)
		
		// Get specific provider
		provider, err = cfg.GetProvider("openai")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		
		// Get non-existent provider
		provider, err = cfg.GetProvider("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("list providers", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]ProviderConfig{
				"openai":     {Type: "openai"},
				"openrouter": {Type: "openrouter"},
				"ollama":     {Type: "ollama"},
			},
		}
		
		providers := cfg.ListProviders()
		assert.Len(t, providers, 3)
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "openrouter")
		assert.Contains(t, providers, "ollama")
	})
}

// Test config validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				ActiveProvider: "openai",
				Providers: map[string]ProviderConfig{
					"openai": {
						Type: "openai",
						Config: map[string]interface{}{
							"api_key": "test-key",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid active provider",
			config: &Config{
				ActiveProvider: "nonexistent",
				Providers: map[string]ProviderConfig{
					"openai": {Type: "openai"},
				},
			},
			expectError: true,
			errorMsg:    "active provider 'nonexistent' not found",
		},
		{
			name: "provider without type",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"invalid": {
						Config: map[string]interface{}{},
					},
				},
			},
			expectError: true,
			errorMsg:    "has no type specified",
		},
		{
			name: "invalid provider type",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"invalid": {
						Type: "unknown",
					},
				},
			},
			expectError: true,
			errorMsg:    "has invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test helper functions
func TestConfig_Helpers(t *testing.T) {
	t.Run("GetStringValue", func(t *testing.T) {
		config := map[string]interface{}{
			"string_key": "value",
			"int_key":    123,
			"bool_key":   true,
		}
		
		assert.Equal(t, "value", GetStringValue(config, "string_key", "default"))
		assert.Equal(t, "123", GetStringValue(config, "int_key", "default"))
		assert.Equal(t, "true", GetStringValue(config, "bool_key", "default"))
		assert.Equal(t, "default", GetStringValue(config, "missing_key", "default"))
	})

	t.Run("MergeConfig", func(t *testing.T) {
		base := map[string]interface{}{
			"key1": "base1",
			"key2": "base2",
		}
		
		override := map[string]interface{}{
			"key2": "override2",
			"key3": "override3",
		}
		
		merged := MergeConfig(base, override)
		
		assert.Equal(t, "base1", merged["key1"])
		assert.Equal(t, "override2", merged["key2"])
		assert.Equal(t, "override3", merged["key3"])
	})
}

// Test default config
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, "openai", cfg.ActiveProvider)
	assert.Len(t, cfg.Providers, 3)
	
	// Check default providers
	assert.Contains(t, cfg.Providers, "openai")
	assert.Contains(t, cfg.Providers, "openrouter")
	assert.Contains(t, cfg.Providers, "ollama")
	
	// Validate default config
	err := cfg.Validate()
	assert.NoError(t, err)
}