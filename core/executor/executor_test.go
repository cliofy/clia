package executor

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutor_SimpleCommand tests basic command execution
func TestExecutor_SimpleCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
		exitCode int
	}{
		{
			name:     "echo command",
			command:  "echo 'hello world'",
			expected: "hello world",
			exitCode: 0,
		},
		{
			name:     "pwd command",
			command:  "pwd",
			expected: "/", // should contain forward slash
			exitCode: 0,
		},
		{
			name:     "exit code test",
			command:  "exit 42",
			expected: "",
			exitCode: 42,
		},
	}

	exec := NewExecutor()
	require.NotNil(t, exec)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exec.Execute(tt.command)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check exit code
			assert.Equal(t, tt.exitCode, result.ExitCode)

			// Check output contains expected string
			if tt.expected != "" {
				assert.Contains(t, result.Output, tt.expected)
			}
		})
	}
}

// TestExecutor_ColoredOutput tests commands with ANSI color codes
func TestExecutor_ColoredOutput(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		checkANSI    bool
		checkContent bool
	}{
		{
			name:         "ls with color",
			command:      "ls --color=always /tmp",
			checkANSI:    true,
			checkContent: true,
		},
		{
			name:         "grep with color",
			command:      "echo 'test line' | grep --color=always test",
			checkANSI:    true,
			checkContent: true,
		},
	}

	exec := NewExecutor()
	require.NotNil(t, exec)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exec.Execute(tt.command)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Should have successful exit
			assert.Equal(t, 0, result.ExitCode)

			// Check for ANSI escape sequences if expected
			if tt.checkANSI {
				// ANSI color codes start with ESC[
				assert.Contains(t, result.Output, "\x1b[", "Output should contain ANSI escape sequences")
			}

			// Output should not be empty
			if tt.checkContent {
				assert.NotEmpty(t, result.Output)
			}
		})
	}
}

// TestExecutor_TUIApplication tests handling of TUI applications
func TestExecutor_TUIApplication(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		validate func(t *testing.T, result *Result)
	}{
		{
			name:    "clear screen command",
			command: "clear",
			validate: func(t *testing.T, result *Result) {
				// Clear command should succeed
				assert.Equal(t, 0, result.ExitCode)
				// Should contain clear screen sequence or be empty
				// Clear typically sends \x1b[H\x1b[2J or similar
				if result.Output != "" {
					assert.Contains(t, result.Output, "\x1b[")
				}
			},
		},
		{
			name:    "tput commands",
			command: "tput cols && tput lines",
			validate: func(t *testing.T, result *Result) {
				assert.Equal(t, 0, result.ExitCode)
				// Should return terminal dimensions
				// Output should contain two numbers
				assert.Contains(t, result.Output, "80")  // Default PTY width
				assert.Contains(t, result.Output, "24")  // Default PTY height
			},
		},
		{
			name:    "cursor movement",
			command: "echo -e '\\033[2J\\033[H' && echo 'Screen cleared'",
			validate: func(t *testing.T, result *Result) {
				assert.Equal(t, 0, result.ExitCode)
				// Should contain ANSI escape sequences for clear and home
				assert.Contains(t, result.Output, "\x1b[2J") // Clear screen
				assert.Contains(t, result.Output, "\x1b[H")  // Home cursor
				assert.Contains(t, result.Output, "Screen cleared")
				// Use strings to keep import
				_ = strings.TrimSpace(result.Output)
			},
		},
	}

	exec := NewExecutor()
	require.NotNil(t, exec)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exec.Execute(tt.command)
			require.NoError(t, err)
			require.NotNil(t, result)

			tt.validate(t, result)
		})
	}
}

// TestExecutor_Timeout tests command execution with timeout
func TestExecutor_Timeout(t *testing.T) {
	exec := NewExecutor()
	require.NotNil(t, exec)

	t.Run("command completes before timeout", func(t *testing.T) {
		result, err := exec.ExecuteWithTimeout("echo 'quick'", 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Output, "quick")
	})

	t.Run("command times out", func(t *testing.T) {
		result, err := exec.ExecuteWithTimeout("sleep 10", 100*time.Millisecond)
		// Should either error or have non-zero exit code
		if err == nil {
			assert.NotEqual(t, 0, result.ExitCode)
		}
	})
}