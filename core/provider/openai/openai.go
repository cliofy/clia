package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/yourusername/clia/core/config"
)

// Common errors
var (
	ErrProviderNotAvailable = errors.New("provider not available")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrAPIKeyMissing        = errors.New("API key missing")
	ErrModelNotSupported    = errors.New("model not supported")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrTimeout              = errors.New("request timeout")
)

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

// Provider implements the OpenAI provider
type Provider struct {
	client   *openai.Client
	model    string
	name     string
	config   map[string]interface{}
	baseURL  string
	apiKey   string
}

// NewProvider creates a new OpenAI provider
func NewProvider(name string, cfg map[string]interface{}) (*Provider, error) {
	apiKey := config.GetStringValue(cfg, "api_key", "")
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	baseURL := config.GetStringValue(cfg, "base_url", "")
	model := config.GetStringValue(cfg, "model", "gpt-3.5-turbo")

	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(apiKey)
	
	// Set custom base URL if provided
	if baseURL != "" && baseURL != "https://api.openai.com/v1" {
		clientConfig.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &Provider{
		client:  client,
		model:   model,
		name:    name,
		config:  cfg,
		baseURL: clientConfig.BaseURL,
		apiKey:  apiKey,
	}, nil
}

// Chat implements the Provider interface
func (p *Provider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert our messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Build OpenAI request
	openaiReq := openai.ChatCompletionRequest{
		Model:    p.getModel(req.Model),
		Messages: messages,
	}

	// Apply options if provided
	if req.Options != nil {
		if req.Options.Temperature != nil {
			openaiReq.Temperature = float32(*req.Options.Temperature)
		}
		if req.Options.MaxTokens != nil {
			openaiReq.MaxTokens = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			openaiReq.TopP = float32(*req.Options.TopP)
		}
	}

	// Call OpenAI API
	resp, err := p.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, p.handleError(err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Convert response
	result := &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: resp.Choices[0].Message.Content,
		Created: time.Unix(int64(resp.Created), 0),
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	return result, nil
}

// ChatStream implements the Provider interface
func (p *Provider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Convert our messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Build OpenAI streaming request
	openaiReq := openai.ChatCompletionRequest{
		Model:    p.getModel(req.Model),
		Messages: messages,
		Stream:   true,
	}

	// Apply options if provided
	if req.Options != nil {
		if req.Options.Temperature != nil {
			openaiReq.Temperature = float32(*req.Options.Temperature)
		}
		if req.Options.MaxTokens != nil {
			openaiReq.MaxTokens = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			openaiReq.TopP = float32(*req.Options.TopP)
		}
	}

	// Create stream
	stream, err := p.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, p.handleError(err)
	}

	// Create output channel
	ch := make(chan *ChatChunk)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err == io.EOF {
				// Send final chunk with Done=true
				select {
				case <-ctx.Done():
					return
				case ch <- &ChatChunk{
					ID:      response.ID,
					Model:   response.Model,
					Content: "",
					Done:    true,
				}:
				}
				return
			}

			if err != nil {
				// Log error but don't send it through the channel
				// The stream will be closed and the receiver can detect that
				return
			}

			if len(response.Choices) > 0 {
				chunk := &ChatChunk{
					ID:      response.ID,
					Model:   response.Model,
					Content: response.Choices[0].Delta.Content,
					Done:    false,
				}

				select {
				case <-ctx.Done():
					return
				case ch <- chunk:
				}
			}
		}
	}()

	return ch, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.name
}

// Available checks if the provider is available
func (p *Provider) Available() bool {
	// Basic check - ensure we have an API key and client
	return p.client != nil && p.apiKey != ""
}

// getModel returns the model to use for the request
func (p *Provider) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return p.model
}

// handleError converts OpenAI errors to our error types
func (p *Provider) handleError(err error) error {
	// Check for specific OpenAI error types
	if apiErr, ok := err.(*openai.APIError); ok {
		switch apiErr.HTTPStatusCode {
		case 401:
			return ErrAPIKeyMissing
		case 429:
			return ErrRateLimitExceeded
		case 404:
			if apiErr.Code == "model_not_found" {
				return ErrModelNotSupported
			}
		}
		return fmt.Errorf("OpenAI API error: %s", apiErr.Message)
	}

	// Check for context errors
	if err == context.DeadlineExceeded {
		return ErrTimeout
	}

	// Return generic error
	return fmt.Errorf("OpenAI error: %w", err)
}

// GetConfig returns the provider configuration
func (p *Provider) GetConfig() map[string]interface{} {
	return p.config
}

// GetModel returns the configured model
func (p *Provider) GetModel() string {
	return p.model
}

// GetBaseURL returns the base URL
func (p *Provider) GetBaseURL() string {
	return p.baseURL
}