package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Creation(t *testing.T) {
	t.Run("create provider with valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":  "test-key",
			"base_url": "https://api.openai.com/v1",
			"model":    "gpt-4",
		}

		provider, err := NewProvider("test", config)
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "test", provider.Name())
		assert.True(t, provider.Available())
		assert.Equal(t, "gpt-4", provider.GetModel())
		assert.Equal(t, "https://api.openai.com/v1", provider.GetBaseURL())
	})

	t.Run("create provider with missing API key", func(t *testing.T) {
		config := map[string]interface{}{
			"model": "gpt-4",
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
		assert.Equal(t, "gpt-3.5-turbo", provider.GetModel())
		assert.Contains(t, provider.GetBaseURL(), "api.openai.com")
	})
}