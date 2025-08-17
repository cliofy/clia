package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clia/core/config"
)

// Test Case 1: Provider factory functionality
func TestFactory_CreateProvider(t *testing.T) {
	factory := NewFactory()
	
	tests := []struct {
		name         string
		providerName string
		config       *config.ProviderConfig
		expectError  bool
		validate     func(t *testing.T, provider Provider)
	}{
		{
			name:         "create OpenAI provider",
			providerName: "test-openai",
			config: &config.ProviderConfig{
				Type: "openai",
				Config: map[string]interface{}{
					"api_key":  "sk-test1234567890abcdef1234567890abcdef1234567890abcdef",
					"base_url": "https://api.openai.com/v1",
					"model":    "gpt-4",
				},
			},
			expectError: false,
			validate: func(t *testing.T, provider Provider) {
				assert.Equal(t, "openai", provider.Name())
				assert.True(t, provider.Available())
				// Note: We don't test actual API calls in unit tests
				// Real API functionality would be tested in integration tests
			},
		},
		{
			name:         "create OpenRouter provider",
			providerName: "test-openrouter",
			config: &config.ProviderConfig{
				Type: "openrouter",
				Config: map[string]interface{}{
					"api_key":  "test-key",
					"base_url": "https://openrouter.ai/api/v1",
					"model":    "anthropic/claude-3-opus",
				},
			},
			expectError: false,
			validate: func(t *testing.T, provider Provider) {
				assert.Equal(t, "openrouter", provider.Name())
				assert.True(t, provider.Available())
			},
		},
		{
			name:         "create Ollama provider",
			providerName: "test-ollama",
			config: &config.ProviderConfig{
				Type: "ollama",
				Config: map[string]interface{}{
					"base_url": "http://localhost:11434",
					"model":    "llama2",
				},
			},
			expectError: false,
			validate: func(t *testing.T, provider Provider) {
				assert.Equal(t, "ollama", provider.Name())
				// Ollama's Available() checks if service is running
				// In test environment, it may not be running
				// So we just check that the provider was created successfully
				assert.NotNil(t, provider)
			},
		},
		{
			name:         "missing API key for OpenAI",
			providerName: "test-openai-no-key",
			config: &config.ProviderConfig{
				Type: "openai",
				Config: map[string]interface{}{
					"base_url": "https://api.openai.com/v1",
					"model":    "gpt-4",
				},
			},
			expectError: true,
		},
		{
			name:         "unsupported provider type",
			providerName: "test-unknown",
			config: &config.ProviderConfig{
				Type: "unknown",
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
			expectError: true,
		},
		{
			name:         "nil config",
			providerName: "test-nil",
			config:       nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.providerName, tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, provider)
			
			if tt.validate != nil {
				tt.validate(t, provider)
			}
		})
	}
}

// Test Case 2: Provider manager functionality
func TestManager_ProviderManagement(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {
				Type: "openai",
				Config: map[string]interface{}{
					"api_key": "test-openai-key",
					"model":   "gpt-3.5-turbo",
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
	}

	manager := NewManager(cfg)

	t.Run("get active provider", func(t *testing.T) {
		provider, err := manager.GetActiveProvider()
		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
	})

	t.Run("get provider by name", func(t *testing.T) {
		provider, err := manager.GetProvider("ollama")
		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, "ollama", provider.Name())
	})

	t.Run("get non-existent provider", func(t *testing.T) {
		provider, err := manager.GetProvider("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("set active provider", func(t *testing.T) {
		err := manager.SetActiveProvider("ollama")
		require.NoError(t, err)
		
		provider, err := manager.GetActiveProvider()
		require.NoError(t, err)
		assert.Equal(t, "ollama", provider.Name())
	})

	t.Run("set non-existent active provider", func(t *testing.T) {
		err := manager.SetActiveProvider("nonexistent")
		assert.Error(t, err)
	})

	t.Run("list providers", func(t *testing.T) {
		providers := manager.ListProviders()
		assert.Len(t, providers, 2)
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "ollama")
	})

	t.Run("provider caching", func(t *testing.T) {
		// Get provider twice, should return same instance
		provider1, err := manager.GetProvider("openai")
		require.NoError(t, err)
		
		provider2, err := manager.GetProvider("openai")
		require.NoError(t, err)
		
		// Should be the same instance (cached) - compare pointers
		assert.Same(t, provider1, provider2)
	})

	t.Run("refresh provider", func(t *testing.T) {
		// Get provider
		provider1, err := manager.GetProvider("openai")
		require.NoError(t, err)
		
		// Refresh it
		manager.RefreshProvider("openai")
		
		// Get again, should be different instance
		provider2, err := manager.GetProvider("openai")
		require.NoError(t, err)
		
		// Should be different instances (compare pointers)
		assert.NotSame(t, provider1, provider2)
	})

	t.Run("refresh all providers", func(t *testing.T) {
		// Get providers
		_, err := manager.GetProvider("openai")
		require.NoError(t, err)
		_, err = manager.GetProvider("ollama")
		require.NoError(t, err)
		
		// Refresh all
		manager.RefreshAll()
		
		// Internal cache should be empty
		assert.Empty(t, manager.providers)
	})

	t.Run("update config", func(t *testing.T) {
		newCfg := &config.Config{
			ActiveProvider: "ollama",
			Providers: map[string]config.ProviderConfig{
				"ollama": {
					Type: "ollama",
					Config: map[string]interface{}{
						"base_url": "http://localhost:11434",
						"model":    "llama3",
					},
				},
			},
		}
		
		manager.UpdateConfig(newCfg)
		
		// Should use new config
		providers := manager.ListProviders()
		assert.Len(t, providers, 1)
		assert.Contains(t, providers, "ollama")
		
		// Cache should be cleared
		assert.Empty(t, manager.providers)
	})
}

// Test Case 3: Integration test with placeholder providers
func TestManager_Integration(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)

	t.Run("end-to-end chat", func(t *testing.T) {
		// Skip this test when using real OpenAI provider without API key
		// This would be tested in integration tests with mock servers
		t.Skip("Skipping end-to-end test with real API - would need integration test setup")
	})

	t.Run("end-to-end streaming", func(t *testing.T) {
		// Skip this test when Ollama is not running
		// This would be tested in integration tests with mock servers
		t.Skip("Skipping end-to-end streaming test - requires Ollama service running")
	})

	t.Run("switch providers during runtime", func(t *testing.T) {
		// Start with default (OpenAI)
		provider1, err := manager.GetActiveProvider()
		require.NoError(t, err)
		assert.Equal(t, "openai", provider1.Name())
		
		// Switch to Ollama
		err = manager.SetActiveProvider("ollama")
		require.NoError(t, err)
		
		provider2, err := manager.GetActiveProvider()
		require.NoError(t, err)
		assert.Equal(t, "ollama", provider2.Name())
		
		// Switch back to OpenAI
		err = manager.SetActiveProvider("openai")
		require.NoError(t, err)
		
		provider3, err := manager.GetActiveProvider()
		require.NoError(t, err)
		assert.Equal(t, "openai", provider3.Name())
	})
}