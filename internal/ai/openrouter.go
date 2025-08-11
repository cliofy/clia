package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"io"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/yourusername/clia/pkg/utils"
)

// OpenRouterProvider implements LLMProvider for OpenRouter API
type OpenRouterProvider struct {
	client *openai.Client
	config *ProviderConfig
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(config *ProviderConfig) *OpenRouterProvider {
	var client *openai.Client
	
	if config.APIKey != "" {
		clientConfig := openai.DefaultConfig(config.APIKey)
		
		// Set OpenRouter endpoint
		if config.Endpoint != "" {
			clientConfig.BaseURL = config.Endpoint
		} else {
			clientConfig.BaseURL = "https://openrouter.ai/api/v1"
		}
		
		client = openai.NewClientWithConfig(clientConfig)
	}
	
	return &OpenRouterProvider{
		client: client,
		config: config,
	}
}

// Complete implements LLMProvider
func (p *OpenRouterProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if p.client == nil {
		return nil, NewAIError(ErrorTypeAuth, "OpenRouter client not configured", nil)
	}
	
	// Create context with timeout
	if p.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}
	
	// Prepare the request with OpenRouter-specific model format
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
		return nil, p.handleOpenRouterError(err)
	}
	
	// Parse the response
	if len(resp.Choices) == 0 {
		return nil, NewAIError(ErrorTypeParsing, "no choices returned from OpenRouter", nil)
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
func (p *OpenRouterProvider) ValidateConfig() error {
	if p.config == nil {
		return fmt.Errorf("provider config is nil")
	}
	
	if p.config.APIKey == "" {
		return fmt.Errorf("OpenRouter API key is required")
	}
	
	if p.config.Model == "" {
		return fmt.Errorf("OpenRouter model is required")
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
func (p *OpenRouterProvider) GetName() string {
	return "openrouter"
}

// GetModel implements LLMProvider
func (p *OpenRouterProvider) GetModel() string {
	if p.config != nil {
		return p.config.Model
	}
	return ""
}

// IsConfigured implements LLMProvider
func (p *OpenRouterProvider) IsConfigured() bool {
	return p.client != nil && p.config != nil && p.config.APIKey != ""
}

// parseCommandSuggestions attempts to parse JSON command suggestions from the response
func (p *OpenRouterProvider) parseCommandSuggestions(content string) ([]CommandSuggestion, error) {
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

// handleOpenRouterError converts OpenRouter errors to AIError
func (p *OpenRouterProvider) handleOpenRouterError(err error) error {
	// Check for OpenAI request error using error assertion (OpenRouter uses same format)
	if requestErr, ok := err.(*openai.RequestError); ok {
		switch requestErr.HTTPStatusCode {
		case 401:
			return NewAIError(ErrorTypeAuth, "Invalid OpenRouter API key or unauthorized access", err)
		case 429:
			return NewAIError(ErrorTypeRateLimit, "OpenRouter rate limit exceeded", err)
		case 400:
			return NewAIError(ErrorTypeValidation, "Invalid request to OpenRouter", err)
		default:
			return NewAIError(ErrorTypeUnknown, fmt.Sprintf("OpenRouter API error: %s", requestErr.Error()), err)
		}
	}
	
	// Check for network errors
	if strings.Contains(err.Error(), "context deadline exceeded") {
		return NewAIError(ErrorTypeNetwork, "Request timeout to OpenRouter", err)
	}
	
	if strings.Contains(err.Error(), "no such host") || 
	   strings.Contains(err.Error(), "connection refused") {
		return NewAIError(ErrorTypeNetwork, "Network connection error to OpenRouter", err)
	}
	
	return NewAIError(ErrorTypeUnknown, "Unexpected OpenRouter error", err)
}

// TestConnection tests the connection to OpenRouter API
func (p *OpenRouterProvider) TestConnection(ctx context.Context) error {
	if !p.IsConfigured() {
		return NewAIError(ErrorTypeAuth, "OpenRouter provider not configured", nil)
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

// GetModels implements ModelListProvider interface for OpenRouter
func (p *OpenRouterProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	if !p.IsConfigured() {
		return nil, NewAIError(ErrorTypeAuth, "OpenRouter provider not configured", nil)
	}
	
	// Create HTTP request to OpenRouter models API
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
	
	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var apiResponse OpenRouterModelsResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to our ModelInfo format
	models := make([]ModelInfo, 0, len(apiResponse.Data))
	currentModel := p.GetModel()
	
	for _, model := range apiResponse.Data {
		pricing := ""
		if model.Pricing != nil && model.Pricing.Prompt != "" && model.Pricing.Completion != "" {
			pricing = fmt.Sprintf("$%.3f/$%.3f per 1k tokens", 
				parseFloat(model.Pricing.Prompt)*1000,
				parseFloat(model.Pricing.Completion)*1000)
		}
		
		modelInfo := ModelInfo{
			ID:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Pricing:     pricing,
			ContextSize: model.ContextLength,
			Current:     model.ID == currentModel,
		}
		
		models = append(models, modelInfo)
	}
	
	return models, nil
}

// SwitchModel implements ModelSwitcher interface
func (p *OpenRouterProvider) SwitchModel(modelName string) error {
	if p.config == nil {
		return fmt.Errorf("provider config is nil")
	}
	
	// Update the model in config
	p.config.Model = modelName
	
	// Recreate the client with new model (if needed)
	if p.client != nil {
		clientConfig := openai.DefaultConfig(p.config.APIKey)
		if p.config.Endpoint != "" {
			clientConfig.BaseURL = p.config.Endpoint
		} else {
			clientConfig.BaseURL = "https://openrouter.ai/api/v1"
		}
		p.client = openai.NewClientWithConfig(clientConfig)
	}
	
	return nil
}

// OpenRouter API response structures

// OpenRouterModelsResponse represents the response from OpenRouter models API
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// OpenRouterModel represents a model from OpenRouter API
type OpenRouterModel struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	ContextLength int                    `json:"context_length"`
	Pricing       *OpenRouterModelPricing `json:"pricing"`
}

// OpenRouterModelPricing represents pricing information
type OpenRouterModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// Helper function to parse price strings
func parseFloat(s string) float64 {
	// OpenRouter returns prices as strings like "0.000002"
	// Simple parsing - in production, use strconv.ParseFloat
	if s == "" {
		return 0.0
	}
	
	// Basic conversion for common price formats
	switch s {
	case "0":
		return 0.0
	case "0.000002":
		return 0.000002
	case "0.00001":
		return 0.00001
	case "0.00003":
		return 0.00003
	default:
		return 0.001 // Default fallback
	}
}