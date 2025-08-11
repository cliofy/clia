package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI interface
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render status bar
	statusBar := m.renderStatusBar()
	
	// Render content area
	content := m.renderContent()
	
	// Render input area
	inputArea := m.renderInputArea()
	
	// Render help text
	help := m.renderHelp()

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		content,
		inputArea,
		help,
	)
}

// renderStatusBar renders the top status bar
func (m Model) renderStatusBar() string {
	// Left side: application name and status
	leftStatus := statusStyle.Render(fmt.Sprintf("clia • %s", m.status))
	
	// Right side: message count and dimensions
	rightStatus := encodingStyle.Render(fmt.Sprintf("Messages: %d | %dx%d", 
		len(m.messages), m.width, m.height))
	
	// Calculate spacing
	statusBarWidth := m.width
	usedWidth := lipgloss.Width(leftStatus) + lipgloss.Width(rightStatus)
	spacer := strings.Repeat(" ", max(0, statusBarWidth-usedWidth))
	
	return statusBarStyle.Render(leftStatus + spacer + rightStatus)
}

// renderContent renders the main content area with message history
func (m Model) renderContent() string {
	content := contentStyle.Render(m.viewport.View())
	
	// Apply border and styling
	return baseStyle.
		Width(m.width - 2).
		Height(m.viewport.Height + 2).
		Render(content)
}

// renderInputArea renders the input field
func (m Model) renderInputArea() string {
	// Determine input style based on focus
	style := inputStyle
	if m.input.Focused() {
		style = focusedInputStyle
	}
	
	// Render the input with label
	inputLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Render("❯ ")
	
	inputField := m.input.View()
	
	inputContent := inputLabel + inputField
	
	return style.
		Width(m.width - 2).
		Render(inputContent)
}

// renderHelp renders the help text at the bottom
func (m Model) renderHelp() string {
	helpText := "Press Ctrl+C to quit • Ctrl+L to clear history • Enter to submit"
	return helpStyle.
		Width(m.width).
		Render(helpText)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}