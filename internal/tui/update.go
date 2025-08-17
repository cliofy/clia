package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/ai"
)

// Update handles all incoming messages and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+l":
			m.clearMessages()
			return m, nil

		case "enter":
			if cmd := m.handleInputSubmit(); cmd != nil {
				cmds = append(cmds, cmd)
			}

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Handle number key selection when in selection mode
			if m.inSelectionMode {
				// Convert string to int and adjust for 0-based indexing
				index := int(msg.String()[0] - '1') // '1' -> 0, '2' -> 1, etc.
				if cmd := m.handleCommandSelection(index); cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else {
				// Not in selection mode, handle as regular input
				m.input, cmd = m.input.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "e":
			// Handle edit command when in selection mode
			if m.inSelectionMode {
				// Enter edit mode with the first available suggestion
				if len(m.availableSuggestions) > 0 {
					m.enterEditMode(m.availableSuggestions[0])
				} else {
					m.addMessage("❌ No commands available to edit", MessageTypeError)
				}
			} else {
				// Not in selection mode, handle as regular input
				m.input, cmd = m.input.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "escape":
			// Handle escape key
			if m.inEditMode {
				// Exit edit mode without saving
				if cmd := m.exitEditMode(false); cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else if m.inSelectionMode {
				// Exit selection mode
				m.inSelectionMode = false
				m.availableSuggestions = []aiSuggestion{}
				m.addMessage("Selection mode cancelled", MessageTypeSystem)
			} else {
				// Clear input in normal mode
				m.input.SetValue("")
			}

		case "y", "Y":
			// Handle confirmation - confirm command execution
			if m.inConfirmationMode {
				if cmd := m.handleConfirmationResponse(true); cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else {
				// Not in confirmation mode, handle as regular input
				m.input, cmd = m.input.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "n", "N":
			// Handle confirmation - cancel command execution
			if m.inConfirmationMode {
				if cmd := m.handleConfirmationResponse(false); cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else {
				// Not in confirmation mode, handle as regular input
				m.input, cmd = m.input.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		default:
			// Handle regular input
			m.input, cmd = m.input.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case clearHistoryMsg:
		m.clearMessages()

	case addMessageMsg:
		m.addMessage(msg.Content, msg.Type)

	case aiProcessingMsg:
		// Just update UI, processing state already set

	case aiResponseMsg:
		m.handleAIResponse(msg)

	case commandMsg:
		// Command messages are handled in handleInputSubmit

	case providerSwitchMsg:
		m.handleProviderSwitchMsg(msg)

	case modelListMsg:
		m.handleModelListMsg(msg)

	case modelSwitchMsg:
		m.handleModelSwitchMsg(msg)

	case apiKeyInputMsg:
		m.handleAPIKeyInputMsg(msg)

	case apiKeySubmitMsg:
		return m.handleAPIKeySubmitMsg(msg)

	case commandSelectionMsg:
		if cmd := m.handleCommandSelection(msg.index); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case commandExecutionMsg:
		if cmd := m.handleCommandExecution(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case confirmationRequestMsg:
		m.handleConfirmationRequest(msg)

	case confirmationResponseMsg:
		m.handleConfirmationResponseMsg(msg)

	case commandStartMsg:
		m.handleCommandStart(msg)

	case commandOutputMsg:
		m.handleCommandOutput(msg)

	case commandCompleteMsg:
		m.handleCommandComplete(msg)

	case commandErrorMsg:
		m.handleCommandError(msg)

	case commandStreamStartMsg:
		if cmd := m.handleCommandStreamStart(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case streamTickMsg:
		if cmd := m.handleStreamTick(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case streamEndMsg:
		m.handleStreamEnd(msg)

	// Memory-related messages
	case memoryResultsMsg:
		m.handleMemoryResults(msg)

	case memorySaveMsg:
		if cmd := m.handleMemorySave(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case memorySaveResultMsg:
		m.handleMemorySaveResult(msg)

	case memorySelectionMsg:
		if cmd := m.handleMemorySelection(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case combinedSuggestionsMsg:
		m.handleCombinedSuggestions(msg)

	// PTY execution messages
	case ptyExecutionRequestMsg:
		if cmd := m.handlePTYExecutionRequest(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ptyExecutionCompleteMsg:
		m.handlePTYExecutionComplete(msg)

	case SpinnerTickMsg:
		if m.showSpinner {
			m.spinner = m.spinner.NextFrame()
			cmds = append(cmds, m.spinner.TickCmd())
		}
		// Update pulse frame for input animation
		if m.processing {
			m.pulseFrame++
			// Update thinking dots animation
			dotsCount := (m.pulseFrame / 10) % 4 // Change every 10 ticks, cycle through 0-3 dots
			switch dotsCount {
			case 0:
				m.thinkingDots = ""
			case 1:
				m.thinkingDots = "."
			case 2:
				m.thinkingDots = ".."
			case 3:
				m.thinkingDots = "..."
			}
			// Update the last message if it's a thinking message
			if len(m.messages) > 0 && strings.Contains(m.messages[len(m.messages)-1].Content, "思考中") {
				m.messages[len(m.messages)-1].Content = fmt.Sprintf("🤖 思考中%s", m.thinkingDots)
				m.updateViewportContent()
			}
		}

	case startAnimationMsg:
		// Animation started, no additional action needed

	case stopAnimationMsg:
		m.showSpinner = false
		m.pulseFrame = 0 // Reset pulse frame

	default:
		// Update input component
		m.input, cmd = m.input.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Update viewport component
		m.viewport, cmd = m.viewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleProviderSwitchMsg handles provider switch results
func (m *Model) handleProviderSwitchMsg(msg providerSwitchMsg) {
	if msg.needsAPIKey {
		m.addMessage(fmt.Sprintf("🔑 %s provider needs API key configuration", msg.providerType), MessageTypeSystem)
		return
	}

	if msg.success {
		m.currentProvider = msg.providerType
		// Get the new model from the provider
		providerInfo := m.aiService.GetProviderInfo()
		if model, ok := providerInfo["model"].(string); ok {
			m.currentModel = model
		}
		m.status = fmt.Sprintf("Ready - %s • %s", m.currentProvider, m.currentModel)
		m.addMessage(fmt.Sprintf("✅ Switched to %s provider", msg.providerType), MessageTypeSystem)
	} else {
		errorMsg := "Failed to switch provider"
		if msg.error != nil {
			errorMsg += ": " + msg.error.Error()
		}
		m.addMessage("❌ "+errorMsg, MessageTypeError)
	}
}

// handleModelListMsg handles model list results
func (m *Model) handleModelListMsg(msg modelListMsg) {
	if msg.error != nil {
		m.addMessage("❌ Failed to fetch models: "+msg.error.Error(), MessageTypeError)
		return
	}

	if len(msg.models) == 0 {
		m.addMessage("No models available for current provider", MessageTypeSystem)
		return
	}

	formatted := FormatModelList(msg.models, m.currentModel, 15) // Show first 15 models
	m.addMessage(formatted, MessageTypeSystem)

	if len(msg.models) > 15 {
		m.addMessage(fmt.Sprintf("Showing 15 of %d models. Use '/model <name>' to switch.", len(msg.models)), MessageTypeSystem)
	}
}

// handleModelSwitchMsg handles model switch results
func (m *Model) handleModelSwitchMsg(msg modelSwitchMsg) {
	if msg.success {
		m.currentModel = msg.modelName
		m.status = fmt.Sprintf("Ready - %s • %s", m.currentProvider, m.currentModel)
		m.addMessage(fmt.Sprintf("✅ Switched to model: %s", msg.modelName), MessageTypeSystem)
	} else {
		errorMsg := "Failed to switch model"
		if msg.error != nil {
			errorMsg += ": " + msg.error.Error()
		}
		m.addMessage("❌ "+errorMsg, MessageTypeError)
	}
}

// handleAPIKeyInputMsg handles API key input requests
func (m *Model) handleAPIKeyInputMsg(msg apiKeyInputMsg) {
	m.addMessage(msg.prompt, MessageTypeSystem)
	m.waitingAPIKey = true
	m.apiKeyProvider = msg.providerType
	m.input.EchoMode = textinput.EchoPassword
	m.input.Placeholder = "Enter API key (input hidden for security)..."
}

// handleAPIKeySubmitMsg handles API key submissions
func (m *Model) handleAPIKeySubmitMsg(msg apiKeySubmitMsg) (tea.Model, tea.Cmd) {
	// Validate and configure provider with the API key
	return *m, tea.Cmd(func() tea.Msg {
		providerType := ai.ProviderType(msg.providerType)

		// Validate API key
		err := m.aiService.ValidateAPIKey(providerType, msg.apiKey)
		if err != nil {
			return providerSwitchMsg{
				providerType: msg.providerType,
				success:      false,
				error:        fmt.Errorf("invalid API key: %w", err),
			}
		}

		// Create config and switch provider
		config := ai.DefaultProviderConfig(providerType)
		config.APIKey = msg.apiKey

		err = m.aiService.SwitchProvider(providerType, config)
		return providerSwitchMsg{
			providerType: msg.providerType,
			success:      err == nil,
			error:        err,
		}
	})
}
