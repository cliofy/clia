package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg == nil {
		t.Fatal("DefaultConfig() should not return nil")
	}
	
	// Test API config defaults
	if cfg.API.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", cfg.API.Provider)
	}
	
	if cfg.API.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model 'gpt-3.5-turbo', got '%s'", cfg.API.Model)
	}
	
	if cfg.API.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", cfg.API.Timeout)
	}
	
	// Test UI config defaults
	if cfg.UI.Theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", cfg.UI.Theme)
	}
	
	if cfg.UI.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", cfg.UI.Language)
	}
	
	if cfg.UI.HistorySize != 100 {
		t.Errorf("Expected history size 100, got %d", cfg.UI.HistorySize)
	}
	
	// Test Behavior config defaults
	if cfg.Behavior.AutoExecuteSafeCommands {
		t.Error("AutoExecuteSafeCommands should be false by default")
	}
	
	if !cfg.Behavior.ConfirmDangerousCommands {
		t.Error("ConfirmDangerousCommands should be true by default")
	}
	
	if cfg.Behavior.CollectUsageStats {
		t.Error("CollectUsageStats should be false by default")
	}
	
	// Test Context config defaults
	if cfg.Context.IncludeHiddenFiles {
		t.Error("IncludeHiddenFiles should be false by default")
	}
	
	if cfg.Context.MaxFilesInContext != 50 {
		t.Errorf("Expected MaxFilesInContext 50, got %d", cfg.Context.MaxFilesInContext)
	}
	
	if cfg.Context.IncludeEnvVars {
		t.Error("IncludeEnvVars should be false by default")
	}
}