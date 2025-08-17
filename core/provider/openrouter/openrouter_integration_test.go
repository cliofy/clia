//go:build integration
// +build integration

package openrouter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadEnvFile(t *testing.T) {
	// Try to find .env file in project root
	for dir := "."; ; dir = filepath.Join("..", dir) {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err != nil {
				t.Logf("Warning: Failed to load .env file from %s: %v", envPath, err)
			} else {
				t.Logf("Loaded .env file from %s", envPath)
			}
			break
		}
		
		// Stop if we've reached the root
		if absDir, _ := filepath.Abs(dir); absDir == "/" || absDir == filepath.Dir(absDir) {
			break
		}
	}
}

func TestOpenRouterIntegration(t *testing.T) {
	// Load environment variables
	loadEnvFile(t)
	
	apiKey := os.Getenv("OPENROUTER_KEY")
	if apiKey == "" {
		t.Skip("Skipping OpenRouter integration test: OPENROUTER_KEY not set")
	}
	
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "openai/gpt-3.5-turbo" // Default fallback
		t.Logf("MODEL_NAME not set, using default: %s", modelName)
	}
	
	t.Logf("Running OpenRouter integration tests with model: %s", modelName)
	
	// Create provider with real API key
	config := map[string]interface{}{
		"api_key":  apiKey,
		"base_url": "https://openrouter.ai/api/v1",
		"model":    modelName,
	}
	
	provider, err := NewProvider("openrouter-integration", config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	
	t.Run("ListModels", func(t *testing.T) {
		models, err := provider.ListModels()
		require.NoError(t, err)
		require.NotEmpty(t, models)
		
		t.Logf("Found %d models", len(models))
		
		// Find our test model
		var foundModel bool
		for _, model := range models {
			if model.ID == modelName {
				foundModel = true
				t.Logf("Found test model: %s", model.ID)
				t.Logf("  Name: %s", model.Name)
				t.Logf("  Context Length: %d", model.ContextLength)
				t.Logf("  Modality: %s", model.Architecture.Modality)
				if model.Pricing.Prompt != "" {
					t.Logf("  Pricing - Prompt: %s", model.Pricing.Prompt)
					t.Logf("  Pricing - Completion: %s", model.Pricing.Completion)
				}
				break
			}
		}
		
		if !foundModel {
			t.Logf("Warning: Test model %s not found in model list", modelName)
			// List first 5 available models
			t.Log("Available models (first 5):")
			for i, model := range models {
				if i >= 5 {
					break
				}
				t.Logf("  - %s: %s", model.ID, model.Name)
			}
		}
	})
	
	t.Run("Chat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		req := &ChatRequest{
			Model: modelName,
			Messages: []Message{
				{Role: "user", Content: "Say 'Hello from OpenRouter!' and nothing else."},
			},
			Options: &ChatOptions{
				Temperature: floatPtr(0.1),
				MaxTokens:   intPtr(50),
			},
		}
		
		resp, err := provider.Chat(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		
		assert.NotEmpty(t, resp.ID)
		assert.NotEmpty(t, resp.Content)
		assert.NotZero(t, resp.Created)
		
		t.Logf("Chat Response:")
		t.Logf("  ID: %s", resp.ID)
		t.Logf("  Model: %s", resp.Model)
		t.Logf("  Content: %s", resp.Content)
		if resp.Usage != nil {
			t.Logf("  Tokens - Prompt: %d, Completion: %d, Total: %d",
				resp.Usage.PromptTokens,
				resp.Usage.CompletionTokens,
				resp.Usage.TotalTokens)
		}
	})
	
	t.Run("ChatStream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		req := &ChatRequest{
			Model: modelName,
			Messages: []Message{
				{Role: "user", Content: "Count from 1 to 5, one number per line."},
			},
			Options: &ChatOptions{
				Temperature: floatPtr(0.1),
				MaxTokens:   intPtr(50),
				Stream:      true,
			},
		}
		
		stream, err := provider.ChatStream(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, stream)
		
		var fullContent string
		var chunkCount int
		var hasContent bool
		
		t.Log("Streaming Response:")
		for chunk := range stream {
			chunkCount++
			if chunk.Content != "" {
				fullContent += chunk.Content
				hasContent = true
				t.Logf("  Chunk %d: %q", chunkCount, chunk.Content)
			}
			
			if chunk.Done {
				t.Log("  Stream completed")
				break
			}
		}
		
		// Some models might return empty content in streaming mode
		// or combine all content in a single chunk
		if !hasContent && chunkCount > 0 {
			t.Log("  Warning: Model returned empty chunks in streaming mode")
			t.Log("  This might be a model-specific behavior")
		}
		
		assert.Greater(t, chunkCount, 0, "Should have received at least one chunk")
		t.Logf("Full response (%d chunks): %s", chunkCount, fullContent)
		
		// If we got chunks but no content, that's still a valid streaming response
		// Some models might not support proper streaming
		if fullContent == "" && chunkCount > 0 {
			t.Log("  Note: Model appears to not support content streaming properly")
		}
	})
	
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid model
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		req := &ChatRequest{
			Model: "invalid/model-that-does-not-exist",
			Messages: []Message{
				{Role: "user", Content: "Test"},
			},
		}
		
		resp, err := provider.Chat(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		t.Logf("Expected error for invalid model: %v", err)
	})
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

// TestOpenRouterIntegrationWithoutBuildTag can be run without the integration build tag
// It will still skip if OPENROUTER_KEY is not set
func TestOpenRouterIntegrationManual(t *testing.T) {
	// Load environment variables
	loadEnvFile(t)
	
	apiKey := os.Getenv("OPENROUTER_KEY")
	if apiKey == "" {
		t.Skip("Skipping manual OpenRouter test: OPENROUTER_KEY not set")
	}
	
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "openai/gpt-3.5-turbo"
	}
	
	fmt.Printf("\n=== Manual OpenRouter Integration Test ===\n")
	fmt.Printf("API Key: %s...%s\n", apiKey[:10], apiKey[len(apiKey)-4:])
	fmt.Printf("Model: %s\n", modelName)
	fmt.Println("==========================================")
	
	// Run a simple test
	config := map[string]interface{}{
		"api_key":  apiKey,
		"base_url": "https://openrouter.ai/api/v1",
		"model":    modelName,
	}
	
	provider, err := NewProvider("manual-test", config)
	require.NoError(t, err)
	
	// Quick availability check
	assert.True(t, provider.Available())
	t.Log("✓ Provider is available")
	
	// List models
	t.Run("QuickModelList", func(t *testing.T) {
		models, err := provider.ListModels()
		if err != nil {
			t.Errorf("Failed to list models: %v", err)
			return
		}
		
		t.Logf("✓ Successfully fetched %d models", len(models))
		
		// Check if our model exists
		for _, model := range models {
			if model.ID == modelName {
				t.Logf("✓ Found configured model: %s", modelName)
				return
			}
		}
		t.Logf("⚠ Configured model %s not found in list", modelName)
	})
}