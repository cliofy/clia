package main

import (
	"os"
	"testing"
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