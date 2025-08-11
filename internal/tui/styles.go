package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Style definitions for the TUI
var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("253")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Align(lipgloss.Left).
			Width(20)

	encodingStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("174")).
			Align(lipgloss.Right)

	// Content area styles
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Message styles
	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)

	systemMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)

	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117"))

	errorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("204")).
				Bold(true)

	// Input styles
	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("57")).
			Padding(0, 1)

	focusedInputStyle = inputStyle.BorderForeground(lipgloss.Color("69"))

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			Margin(0, 2)
)

// GetMessageStyle returns the appropriate style for a message type
func GetMessageStyle(msgType MessageType) lipgloss.Style {
	switch msgType {
	case MessageTypeUser:
		return userMessageStyle
	case MessageTypeSystem:
		return systemMessageStyle
	case MessageTypeAssistant:
		return assistantMessageStyle
	case MessageTypeError:
		return errorMessageStyle
	default:
		return systemMessageStyle
	}
}

// FormatMessage formats a message with the appropriate style and prefix
func FormatMessage(msg Message) string {
	style := GetMessageStyle(msg.Type)
	
	var prefix string
	switch msg.Type {
	case MessageTypeUser:
		prefix = "❯ "
	case MessageTypeSystem:
		prefix = "⚠ "
	case MessageTypeAssistant:
		prefix = "🤖 "
	case MessageTypeError:
		prefix = "✗ "
	default:
		prefix = "• "
	}
	
	return style.Render(prefix + msg.Content)
}