package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/yourusername/clia/pkg/utils"
)

// OpenAIProvider implements LLMProvider for OpenAI API
type OpenAIProvider struct {
	client *openai.Client
	config *ProviderConfig
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config *ProviderConfig) *OpenAIProvider {
	var client *openai.Client
	
	if config.APIKey != "" {
		clientConfig := openai.DefaultConfig(config.APIKey)
		
		// Set custom endpoint if provided
		if config.Endpoint != "" && config.Endpoint != "https://api.openai.com/v1" {
			clientConfig.BaseURL = config.Endpoint
		}
		
		client = openai.NewClientWithConfig(clientConfig)
	}
	
	return &OpenAIProvider{
		client: client,
		config: config,
	}
}

// Complete implements LLMProvider
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if p.client == nil {
		return nil, NewAIError(ErrorTypeAuth, "OpenAI client not configured", nil)
	}
	
	// Create context with timeout
	if p.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}
	
	// Prepare the request
	chatReq := openai.ChatCompletionRequest{
		Model: p.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Prompt,
			},
		},
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}
	
	// Make the API call
	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, p.handleOpenAIError(err)
	}
	
	// Parse the response
	if len(resp.Choices) == 0 {
		return nil, NewAIError(ErrorTypeParsing, "no choices returned from OpenAI", nil)
	}
	
	content := resp.Choices[0].Message.Content
	suggestions, parseErr := p.parseCommandSuggestions(content)
	if parseErr != nil {
		// If parsing fails, treat the content as a plain text response
		suggestions = []CommandSuggestion{
			{
				Command:     strings.TrimSpace(content),
				Description: "AI suggested command",
				Confidence:  0.8,
				Safe:        utils.IsCommandSafe(content),
				Category:    "general",
			},
		}
	}
	
	return &CompletionResponse{
		Content:     content,
		Suggestions: suggestions,
		Usage: &UsageInfo{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Model:    p.config.Model,
		Provider: p.GetName(),
	}, nil
}

// ValidateConfig implements LLMProvider
func (p *OpenAIProvider) ValidateConfig() error {
	if p.config == nil {
		return fmt.Errorf("provider config is nil")
	}
	
	if p.config.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	
	if p.config.Model == "" {
		return fmt.Errorf("OpenAI model is required")
	}
	
	if p.config.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}
	
	if p.config.Temperature < 0 || p.config.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	
	return nil
}

// GetName implements LLMProvider
func (p *OpenAIProvider) GetName() string {
	return "openai"
}

// GetModel implements LLMProvider
func (p *OpenAIProvider) GetModel() string {
	if p.config != nil {
		return p.config.Model
	}
	return ""
}

// IsConfigured implements LLMProvider
func (p *OpenAIProvider) IsConfigured() bool {
	return p.client != nil && p.config != nil && p.config.APIKey != ""
}

// parseCommandSuggestions attempts to parse JSON command suggestions from the response
func (p *OpenAIProvider) parseCommandSuggestions(content string) ([]CommandSuggestion, error) {
	// Try to find JSON in the response
	content = strings.TrimSpace(content)
	
	// Look for JSON block markers
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end != -1 {
			content = content[start : start+end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.Index(content[start:], "```")
		if end != -1 {
			content = content[start : start+end]
		}
	}
	
	// Try to parse as JSON
	var result struct {
		Commands []CommandSuggestion `json:"commands"`
	}
	
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}
	
	// Validate and enhance suggestions
	for i := range result.Commands {
		if result.Commands[i].Command == "" {
			continue
		}
		
		// Set safety flag based on command analysis
		result.Commands[i].Safe = utils.IsCommandSafe(result.Commands[i].Command) && 
								  !utils.IsDangerousCommand(result.Commands[i].Command)
		
		// Set default confidence if not provided
		if result.Commands[i].Confidence == 0 {
			result.Commands[i].Confidence = 0.7
		}
		
		// Set default category
		if result.Commands[i].Category == "" {
			result.Commands[i].Category = "general"
		}
	}
	
	return result.Commands, nil
}

// handleOpenAIError converts OpenAI errors to AIError
func (p *OpenAIProvider) handleOpenAIError(err error) error {
	// Check for OpenAI request error using error assertion
	if requestErr, ok := err.(*openai.RequestError); ok {
		switch requestErr.HTTPStatusCode {
		case 401:
			return NewAIError(ErrorTypeAuth, "Invalid API key or unauthorized access", err)
		case 429:
			return NewAIError(ErrorTypeRateLimit, "Rate limit exceeded", err)
		case 400:
			return NewAIError(ErrorTypeValidation, "Invalid request", err)
		default:
			return NewAIError(ErrorTypeUnknown, fmt.Sprintf("OpenAI API error: %s", requestErr.Error()), err)
		}
	}
	
	// Check for network errors
	if strings.Contains(err.Error(), "context deadline exceeded") {
		return NewAIError(ErrorTypeNetwork, "Request timeout", err)
	}
	
	if strings.Contains(err.Error(), "no such host") || 
	   strings.Contains(err.Error(), "connection refused") {
		return NewAIError(ErrorTypeNetwork, "Network connection error", err)
	}
	
	return NewAIError(ErrorTypeUnknown, "Unexpected error", err)
}

// TestConnection tests the connection to OpenAI API
func (p *OpenAIProvider) TestConnection(ctx context.Context) error {
	if !p.IsConfigured() {
		return NewAIError(ErrorTypeAuth, "Provider not configured", nil)
	}
	
	// Create a simple test request
	testReq := &CompletionRequest{
		Prompt:    "Say 'connection test successful' if you can read this.",
		MaxTokens: 10,
	}
	
	// Set a short timeout for connection test
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	_, err := p.Complete(testCtx, testReq)
	return err
}

// SwitchModel implements ModelSwitcher interface for OpenAI
func (p *OpenAIProvider) SwitchModel(modelName string) error {
	if p.config == nil {
		return fmt.Errorf("provider config is nil")
	}
	
	// Update the model in config
	p.config.Model = modelName
	
	return nil
}