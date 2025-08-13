package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/config"
	"github.com/yourusername/clia/internal/executor"
	"github.com/yourusername/clia/internal/version"
	"github.com/yourusername/clia/pkg/memory"
	"github.com/yourusername/clia/pkg/utils"
)

// executionResult represents a command execution result for TUI display
type executionResult struct {
	Command  string
	ExitCode int
	Duration time.Duration
	Error    error
}

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
	
	// Command executor
	executor *executor.Executor
	
	// Animation state
	spinner         Spinner
	animationTicker int
	showSpinner     bool
	pulseFrame      int
	thinkingDots    string
	
	// Command state
	commandMode     bool
	waitingAPIKey   bool
	apiKeyProvider  string
	
	// Selection state
	inSelectionMode bool
	availableSuggestions []aiSuggestion
	lastSelectedIndex    int
	
	// Confirmation dialog state
	inConfirmationMode bool
	pendingCommand     commandExecutionMsg
	
	// Edit mode state
	inEditMode         bool
	editingCommand     string
	editingDescription string
	editingSafe        bool
	editingConfidence  float64
	originalCommand    string
	
	// Command execution state
	executingCommand bool
	currentCommand   string
	currentPID       int
	executionOutput  []string
	executionResult  *executionResult
	outputStream     <-chan executor.OutputLine
	streamActive     bool
	
	// Configuration
	configManager *config.Manager
	
	// Current provider and model info
	currentProvider string
	currentModel    string

	// Memory management
	memoryManager       *memory.Manager
	memorySuggestions   []memorySuggestion
	combinedSuggestions []interface{} // Mix of aiSuggestion and memorySuggestion
	lastUserRequest     string         // Store for memory saving
	memoryEnabled       bool          // Whether memory is functional
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
	
	// Initialize executor
	cmdExecutor := executor.New()
	
	// Initialize memory manager
	memoryManager, memoryErr := memory.NewManager()
	memoryEnabled := memoryErr == nil
	if memoryErr != nil {
		fmt.Printf("Warning: Failed to initialize memory manager: %v\n", memoryErr)
	}
	
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
		executor:        cmdExecutor,
		commandMode:     false,
		waitingAPIKey:   false,
		configManager:   configManager,
		currentProvider: currentProvider,
		currentModel:    currentModel,
		// Animation state
		spinner:         ProcessingSpinner,
		animationTicker: 0,
		showSpinner:     false,
		pulseFrame:      0,
		thinkingDots:    "",
		// Selection state
		inSelectionMode:      false,
		availableSuggestions: []aiSuggestion{},
		lastSelectedIndex:    -1,
		// Confirmation state
		inConfirmationMode:   false,
		pendingCommand:       commandExecutionMsg{},
		// Edit mode state
		inEditMode:         false,
		editingCommand:     "",
		editingDescription: "",
		editingSafe:        true,
		editingConfidence:  0.0,
		originalCommand:    "",
		// Execution state
		executingCommand:     false,
		currentCommand:       "",
		currentPID:           0,
		executionOutput:      []string{},
		executionResult:      nil,
		outputStream:         nil,
		streamActive:         false,
		// Memory state
		memoryManager:       memoryManager,
		memorySuggestions:   []memorySuggestion{},
		combinedSuggestions: []interface{}{},
		lastUserRequest:     "",
		memoryEnabled:       memoryEnabled,
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
	
	// Show memory status
	if memoryEnabled {
		if memoryManager != nil {
			stats := memoryManager.GetStats()
			totalEntries := stats["total_entries"].(int)
			model.addMessage(fmt.Sprintf("💭 Memory initialized: %d stored commands", totalEntries), MessageTypeSystem)
		}
	} else {
		model.addMessage("⚠️  Memory disabled due to initialization error", MessageTypeError)
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

// removeLastMessage removes the last message (used for removing thinking bubble)
func (m *Model) removeLastMessage() {
	if len(m.messages) > 0 {
		m.messages = m.messages[:len(m.messages)-1]
		m.updateViewportContent()
	}
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

	// Handle edit mode input
	if m.inEditMode {
		return m.handleEditModeInput()
	}

	// Handle API key input mode
	if m.waitingAPIKey {
		return m.handleAPIKeyInput(input)
	}

	// Handle direct command execution (starting with '!')
	if strings.HasPrefix(input, "!") {
		return m.handleDirectCommand(input)
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

// handleAIRequest processes regular AI requests with memory integration
func (m *Model) handleAIRequest(input string) tea.Cmd {
	// Add user message to history
	m.addMessage(input, MessageTypeUser)
	
	// Store the user request for later memory saving
	m.lastUserRequest = input
	
	// Clear input and set processing state
	m.input.SetValue("")
	m.processing = true
	m.showSpinner = true
	m.spinner = m.spinner.Reset() // Reset spinner to start fresh
	m.thinkingDots = ""
	m.status = "Processing..."

	// Search memory first if enabled
	var memoryCmd tea.Cmd
	if m.memoryEnabled && m.memoryManager != nil {
		memoryCmd = tea.Cmd(func() tea.Msg {
			options := memory.DefaultSearchOptions()
			options.MaxResults = 3 // Limit memory suggestions
			
			results, err := m.memoryManager.Search(input, options)
			if err != nil {
				// If memory search fails, just log and continue
				log.Printf("Memory search failed: %v", err)
				return memoryResultsMsg{query: input, results: []memory.SearchResult{}, error: err}
			}
			
			return memoryResultsMsg{query: input, results: results, error: nil}
		})
	}
	
	// AI request command
	aiCmd := tea.Cmd(func() tea.Msg {
		// Add thinking bubble message
		return addMessageMsg{Content: "🤖 思考中...", Type: MessageTypeSystem}
	})
	
	// Start AI processing in background
	aiProcessingCmd := tea.Cmd(func() tea.Msg {
		// Small delay to allow memory results to show first
		time.Sleep(100 * time.Millisecond)
		
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
	})

	// Return combined commands
	var cmds []tea.Cmd
	if memoryCmd != nil {
		cmds = append(cmds, memoryCmd)
	}
	cmds = append(cmds, aiCmd, aiProcessingCmd, AIProcessingCmd(), StartAnimationCmd(), m.spinner.TickCmd())
	
	return tea.Batch(cmds...)
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
	m.showSpinner = false // Stop the spinner
	
	// Remove thinking bubble message
	m.removeLastMessage()
	
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
		m.status = fmt.Sprintf("Ready - %s • %s", m.currentProvider, m.currentModel)
		return
	}
	
	// Store suggestions for potential selection
	m.suggestions = msg.suggestions
	m.availableSuggestions = msg.suggestions
	m.inSelectionMode = true // Enable selection mode
	
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
	m.addMessage("💡 Use 1-9 to select a command, 'e' to edit first command, or type a new request", MessageTypeSystem)
	m.status = fmt.Sprintf("Ready - %s • %s", m.currentProvider, m.currentModel)
}

// handleCommandSelection handles when user selects a command by number
func (m *Model) handleCommandSelection(index int) tea.Cmd {
	// Check if we're in selection mode
	if !m.inSelectionMode {
		m.addMessage("❌ No commands available to select", MessageTypeError)
		return nil
	}
	
	// First check if user is selecting from memory suggestions (they use M1, M2, etc format)
	// For simplicity, we'll handle this in the number input parsing instead
	
	// Handle memory suggestions first (if any)
	if len(m.memorySuggestions) > 0 {
		// If index is within memory range, select from memory
		if index < len(m.memorySuggestions) {
			return m.handleMemorySelection(memorySelectionMsg{index: index})
		}
		// Adjust index for AI suggestions (they come after memory suggestions)
		index -= len(m.memorySuggestions)
	}
	
	// Handle AI suggestions
	if len(m.availableSuggestions) == 0 {
		m.addMessage("❌ No AI commands available to select", MessageTypeError)
		return nil
	}
	
	// Check if adjusted index is valid for AI suggestions
	if index < 0 || index >= len(m.availableSuggestions) {
		totalSuggestions := len(m.memorySuggestions) + len(m.availableSuggestions)
		m.addMessage(fmt.Sprintf("❌ Invalid selection. Please choose 1-%d", totalSuggestions), MessageTypeError)
		return nil
	}
	
	// Track the last selected index
	m.lastSelectedIndex = index
	
	// Get the selected AI suggestion
	selectedSuggestion := m.availableSuggestions[index]
	
	// Add confirmation message
	safetyIcon := "✓"
	if !selectedSuggestion.Safe {
		safetyIcon = "⚠️"
	}
	
	m.addMessage(fmt.Sprintf("Selected: %s %s", safetyIcon, selectedSuggestion.Command), MessageTypeUser)
	
	// Clear selection mode
	m.inSelectionMode = false
	m.availableSuggestions = []aiSuggestion{}
	m.memorySuggestions = []memorySuggestion{}
	
	// Return command to execute the selected command
	return CommandExecutionCmd(
		selectedSuggestion.Command,
		selectedSuggestion.Description,
		selectedSuggestion.Safe,
		selectedSuggestion.Confidence,
	)
}

// handleCommandExecution handles the execution of a selected command
func (m *Model) handleCommandExecution(msg commandExecutionMsg) tea.Cmd {
	// Perform detailed safety analysis using utils package
	isDangerous := utils.IsDangerousCommand(msg.command)
	
	// If command is dangerous or AI marked it as unsafe, request confirmation
	if isDangerous || !msg.safe {
		var reason string
		if isDangerous {
			reason = "Command contains potentially dangerous operations"
		} else {
			reason = "AI confidence indicates this command may be risky"
		}
		
		// Store the pending command and enter confirmation mode
		m.pendingCommand = msg
		m.inConfirmationMode = true
		
		// Display confirmation dialog
		m.addMessage(fmt.Sprintf("⚠️  SAFETY WARNING: %s", reason), MessageTypeError)
		m.addMessage(fmt.Sprintf("🔍 Command: %s", msg.command), MessageTypeSystem)
		
		if msg.description != "" {
			m.addMessage(fmt.Sprintf("📝 Description: %s", msg.description), MessageTypeSystem)
		}
		
		confidencePercent := int(msg.confidence * 100)
		m.addMessage(fmt.Sprintf("🎯 AI Confidence: %d%%", confidencePercent), MessageTypeSystem)
		
		m.addMessage("❓ Do you want to proceed?", MessageTypeSystem)
		m.addMessage("💡 Press 'y' to confirm, 'n' to cancel", MessageTypeSystem)
		return nil
	}
	
	// Command is safe, proceed with execution
	m.addMessage(fmt.Sprintf("✅ Executing safe command: %s", msg.command), MessageTypeSystem)
	
	if msg.description != "" {
		m.addMessage(fmt.Sprintf("📝 %s", msg.description), MessageTypeSystem)
	}
	
	confidencePercent := int(msg.confidence * 100)
	m.addMessage(fmt.Sprintf("🎯 Confidence: %d%%", confidencePercent), MessageTypeSystem)
	
	// Save to memory before execution
	var memorySaveCmd tea.Cmd
	if m.lastUserRequest != "" && m.memoryEnabled {
		memorySaveCmd = MemorySaveCmd(
			m.lastUserRequest,
			msg.command,
			msg.description,
			"ai", // Source
			true, // Initial assumption - will be updated after execution
		)
	}
	
	// Execute the command
	executeCmd := m.executeCommand(msg.command, msg.description)
	
	if memorySaveCmd != nil {
		return tea.Batch(memorySaveCmd, executeCmd)
	}
	
	return executeCmd
}

// handleConfirmationResponse handles user's response to confirmation dialog
func (m *Model) handleConfirmationResponse(confirmed bool) tea.Cmd {
	if !m.inConfirmationMode {
		m.addMessage("❌ No confirmation dialog active", MessageTypeError)
		return nil
	}
	
	// Exit confirmation mode
	m.inConfirmationMode = false
	
	if confirmed {
		m.addMessage("✅ Command confirmed by user", MessageTypeSystem)
		
		// Execute the confirmed command (bypass safety check)
		cmd := m.pendingCommand
		m.addMessage(fmt.Sprintf("🚀 Executing confirmed command: %s", cmd.command), MessageTypeSystem)
		
		if cmd.description != "" {
			m.addMessage(fmt.Sprintf("📝 %s", cmd.description), MessageTypeSystem)
		}
		
		confidencePercent := int(cmd.confidence * 100)
		m.addMessage(fmt.Sprintf("🎯 Confidence: %d%%", confidencePercent), MessageTypeSystem)
		
		// Execute the command - return the command for execution
		return m.executeCommand(cmd.command, cmd.description)
		
	} else {
		m.addMessage("❌ Command execution cancelled by user", MessageTypeSystem)
		m.addMessage("💡 You can type a new request or select a different command", MessageTypeSystem)
	}
	
	// Clear pending command
	m.pendingCommand = commandExecutionMsg{}
	
	return nil
}

// handleConfirmationRequest handles a confirmation request message
func (m *Model) handleConfirmationRequest(msg confirmationRequestMsg) {
	// This method could be used for external confirmation requests
	// For now, it's mainly a placeholder for completeness
	m.addMessage(fmt.Sprintf("⚠️  Confirmation requested: %s", msg.reason), MessageTypeError)
	m.addMessage(fmt.Sprintf("🔍 Command: %s", msg.command), MessageTypeSystem)
	
	if msg.description != "" {
		m.addMessage(fmt.Sprintf("📝 Description: %s", msg.description), MessageTypeSystem)
	}
	
	m.addMessage("❓ Do you want to proceed?", MessageTypeSystem)
	m.addMessage("💡 Press 'y' to confirm, 'n' to cancel", MessageTypeSystem)
}

// handleConfirmationResponseMsg handles a confirmation response message
func (m *Model) handleConfirmationResponseMsg(msg confirmationResponseMsg) {
	if msg.confirmed {
		m.addMessage("✅ Command execution confirmed", MessageTypeSystem)
		// Re-execute the command that was confirmed
		m.handleCommandExecution(msg.command)
	} else {
		m.addMessage("❌ Command execution cancelled", MessageTypeSystem)
		m.addMessage("💡 You can type a new request or select a different command", MessageTypeSystem)
	}
}

// executeCommand executes a command using the executor
func (m *Model) executeCommand(command, description string) tea.Cmd {
	// Check if already executing a command
	if m.executingCommand {
		m.addMessage("⚠️  Another command is already running. Please wait for it to complete.", MessageTypeError)
		return nil
	}
	
	// Update execution state
	m.executingCommand = true
	m.currentCommand = command
	m.executionOutput = []string{}
	m.executionResult = nil
	
	// Start streaming command execution
	return m.startStreamingExecution(command, description)
}

// startStreamingExecution starts command execution with real-time output streaming
func (m *Model) startStreamingExecution(command, description string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		
		// Start the command stream
		outputChan, err := m.executor.Stream(ctx, command)
		if err != nil {
			return CommandErrorCmd(command, err)()
		}
		
		// Store the stream channel in model (we'll access it in update loop)
		// Note: This is a hack for the current implementation - in a real app
		// we'd handle this differently
		
		return commandStreamStartMsg{
			command:     command,
			description: description,
			stream:      outputChan,
		}
	})
}

// handleCommandStart handles command start message
func (m *Model) handleCommandStart(msg commandStartMsg) {
	m.addMessage(fmt.Sprintf("🚀 Started: %s (PID: %d)", msg.command, msg.pid), MessageTypeSystem)
	m.currentPID = msg.pid
}

// handleCommandOutput handles command output message  
func (m *Model) handleCommandOutput(msg commandOutputMsg) {
	// Add output to buffer
	m.executionOutput = append(m.executionOutput, msg.content)
	
	// Display output in TUI
	outputType := MessageTypeAssistant
	if msg.isStderr {
		outputType = MessageTypeError
	}
	
	// Add timestamp prefix for output
	prefix := "📤"
	if msg.isStderr {
		prefix = "❌"
	}
	
	if strings.TrimSpace(msg.content) != "" {
		m.addMessage(fmt.Sprintf("%s %s", prefix, msg.content), outputType)
	}
}

// handleCommandComplete handles command completion message
func (m *Model) handleCommandComplete(msg commandCompleteMsg) {
	// Update execution state
	m.executingCommand = false
	m.executionResult = &executionResult{
		Command:  msg.command,
		ExitCode: msg.exitCode,
		Duration: msg.duration,
		Error:    msg.error,
	}
	
	// Display stdout output if available
	if msg.stdout != "" {
		lines := strings.Split(strings.TrimSpace(msg.stdout), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				m.addMessage(fmt.Sprintf("📤 %s", line), MessageTypeAssistant)
			}
		}
	}
	
	// Display stderr output if available
	if msg.stderr != "" {
		lines := strings.Split(strings.TrimSpace(msg.stderr), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				m.addMessage(fmt.Sprintf("❌ %s", line), MessageTypeError)
			}
		}
	}
	
	// Display completion message
	if msg.exitCode == 0 {
		m.addMessage(fmt.Sprintf("✅ Command completed successfully (%.2fs)", msg.duration.Seconds()), MessageTypeSystem)
	} else {
		m.addMessage(fmt.Sprintf("❌ Command failed with exit code %d (%.2fs)", msg.exitCode, msg.duration.Seconds()), MessageTypeError)
		if msg.error != nil {
			m.addMessage(fmt.Sprintf("Error: %s", msg.error.Error()), MessageTypeError)
		}
	}
	
	// Reset current command tracking
	m.currentCommand = ""
	m.currentPID = 0
}

// handleCommandError handles command execution error message
func (m *Model) handleCommandError(msg commandErrorMsg) {
	// Update execution state
	m.executingCommand = false
	m.executionResult = &executionResult{
		Command: msg.command,
		Error:   msg.error,
	}
	
	m.addMessage(fmt.Sprintf("❌ Execution error: %s", msg.error.Error()), MessageTypeError)
	
	// Reset current command tracking
	m.currentCommand = ""
	m.currentPID = 0
}

// enterEditMode enters edit mode for a selected command
func (m *Model) enterEditMode(suggestion aiSuggestion) {
	m.inEditMode = true
	m.inSelectionMode = false
	m.editingCommand = suggestion.Command
	m.editingDescription = suggestion.Description
	m.editingSafe = suggestion.Safe
	m.editingConfidence = suggestion.Confidence
	m.originalCommand = suggestion.Command
	
	// Clear available suggestions since we're now editing
	m.availableSuggestions = []aiSuggestion{}
	
	// Update input to show the command being edited
	m.input.SetValue(suggestion.Command)
	m.input.Focus()
	
	// Show edit mode instructions
	safetyIcon := "✓"
	if !suggestion.Safe {
		safetyIcon = "⚠️"
	}
	
	m.addMessage(fmt.Sprintf("📝 Edit Mode: %s %s", safetyIcon, suggestion.Command), MessageTypeSystem)
	m.addMessage(fmt.Sprintf("📋 Original: %s", suggestion.Description), MessageTypeSystem)
	m.addMessage("💡 Edit the command above, then press Enter to execute or Escape to cancel", MessageTypeSystem)
}

// exitEditMode exits edit mode and returns to normal mode
func (m *Model) exitEditMode(save bool) tea.Cmd {
	if !m.inEditMode {
		return nil
	}
	
	var cmd tea.Cmd
	
	if save {
		// Save the edited command and execute it
		editedCommand := m.input.Value()
		if editedCommand != m.originalCommand {
			m.addMessage(fmt.Sprintf("✏️ Command edited: %s → %s", m.originalCommand, editedCommand), MessageTypeSystem)
		} else {
			m.addMessage(fmt.Sprintf("✅ Command unchanged: %s", editedCommand), MessageTypeSystem)
		}
		
		// Create execution message with edited command
		cmd = CommandExecutionCmd(
			editedCommand,
			m.editingDescription,
			m.editingSafe, // Keep original safety assessment
			m.editingConfidence,
		)
	} else {
		// Cancel editing
		m.addMessage("❌ Edit cancelled", MessageTypeSystem)
		m.addMessage("💡 You can type a new request or select a different command", MessageTypeSystem)
	}
	
	// Reset edit mode state
	m.inEditMode = false
	m.editingCommand = ""
	m.editingDescription = ""
	m.editingSafe = true
	m.editingConfidence = 0.0
	m.originalCommand = ""
	
	// Clear input and reset placeholder
	m.input.SetValue("")
	m.input.Placeholder = "Type your command request here..."
	
	return cmd
}

// handleEditModeInput handles input when in edit mode
func (m *Model) handleEditModeInput() tea.Cmd {
	if !m.inEditMode {
		return nil
	}
	
	// In edit mode, Enter saves and executes the command
	return m.exitEditMode(true)
}

// handleDirectCommand handles direct command execution (starting with '!')
func (m *Model) handleDirectCommand(input string) tea.Cmd {
	// Extract the actual command by removing the '!' prefix
	if len(input) < 2 {
		m.addMessage("❌ Empty direct command. Usage: !<command>", MessageTypeError)
		m.input.SetValue("")
		return nil
	}
	
	command := strings.TrimSpace(input[1:])
	if command == "" {
		m.addMessage("❌ Empty direct command. Usage: !<command>", MessageTypeError)
		m.input.SetValue("")
		return nil
	}
	
	// Add user input to history
	m.addMessage(input, MessageTypeUser)
	m.input.SetValue("")
	
	// Show direct execution message (no safety checks warning)
	m.addMessage(fmt.Sprintf("⚡ Direct execution (no safety checks): %s", command), MessageTypeSystem)
	
	// Execute command directly without any safety checks or confirmations
	return m.executeCommand(command, "Direct command execution")
}

// handleCommandStreamStart handles the start of a command stream
func (m *Model) handleCommandStreamStart(msg commandStreamStartMsg) tea.Cmd {
	m.outputStream = msg.stream
	m.streamActive = true
	
	m.addMessage(fmt.Sprintf("🚀 Streaming: %s", msg.command), MessageTypeSystem)
	if msg.description != "" {
		m.addMessage(fmt.Sprintf("📝 %s", msg.description), MessageTypeSystem)
	}
	
	// Start stream tick processing
	return StreamTickCmd()
}

// handleStreamTick processes stream output from the running command
func (m *Model) handleStreamTick() tea.Cmd {
	if !m.streamActive || m.outputStream == nil {
		return nil
	}
	
	// Non-blocking read from stream
	select {
	case output, ok := <-m.outputStream:
		if !ok {
			// Stream closed - reset all execution state
			m.streamActive = false
			m.outputStream = nil
			m.executingCommand = false
			
			// Reset command tracking
			command := m.currentCommand
			m.currentCommand = ""
			m.currentPID = 0
			
			// Add completion message
			if command != "" {
				m.addMessage(fmt.Sprintf("✅ Command completed: %s", command), MessageTypeSystem)
			}
			
			return nil
		}
		
		// Process the output line
		if strings.TrimSpace(output.Content) != "" {
			outputType := MessageTypeAssistant
			prefix := "📤"
			if output.IsStderr {
				outputType = MessageTypeError
				prefix = "❌"
			}
			
			m.addMessage(fmt.Sprintf("%s %s", prefix, output.Content), outputType)
		}
		
		// Continue ticking
		return StreamTickCmd()
		
	default:
		// No data available, continue ticking
		return StreamTickCmd()
	}
}

// handleStreamEnd handles the end of a command stream
func (m *Model) handleStreamEnd(msg streamEndMsg) {
	m.streamActive = false
	m.outputStream = nil
	m.executingCommand = false
	
	// Update memory with execution result
	if m.currentCommand != "" {
		m.updateMemoryWithResult(m.currentCommand, msg.exitCode == 0)
	}
	
	// Display completion message
	if msg.exitCode == 0 {
		m.addMessage(fmt.Sprintf("✅ Command completed successfully (%.2fs)", msg.duration.Seconds()), MessageTypeSystem)
	} else {
		m.addMessage(fmt.Sprintf("❌ Command failed with exit code %d (%.2fs)", msg.exitCode, msg.duration.Seconds()), MessageTypeError)
		if msg.error != nil {
			m.addMessage(fmt.Sprintf("Error: %s", msg.error.Error()), MessageTypeError)
		}
	}
	
	// Reset current command tracking
	m.currentCommand = ""
	m.currentPID = 0
}

// Memory-related handlers

// handleMemoryResults processes memory search results
func (m *Model) handleMemoryResults(msg memoryResultsMsg) {
	if msg.error != nil {
		log.Printf("Memory search error: %v", msg.error)
		return
	}

	if len(msg.results) == 0 {
		// No memory suggestions found, just wait for AI
		return
	}

	// Convert memory results to memory suggestions
	m.memorySuggestions = make([]memorySuggestion, 0, len(msg.results))
	for _, result := range msg.results {
		suggestion := memorySuggestion{
			Entry:      result.Entry,
			Score:      result.Score,
			Reason:     result.Reason,
			MatchType:  result.MatchType,
			UsageCount: result.Entry.UsageCount,
			LastUsed:   result.Entry.Timestamp,
		}
		m.memorySuggestions = append(m.memorySuggestions, suggestion)
	}

	// Display memory suggestions immediately
	m.addMessage("💭 Memory suggestions:", MessageTypeSystem)
	for i, suggestion := range m.memorySuggestions {
		safetyIcon := "✓"
		if !suggestion.Entry.Success {
			safetyIcon = "⚠"
		}
		
		// Show usage count and last used time
		timeAgo := time.Since(suggestion.LastUsed)
		var timeStr string
		if timeAgo < time.Hour {
			timeStr = fmt.Sprintf("%.0fm ago", timeAgo.Minutes())
		} else if timeAgo < 24*time.Hour {
			timeStr = fmt.Sprintf("%.0fh ago", timeAgo.Hours())
		} else {
			timeStr = fmt.Sprintf("%.0fd ago", timeAgo.Hours()/24)
		}

		suggestionText := fmt.Sprintf("M%d. %s %s (used %dx, %s)\n    %s", 
			i+1, safetyIcon, suggestion.Entry.SelectedCommand, 
			suggestion.UsageCount, timeStr, suggestion.Entry.Description)
		
		m.addMessage(suggestionText, MessageTypeAssistant)
	}

	// Update selection mode to include memory suggestions
	m.inSelectionMode = true
}

// handleMemorySave processes memory save requests
func (m *Model) handleMemorySave(msg memorySaveMsg) tea.Cmd {
	if !m.memoryEnabled || m.memoryManager == nil {
		return nil
	}

	return tea.Cmd(func() tea.Msg {
		err := m.memoryManager.Add(
			msg.userRequest,
			msg.selectedCommand,
			msg.description,
			msg.source,
			msg.success,
		)
		
		return memorySaveResultMsg{
			success: err == nil,
			error:   err,
		}
	})
}

// handleMemorySaveResult processes memory save results
func (m *Model) handleMemorySaveResult(msg memorySaveResultMsg) {
	if msg.error != nil {
		log.Printf("Failed to save to memory: %v", msg.error)
		// Don't show error to user unless it's critical
	}
	// Successful saves are silent - no need to notify user
}

// handleMemorySelection processes memory suggestion selection
func (m *Model) handleMemorySelection(msg memorySelectionMsg) tea.Cmd {
	if msg.index < 0 || msg.index >= len(m.memorySuggestions) {
		m.addMessage("❌ Invalid memory selection", MessageTypeError)
		return nil
	}

	selectedMemory := m.memorySuggestions[msg.index]
	
	// Add confirmation message
	m.addMessage(fmt.Sprintf("Selected from memory: %s", selectedMemory.Entry.SelectedCommand), MessageTypeUser)
	
	// Clear selection mode
	m.inSelectionMode = false
	m.memorySuggestions = []memorySuggestion{}
	
	// Execute the selected command
	return CommandExecutionCmd(
		selectedMemory.Entry.SelectedCommand,
		selectedMemory.Entry.Description,
		selectedMemory.Entry.Success, // Use success history as safety indicator
		selectedMemory.Score,
	)
}

// handleCombinedSuggestions processes combined AI and memory suggestions
func (m *Model) handleCombinedSuggestions(msg combinedSuggestionsMsg) {
	// This is called when we have both AI and memory suggestions
	// For now, we display them separately, but this could be enhanced
	// to merge and rank them together
	
	// AI suggestions are handled by existing handleAIResponse
	// Memory suggestions are already displayed by handleMemoryResults
}

// Helper function to update memory after command execution
func (m *Model) updateMemoryWithResult(command string, success bool) {
	if !m.memoryEnabled || m.memoryManager == nil || m.lastUserRequest == "" {
		return
	}
	
	// Update the memory entry with the actual execution result
	// This is done asynchronously to not block the UI
	go func() {
		// Find recent entries that match this command and update success status
		entries := m.memoryManager.GetAll()
		for _, entry := range entries {
			if entry.SelectedCommand == command && 
			   entry.UserRequest == m.lastUserRequest &&
			   time.Since(entry.Timestamp) < 5*time.Minute { // Recent entry
				
				// Update the entry
				updates := map[string]interface{}{
					"success": success,
				}
				m.memoryManager.Update(entry.ID, updates)
				break
			}
		}
	}()
}