package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/executor"
	"github.com/yourusername/clia/pkg/memory"
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

// Command selection messages

// commandSelectionMsg represents a command selection by number
type commandSelectionMsg struct {
	index int // 0-based index (user inputs 1-9, we convert to 0-8)
}

// CommandSelectionCmd returns a command to select a suggestion by index
func CommandSelectionCmd(index int) tea.Cmd {
	return func() tea.Msg {
		return commandSelectionMsg{
			index: index,
		}
	}
}

// commandExecutionMsg represents a command execution request
type commandExecutionMsg struct {
	command     string
	description string
	safe        bool
	confidence  float64
}

// CommandExecutionCmd returns a command to execute a selected command
func CommandExecutionCmd(command, description string, safe bool, confidence float64) tea.Cmd {
	return func() tea.Msg {
		return commandExecutionMsg{
			command:     command,
			description: description,
			safe:        safe,
			confidence:  confidence,
		}
	}
}

// Confirmation dialog messages

// confirmationRequestMsg represents a confirmation request for a command
type confirmationRequestMsg struct {
	command     string
	description string
	reason      string // Why confirmation is needed
}

// ConfirmationRequestCmd returns a command to request confirmation
func ConfirmationRequestCmd(command, description, reason string) tea.Cmd {
	return func() tea.Msg {
		return confirmationRequestMsg{
			command:     command,
			description: description,
			reason:      reason,
		}
	}
}

// confirmationResponseMsg represents user's response to confirmation dialog
type confirmationResponseMsg struct {
	confirmed bool
	command   commandExecutionMsg // The command that was confirmed/rejected
}

// ConfirmationResponseCmd returns a command with confirmation response
func ConfirmationResponseCmd(confirmed bool, command commandExecutionMsg) tea.Cmd {
	return func() tea.Msg {
		return confirmationResponseMsg{
			confirmed: confirmed,
			command:   command,
		}
	}
}

// Command execution lifecycle messages

// commandStartMsg represents the start of command execution
type commandStartMsg struct {
	command     string
	pid         int
	description string
}

// CommandStartCmd returns a command indicating command execution has started
func CommandStartCmd(command string, pid int, description string) tea.Cmd {
	return func() tea.Msg {
		return commandStartMsg{
			command:     command,
			pid:         pid,
			description: description,
		}
	}
}

// commandOutputMsg represents output from a running command
type commandOutputMsg struct {
	content   string
	isStderr  bool
	timestamp time.Time
}

// CommandOutputCmd returns a command with command output
func CommandOutputCmd(content string, isStderr bool) tea.Cmd {
	return func() tea.Msg {
		return commandOutputMsg{
			content:   content,
			isStderr:  isStderr,
			timestamp: time.Now(),
		}
	}
}

// commandCompleteMsg represents command execution completion
type commandCompleteMsg struct {
	command  string
	exitCode int
	duration time.Duration
	stdout   string
	stderr   string
	error    error
}

// CommandCompleteCmd returns a command indicating command execution is complete
func CommandCompleteCmd(command string, exitCode int, duration time.Duration, stdout, stderr string, err error) tea.Cmd {
	return func() tea.Msg {
		return commandCompleteMsg{
			command:  command,
			exitCode: exitCode,
			duration: duration,
			stdout:   stdout,
			stderr:   stderr,
			error:    err,
		}
	}
}

// commandErrorMsg represents an error during command execution
type commandErrorMsg struct {
	command string
	error   error
}

// CommandErrorCmd returns a command with execution error
func CommandErrorCmd(command string, err error) tea.Cmd {
	return func() tea.Msg {
		return commandErrorMsg{
			command: command,
			error:   err,
		}
	}
}

// Stream processing messages

// streamTickMsg represents a tick to check for new output
type streamTickMsg struct{}

// StreamTickCmd returns a command to check for stream output
func StreamTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return streamTickMsg{}
	})
}

// streamEndMsg represents the end of stream
type streamEndMsg struct {
	command  string
	exitCode int
	duration time.Duration
	error    error
}

// StreamEndCmd returns a command indicating stream has ended
func StreamEndCmd(command string, exitCode int, duration time.Duration, err error) tea.Cmd {
	return func() tea.Msg {
		return streamEndMsg{
			command:  command,
			exitCode: exitCode,
			duration: duration,
			error:    err,
		}
	}
}

// commandStreamStartMsg represents the start of a command stream
type commandStreamStartMsg struct {
	command     string
	description string
	stream      <-chan executor.OutputLine
}

// CommandStreamStartCmd returns a command to start a command stream
func CommandStreamStartCmd(command, description string, stream <-chan executor.OutputLine) tea.Cmd {
	return func() tea.Msg {
		return commandStreamStartMsg{
			command:     command,
			description: description,
			stream:      stream,
		}
	}
}

// Memory-related messages

// memorySearchMsg represents a memory search request
type memorySearchMsg struct {
	query string
}

// MemorySearchCmd returns a command to search memory
func MemorySearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		return memorySearchMsg{
			query: query,
		}
	}
}

// memoryResultsMsg represents memory search results
type memoryResultsMsg struct {
	query   string
	results []memory.SearchResult
	error   error
}

// MemoryResultsCmd returns a command with memory search results
func MemoryResultsCmd(query string, results []memory.SearchResult, err error) tea.Cmd {
	return func() tea.Msg {
		return memoryResultsMsg{
			query:   query,
			results: results,
			error:   err,
		}
	}
}

// memorySuggestion represents a memory-based command suggestion
type memorySuggestion struct {
	Entry      memory.MemoryEntry
	Score      float64
	Reason     string
	MatchType  memory.MatchType
	UsageCount int
	LastUsed   time.Time
}

// memorySelectionMsg represents selection of a memory suggestion
type memorySelectionMsg struct {
	index int // 0-based index of selected memory suggestion
}

// MemorySelectionCmd returns a command to select a memory suggestion
func MemorySelectionCmd(index int) tea.Cmd {
	return func() tea.Msg {
		return memorySelectionMsg{
			index: index,
		}
	}
}

// memorySaveMsg represents a request to save a command to memory
type memorySaveMsg struct {
	userRequest     string
	selectedCommand string
	description     string
	source          string
	success         bool
}

// MemorySaveCmd returns a command to save a command to memory
func MemorySaveCmd(userRequest, selectedCommand, description, source string, success bool) tea.Cmd {
	return func() tea.Msg {
		return memorySaveMsg{
			userRequest:     userRequest,
			selectedCommand: selectedCommand,
			description:     description,
			source:          source,
			success:         success,
		}
	}
}

// memorySaveResultMsg represents the result of saving to memory
type memorySaveResultMsg struct {
	success bool
	error   error
}

// MemorySaveResultCmd returns a command with memory save result
func MemorySaveResultCmd(success bool, err error) tea.Cmd {
	return func() tea.Msg {
		return memorySaveResultMsg{
			success: success,
			error:   err,
		}
	}
}

// memoryStatsMsg represents memory statistics
type memoryStatsMsg struct {
	stats map[string]interface{}
	error error
}

// MemoryStatsCmd returns a command with memory statistics
func MemoryStatsCmd(stats map[string]interface{}, err error) tea.Cmd {
	return func() tea.Msg {
		return memoryStatsMsg{
			stats: stats,
			error: err,
		}
	}
}

// memoryDeleteMsg represents a request to delete a memory entry
type memoryDeleteMsg struct {
	entryID string
}

// MemoryDeleteCmd returns a command to delete a memory entry
func MemoryDeleteCmd(entryID string) tea.Cmd {
	return func() tea.Msg {
		return memoryDeleteMsg{
			entryID: entryID,
		}
	}
}

// memoryDeleteResultMsg represents the result of deleting a memory entry
type memoryDeleteResultMsg struct {
	entryID string
	success bool
	error   error
}

// MemoryDeleteResultCmd returns a command with memory delete result
func MemoryDeleteResultCmd(entryID string, success bool, err error) tea.Cmd {
	return func() tea.Msg {
		return memoryDeleteResultMsg{
			entryID: entryID,
			success: success,
			error:   err,
		}
	}
}

// combinedSuggestionsMsg represents combined AI and memory suggestions
type combinedSuggestionsMsg struct {
	aiSuggestions     []aiSuggestion
	memorySuggestions []memorySuggestion
	userRequest       string
}

// CombinedSuggestionsCmd returns a command with combined suggestions
func CombinedSuggestionsCmd(userRequest string, aiSuggestions []aiSuggestion, memorySuggestions []memorySuggestion) tea.Cmd {
	return func() tea.Msg {
		return combinedSuggestionsMsg{
			userRequest:       userRequest,
			aiSuggestions:     aiSuggestions,
			memorySuggestions: memorySuggestions,
		}
	}
}

// PTY execution messages

// ptyExecutionRequestMsg represents a request to execute a command with PTY
type ptyExecutionRequestMsg struct {
	command     string
	description string
}

// PTYExecutionRequestCmd returns a command to execute with PTY
func PTYExecutionRequestCmd(command, description string) tea.Cmd {
	return func() tea.Msg {
		return ptyExecutionRequestMsg{
			command:     command,
			description: description,
		}
	}
}

// ptyExecutionCompleteMsg represents completion of PTY execution
type ptyExecutionCompleteMsg struct {
	command  string
	exitCode int
	duration time.Duration
	error    error
}

// PTYExecutionCompleteCmd returns a command indicating PTY execution completion
func PTYExecutionCompleteCmd(command string, exitCode int, duration time.Duration, err error) tea.Cmd {
	return func() tea.Msg {
		return ptyExecutionCompleteMsg{
			command:  command,
			exitCode: exitCode,
			duration: duration,
			error:    err,
		}
	}
}
