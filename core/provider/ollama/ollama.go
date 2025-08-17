package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ErrOllamaNotRunning     = errors.New("ollama service not running")
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

// Provider implements the Ollama provider
type Provider struct {
	baseURL string
	model   string
	name    string
	config  map[string]interface{}
	client  *http.Client
}

// NewProvider creates a new Ollama provider
func NewProvider(name string, cfg map[string]interface{}) (*Provider, error) {
	baseURL := config.GetStringValue(cfg, "base_url", "http://localhost:11434")
	model := config.GetStringValue(cfg, "model", "llama2")

	// Note: Ollama doesn't require an API key
	provider := &Provider{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		name:    name,
		config:  cfg,
		client: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for local models
		},
	}

	return provider, nil
}

// Chat implements the Provider interface
func (p *Provider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Check if Ollama is available
	if !p.Available() {
		return nil, ErrOllamaNotRunning
	}

	// Build Ollama request
	ollamaReq := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": p.convertMessages(req.Messages),
		"stream":   false,
	}

	// Apply options if provided
	if req.Options != nil {
		options := make(map[string]interface{})
		if req.Options.Temperature != nil {
			options["temperature"] = *req.Options.Temperature
		}
		if req.Options.MaxTokens != nil {
			options["num_predict"] = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			options["top_p"] = *req.Options.TopP
		}
		if len(options) > 0 {
			ollamaReq["options"] = options
		}
	}

	// Marshal request body
	jsonBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, p.handleError(err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp.StatusCode)
	}

	// Parse response
	var ollamaResp struct {
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Message   struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done               bool `json:"done"`
		TotalDuration      int64 `json:"total_duration"`
		LoadDuration       int64 `json:"load_duration"`
		PromptEvalCount    int  `json:"prompt_eval_count"`
		PromptEvalDuration int64 `json:"prompt_eval_duration"`
		EvalCount          int  `json:"eval_count"`
		EvalDuration       int64 `json:"eval_duration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our response format
	createdAt := time.Now()
	if ollamaResp.CreatedAt != "" {
		if parsedTime, err := time.Parse(time.RFC3339, ollamaResp.CreatedAt); err == nil {
			createdAt = parsedTime
		}
	}

	result := &ChatResponse{
		ID:      fmt.Sprintf("ollama-%d", time.Now().UnixNano()),
		Model:   ollamaResp.Model,
		Content: ollamaResp.Message.Content,
		Created: createdAt,
		Usage: &Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}

	return result, nil
}

// ChatStream implements the Provider interface for streaming responses
func (p *Provider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Check if Ollama is available
	if !p.Available() {
		return nil, ErrOllamaNotRunning
	}

	// Build Ollama request
	ollamaReq := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": p.convertMessages(req.Messages),
		"stream":   true,
	}

	// Apply options if provided
	if req.Options != nil {
		options := make(map[string]interface{})
		if req.Options.Temperature != nil {
			options["temperature"] = *req.Options.Temperature
		}
		if req.Options.MaxTokens != nil {
			options["num_predict"] = *req.Options.MaxTokens
		}
		if req.Options.TopP != nil {
			options["top_p"] = *req.Options.TopP
		}
		if len(options) > 0 {
			ollamaReq["options"] = options
		}
	}

	// Marshal request body
	jsonBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, p.handleError(err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.handleHTTPError(resp.StatusCode)
	}

	// Create output channel
	ch := make(chan *ChatChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		
		for scanner.Scan() {
			line := scanner.Text()
			
			// Parse JSON response
			var chunk struct {
				Model     string `json:"model"`
				CreatedAt string `json:"created_at"`
				Message   struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}

			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue // Skip invalid JSON
			}

			// Send chunk
			select {
			case <-ctx.Done():
				return
			case ch <- &ChatChunk{
				ID:      fmt.Sprintf("ollama-stream-%d", time.Now().UnixNano()),
				Model:   chunk.Model,
				Content: chunk.Message.Content,
				Done:    chunk.Done,
			}:
			}

			if chunk.Done {
				return
			}
		}
	}()

	return ch, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.name
}

// Available checks if the Ollama service is available
func (p *Provider) Available() bool {
	// Try to connect to Ollama API
	resp, err := p.client.Get(p.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ListModels returns a list of available models from Ollama
func (p *Provider) ListModels() ([]string, error) {
	resp, err := p.client.Get(p.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, model := range result.Models {
		models[i] = model.Name
	}

	return models, nil
}

// PullModel pulls a model from the Ollama registry
func (p *Provider) PullModel(modelName string) error {
	reqBody := map[string]string{
		"name": modelName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.baseURL+"/api/pull", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to pull model: status %d", resp.StatusCode)
	}

	// Read streaming response (pull progress)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// In a real implementation, we might want to report progress
		// For now, just consume the stream
		_ = scanner.Text()
	}

	return nil
}

// getModel returns the model to use for the request
func (p *Provider) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return p.model
}

// convertMessages converts messages to the format expected by Ollama
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
	
	// Check if Ollama is not running
	if strings.Contains(err.Error(), "connection refused") {
		return ErrOllamaNotRunning
	}
	
	return fmt.Errorf("Ollama error: %w", err)
}

// handleHTTPError handles HTTP error responses
func (p *Provider) handleHTTPError(statusCode int) error {
	switch statusCode {
	case http.StatusNotFound:
		return ErrModelNotSupported
	case http.StatusTooManyRequests:
		return ErrRateLimitExceeded
	case http.StatusServiceUnavailable:
		return ErrOllamaNotRunning
	default:
		return fmt.Errorf("Ollama HTTP error: status %d", statusCode)
	}
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