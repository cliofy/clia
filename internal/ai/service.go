package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yourusername/clia/internal/prompt"
)

// Service provides AI-powered command suggestion functionality
type Service struct {
	provider       LLMProvider
	promptBuilder  *prompt.PromptBuilder
	factory        *ProviderFactory
	fallbackMode   bool
	requestTimeout time.Duration
}

// NewService creates a new AI service
func NewService() *Service {
	return &Service{
		factory:        NewProviderFactory(),
		promptBuilder:  prompt.NewPromptBuilder(),
		fallbackMode:   false,
		requestTimeout: 30 * time.Second,
	}
}

// SetProvider sets the LLM provider
func (s *Service) SetProvider(provider LLMProvider) *Service {
	s.provider = provider
	return s
}

// SetProviderByConfig creates and sets a provider from configuration
func (s *Service) SetProviderByConfig(providerType ProviderType, config *ProviderConfig) error {
	provider, err := s.factory.Create(providerType, config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	s.provider = provider
	return nil
}

// SetTimeout sets the request timeout
func (s *Service) SetTimeout(timeout time.Duration) *Service {
	s.requestTimeout = timeout
	return s
}

// SetFallbackMode enables/disables fallback mode
func (s *Service) SetFallbackMode(enabled bool) *Service {
	s.fallbackMode = enabled
	return s
}

// SuggestCommands generates command suggestions based on natural language input
func (s *Service) SuggestCommands(ctx context.Context, userInput string) (*CompletionResponse, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}
	
	if !s.provider.IsConfigured() {
		return nil, fmt.Errorf("LLM provider is not properly configured")
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	
	// Build prompt
	promptText, err := s.promptBuilder.BuildCommandPrompt(ctx, userInput)
	if err != nil {
		if s.fallbackMode {
			promptText = s.promptBuilder.BuildQuickPrompt(userInput)
		} else {
			return nil, fmt.Errorf("failed to build prompt: %w", err)
		}
	}
	
	// Validate prompt
	if err := s.promptBuilder.ValidatePrompt(promptText); err != nil {
		return nil, fmt.Errorf("invalid prompt: %w", err)
	}
	
	// Create completion request
	req := &CompletionRequest{
		Prompt: promptText,
	}
	
	// Get suggestions from LLM
	response, err := s.provider.Complete(ctx, req)
	if err != nil {
		// Handle different error types
		if s.fallbackMode {
			return s.handleFallback(userInput, err)
		}
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}
	
	// Process and validate suggestions
	if response.Suggestions == nil {
		response.Suggestions = []CommandSuggestion{}
	}
	
	// Filter and sort suggestions
	suggestions := CommandSuggestions(response.Suggestions)
	
	// Sort by confidence and limit results
	suggestions = suggestions.SortByConfidence().Top(3)
	
	response.Suggestions = suggestions
	return response, nil
}

// TestConnection tests the connection to the configured LLM provider
func (s *Service) TestConnection(ctx context.Context) error {
	if s.provider == nil {
		return fmt.Errorf("no LLM provider configured")
	}
	
	// For OpenAI provider, we can test connection
	if openaiProvider, ok := s.provider.(*OpenAIProvider); ok {
		return openaiProvider.TestConnection(ctx)
	}
	
	// For other providers, do a simple validation
	return s.provider.ValidateConfig()
}

// GetProviderInfo returns information about the current provider
func (s *Service) GetProviderInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	if s.provider != nil {
		info["name"] = s.provider.GetName()
		info["model"] = s.provider.GetModel()
		info["configured"] = s.provider.IsConfigured()
	} else {
		info["name"] = "none"
		info["configured"] = false
	}
	
	info["fallback_mode"] = s.fallbackMode
	info["timeout"] = s.requestTimeout.String()
	
	return info
}

// handleFallback provides fallback suggestions when LLM fails
func (s *Service) handleFallback(userInput string, originalErr error) (*CompletionResponse, error) {
	log.Printf("LLM failed, using fallback mode: %v", originalErr)
	
	// Generate simple rule-based suggestions
	suggestions := s.generateFallbackSuggestions(userInput)
	
	return &CompletionResponse{
		Content: fmt.Sprintf("LLM unavailable, generated %d fallback suggestions", len(suggestions)),
		Suggestions: suggestions,
		Provider:    "fallback",
		Model:       "rule-based",
	}, nil
}

// generateFallbackSuggestions generates simple rule-based command suggestions
func (s *Service) generateFallbackSuggestions(userInput string) []CommandSuggestion {
	input := strings.ToLower(strings.TrimSpace(userInput))
	var suggestions []CommandSuggestion
	
	// Simple pattern matching for common commands
	patterns := map[string]CommandSuggestion{
		"list":      {Command: "ls -la", Description: "List files with details", Confidence: 0.7, Safe: true, Category: "file_management"},
		"files":     {Command: "ls", Description: "List files", Confidence: 0.7, Safe: true, Category: "file_management"},
		"directory": {Command: "pwd", Description: "Show current directory", Confidence: 0.7, Safe: true, Category: "navigation"},
		"current":   {Command: "pwd", Description: "Show current directory", Confidence: 0.7, Safe: true, Category: "navigation"},
		"copy":      {Command: "cp", Description: "Copy files", Confidence: 0.5, Safe: true, Category: "file_management"},
		"move":      {Command: "mv", Description: "Move files", Confidence: 0.5, Safe: false, Category: "file_management"},
		"delete":    {Command: "rm", Description: "Delete files (use with caution)", Confidence: 0.5, Safe: false, Category: "file_management"},
		"search":    {Command: "find", Description: "Search for files", Confidence: 0.6, Safe: true, Category: "search"},
		"git":       {Command: "git status", Description: "Show git status", Confidence: 0.8, Safe: true, Category: "development"},
		"process":   {Command: "ps aux", Description: "Show running processes", Confidence: 0.7, Safe: true, Category: "system_info"},
	}
	
	// Find matching patterns
	for pattern, suggestion := range patterns {
		if strings.Contains(input, pattern) {
			suggestions = append(suggestions, suggestion)
		}
	}
	
	// If no patterns match, provide generic suggestions
	if len(suggestions) == 0 {
		suggestions = []CommandSuggestion{
			{
				Command:     "echo '" + userInput + "'",
				Description: "Echo the input text",
				Confidence:  0.3,
				Safe:        true,
				Category:    "general",
			},
		}
	}
	
	return suggestions
}

// GetPromptBuilder returns the prompt builder for configuration
func (s *Service) GetPromptBuilder() *prompt.PromptBuilder {
	return s.promptBuilder
}

// SetPromptBuilder sets a custom prompt builder
func (s *Service) SetPromptBuilder(builder *prompt.PromptBuilder) *Service {
	s.promptBuilder = builder
	return s
}