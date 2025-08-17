package openrouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Case 1: Provider creation
func TestOpenRouterProvider_Creation(t *testing.T) {
	t.Run("create provider with valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":  "test-key",
			"base_url": "https://openrouter.ai/api/v1",
			"model":    "anthropic/claude-3-opus",
			"referer":  "https://example.com",
			"app_name": "TestApp",
		}

		provider, err := NewProvider("test-openrouter", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "test-openrouter", provider.Name())
		assert.True(t, provider.Available())
		assert.Equal(t, "anthropic/claude-3-opus", provider.GetModel())
		assert.Equal(t, "https://openrouter.ai/api/v1", provider.GetBaseURL())
		assert.Equal(t, "https://example.com", provider.referer)
		assert.Equal(t, "TestApp", provider.appName)
	})

	t.Run("create provider with missing API key", func(t *testing.T) {
		config := map[string]interface{}{
			"model": "openai/gpt-4",
		}

		provider, err := NewProvider("test", config)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyMissing)
		assert.Nil(t, provider)
	})

	t.Run("create provider with defaults", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test-key",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Should use defaults
		assert.Equal(t, "openai/gpt-3.5-turbo", provider.GetModel())
		assert.Equal(t, "https://openrouter.ai/api/v1", provider.GetBaseURL())
		assert.Equal(t, "https://github.com/yourusername/clia", provider.referer)
		assert.Equal(t, "CLIA", provider.appName)
	})

	t.Run("create provider with custom base URL", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":  "test-key",
			"base_url": "https://custom.openrouter.ai/v1/",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Should trim trailing slash
		assert.Equal(t, "https://custom.openrouter.ai/v1", provider.GetBaseURL())
	})
}

// Test Case 2: Model routing support
func TestOpenRouterProvider_ModelRouting(t *testing.T) {
	config := map[string]interface{}{
		"api_key": "test-key",
	}

	provider, err := NewProvider("test", config)
	require.NoError(t, err)

	tests := []struct {
		name          string
		requestModel  string
		expectedModel string
	}{
		{
			name:          "OpenAI model",
			requestModel:  "openai/gpt-4",
			expectedModel: "openai/gpt-4",
		},
		{
			name:          "Anthropic model",
			requestModel:  "anthropic/claude-3-opus",
			expectedModel: "anthropic/claude-3-opus",
		},
		{
			name:          "Google model",
			requestModel:  "google/gemini-pro",
			expectedModel: "google/gemini-pro",
		},
		{
			name:          "Meta model",
			requestModel:  "meta-llama/llama-2-70b-chat",
			expectedModel: "meta-llama/llama-2-70b-chat",
		},
		{
			name:          "empty model uses default",
			requestModel:  "",
			expectedModel: "openai/gpt-3.5-turbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := provider.getModel(tt.requestModel)
			assert.Equal(t, tt.expectedModel, model)
		})
	}
}

// Test Case 3: Message conversion
func TestOpenRouterProvider_MessageConversion(t *testing.T) {
	config := map[string]interface{}{
		"api_key": "test-key",
	}

	provider, err := NewProvider("test", config)
	require.NoError(t, err)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	converted := provider.convertMessages(messages)

	assert.Len(t, converted, 4)
	assert.Equal(t, "system", converted[0]["role"])
	assert.Equal(t, "You are a helpful assistant", converted[0]["content"])
	assert.Equal(t, "user", converted[1]["role"])
	assert.Equal(t, "Hello", converted[1]["content"])
	assert.Equal(t, "assistant", converted[2]["role"])
	assert.Equal(t, "Hi there!", converted[2]["content"])
	assert.Equal(t, "user", converted[3]["role"])
	assert.Equal(t, "How are you?", converted[3]["content"])
}