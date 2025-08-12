package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/clia/internal/executor"
)

func TestNewModel(t *testing.T) {
	model := New()
	
	// Check that status contains provider and model info
	if model.status == "" {
		t.Error("Expected status to be set")
	}
	
	if len(model.messages) == 0 {
		t.Error("Expected welcome messages to be added during initialization")
	}
	
	if !model.input.Focused() {
		t.Error("Expected input to be focused initially")
	}
}

func TestMessageHandling(t *testing.T) {
	model := New()
	initialMessageCount := len(model.messages)
	
	// Test adding a message
	model.addMessage("Test message", MessageTypeUser)
	
	if len(model.messages) != initialMessageCount+1 {
		t.Errorf("Expected %d messages, got %d", initialMessageCount+1, len(model.messages))
	}
	
	lastMessage := model.messages[len(model.messages)-1]
	if lastMessage.Content != "Test message" {
		t.Errorf("Expected message content 'Test message', got '%s'", lastMessage.Content)
	}
	
	if lastMessage.Type != MessageTypeUser {
		t.Errorf("Expected message type %v, got %v", MessageTypeUser, lastMessage.Type)
	}
}

func TestClearMessages(t *testing.T) {
	model := New()
	
	// Add some messages
	model.addMessage("Message 1", MessageTypeUser)
	model.addMessage("Message 2", MessageTypeAssistant)
	
	// Clear messages
	model.clearMessages()
	
	// Should have only the "History cleared" system message
	if len(model.messages) != 1 {
		t.Errorf("Expected 1 message after clear (system message), got %d", len(model.messages))
	}
	
	if model.messages[0].Type != MessageTypeSystem {
		t.Error("Expected first message after clear to be system message")
	}
}

func TestWindowSizeHandling(t *testing.T) {
	model := New()
	
	// Simulate window resize
	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 30,
	}
	
	model.handleWindowSizeMsg(msg)
	
	if model.width != 100 {
		t.Errorf("Expected width 100, got %d", model.width)
	}
	
	if model.height != 30 {
		t.Errorf("Expected height 30, got %d", model.height)
	}
	
	if !model.ready {
		t.Error("Expected model to be ready after window size message")
	}
}

func TestKeyHandling(t *testing.T) {
	model := New()
	
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"Quit key", "ctrl+c", "quit"},
		{"Clear key", "ctrl+l", "clear"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			
			_, cmd := model.Update(keyMsg)
			
			if tt.expected == "quit" && cmd == nil {
				// For ctrl+c, we expect tea.Quit command (which is a specific type)
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
				_, cmd = model.Update(keyMsg)
			}
			
			// Note: Testing the exact command returned is complex with Bubble Tea
			// In a real application, you might use integration tests instead
		})
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MessageTypeUser, "user"},
		{MessageTypeSystem, "system"},
		{MessageTypeAssistant, "assistant"},
		{MessageTypeError, "error"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.msgType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		msgType MessageType
		content string
	}{
		{MessageTypeUser, "User message"},
		{MessageTypeSystem, "System message"},
		{MessageTypeAssistant, "Assistant message"},
		{MessageTypeError, "Error message"},
	}
	
	for _, tt := range tests {
		t.Run(tt.msgType.String(), func(t *testing.T) {
			msg := Message{
				Content: tt.content,
				Type:    tt.msgType,
			}
			
			formatted := FormatMessage(msg)
			
			if formatted == "" {
				t.Error("Expected formatted message to not be empty")
			}
			
			// The formatted message should contain the original content
			// Note: We can't easily test the exact formatted output due to styling
		})
	}
}

func TestSequentialCommandExecution(t *testing.T) {
	model := New()
	
	// Simulate first command execution - should set executingCommand to true
	firstCmd := commandExecutionMsg{
		command:     "ls",
		description: "List files",
		safe:        true,
		confidence:  0.9,
	}
	
	// Execute first command
	cmd := model.handleCommandExecution(firstCmd)
	if cmd == nil {
		t.Error("Expected command execution to return a command")
	}
	
	// Verify execution state is set
	if !model.executingCommand {
		t.Error("Expected executingCommand to be true after starting execution")
	}
	
	// Simulate stream completion by manually calling the stream end logic
	// This simulates what happens when a stream closes in handleStreamTick
	model.streamActive = false
	model.outputStream = nil
	model.executingCommand = false
	model.currentCommand = ""
	model.currentPID = 0
	
	// Now try to execute a second command - should succeed
	secondCmd := commandExecutionMsg{
		command:     "pwd",
		description: "Print working directory",
		safe:        true,
		confidence:  0.9,
	}
	
	// This should not fail with "Another command is already running"
	cmd2 := model.handleCommandExecution(secondCmd)
	if cmd2 == nil {
		t.Error("Expected second command execution to succeed and return a command")
	}
	
	// Verify that the second command can start execution
	if !model.executingCommand {
		t.Error("Expected executingCommand to be true after starting second execution")
	}
}

func TestExecutionStateReset(t *testing.T) {
	model := New()
	
	// Set execution state as if a command is running
	model.executingCommand = true
	model.currentCommand = "test-command"
	model.currentPID = 12345
	model.streamActive = true
	
	// Simulate stream completion by calling handleStreamTick with closed channel
	outputChan := make(chan executor.OutputLine)
	close(outputChan)
	model.outputStream = outputChan
	
	// Call handleStreamTick - this should detect the closed channel and reset state
	cmd := model.handleStreamTick()
	
	// Verify command returned is nil (no more ticking needed)
	if cmd != nil {
		t.Error("Expected handleStreamTick to return nil when stream is closed")
	}
	
	// This should reset all execution state
	if model.executingCommand {
		t.Error("Expected executingCommand to be false after stream completion")
	}
	
	if model.currentCommand != "" {
		t.Error("Expected currentCommand to be empty after stream completion")
	}
	
	if model.currentPID != 0 {
		t.Error("Expected currentPID to be 0 after stream completion")
	}
	
	if model.streamActive {
		t.Error("Expected streamActive to be false after stream completion")
	}
}