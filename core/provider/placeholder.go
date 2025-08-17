package provider

import (
	"context"
	"fmt"
	"time"
)

// PlaceholderProvider is a placeholder implementation used during development
type PlaceholderProvider struct {
	name         string
	providerType ProviderType
	config       map[string]interface{}
	baseURL      string
	model        string
	available    bool
}

// NewPlaceholderProvider creates a new placeholder provider
func NewPlaceholderProvider(name string, providerType ProviderType) *PlaceholderProvider {
	return &PlaceholderProvider{
		name:         name,
		providerType: providerType,
		available:    true,
	}
}

// Chat implements the Provider interface
func (p *PlaceholderProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if !p.available {
		return nil, ErrProviderNotAvailable
	}

	// Simulate processing time
	time.Sleep(10 * time.Millisecond)

	// Generate a mock response
	return &ChatResponse{
		ID:      fmt.Sprintf("%s-mock-id", p.name),
		Model:   req.Model,
		Content: fmt.Sprintf("Mock response from %s provider", p.name),
		Created: time.Now(),
		Usage: &Usage{
			PromptTokens:     estimateTokens(req.Messages),
			CompletionTokens: 20,
			TotalTokens:      estimateTokens(req.Messages) + 20,
		},
	}, nil
}

// ChatStream implements the Provider interface
func (p *PlaceholderProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	if !p.available {
		return nil, ErrProviderNotAvailable
	}

	ch := make(chan *ChatChunk)
	
	go func() {
		defer close(ch)
		
		words := []string{"Streaming", "response", "from", p.name, "provider"}
		
		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case ch <- &ChatChunk{
				ID:      fmt.Sprintf("%s-stream", p.name),
				Model:   req.Model,
				Content: word + " ",
				Done:    i == len(words)-1,
			}:
				time.Sleep(50 * time.Millisecond) // Simulate streaming delay
			}
		}
	}()
	
	return ch, nil
}

// Name returns the provider name
func (p *PlaceholderProvider) Name() string {
	return p.name
}

// Available returns whether the provider is available
func (p *PlaceholderProvider) Available() bool {
	return p.available
}

// SetAvailable sets the availability status (for testing)
func (p *PlaceholderProvider) SetAvailable(available bool) {
	p.available = available
}

// GetConfig returns the provider configuration
func (p *PlaceholderProvider) GetConfig() map[string]interface{} {
	return p.config
}

// GetBaseURL returns the base URL
func (p *PlaceholderProvider) GetBaseURL() string {
	return p.baseURL
}

// GetModel returns the model
func (p *PlaceholderProvider) GetModel() string {
	return p.model
}

// estimateTokens estimates the number of tokens in messages (rough approximation)
func estimateTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Rough estimation: ~4 characters per token
		total += len(msg.Content) / 4
		if total == 0 {
			total = 1 // Minimum 1 token
		}
	}
	return total
}