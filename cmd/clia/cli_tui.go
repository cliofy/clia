package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/executor"
	"github.com/yourusername/clia/pkg/utils"
)

// CLITUIState represents the current state of the CLI TUI
type CLITUIState int

const (
	StateSelecting  CLITUIState = iota // Selecting from suggestions
	StateEditing                       // Editing the selected command
	StateCompleting                    // Path completion mode
	StateExecuting                     // Executing command
	StateCompleted                     // Command completed, ready to exit
)

// CLITUIModel represents the CLI-specific TUI model
type CLITUIModel struct {
	// State management
	state        CLITUIState
	userRequest  string
	suggestions  []ai.CommandSuggestion
	
	// Selection state
	selectedIndex int
	
	// Editing state
	input         textinput.Model
	editingCommand string
	
	// Path completion state
	completionCandidates []string                      // List of completion candidates
	completionIndex      int                           // Currently selected completion index
	completionContext    *utils.PathCompletionContext // Context for current completion
	inCompletionMode     bool                          // Whether we're in completion mode
	
	// Execution state
	executor      *executor.Executor
	executing     bool
	executionOutput []string
	commandResult *executor.ExecutionResult
	
	// UI state
	width  int
	height int
	ready  bool
}

// NewCLITUIModel creates a new CLI TUI model
func NewCLITUIModel(userRequest string, suggestions []ai.CommandSuggestion) CLITUIModel {
	// Initialize text input for editing
	input := textinput.New()
	input.CharLimit = 500
	input.Width = 80
	
	return CLITUIModel{
		state:         StateSelecting,
		userRequest:   userRequest,
		suggestions:   suggestions,
		selectedIndex: 0,
		input:         input,
		// Initialize path completion state
		completionCandidates: []string{},
		completionIndex:      0,
		completionContext:    nil,
		inCompletionMode:     false,
		// Initialize execution state
		executor:        executor.New(),
		executionOutput: []string{},
	}
}

// Init initializes the CLI TUI model
func (m CLITUIModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state
func (m CLITUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		
	case tea.KeyMsg:
		switch m.state {
		case StateSelecting:
			return m.updateSelecting(msg)
		case StateEditing:
			return m.updateEditing(msg)
		case StateCompleting:
			return m.updateCompleting(msg)
		case StateExecuting:
			return m.updateExecuting(msg)
		case StateCompleted:
			return m.updateCompleted(msg)
		}
		
	case commandCompleteMsg:
		m.executing = false
		m.commandResult = &msg.result
		m.state = StateCompleted
		return m, nil
	}
	
	return m, nil
}

// updateSelecting handles updates in selection state
func (m CLITUIModel) updateSelecting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case "down", "j":
		if m.selectedIndex < len(m.suggestions)-1 {
			m.selectedIndex++
		}
	case "enter":
		// Move to editing state with selected command
		if len(m.suggestions) > 0 {
			selectedSuggestion := m.suggestions[m.selectedIndex]
			m.editingCommand = selectedSuggestion.Command
			m.input.SetValue(selectedSuggestion.Command)
			m.input.Focus()
			m.state = StateEditing
		}
	case "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateEditing handles updates in editing state
func (m CLITUIModel) updateEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute the edited command
		command := m.input.Value()
		if command != "" {
			m.state = StateExecuting
			m.executing = true
			m.executionOutput = []string{}
			return m, m.executeCommand(command)
		}
	case "tab":
		// Trigger path completion
		return m.handleTabCompletion()
	case "esc":
		// Go back to selection
		m.state = StateSelecting
		m.input.Blur()
	case "ctrl+c":
		return m, tea.Quit
	default:
		// Update text input
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateExecuting handles updates in executing state
func (m CLITUIModel) updateExecuting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateCompleted handles updates in completed state
func (m CLITUIModel) updateCompleted(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key exits the program
	_ = msg // ignore the specific key pressed
	return m, tea.Quit
}

// executeCommand executes a command and returns the result
func (m CLITUIModel) executeCommand(command string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		result, err := m.executor.Execute(ctx, command)
		
		return commandCompleteMsg{
			command: command,
			result:  *result,
			error:   err,
		}
	})
}

// View renders the CLI-style interface
func (m CLITUIModel) View() string {
	switch m.state {
	case StateSelecting:
		return m.viewSelecting()
	case StateEditing:
		return m.viewEditing()
	case StateCompleting:
		return m.viewCompleting()
	case StateExecuting:
		return m.viewExecuting()
	case StateCompleted:
		return m.viewCompleted()
	default:
		return ""
	}
}

// viewSelecting renders the CLI-style suggestion selection interface
func (m CLITUIModel) viewSelecting() string {
	header := fmt.Sprintf("🤖 Processing: %s\n\nSelect a command:\n\n", m.userRequest)
	
	var choices strings.Builder
	for i, suggestion := range m.suggestions {
		checkbox := "[ ]"
		if i == m.selectedIndex {
			checkbox = "[●]"
		}
		
		// Safety indicator
		safetyIcon := "✅"
		if !suggestion.Safe {
			safetyIcon = "⚠️"
		}
		
		confidencePercent := int(suggestion.Confidence * 100)
		choice := fmt.Sprintf("%s %s %s (%d%%)\n    %s\n", 
			checkbox, safetyIcon, suggestion.Command, confidencePercent, 
			subtleStyle.Render(suggestion.Description))
		
		choices.WriteString(choice)
	}
	
	footer := "\n" + subtleStyle.Render("↑/↓, j/k: select") + dotStyle + 
			subtleStyle.Render("enter: edit command") + dotStyle + 
			subtleStyle.Render("esc: quit") + "\n"
	
	return header + choices.String() + footer
}

// viewEditing renders the CLI-style command editing interface
func (m CLITUIModel) viewEditing() string {
	header := "✏️  Edit command (press Enter to execute, Tab for path completion):\n\n"
	
	inputLine := fmt.Sprintf("$ %s", m.input.View())
	
	footer := "\n" + subtleStyle.Render("enter: execute") + dotStyle + 
			subtleStyle.Render("tab: complete") + dotStyle + 
			subtleStyle.Render("esc: back") + "\n"
	
	return header + inputLine + footer
}

// viewExecuting renders the CLI-style command execution interface
func (m CLITUIModel) viewExecuting() string {
	return "🚀 Executing command...\n"
}

// viewCompleted renders the command completion with raw output
func (m CLITUIModel) viewCompleted() string {
	if m.commandResult == nil {
		return "No result available.\nPress any key to exit.\n"
	}
	
	result := m.commandResult
	var content strings.Builder
	
	// Show raw stdout (completely unformatted)
	if result.Stdout != "" {
		content.WriteString(result.Stdout)
		if !strings.HasSuffix(result.Stdout, "\n") {
			content.WriteString("\n")
		}
	}
	
	// Show raw stderr (completely unformatted)
	if result.Stderr != "" {
		content.WriteString(result.Stderr)
		if !strings.HasSuffix(result.Stderr, "\n") {
			content.WriteString("\n")
		}
	}
	
	// Simple completion indicator
	if result.ExitCode == 0 {
		content.WriteString(fmt.Sprintf("\n[Completed in %.2fs]\n", result.Duration.Seconds()))
	} else {
		content.WriteString(fmt.Sprintf("\n[Failed with exit code %d in %.2fs]\n", result.ExitCode, result.Duration.Seconds()))
	}
	
	content.WriteString("Press any key to exit.\n")
	
	return content.String()
}

// Message types for CLI TUI
type commandCompleteMsg struct {
	command string
	result  executor.ExecutionResult
	error   error
}

// updateCompleting handles updates in path completion state
func (m CLITUIModel) updateCompleting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Cycle to next completion candidate
		m.cycleCompletion()
		return m, nil
	case "enter":
		// Apply selected completion and return to editing
		m.applySelectedCompletion()
		m.exitCompletionMode()
		return m, nil
	case "esc":
		// Exit completion mode without applying
		m.exitCompletionMode()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	default:
		// Any other key exits completion mode and handles the key in edit mode
		m.exitCompletionMode()
		return m.updateEditing(msg)
	}
}

// handleTabCompletion triggers path completion
func (m CLITUIModel) handleTabCompletion() (CLITUIModel, tea.Cmd) {
	command := m.input.Value()
	cursorPos := m.input.Position()
	
	// Extract path context
	context, err := utils.ExtractPathContext(command, cursorPos)
	if err != nil || context == nil {
		// No valid path context, ignore tab
		return m, nil
	}
	
	// Get completion candidates
	candidates, err := utils.ScanDirectoryForCompletion(context.Directory, context.Prefix)
	if err != nil || len(candidates) == 0 {
		// No completions available
		return m, nil
	}
	
	// Handle single candidate - auto complete
	if len(candidates) == 1 {
		// Apply completion directly
		newCommand, newPos := utils.ApplyCompletion(command, candidates[0], context.StartPos, context.EndPos)
		m.input.SetValue(newCommand)
		m.input.SetCursor(newPos)
		return m, nil
	}
	
	// Multiple candidates - enter completion mode
	m.completionCandidates = candidates
	m.completionIndex = 0
	m.completionContext = context
	m.inCompletionMode = true
	m.state = StateCompleting
	
	return m, nil
}

// cycleCompletion cycles through completion candidates
func (m *CLITUIModel) cycleCompletion() {
	if len(m.completionCandidates) == 0 {
		return
	}
	
	m.completionIndex = (m.completionIndex + 1) % len(m.completionCandidates)
}

// applySelectedCompletion applies the currently selected completion
func (m *CLITUIModel) applySelectedCompletion() {
	if len(m.completionCandidates) == 0 || m.completionContext == nil {
		return
	}
	
	selectedCompletion := m.completionCandidates[m.completionIndex]
	command := m.input.Value()
	
	newCommand, newPos := utils.ApplyCompletion(
		command,
		selectedCompletion,
		m.completionContext.StartPos,
		m.completionContext.EndPos,
	)
	
	m.input.SetValue(newCommand)
	m.input.SetCursor(newPos)
}

// exitCompletionMode exits path completion mode and returns to editing
func (m *CLITUIModel) exitCompletionMode() {
	m.state = StateEditing
	m.inCompletionMode = false
	m.completionCandidates = []string{}
	m.completionIndex = 0
	m.completionContext = nil
}

// viewCompleting renders the path completion interface
func (m CLITUIModel) viewCompleting() string {
	if !m.inCompletionMode || len(m.completionCandidates) == 0 {
		return m.viewEditing()
	}
	
	// Header
	header := "✏️  Edit command (tab to cycle, enter to select):\n\n"
	
	// Input line with command
	inputLine := fmt.Sprintf("$ %s\n\n", m.input.View())
	
	// Completion candidates
	candidatesHeader := "Path completions:\n"
	var candidates strings.Builder
	
	for i, candidate := range m.completionCandidates {
		checkbox := "[ ]"
		if i == m.completionIndex {
			checkbox = "[●]"
		}
		
		displayName := utils.GetCompletionDisplayName(candidate)
		candidates.WriteString(fmt.Sprintf("%s %s\n", checkbox, displayName))
	}
	
	// Footer with help
	footer := "\n" + subtleStyle.Render("tab: cycle") + dotStyle + 
		subtleStyle.Render("enter: select") + dotStyle + 
		subtleStyle.Render("esc: cancel") + "\n"
	
	return header + inputLine + candidatesHeader + candidates.String() + footer
}

// Styles for CLI-style interface (inspired by bubbletea example)
var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dotStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(" • ")
)