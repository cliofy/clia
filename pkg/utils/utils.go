package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetHomeDir returns the user's home directory
func GetHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

// GetConfigDir returns the configuration directory for the application
func GetConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			home, err := GetHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, "AppData", "Roaming")
		}
	case "darwin":
		home, err := GetHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	default: // Linux and other Unix-like systems
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := GetHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config")
		}
	}

	cliaConfigDir := filepath.Join(configDir, "clia")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(cliaConfigDir, 0755); err != nil {
		return "", err
	}

	return cliaConfigDir, nil
}

// SanitizeAPIKey masks an API key for safe logging
func SanitizeAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// IsCommandSafe checks if a command is considered safe to execute
func IsCommandSafe(command string) bool {
	command = strings.TrimSpace(strings.ToLower(command))

	// List of safe commands that don't modify the system
	safeCommands := []string{
		"ls", "ll", "la", "dir",
		"pwd", "cd",
		"cat", "less", "more", "head", "tail",
		"echo", "printf",
		"date", "cal",
		"whoami", "id",
		"ps", "top", "htop",
		"df", "du", "free",
		"which", "whereis", "type",
		"history",
		"env", "printenv",
		"uname", "hostname",
		"uptime", "w", "who",
	}

	for _, safe := range safeCommands {
		if strings.HasPrefix(command, safe+" ") || command == safe {
			return true
		}
	}

	return false
}

// IsDangerousCommand checks if a command is potentially dangerous
func IsDangerousCommand(command string) bool {
	command = strings.TrimSpace(strings.ToLower(command))

	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf *",
		":(){ :|:& };:", // Fork bomb
		"dd if=/dev/zero",
		"mkfs",
		"fdisk",
		"format c:",
		"> /dev/sda",
		"curl", // Could download malicious content
		"wget", // Could download malicious content
		"chmod 777",
		"chown -R",
		"shutdown",
		"reboot",
		"halt",
		"poweroff",
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(command, dangerous) {
			return true
		}
	}

	return false
}
