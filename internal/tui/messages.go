package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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