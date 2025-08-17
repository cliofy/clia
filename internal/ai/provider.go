package ai

import (
	"context"
	"fmt"
	"time"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Complete performs a completion request
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// GetName returns the provider name
	GetName() string

	// GetModel returns the current model being used
	GetModel() string

	// IsConfigured returns true if the provider is properly configured
	IsConfigured() bool
}

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeOllama     ProviderType = "ollama"
	ProviderTypeOpenRouter ProviderType = "openrouter"
)

// ProviderFactory creates LLM providers
type ProviderFactory struct {
	providers map[ProviderType]func(*ProviderConfig) LLMProvider
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	factory := &ProviderFactory{
		providers: make(map[ProviderType]func(*ProviderConfig) LLMProvider),
	}

	// Register built-in providers
	factory.Register(ProviderTypeOpenAI, func(config *ProviderConfig) LLMProvider {
		return NewOpenAIProvider(config)
	})

	factory.Register(ProviderTypeOpenRouter, func(config *ProviderConfig) LLMProvider {
		return NewOpenRouterProvider(config)
	})

	return factory
}

// Register registers a new provider type
func (f *ProviderFactory) Register(providerType ProviderType, constructor func(*ProviderConfig) LLMProvider) {
	f.providers[providerType] = constructor
}

// Create creates a new provider instance
func (f *ProviderFactory) Create(providerType ProviderType, config *ProviderConfig) (LLMProvider, error) {
	constructor, exists := f.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	provider := constructor(config)
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider configuration invalid: %w", err)
	}

	return provider, nil
}

// GetSupportedProviders returns a list of supported provider types
func (f *ProviderFactory) GetSupportedProviders() []ProviderType {
	var types []ProviderType
	for providerType := range f.providers {
		types = append(types, providerType)
	}
	return types
}

// DefaultProviderConfig returns default configuration for a provider type
func DefaultProviderConfig(providerType ProviderType) *ProviderConfig {
	base := &ProviderConfig{
		Name:        string(providerType),
		Timeout:     30 * time.Second, // 30 seconds
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	switch providerType {
	case ProviderTypeOpenAI:
		base.Model = "gpt-3.5-turbo"
		base.Endpoint = "https://api.openai.com/v1"
	case ProviderTypeAnthropic:
		base.Model = "claude-3-sonnet-20240229"
		base.Endpoint = "https://api.anthropic.com"
	case ProviderTypeOllama:
		base.Model = "llama2"
		base.Endpoint = "http://localhost:11434"
	case ProviderTypeOpenRouter:
		base.Model = "openai/gpt-3.5-turbo"
		base.Endpoint = "https://openrouter.ai/api/v1"
	}

	return base
}

// MockProvider is a mock implementation for testing
type MockProvider struct {
	name         string
	model        string
	configured   bool
	mockError    error
	mockResponse *CompletionResponse
}

// NewMockProvider creates a new mock provider
func NewMockProvider(name, model string) *MockProvider {
	return &MockProvider{
		name:       name,
		model:      model,
		configured: true,
	}
}

// Complete implements LLMProvider
func (m *MockProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if m.mockError != nil {
		return nil, m.mockError
	}

	if m.mockResponse != nil {
		return m.mockResponse, nil
	}

	// Default mock response
	return &CompletionResponse{
		Content: "Mock response for: " + req.Prompt,
		Suggestions: []CommandSuggestion{
			{
				Command:     "echo 'mock command'",
				Description: "Mock command suggestion",
				Confidence:  0.9,
				Safe:        true,
				Category:    "test",
			},
		},
		Provider: m.name,
		Model:    m.model,
	}, nil
}

// ValidateConfig implements LLMProvider
func (m *MockProvider) ValidateConfig() error {
	if !m.configured {
		return fmt.Errorf("mock provider not configured")
	}
	return nil
}

// GetName implements LLMProvider
func (m *MockProvider) GetName() string {
	return m.name
}

// GetModel implements LLMProvider
func (m *MockProvider) GetModel() string {
	return m.model
}

// IsConfigured implements LLMProvider
func (m *MockProvider) IsConfigured() bool {
	return m.configured
}

// SetMockError sets an error to be returned by Complete
func (m *MockProvider) SetMockError(err error) {
	m.mockError = err
}

// SetMockResponse sets a response to be returned by Complete
func (m *MockProvider) SetMockResponse(resp *CompletionResponse) {
	m.mockResponse = resp
}
