package ai

import (
	"context"
	"time"
)

// CompletionRequest represents a request to an LLM provider
type CompletionRequest struct {
	Prompt      string            `json:"prompt"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// CompletionResponse represents a response from an LLM provider
type CompletionResponse struct {
	Content     string              `json:"content"`
	Suggestions []CommandSuggestion `json:"suggestions,omitempty"`
	Usage       *UsageInfo          `json:"usage,omitempty"`
	Model       string              `json:"model,omitempty"`
	Provider    string              `json:"provider,omitempty"`
}

// CommandSuggestion represents a suggested command
type CommandSuggestion struct {
	Command     string  `json:"cmd"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence,omitempty"`
	Safe        bool    `json:"safe"`
	Category    string  `json:"category,omitempty"`
}

// UsageInfo represents token usage information
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderConfig represents configuration for an LLM provider
type ProviderConfig struct {
	Name        string        `json:"name"`
	APIKey      string        `json:"api_key"`
	Model       string        `json:"model"`
	Endpoint    string        `json:"endpoint,omitempty"`
	Timeout     time.Duration `json:"timeout"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float32       `json:"temperature"`
}

// Error types for AI operations
type ErrorType string

const (
	ErrorTypeAuth       ErrorType = "auth_error"
	ErrorTypeNetwork    ErrorType = "network_error"
	ErrorTypeRateLimit  ErrorType = "rate_limit_error"
	ErrorTypeValidation ErrorType = "validation_error"
	ErrorTypeParsing    ErrorType = "parsing_error"
	ErrorTypeUnknown    ErrorType = "unknown_error"
)

// AIError represents an AI-specific error
type AIError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    int       `json:"code,omitempty"`
	Err     error     `json:"-"`
}

func (e *AIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AIError) Unwrap() error {
	return e.Err
}

// NewAIError creates a new AI error
func NewAIError(errType ErrorType, message string, err error) *AIError {
	return &AIError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// CommandSuggestions is a slice of CommandSuggestion with helper methods
type CommandSuggestions []CommandSuggestion

// FilterSafe returns only safe commands
func (cs CommandSuggestions) FilterSafe() CommandSuggestions {
	var safe CommandSuggestions
	for _, cmd := range cs {
		if cmd.Safe {
			safe = append(safe, cmd)
		}
	}
	return safe
}

// SortByConfidence sorts suggestions by confidence (highest first)
func (cs CommandSuggestions) SortByConfidence() CommandSuggestions {
	if len(cs) <= 1 {
		return cs
	}

	// Simple bubble sort by confidence
	sorted := make(CommandSuggestions, len(cs))
	copy(sorted, cs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Confidence < sorted[j+1].Confidence {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// Top returns the top n suggestions
func (cs CommandSuggestions) Top(n int) CommandSuggestions {
	if n >= len(cs) {
		return cs
	}
	return cs[:n]
}

// ModelInfo represents information about an AI model
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pricing     string `json:"pricing"`
	ContextSize int    `json:"context_length"`
	Current     bool   `json:"current"`
}

// ProviderStatusInfo represents the status of a provider
type ProviderStatusInfo struct {
	Type       ProviderType `json:"type"`
	Available  bool         `json:"available"`
	Configured bool         `json:"configured"`
	Current    bool         `json:"current"`
}

// Extended interfaces for providers

// ModelListProvider interface for providers that support listing models
type ModelListProvider interface {
	GetModels(ctx context.Context) ([]ModelInfo, error)
}

// ModelSwitcher interface for providers that support dynamic model switching
type ModelSwitcher interface {
	SwitchModel(modelName string) error
}

// ConnectionTester interface for providers that support connection testing
type ConnectionTester interface {
	TestConnection(ctx context.Context) error
}
