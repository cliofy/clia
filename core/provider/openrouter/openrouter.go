package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

// Provider implements the OpenRouter provider
type Provider struct {
	apiKey     string
	baseURL    string
	model      string
	name       string
	config     map[string]interface{}
	client     *http.Client
	referer    string // HTTP-Referer header for OpenRouter
	appName    string // X-Title header for OpenRouter
}

// NewProvider creates a new OpenRouter provider
func NewProvider(name string, cfg map[string]interface{}) (*Provider, error) {
	apiKey := config.GetStringValue(cfg, "api_key", "")
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	baseURL := config.GetStringValue(cfg, "base_url", "https://openrouter.ai/api/v1")
	model := config.GetStringValue(cfg, "model", "openai/gpt-3.5-turbo")
	referer := config.GetStringValue(cfg, "referer", "https://github.com/yourusername/clia")
	appName := config.GetStringValue(cfg, "app_name", "CLIA")

	return &Provider{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		name:    name,
		config:  cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		referer: referer,
		appName: appName,
	}, nil
}

// Chat implements the Provider interface
func (p *Provider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build request body
	requestBody := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": p.convertMessages(req.Messages),
		"stream":   false,
	}

	// Apply options if provided
	if req.Options != nil {
		if req.Options.Temperature != nil {
			requestBody["temperature"] = *req.Options.Temperature
		}
		if req.Options.MaxTokens != nil {
			requestBody["max_tokens"] = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			requestBody["top_p"] = *req.Options.TopP
		}
	}

	// Marshal request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	p.setHeaders(httpReq)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, p.handleError(err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp)
	}

	// Parse response
	var openRouterResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenRouter")
	}

	// Convert to our response format
	result := &ChatResponse{
		ID:      openRouterResp.ID,
		Model:   openRouterResp.Model,
		Content: openRouterResp.Choices[0].Message.Content,
		Created: time.Unix(openRouterResp.Created, 0),
		Usage: &Usage{
			PromptTokens:     openRouterResp.Usage.PromptTokens,
			CompletionTokens: openRouterResp.Usage.CompletionTokens,
			TotalTokens:      openRouterResp.Usage.TotalTokens,
		},
	}

	return result, nil
}

// ChatStream implements the Provider interface for streaming responses
func (p *Provider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Build request body
	requestBody := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": p.convertMessages(req.Messages),
		"stream":   true,
	}

	// Apply options if provided
	if req.Options != nil {
		if req.Options.Temperature != nil {
			requestBody["temperature"] = *req.Options.Temperature
		}
		if req.Options.MaxTokens != nil {
			requestBody["max_tokens"] = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			requestBody["top_p"] = *req.Options.TopP
		}
	}

	// Marshal request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	p.setHeaders(httpReq)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, p.handleError(err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.handleHTTPError(resp)
	}

	// Create output channel
	ch := make(chan *ChatChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := resp.Body
		buffer := make([]byte, 4096)

		for {
			n, err := reader.Read(buffer)
			if err == io.EOF {
				// Send final chunk
				select {
				case <-ctx.Done():
					return
				case ch <- &ChatChunk{
					Done: true,
				}:
				}
				return
			}

			if err != nil {
				// Error reading stream
				return
			}

			// Parse SSE data
			data := string(buffer[:n])
			lines := strings.Split(data, "\n")

			for _, line := range lines {
				if !strings.HasPrefix(line, "data: ") {
					continue
				}

				jsonData := strings.TrimPrefix(line, "data: ")
				if jsonData == "[DONE]" {
					select {
					case <-ctx.Done():
						return
					case ch <- &ChatChunk{
						Done: true,
					}:
					}
					return
				}

				var chunk struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					Created int64  `json:"created"`
					Model   string `json:"model"`
					Choices []struct {
						Index int `json:"index"`
						Delta struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						} `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					} `json:"choices"`
				}

				if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
					continue // Skip invalid JSON
				}

				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					select {
					case <-ctx.Done():
						return
					case ch <- &ChatChunk{
						ID:      chunk.ID,
						Model:   chunk.Model,
						Content: chunk.Choices[0].Delta.Content,
						Done:    false,
					}:
					}
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
	return p.apiKey != ""
}

// setHeaders sets the required headers for OpenRouter API
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	// OpenRouter specific headers
	req.Header.Set("HTTP-Referer", p.referer)
	req.Header.Set("X-Title", p.appName)
}

// getModel returns the model to use for the request
func (p *Provider) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return p.model
}

// convertMessages converts messages to the format expected by OpenRouter
func (p *Provider) convertMessages(messages []Message) []map[string]string {
	result := make([]map[string]string, len(messages))
	for i, msg := range messages {
		result[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

// handleError converts errors to our error types
func (p *Provider) handleError(err error) error {
	if err == context.DeadlineExceeded {
		return ErrTimeout
	}
	return fmt.Errorf("OpenRouter error: %w", err)
}

// handleHTTPError handles HTTP error responses
func (p *Provider) handleHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return ErrAPIKeyMissing
		case http.StatusTooManyRequests:
			return ErrRateLimitExceeded
		case http.StatusNotFound:
			if strings.Contains(errorResp.Error.Message, "model") {
				return ErrModelNotSupported
			}
		}
		return fmt.Errorf("OpenRouter API error: %s", errorResp.Error.Message)
	}

	return fmt.Errorf("OpenRouter HTTP error: status %d", resp.StatusCode)
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

// ListModels returns a list of available models from OpenRouter
func (p *Provider) ListModels() ([]ModelInfo, error) {
	req, err := http.NewRequest("GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, p.handleError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp)
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}

	return result.Data, nil
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Pricing      Pricing `json:"pricing"`
	ContextLength int    `json:"context_length"`
	Architecture Architecture `json:"architecture"`
}

// Pricing represents the pricing information for a model
type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request"`
	Image      string `json:"image"`
}

// Architecture represents the architecture information for a model
type Architecture struct {
	Modality      string `json:"modality"`
	Tokenizer     string `json:"tokenizer"`
	InstructType  string `json:"instruct_type"`
}