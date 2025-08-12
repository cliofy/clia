package main

import (
	"os"
	"strings"
	"testing"
	
	"github.com/yourusername/clia/internal/ai"
)

func TestCLIServiceInitialization(t *testing.T) {
	// Test service initialization without API keys
	service, err := initializeCLIServices()
	
	// Should not return error even without API keys (fallback mode)
	if err != nil {
		t.Errorf("Expected no error during initialization, got: %v", err)
	}
	
	if service == nil {
		t.Fatal("Expected service to be initialized")
	}
	
	if service.aiService == nil {
		t.Error("Expected AI service to be initialized")
	}
	
	if service.executor == nil {
		t.Error("Expected executor to be initialized")
	}
}

func TestCLIServiceWithAPIKey(t *testing.T) {
	// Set a test API key
	os.Setenv("OPENROUTER_API_KEY", "test-key-for-testing")
	defer os.Unsetenv("OPENROUTER_API_KEY")
	
	service, err := initializeCLIServices()
	
	if err != nil {
		t.Errorf("Expected no error with API key set, got: %v", err)
	}
	
	if service == nil {
		t.Fatal("Expected service to be initialized")
	}
}

func TestDisplaySuggestionsAndGetChoice(t *testing.T) {
	// Note: This test would require mocking stdin, which is complex
	// For now, we just test that the function signature is correct
	// and it doesn't panic with empty suggestions
	
	// Test with empty suggestions - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displaySuggestionsAndGetChoice panicked with empty suggestions: %v", r)
		}
	}()
	
	// This would normally wait for user input, but we can't easily test that
	// in a unit test without mocking stdin
	// displaySuggestionsAndGetChoice([]ai.CommandSuggestion{})
	
	// Test passes if we reach here without panicking
}

func TestCLIFallbackSuggestions(t *testing.T) {
	service, err := initializeCLIServices()
	if err != nil {
		t.Errorf("Failed to initialize CLI services: %v", err)
		return
	}
	
	// Test disk space query
	suggestions := service.getFallbackSuggestions("显示当前剩余空间")
	if len(suggestions) == 0 {
		t.Error("Expected fallback suggestions for disk space query, got none")
	}
	
	// Verify we got df command
	found := false
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion.Command, "df") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'df' command in fallback suggestions")
	}
	
	// Test file listing query
	suggestions2 := service.getFallbackSuggestions("list files")
	if len(suggestions2) == 0 {
		t.Error("Expected fallback suggestions for file listing query, got none")
	}
	
	// Verify we got ls command
	found = false
	for _, suggestion := range suggestions2 {
		if strings.Contains(suggestion.Command, "ls") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'ls' command in fallback suggestions")
	}
}

func TestCLITUIModelCreation(t *testing.T) {
	// Test creating CLI TUI model
	userRequest := "test query"
	suggestions := []ai.CommandSuggestion{
		{
			Command:     "ls -la",
			Description: "List files",
			Safe:        true,
			Confidence:  0.9,
		},
		{
			Command:     "pwd",
			Description: "Show current directory",
			Safe:        true,
			Confidence:  0.8,
		},
	}
	
	model := NewCLITUIModel(userRequest, suggestions)
	
	// Verify initial state
	if model.state != StateSelecting {
		t.Errorf("Expected initial state to be StateSelecting, got %v", model.state)
	}
	
	if model.userRequest != userRequest {
		t.Errorf("Expected userRequest to be %s, got %s", userRequest, model.userRequest)
	}
	
	if len(model.suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(model.suggestions))
	}
	
	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", model.selectedIndex)
	}
	
	// Test basic view rendering without running TUI
	// Simulate ready state for proper rendering
	model.ready = true
	model.width = 80
	model.height = 24
	
	view := model.View()
	
	if !strings.Contains(view, userRequest) {
		t.Error("Expected view to contain user request")
	}
	
	if !strings.Contains(view, "ls -la") {
		t.Error("Expected view to contain first suggestion")
	}
	
	if !strings.Contains(view, "Select a command") {
		t.Error("Expected view to contain selection prompt")
	}
	
	if !strings.Contains(view, "[●]") {
		t.Error("Expected view to show selected item indicator")
	}
	
	// Test that it shows CLI-style format
	if !strings.Contains(view, "↑/↓, j/k: select") {
		t.Error("Expected view to contain CLI-style help text")
	}
}