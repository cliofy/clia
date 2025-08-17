package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInteractiveCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// Interactive commands
		{"top command", "top", true},
		{"htop command", "htop", true},
		{"vim command", "vim file.txt", true},
		{"vi command", "vi", true},
		{"nano editor", "nano README.md", true},
		{"less pager", "less file.log", true},
		{"tail follow", "tail -f /var/log/system.log", true},
		{"docker interactive", "docker exec -it container bash", true},
		{"docker attach", "docker attach mycontainer", true},
		{"ssh connection", "ssh user@host", true},
		{"mysql client", "mysql -u root -p", true},
		{"python repl", "python", true},
		{"node repl", "node", true},
		{"bash shell", "bash", true},
		{"watch command", "watch df -h", true},
		{"tmux session", "tmux new-session", true},
		{"screen session", "screen", true},
		{"btop monitor", "btop", true},
		{"lazygit", "lazygit", true},
		{"interactive flag -it", "some_command -it arg", true},
		{"interactive flag -i -t", "command -i -t input", true},
		
		// Non-interactive commands
		{"echo command", "echo hello", false},
		{"ls command", "ls -la", false},
		{"grep command", "grep pattern file", false},
		{"cat command", "cat file.txt", false},
		{"pwd command", "pwd", false},
		{"date command", "date", false},
		{"mkdir command", "mkdir newdir", false},
		{"cp command", "cp source dest", false},
		{"mv command", "mv old new", false},
		{"rm command", "rm file", false},
		{"chmod command", "chmod 755 file", false},
		{"curl command", "curl https://example.com", false},
		{"wget command", "wget https://example.com", false},
		{"git command", "git status", false},
		{"docker ps", "docker ps", false},
		{"python script", "python script.py", false},
		{"node script", "node app.js", false},
		{"topological sort", "topo_sort input.txt", false}, // "top" is substring but not the command
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInteractiveCommand(tt.command)
			assert.Equal(t, tt.expected, result, "Command: %s", tt.command)
		})
	}
}

func TestInteractiveExecutor(t *testing.T) {
	t.Run("NewInteractiveExecutor", func(t *testing.T) {
		exec := NewInteractiveExecutor()
		assert.NotNil(t, exec)
	})

	t.Run("NewExtendedExecutor", func(t *testing.T) {
		exec := NewExtendedExecutor()
		assert.NotNil(t, exec)
		
		// Should implement both interfaces
		_, ok := exec.(Executor)
		assert.True(t, ok, "Should implement Executor interface")
		
		_, ok = exec.(InteractiveExecutor)
		assert.True(t, ok, "Should implement InteractiveExecutor interface")
	})
}