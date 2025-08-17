package provider

import (
	"context"
	"errors"
	"time"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Chat sends a chat request and returns the complete response
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	
	// ChatStream sends a chat request and returns a stream of response chunks
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
	
	// Name returns the provider name
	Name() string
	
	// Available checks if the provider is available and configured
	Available() bool
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Options  *ChatOptions `json:"options,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatOptions contains optional parameters for chat requests
type ChatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"` // 0.0 to 2.0
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
}

// ChatResponse represents a complete chat response
type ChatResponse struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Content string    `json:"content"`
	Usage   *Usage    `json:"usage,omitempty"`
	Created time.Time `json:"created"`
}

// ChatChunk represents a streaming response chunk
type ChatChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"` // Incremental content
	Done    bool   `json:"done"`    // True when stream is complete
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Common errors
var (
	ErrProviderNotAvailable = errors.New("provider not available")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrAPIKeyMissing        = errors.New("API key missing")
	ErrModelNotSupported    = errors.New("model not supported")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrTimeout              = errors.New("request timeout")
)

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeOllama     ProviderType = "ollama"
)