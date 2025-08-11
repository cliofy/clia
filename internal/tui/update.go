package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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

// handleKeyInput processes specific key inputs
func (m *Model) handleKeyInput(key string) tea.Cmd {
	switch key {
	case "ctrl+c":
		return tea.Quit
	case "ctrl+l":
		return ClearHistoryCmd()
	case "enter":
		input := m.input.Value()
		if input != "" {
			m.input.SetValue("")
			return AddMessageCmd(input, MessageTypeUser)
		}
	}
	return nil
}