package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/ai"
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
	
	// AI service
	aiService    *ai.Service
	processing   bool
	suggestions  []aiSuggestion
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

	// Initialize AI service
	aiService := ai.NewService().SetFallbackMode(true)
	
	// Try to configure OpenAI provider if API key is available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenAI)
		config.APIKey = apiKey
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenAI, config); err != nil {
			// Log error but continue without AI (will use fallback)
			fmt.Printf("Warning: Failed to configure AI provider: %v\n", err)
		}
	}
	
	// Create initial model
	model := Model{
		input:       ti,
		viewport:    vp,
		messages:    []Message{},
		status:      "Ready",
		aiService:   aiService,
		processing:  false,
		suggestions: []aiSuggestion{},
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

	// Clear input and set processing state
	m.input.SetValue("")
	m.processing = true
	m.status = "Processing..."

	// Request AI suggestions
	return tea.Sequence(
		AIProcessingCmd(),
		tea.Cmd(func() tea.Msg {
			// Run AI request in background
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			response, err := m.aiService.SuggestCommands(ctx, input)
			if err != nil {
				return aiResponseMsg{error: err}
			}
			
			// Convert AI suggestions to TUI suggestions
			var suggestions []aiSuggestion
			for _, cmd := range response.Suggestions {
				suggestions = append(suggestions, aiSuggestion{
					Command:     cmd.Command,
					Description: cmd.Description,
					Safe:        cmd.Safe,
					Confidence:  cmd.Confidence,
				})
			}
			
			return aiResponseMsg{suggestions: suggestions}
		}),
	)
}

// handleAIResponse handles AI response messages
func (m *Model) handleAIResponse(msg aiResponseMsg) {
	m.processing = false
	
	if msg.error != nil {
		// Handle AI error
		errorMsg := fmt.Sprintf("AI Error: %s", msg.error.Error())
		m.addMessage(errorMsg, MessageTypeError)
		m.status = "Error"
		return
	}
	
	if len(msg.suggestions) == 0 {
		m.addMessage("No command suggestions available", MessageTypeSystem)
		m.status = "Ready"
		return
	}
	
	// Store suggestions for potential selection
	m.suggestions = msg.suggestions
	
	// Display suggestions
	for i, suggestion := range msg.suggestions {
		safetyIndicator := "✓"
		if !suggestion.Safe {
			safetyIndicator = "⚠"
		}
		
		confidencePercent := int(suggestion.Confidence * 100)
		suggestionText := fmt.Sprintf("%d. %s %s (%d%% confidence)\n   %s", 
			i+1, safetyIndicator, suggestion.Command, confidencePercent, suggestion.Description)
		
		m.addMessage(suggestionText, MessageTypeAssistant)
	}
	
	// Add instruction message
	m.addMessage("Use 1-9 to select a command, or type a new request", MessageTypeSystem)
	m.status = "Ready"
}