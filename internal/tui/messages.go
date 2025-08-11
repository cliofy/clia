package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/clia/internal/ai"
)

// Message represents a message in the chat history
type Message struct {
	Content string
	Type    MessageType
}

// MessageType represents the type of message
type MessageType int

const (
	MessageTypeUser MessageType = iota
	MessageTypeSystem
	MessageTypeAssistant
	MessageTypeError
)

// String returns the string representation of the message type
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeUser:
		return "user"
	case MessageTypeSystem:
		return "system"
	case MessageTypeAssistant:
		return "assistant"
	case MessageTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// clearHistoryMsg is a message to clear the chat history
type clearHistoryMsg struct{}

// ClearHistoryCmd returns a command to clear the history
func ClearHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		return clearHistoryMsg{}
	}
}

// addMessageMsg is a message to add a new message to history
type addMessageMsg Message

// AddMessageCmd returns a command to add a message
func AddMessageCmd(content string, msgType MessageType) tea.Cmd {
	return func() tea.Msg {
		return addMessageMsg{
			Content: content,
			Type:    msgType,
		}
	}
}

// aiRequestMsg represents an AI request being processed
type aiRequestMsg struct {
	input string
}

// AIRequestCmd returns a command to process an AI request
func AIRequestCmd(input string) tea.Cmd {
	return func() tea.Msg {
		return aiRequestMsg{input: input}
	}
}

// aiResponseMsg represents an AI response
type aiResponseMsg struct {
	suggestions []aiSuggestion
	error       error
}

// aiSuggestion represents a command suggestion from AI
type aiSuggestion struct {
	Command     string
	Description string
	Safe        bool
	Confidence  float64
}

// AIResponseCmd returns a command with AI response
func AIResponseCmd(suggestions []aiSuggestion, err error) tea.Cmd {
	return func() tea.Msg {
		return aiResponseMsg{
			suggestions: suggestions,
			error:       err,
		}
	}
}

// aiProcessingMsg indicates AI is processing
type aiProcessingMsg struct{}

// AIProcessingCmd returns a processing indicator command
func AIProcessingCmd() tea.Cmd {
	return func() tea.Msg {
		return aiProcessingMsg{}
	}
}

// Command-related messages

// commandMsg represents a command being processed
type commandMsg struct {
	command string
	args    []string
	raw     string
}

// CommandCmd returns a command to process a command
func CommandCmd(command string, args []string, raw string) tea.Cmd {
	return func() tea.Msg {
		return commandMsg{
			command: command,
			args:    args,
			raw:     raw,
		}
	}
}

// providerSwitchMsg represents a provider switch operation
type providerSwitchMsg struct {
	providerType string
	success      bool
	error        error
	needsAPIKey  bool
}

// ProviderSwitchCmd returns a command to switch providers
func ProviderSwitchCmd(providerType string, success bool, err error, needsAPIKey bool) tea.Cmd {
	return func() tea.Msg {
		return providerSwitchMsg{
			providerType: providerType,
			success:      success,
			error:        err,
			needsAPIKey:  needsAPIKey,
		}
	}
}

// modelListMsg represents model list results
type modelListMsg struct {
	models []ai.ModelInfo
	error  error
}

// ModelListCmd returns a command to fetch model list
func ModelListCmd(models []ai.ModelInfo, err error) tea.Cmd {
	return func() tea.Msg {
		return modelListMsg{
			models: models,
			error:  err,
		}
	}
}

// modelSwitchMsg represents a model switch operation
type modelSwitchMsg struct {
	modelName string
	success   bool
	error     error
}

// ModelSwitchCmd returns a command to switch models
func ModelSwitchCmd(modelName string, success bool, err error) tea.Cmd {
	return func() tea.Msg {
		return modelSwitchMsg{
			modelName: modelName,
			success:   success,
			error:     err,
		}
	}
}

// apiKeyInputMsg represents API key input request
type apiKeyInputMsg struct {
	providerType string
	prompt       string
}

// APIKeyInputCmd returns a command to request API key input
func APIKeyInputCmd(providerType, prompt string) tea.Cmd {
	return func() tea.Msg {
		return apiKeyInputMsg{
			providerType: providerType,
			prompt:       prompt,
		}
	}
}

// apiKeySubmitMsg represents API key submission
type apiKeySubmitMsg struct {
	providerType string
	apiKey       string
}

// APIKeySubmitCmd returns a command to submit API key
func APIKeySubmitCmd(providerType, apiKey string) tea.Cmd {
	return func() tea.Msg {
		return apiKeySubmitMsg{
			providerType: providerType,
			apiKey:       apiKey,
		}
	}
}