package prompt

import (
	"context"
	"fmt"
)

// PromptBuilder builds prompts for LLM requests
type PromptBuilder struct {
	collector *ContextCollector
	template  string
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		collector: NewContextCollector(),
		template:  SystemPrompt,
	}
}

// WithTemplate sets a custom template
func (b *PromptBuilder) WithTemplate(template string) *PromptBuilder {
	b.template = template
	return b
}

// WithContextOptions configures the context collector
func (b *PromptBuilder) WithContextOptions(maxFiles int, includeHidden, includeEnvVars bool) *PromptBuilder {
	b.collector.SetMaxFiles(maxFiles).
		SetIncludeHidden(includeHidden).
		SetIncludeEnvVars(includeEnvVars)
	return b
}

// BuildCommandPrompt builds a prompt for command suggestion
func (b *PromptBuilder) BuildCommandPrompt(ctx context.Context, userInput string) (string, error) {
	// Collect context
	envContext, err := b.collector.Collect()
	if err != nil {
		// If context collection fails, use a fallback approach
		return b.buildFallbackPrompt(userInput, err), nil
	}

	// Build template
	template := NewCommandPromptTemplate(userInput, envContext)

	// Enhance with examples if context is rich enough
	if envContext.FileCount > 0 || envContext.DirectoryCount > 0 {
		template = template.EnhanceWithExamples()
	}

	return template.Build(), nil
}

// BuildQuickPrompt builds a minimal prompt without context collection
func (b *PromptBuilder) BuildQuickPrompt(userInput string) string {
	// Use minimal context to avoid delays
	os := "unknown"
	shell := "bash"

	if quickContext, err := b.collector.Collect(); err == nil {
		os = quickContext.OS
		shell = quickContext.Shell
	}

	return QuickCommandPrompt(userInput, os, shell)
}

// BuildCustomPrompt builds a custom prompt with variables
func (b *PromptBuilder) BuildCustomPrompt(template, userInput string, variables map[string]string) (string, error) {
	// Collect context for custom template
	envContext, err := b.collector.Collect()
	if err != nil {
		return "", fmt.Errorf("failed to collect context: %w", err)
	}

	// Build custom template
	customTemplate := NewCustomPromptTemplate(template, userInput, envContext)

	// Set variables
	for key, value := range variables {
		customTemplate.SetVariable(key, value)
	}

	return customTemplate.Build(), nil
}

// buildFallbackPrompt builds a prompt when context collection fails
func (b *PromptBuilder) buildFallbackPrompt(userInput string, contextErr error) string {
	fallbackPrompt := fmt.Sprintf(`You are a command line assistant. The user wants: "%s"

Note: Unable to collect full environment context (%v). Provide general command suggestions.

Respond with JSON format:
{"commands":[{"cmd":"command","description":"description","confidence":0.7,"safe":true,"category":"general"}]}`,
		userInput, contextErr)

	return fallbackPrompt
}

// ValidatePrompt checks if a prompt is valid and not too long
func (b *PromptBuilder) ValidatePrompt(prompt string) error {
	const maxPromptLength = 8000 // Reasonable limit to avoid token limits

	if prompt == "" {
		return fmt.Errorf("prompt is empty")
	}

	if len(prompt) > maxPromptLength {
		return fmt.Errorf("prompt too long (%d characters, max %d)", len(prompt), maxPromptLength)
	}

	return nil
}

// GetContextCollector returns the context collector for configuration
func (b *PromptBuilder) GetContextCollector() *ContextCollector {
	return b.collector
}

// PreviewContext collects and returns current context for debugging
func (b *PromptBuilder) PreviewContext() (*Context, error) {
	return b.collector.Collect()
}
