package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/config"
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
	
	// Command state
	commandMode     bool
	waitingAPIKey   bool
	apiKeyProvider  string
	
	// Configuration
	configManager *config.Manager
	
	// Current provider and model info
	currentProvider string
	currentModel    string
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

	// Initialize configuration manager
	configManager, err := config.NewManager()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize config manager: %v\n", err)
	}
	
	// Initialize AI service
	aiService := ai.NewService().SetFallbackMode(true)
	
	currentProvider := "none"
	currentModel := "none"
	
	var initErrors []string
	
	// Try to configure providers based on available API keys
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenRouter)
		config.APIKey = apiKey
		config.Model = "z-ai/glm-4.5-air:free" // Use user's preferred model
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenRouter, config); err == nil {
			currentProvider = "openrouter"
			currentModel = config.Model
		} else {
			initErrors = append(initErrors, fmt.Sprintf("Failed to configure OpenRouter: %v", err))
		}
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenAI)
		config.APIKey = apiKey
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenAI, config); err == nil {
			currentProvider = "openai"
			currentModel = config.Model
		} else {
			initErrors = append(initErrors, fmt.Sprintf("Failed to configure OpenAI: %v", err))
		}
	} else {
		initErrors = append(initErrors, "No API keys found. Set OPENROUTER_API_KEY or OPENAI_API_KEY in environment.")
	}
	
	// Create initial model
	model := Model{
		input:           ti,
		viewport:        vp,
		messages:        []Message{},
		status:          fmt.Sprintf("Ready - %s • %s", currentProvider, currentModel),
		aiService:       aiService,
		processing:      false,
		suggestions:     []aiSuggestion{},
		commandMode:     false,
		waitingAPIKey:   false,
		configManager:   configManager,
		currentProvider: currentProvider,
		currentModel:    currentModel,
	}

	// Add welcome message
	model.addMessage("Welcome to clia - Command Line Intelligent Assistant", MessageTypeSystem)
	model.addMessage(fmt.Sprintf("Version %s (%s)", version.Version, version.GoVersion), MessageTypeSystem)
	
	// Show initialization errors if any
	for _, err := range initErrors {
		model.addMessage("⚠️  "+err, MessageTypeError)
	}
	
	if currentProvider != "none" {
		model.addMessage(fmt.Sprintf("✅ Provider initialized: %s (model: %s)", currentProvider, currentModel), MessageTypeSystem)
	}
	
	model.addMessage("Type your natural language command and press Enter", MessageTypeSystem)
	model.addMessage("Commands: /provider, /model, /status, /help", MessageTypeSystem)
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

	// Handle API key input mode
	if m.waitingAPIKey {
		return m.handleAPIKeyInput(input)
	}

	// Check if input is a command
	if cmd := ParseCommand(input); cmd != nil {
		return m.handleCommand(cmd)
	}

	// Regular AI request processing
	return m.handleAIRequest(input)
}

// handleCommand processes slash commands
func (m *Model) handleCommand(cmd *Command) tea.Cmd {
	// Add user command to history
	m.addMessage(cmd.Raw, MessageTypeUser)
	m.input.SetValue("")

	switch cmd.Type {
	case CommandTypeHelp:
		return m.handleHelpCommand()
	case CommandTypeStatus:
		return m.handleStatusCommand()
	case CommandTypeProvider:
		return m.handleProviderCommand(cmd.Args)
	case CommandTypeModel:
		return m.handleModelCommand(cmd.Args)
	default:
		m.addMessage("Unknown command: "+cmd.Type+". Type /help for available commands.", MessageTypeError)
		return nil
	}
}

// handleAIRequest processes regular AI requests
func (m *Model) handleAIRequest(input string) tea.Cmd {
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

// handleHelpCommand shows help information
func (m *Model) handleHelpCommand() tea.Cmd {
	helpText := GetCommandHelp()
	m.addMessage(helpText, MessageTypeSystem)
	return nil
}

// handleStatusCommand shows current status
func (m *Model) handleStatusCommand() tea.Cmd {
	providerInfo := m.aiService.GetProviderInfo()
	statusText := FormatStatusInfo(m.currentProvider, m.currentModel, providerInfo)
	m.addMessage(statusText, MessageTypeSystem)
	return nil
}

// handleProviderCommand handles provider switching
func (m *Model) handleProviderCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		// List providers
		return tea.Cmd(func() tea.Msg {
			status := m.aiService.GetProviderStatus()
			
			// Convert to ProviderStatus map for display
			providerStatus := make(map[string]ProviderStatus)
			for providerType, statusInfo := range status {
				providerStatus[string(providerType)] = ProviderStatus{
					Name:       string(providerType),
					Configured: statusInfo.Configured,
					Current:    statusInfo.Current,
					Available:  statusInfo.Available,
				}
			}
			
			formatted := FormatProviderList(providerStatus, m.currentProvider)
			return addMessageMsg{Content: formatted, Type: MessageTypeSystem}
		})
	}
	
	// Switch provider
	providerName := args[0]
	return tea.Cmd(func() tea.Msg {
		// Check if API key is available
		providerType := ai.ProviderType(providerName)
		config := ai.DefaultProviderConfig(providerType)
		
		// Try to get API key from environment
		if m.configManager != nil {
			if apiKey := m.configManager.GetAPIKeyFromEnv(); apiKey != "" {
				config.APIKey = apiKey
			}
		}
		
		if config.APIKey == "" {
			// Need API key
			return apiKeyInputMsg{
				providerType: providerName,
				prompt:       fmt.Sprintf("🔑 %s API key not found. Please enter your API key:", providerName),
			}
		}
		
		// Try to switch provider
		err := m.aiService.SwitchProvider(providerType, config)
		if err != nil {
			return providerSwitchMsg{
				providerType: providerName,
				success:      false,
				error:        err,
			}
		}
		
		return providerSwitchMsg{
			providerType: providerName,
			success:      true,
		}
	})
}

// handleModelCommand handles model listing and switching
func (m *Model) handleModelCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		// List models
		return tea.Cmd(func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			models, err := m.aiService.GetAvailableModels(ctx)
			return modelListMsg{models: models, error: err}
		})
	}
	
	// Switch model
	modelName := args[0]
	return tea.Cmd(func() tea.Msg {
		err := m.aiService.SwitchModel(modelName)
		return modelSwitchMsg{
			modelName: modelName,
			success:   err == nil,
			error:     err,
		}
	})
}

// handleAPIKeyInput processes API key input
func (m *Model) handleAPIKeyInput(input string) tea.Cmd {
	apiKey := input
	providerType := m.apiKeyProvider
	
	m.input.SetValue("")
	m.waitingAPIKey = false
	m.input.EchoMode = textinput.EchoNormal
	m.input.Placeholder = "Type your command request here..."
	
	return tea.Cmd(func() tea.Msg {
		return apiKeySubmitMsg{
			providerType: providerType,
			apiKey:       apiKey,
		}
	})
}

// handleAIResponse handles AI response messages
func (m *Model) handleAIResponse(msg aiResponseMsg) {
	m.processing = false
	
	if msg.error != nil {
		// Handle AI error with more context
		errorMsg := fmt.Sprintf("❌ AI Request Failed: %s", msg.error.Error())
		m.addMessage(errorMsg, MessageTypeError)
		
		// Provide helpful suggestions based on error type
		if strings.Contains(msg.error.Error(), "not configured") || strings.Contains(msg.error.Error(), "API key") {
			m.addMessage("💡 Try: /provider openrouter (to configure provider with API key)", MessageTypeSystem)
		} else if strings.Contains(msg.error.Error(), "timeout") || strings.Contains(msg.error.Error(), "context deadline") {
			m.addMessage("💡 Network timeout - check your internet connection and try again", MessageTypeSystem)
		} else if strings.Contains(msg.error.Error(), "rate limit") {
			m.addMessage("💡 Rate limit exceeded - please wait a moment and try again", MessageTypeSystem)
		}
		
		m.status = fmt.Sprintf("Error - %s • %s", m.currentProvider, m.currentModel)
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