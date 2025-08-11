package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/version"
)

// Model represents the main TUI model
type Model struct {
	// UI Components
	input    textinput.Model
	viewport viewport.Model
	
	// Application state
	messages []Message
	ready    bool
	width    int
	height   int
	
	// Status information
	status string
}

// New creates a new TUI model
func New() Model {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Type your command request here..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 50

	// Initialize viewport
	vp := viewport.New(80, 20)
	vp.SetContent("")

	// Create initial model
	model := Model{
		input:    ti,
		viewport: vp,
		messages: []Message{},
		status:   "Ready",
	}

	// Add welcome message
	model.addMessage("Welcome to clia - Command Line Intelligent Assistant", MessageTypeSystem)
	model.addMessage(fmt.Sprintf("Version %s (%s)", version.Version, version.GoVersion), MessageTypeSystem)
	model.addMessage("Type your natural language command and press Enter", MessageTypeSystem)
	model.addMessage("Shortcuts: Ctrl+C (quit), Ctrl+L (clear history)", MessageTypeSystem)

	return model
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// addMessage adds a message to the history
func (m *Model) addMessage(content string, msgType MessageType) {
	msg := Message{
		Content: content,
		Type:    msgType,
	}
	m.messages = append(m.messages, msg)
	m.updateViewportContent()
}

// clearMessages clears all messages from history
func (m *Model) clearMessages() {
	m.messages = []Message{}
	m.addMessage("History cleared", MessageTypeSystem)
}

// updateViewportContent updates the viewport with current messages
func (m *Model) updateViewportContent() {
	var content string
	for i, msg := range m.messages {
		if i > 0 {
			content += "\n"
		}
		content += FormatMessage(msg)
	}
	m.viewport.SetContent(content)
	// Scroll to bottom
	m.viewport.GotoBottom()
}

// handleWindowSizeMsg handles terminal window resize
func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true

	// Calculate dimensions for components
	headerHeight := 3  // Status bar + borders
	footerHeight := 3  // Input area + borders  
	contentHeight := msg.Height - headerHeight - footerHeight

	// Update viewport size
	m.viewport.Width = msg.Width - 4  // Account for padding
	m.viewport.Height = contentHeight
	
	// Update input width
	m.input.Width = msg.Width - 8  // Account for borders and padding

	// Refresh content
	m.updateViewportContent()
}

// handleInputSubmit processes user input submission
func (m *Model) handleInputSubmit() tea.Cmd {
	input := m.input.Value()
	if input == "" {
		return nil
	}

	// Add user message to history
	m.addMessage(input, MessageTypeUser)

	// Clear input
	m.input.SetValue("")

	// For now, just echo the input as a system response
	// This will be replaced with actual LLM integration in Phase 2
	m.status = "Processing..."
	
	return tea.Sequence(
		tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return echoResponseMsg{input}
		}),
	)
}

// echoResponseMsg represents a mock response for Phase 1
type echoResponseMsg struct {
	input string
}

// handleEchoResponse handles the mock echo response
func (m *Model) handleEchoResponse(msg echoResponseMsg) {
	response := fmt.Sprintf("Echo: %s (Phase 2 will add LLM processing)", msg.input)
	m.addMessage(response, MessageTypeAssistant)
	m.status = "Ready"
}