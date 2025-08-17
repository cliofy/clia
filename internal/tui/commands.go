package tui

import (
	"github.com/yourusername/clia/internal/ai"
	"strings"
)

// Command represents a parsed command
type Command struct {
	Type string   // "provider", "model", "help"
	Args []string // command arguments
	Raw  string   // original input
}

// CommandType constants
const (
	CommandTypeProvider = "provider"
	CommandTypeModel    = "model"
	CommandTypeHelp     = "help"
	CommandTypeStatus   = "status"
)

// ParseCommand parses user input to extract commands
func ParseCommand(input string) *Command {
	input = strings.TrimSpace(input)

	// Check if it's a command (starts with /)
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	// Remove the leading slash and split by spaces
	cmdStr := input[1:]
	parts := strings.Fields(cmdStr)

	if len(parts) == 0 {
		return nil
	}

	command := &Command{
		Type: strings.ToLower(parts[0]),
		Raw:  input,
	}

	// Extract arguments
	if len(parts) > 1 {
		command.Args = parts[1:]
	}

	return command
}

// IsValidCommand checks if a command type is valid
func IsValidCommand(cmdType string) bool {
	switch cmdType {
	case CommandTypeProvider, CommandTypeModel, CommandTypeHelp, CommandTypeStatus:
		return true
	default:
		return false
	}
}

// GetCommandHelp returns help text for commands
func GetCommandHelp() string {
	return `Available commands:
  /provider              - List available providers and their status
  /provider <name>       - Switch to specified provider (openai, openrouter, anthropic, ollama)
  /model                 - List available models for current provider
  /model <name>          - Switch to specified model
  /status                - Show current configuration status
  /help                  - Show this help message

Direct command execution:
  !<command>             - Execute command directly without AI processing or safety checks

Examples:
  /provider openrouter   - Switch to OpenRouter provider
  /model openai/gpt-4    - Switch to GPT-4 model via OpenRouter
  /status                - Show current provider and model
  !ls -la                - Execute 'ls -la' command directly
  !pwd                   - Execute 'pwd' command directly`
}

// ValidateProviderCommand validates provider command arguments
func ValidateProviderCommand(args []string) error {
	if len(args) == 0 {
		return nil // List providers command
	}

	if len(args) != 1 {
		return ErrInvalidProviderArgs
	}

	providerName := strings.ToLower(args[0])
	validProviders := []string{"openai", "openrouter", "anthropic", "ollama"}

	for _, valid := range validProviders {
		if providerName == valid {
			return nil
		}
	}

	return ErrInvalidProviderName
}

// ValidateModelCommand validates model command arguments
func ValidateModelCommand(args []string) error {
	if len(args) == 0 {
		return nil // List models command
	}

	if len(args) != 1 {
		return ErrInvalidModelArgs
	}

	// Model name validation (basic check)
	modelName := strings.TrimSpace(args[0])
	if modelName == "" {
		return ErrEmptyModelName
	}

	return nil
}

// Command errors
var (
	ErrInvalidProviderArgs = &CommandError{Type: "validation", Message: "Invalid provider command. Usage: /provider [provider_name]"}
	ErrInvalidProviderName = &CommandError{Type: "validation", Message: "Invalid provider name. Available: openai, openrouter, anthropic, ollama"}
	ErrInvalidModelArgs    = &CommandError{Type: "validation", Message: "Invalid model command. Usage: /model [model_name]"}
	ErrEmptyModelName      = &CommandError{Type: "validation", Message: "Model name cannot be empty"}
	ErrCommandNotSupported = &CommandError{Type: "unsupported", Message: "Command not supported"}
)

// CommandError represents a command-specific error
type CommandError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *CommandError) Error() string {
	return e.Message
}

// CommandHandler interface for handling different command types
type CommandHandler interface {
	HandleCommand(cmd *Command) error
}

// FormatProviderList formats a list of providers for display
func FormatProviderList(providers map[string]ProviderStatus, currentProvider string) string {
	var lines []string
	lines = append(lines, "Available providers:")

	for name, status := range providers {
		indicator := "✗"
		statusText := "not configured"

		if status.Configured {
			indicator = "✓"
			statusText = "configured"
		}

		current := ""
		if name == currentProvider {
			current = " - Current"
		}

		line := "  " + indicator + " " + name + " (" + statusText + ")" + current
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// FormatModelList formats a list of models for display
func FormatModelList(models []ai.ModelInfo, currentModel string, limit int) string {
	var lines []string
	lines = append(lines, "Available models:")

	displayed := 0
	for _, model := range models {
		if limit > 0 && displayed >= limit {
			remaining := len(models) - displayed
			lines = append(lines, "  ... and "+string(rune(remaining))+" more models")
			break
		}

		current := ""
		if model.ID == currentModel || model.Name == currentModel {
			current = " - Current"
		}

		pricing := ""
		if model.Pricing != "" {
			pricing = " (" + model.Pricing + ")"
		}

		line := "  " + model.ID + pricing + current
		if model.Description != "" && model.Description != model.ID {
			line += " - " + model.Description
		}

		lines = append(lines, line)
		displayed++
	}

	return strings.Join(lines, "\n")
}

// ProviderStatus represents the configuration status of a provider
type ProviderStatus struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Current    bool   `json:"current"`
	Available  bool   `json:"available"`
}

// FormatStatusInfo formats current configuration status
func FormatStatusInfo(currentProvider, currentModel string, providerInfo map[string]interface{}) string {
	var lines []string
	lines = append(lines, "Current Configuration:")
	lines = append(lines, "  Provider: "+currentProvider)
	lines = append(lines, "  Model: "+currentModel)

	if configured, ok := providerInfo["configured"].(bool); ok && configured {
		lines = append(lines, "  Status: ✅ Ready")
	} else {
		lines = append(lines, "  Status: ❌ Not configured")
	}

	if timeout, ok := providerInfo["timeout"].(string); ok {
		lines = append(lines, "  Timeout: "+timeout)
	}

	if fallbackMode, ok := providerInfo["fallback_mode"].(bool); ok && fallbackMode {
		lines = append(lines, "  Fallback Mode: Enabled")
	}

	return strings.Join(lines, "\n")
}
