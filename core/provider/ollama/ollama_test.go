package ollama

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Case 1: Provider creation
func TestOllamaProvider_Creation(t *testing.T) {
	t.Run("create provider with valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"base_url": "http://localhost:11434",
			"model":    "llama2",
		}

		provider, err := NewProvider("test-ollama", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "test-ollama", provider.Name())
		assert.Equal(t, "llama2", provider.GetModel())
		assert.Equal(t, "http://localhost:11434", provider.GetBaseURL())
	})

	t.Run("create provider with defaults", func(t *testing.T) {
		config := map[string]interface{}{}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Should use defaults
		assert.Equal(t, "llama2", provider.GetModel())
		assert.Equal(t, "http://localhost:11434", provider.GetBaseURL())
	})

	t.Run("create provider with custom base URL", func(t *testing.T) {
		config := map[string]interface{}{
			"base_url": "http://192.168.1.100:11434/",
			"model":    "mistral",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Should trim trailing slash
		assert.Equal(t, "http://192.168.1.100:11434", provider.GetBaseURL())
		assert.Equal(t, "mistral", provider.GetModel())
	})
}

// Test Case 2: Model management
func TestOllamaProvider_Models(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "http://localhost:11434",
	}

	provider, err := NewProvider("test", config)
	require.NoError(t, err)

	tests := []struct {
		name          string
		requestModel  string
		expectedModel string
	}{
		{
			name:          "llama2 model",
			requestModel:  "llama2",
			expectedModel: "llama2",
		},
		{
			name:          "codellama model",
			requestModel:  "codellama",
			expectedModel: "codellama",
		},
		{
			name:          "mistral model",
			requestModel:  "mistral",
			expectedModel: "mistral",
		},
		{
			name:          "vicuna model",
			requestModel:  "vicuna",
			expectedModel: "vicuna",
		},
		{
			name:          "empty model uses default",
			requestModel:  "",
			expectedModel: "llama2",
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
func TestOllamaProvider_MessageConversion(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "http://localhost:11434",
	}

	provider, err := NewProvider("test", config)
	require.NoError(t, err)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "What is Ollama?"},
		{Role: "assistant", Content: "Ollama is a tool for running large language models locally."},
		{Role: "user", Content: "Tell me more"},
	}

	converted := provider.convertMessages(messages)

	assert.Len(t, converted, 4)
	assert.Equal(t, "system", converted[0]["role"])
	assert.Equal(t, "You are a helpful assistant", converted[0]["content"])
	assert.Equal(t, "user", converted[1]["role"])
	assert.Equal(t, "What is Ollama?", converted[1]["content"])
	assert.Equal(t, "assistant", converted[2]["role"])
	assert.Equal(t, "Ollama is a tool for running large language models locally.", converted[2]["content"])
	assert.Equal(t, "user", converted[3]["role"])
	assert.Equal(t, "Tell me more", converted[3]["content"])
}

// Test Case 4: Ollama-specific features
func TestOllamaProvider_Features(t *testing.T) {
	t.Run("no API key required", func(t *testing.T) {
		// Ollama doesn't require an API key
		config := map[string]interface{}{
			"base_url": "http://localhost:11434",
			"model":    "llama2",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Should create successfully without API key
		assert.Equal(t, "test", provider.Name())
	})

	t.Run("local model support", func(t *testing.T) {
		config := map[string]interface{}{
			"base_url": "http://localhost:11434",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)

		// Test various local models
		localModels := []string{
			"llama2",
			"llama2:7b",
			"llama2:13b",
			"codellama",
			"mistral",
			"mixtral",
			"vicuna",
		}

		for _, model := range localModels {
			result := provider.getModel(model)
			assert.Equal(t, model, result)
		}
	})
}