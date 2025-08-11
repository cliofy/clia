package ai

import (
	"context"
	"testing"
	"time"
)

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory()
	
	// Test getting supported providers
	providers := factory.GetSupportedProviders()
	if len(providers) == 0 {
		t.Error("Expected at least one supported provider")
	}
	
	// Test creating OpenAI provider
	config := DefaultProviderConfig(ProviderTypeOpenAI)
	config.APIKey = "test-key"
	
	provider, err := factory.Create(ProviderTypeOpenAI, config)
	if err != nil {
		t.Errorf("Failed to create OpenAI provider: %v", err)
	}
	
	if provider.GetName() != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", provider.GetName())
	}
}

func TestDefaultProviderConfig(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expectedModel string
	}{
		{ProviderTypeOpenAI, "gpt-3.5-turbo"},
		{ProviderTypeAnthropic, "claude-3-sonnet-20240229"},
		{ProviderTypeOllama, "llama2"},
		{ProviderTypeOpenRouter, "openai/gpt-3.5-turbo"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			config := DefaultProviderConfig(tt.providerType)
			
			if config.Name != string(tt.providerType) {
				t.Errorf("Expected name '%s', got '%s'", tt.providerType, config.Name)
			}
			
			if config.Model != tt.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", tt.expectedModel, config.Model)
			}
			
			if config.MaxTokens <= 0 {
				t.Error("Expected positive max tokens")
			}
			
			if config.Temperature < 0 || config.Temperature > 1 {
				t.Errorf("Expected temperature between 0-1, got %f", config.Temperature)
			}
		})
	}
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider("mock", "test-model")
	
	// Test basic properties
	if provider.GetName() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", provider.GetName())
	}
	
	if provider.GetModel() != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", provider.GetModel())
	}
	
	if !provider.IsConfigured() {
		t.Error("Expected mock provider to be configured")
	}
	
	// Test validation
	if err := provider.ValidateConfig(); err != nil {
		t.Errorf("Mock provider validation failed: %v", err)
	}
	
	// Test completion
	ctx := context.Background()
	req := &CompletionRequest{
		Prompt: "test prompt",
	}
	
	resp, err := provider.Complete(ctx, req)
	if err != nil {
		t.Errorf("Mock provider completion failed: %v", err)
	}
	
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
	
	if len(resp.Suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}
	
	// Test mock error
	testErr := NewAIError(ErrorTypeAuth, "test error", nil)
	provider.SetMockError(testErr)
	
	_, err = provider.Complete(ctx, req)
	if err == nil {
		t.Error("Expected error from mock provider")
	}
	
	// Test custom mock response
	provider.SetMockError(nil)
	customResp := &CompletionResponse{
		Content: "custom response",
		Suggestions: []CommandSuggestion{
			{Command: "custom command", Description: "custom desc", Safe: true, Confidence: 0.95},
		},
	}
	provider.SetMockResponse(customResp)
	
	resp, err = provider.Complete(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if resp.Content != "custom response" {
		t.Errorf("Expected custom response content, got '%s'", resp.Content)
	}
}

func TestCommandSuggestions(t *testing.T) {
	suggestions := CommandSuggestions{
		{Command: "ls", Confidence: 0.8, Safe: true},
		{Command: "rm -rf /", Confidence: 0.9, Safe: false},
		{Command: "pwd", Confidence: 0.7, Safe: true},
	}
	
	// Test FilterSafe
	safe := suggestions.FilterSafe()
	if len(safe) != 2 {
		t.Errorf("Expected 2 safe commands, got %d", len(safe))
	}
	
	// Test SortByConfidence
	sorted := suggestions.SortByConfidence()
	if sorted[0].Confidence != 0.9 {
		t.Errorf("Expected highest confidence first, got %f", sorted[0].Confidence)
	}
	
	// Test Top
	top := suggestions.Top(2)
	if len(top) != 2 {
		t.Errorf("Expected top 2 suggestions, got %d", len(top))
	}
}

func TestAIService(t *testing.T) {
	service := NewService()
	
	// Test without provider
	ctx := context.Background()
	_, err := service.SuggestCommands(ctx, "test input")
	if err == nil {
		t.Error("Expected error when no provider is configured")
	}
	
	// Test with mock provider
	mockProvider := NewMockProvider("test", "test-model")
	service.SetProvider(mockProvider)
	
	// Test connection
	err = service.TestConnection(ctx)
	if err != nil {
		t.Errorf("Test connection failed: %v", err)
	}
	
	// Test suggestions
	resp, err := service.SuggestCommands(ctx, "list files")
	if err != nil {
		t.Errorf("SuggestCommands failed: %v", err)
	}
	
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
	
	if len(resp.Suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}
	
	// Test provider info
	info := service.GetProviderInfo()
	if info["name"] != "test" {
		t.Errorf("Expected provider name 'test', got '%v'", info["name"])
	}
	
	if !info["configured"].(bool) {
		t.Error("Expected provider to be configured")
	}
}

func TestAIServiceWithTimeout(t *testing.T) {
	service := NewService().SetTimeout(1 * time.Millisecond)
	mockProvider := NewMockProvider("test", "test-model")
	
	// Simulate slow response
	mockProvider.SetMockError(context.DeadlineExceeded)
	service.SetProvider(mockProvider)
	
	ctx := context.Background()
	_, err := service.SuggestCommands(ctx, "test")
	
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestAIServiceFallbackMode(t *testing.T) {
	service := NewService().SetFallbackMode(true)
	mockProvider := NewMockProvider("test", "test-model")
	
	// Simulate provider error
	mockProvider.SetMockError(NewAIError(ErrorTypeNetwork, "network error", nil))
	service.SetProvider(mockProvider)
	
	ctx := context.Background()
	resp, err := service.SuggestCommands(ctx, "list files")
	
	if err != nil {
		t.Errorf("Expected fallback to work, got error: %v", err)
	}
	
	if resp == nil {
		t.Fatal("Expected fallback response")
	}
	
	if resp.Provider != "fallback" {
		t.Errorf("Expected fallback provider, got '%s'", resp.Provider)
	}
}

func TestAIError(t *testing.T) {
	originalErr := context.DeadlineExceeded
	aiErr := NewAIError(ErrorTypeNetwork, "timeout occurred", originalErr)
	
	if aiErr.Type != ErrorTypeNetwork {
		t.Errorf("Expected error type %s, got %s", ErrorTypeNetwork, aiErr.Type)
	}
	
	if aiErr.Message != "timeout occurred" {
		t.Errorf("Expected message 'timeout occurred', got '%s'", aiErr.Message)
	}
	
	if aiErr.Unwrap() != originalErr {
		t.Error("Expected to unwrap to original error")
	}
	
	errorString := aiErr.Error()
	if errorString != "timeout occurred: context deadline exceeded" {
		t.Errorf("Unexpected error string: %s", errorString)
	}
}

func TestOpenRouterProvider(t *testing.T) {
	config := &ProviderConfig{
		Name:        "openrouter",
		APIKey:      "test-openrouter-key",
		Model:       "openai/gpt-3.5-turbo",
		Endpoint:    "https://openrouter.ai/api/v1",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30000,
	}
	
	provider := NewOpenRouterProvider(config)
	
	// Test basic properties
	if provider.GetName() != "openrouter" {
		t.Errorf("Expected name 'openrouter', got '%s'", provider.GetName())
	}
	
	if provider.GetModel() != "openai/gpt-3.5-turbo" {
		t.Errorf("Expected model 'openai/gpt-3.5-turbo', got '%s'", provider.GetModel())
	}
	
	if !provider.IsConfigured() {
		t.Error("Expected provider to be configured")
	}
	
	// Test validation
	if err := provider.ValidateConfig(); err != nil {
		t.Errorf("Provider validation failed: %v", err)
	}
}

func TestOpenRouterProviderValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProviderConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &ProviderConfig{
				APIKey:      "test-key",
				Model:       "openai/gpt-3.5-turbo",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: &ProviderConfig{
				Model:       "openai/gpt-3.5-turbo",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			expectError: true,
		},
		{
			name: "missing model",
			config: &ProviderConfig{
				APIKey:      "test-key",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			expectError: true,
		},
		{
			name: "invalid max tokens",
			config: &ProviderConfig{
				APIKey:      "test-key",
				Model:       "openai/gpt-3.5-turbo",
				MaxTokens:   -1,
				Temperature: 0.7,
			},
			expectError: true,
		},
		{
			name: "invalid temperature",
			config: &ProviderConfig{
				APIKey:      "test-key",
				Model:       "openai/gpt-3.5-turbo",
				MaxTokens:   1000,
				Temperature: 3.0,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenRouterProvider(tt.config)
			err := provider.ValidateConfig()
			
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestProviderFactoryOpenRouter(t *testing.T) {
	factory := NewProviderFactory()
	
	// Test OpenRouter provider creation
	config := &ProviderConfig{
		APIKey:      "test-openrouter-key",
		Model:       "openai/gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	
	provider, err := factory.Create(ProviderTypeOpenRouter, config)
	if err != nil {
		t.Errorf("Failed to create OpenRouter provider: %v", err)
	}
	
	if provider.GetName() != "openrouter" {
		t.Errorf("Expected provider name 'openrouter', got '%s'", provider.GetName())
	}
	
	// Test that OpenRouter is in supported providers
	supportedProviders := factory.GetSupportedProviders()
	found := false
	for _, p := range supportedProviders {
		if p == ProviderTypeOpenRouter {
			found = true
			break
		}
	}
	if !found {
		t.Error("OpenRouter provider not found in supported providers")
	}
}

func TestOpenRouterProviderWithoutAPIKey(t *testing.T) {
	config := &ProviderConfig{
		Model:       "openai/gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	
	provider := NewOpenRouterProvider(config)
	
	// Provider should not be configured without API key
	if provider.IsConfigured() {
		t.Error("Expected provider to not be configured without API key")
	}
	
	// Should return error when trying to complete
	ctx := context.Background()
	req := &CompletionRequest{Prompt: "test"}
	
	_, err := provider.Complete(ctx, req)
	if err == nil {
		t.Error("Expected error when completing without API key")
	}
	
	aiErr, ok := err.(*AIError)
	if !ok {
		t.Error("Expected AIError type")
	}
	
	if aiErr.Type != ErrorTypeAuth {
		t.Errorf("Expected auth error type, got %s", aiErr.Type)
	}
}