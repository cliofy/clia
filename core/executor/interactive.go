package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// InteractiveExecutor handles interactive/TUI commands
type InteractiveExecutor interface {
	// ExecuteInteractive runs an interactive command with full terminal control
	ExecuteInteractive(cmd string) error
}

// interactiveExecutor implements InteractiveExecutor
type interactiveExecutor struct {
	shell string
}

// NewInteractiveExecutor creates a new interactive executor
func NewInteractiveExecutor() InteractiveExecutor {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return &interactiveExecutor{
		shell: shell,
	}
}

// ExecuteInteractive runs a command with full PTY pass-through
func (e *interactiveExecutor) ExecuteInteractive(cmd string) error {
	// Create command
	command := exec.Command(e.shell, "-c", cmd)

	// Start the command with a PTY
	ptmx, err := pty.Start(command)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	// Handle PTY size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				// Log error but continue
				_ = err
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

	// Set stdin to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// Create a context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Copy stdin to PTY
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
		cancel()
	}()

	// Copy PTY to stdout
	go func() {
		_, _ = io.Copy(os.Stdout, ptmx)
		cancel()
	}()

	// Wait for the command to complete or context to be done
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return nil
	}
}

// ExtendedExecutor combines both regular and interactive execution
type ExtendedExecutor interface {
	Executor
	InteractiveExecutor
}

// extendedExecutor implements both interfaces
type extendedExecutor struct {
	*ptyExecutor
	*interactiveExecutor
}

// NewExtendedExecutor creates an executor that handles both modes
func NewExtendedExecutor() ExtendedExecutor {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return &extendedExecutor{
		ptyExecutor:         &ptyExecutor{shell: shell},
		interactiveExecutor: &interactiveExecutor{shell: shell},
	}
}

// IsInteractiveCommand checks if a command requires interactive mode
func IsInteractiveCommand(cmd string) bool {
	// List of known interactive commands that always need PTY
	interactiveCommands := []string{
		"top", "htop", "btop", "less", "vim", "vi", "nano", "emacs",
		"tmux", "screen", "watch", "tail -f", "docker attach",
		"ssh", "telnet", "ftp", "mysql", "psql", "redis-cli",
		"mongo", "irb", "bash", "sh", "zsh", "fish", "tcsh",
		"csh", "lazygit", "tig", "mc", "ranger", "nnn", "ncdu",
	}

	// Commands that are interactive only when run without arguments
	replCommands := []string{
		"python", "python3", "node", "ruby", "perl", "lua",
		"php", "r", "julia", "scala", "clojure",
	}

	// Check if the command is an exact match for REPL commands
	for _, repl := range replCommands {
		if cmd == repl {
			return true
		}
	}

	// Check if the command starts with any interactive command
	for _, ic := range interactiveCommands {
		if cmd == ic {
			return true
		}
		// Check if command starts with the interactive command
		if len(cmd) > len(ic) && cmd[:len(ic)] == ic && (cmd[len(ic)] == ' ' || cmd[len(ic)] == '\t') {
			return true
		}
	}

	// Check for docker exec with -it flag
	if contains(cmd, "docker exec") && (contains(cmd, " -it ") || contains(cmd, " -ti ")) {
		return true
	}

	// Check for common interactive flags in other commands
	if contains(cmd, " -it ") || (contains(cmd, " -i ") && contains(cmd, " -t ")) {
		return true
	}

	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
