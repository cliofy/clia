package executor

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// PTYExecutor extends the base executor with PTY support for interactive programs
type PTYExecutor struct {
	*Executor
	tuiPrograms map[string]bool
}

// PTYResult contains the result of a PTY command execution
type PTYResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
	Pid      int           `json:"pid,omitempty"`
}

// NewPTYExecutor creates a new PTY-enabled executor
func NewPTYExecutor() *PTYExecutor {
	baseExecutor := New()

	// Define programs that require PTY for proper operation
	tuiPrograms := map[string]bool{
		// Text editors
		"vim":   true,
		"vi":    true,
		"nano":  true,
		"emacs": true,
		"nvim":  true,

		// System monitors
		"htop":  true,
		"top":   true,
		"btop":  true,
		"atop":  true,
		"nmon":  true,
		"iotop": true,

		// Pagers
		"less": true,
		"more": true,

		// Terminal multiplexers
		"tmux":   true,
		"screen": true,

		// Git tools
		"tig":     true,
		"lazygit": true,
		"gitui":   true,

		// Database clients
		"mysql":     true,
		"psql":      true,
		"redis-cli": true,
		"mongo":     true,
		"sqlite3":   true,

		// REPLs and interpreters
		"python":  true,
		"python3": true,
		"node":    true,
		"irb":     true,
		"php":     true,
		"lua":     true,
		"ghci":    true,

		// File managers
		"mc":     true,
		"ranger": true,
		"nnn":    true,
		"lf":     true,

		// Network tools
		"ssh":    true,
		"telnet": true,

		// Other TUI programs
		"ncdu":      true,
		"nethogs":   true,
		"bandwhich": true,
		"cmatrix":   true,
		"sl":        true,
	}

	return &PTYExecutor{
		Executor:    baseExecutor,
		tuiPrograms: tuiPrograms,
	}
}

// IsTUIProgram checks if a command requires PTY for proper operation
func (e *PTYExecutor) IsTUIProgram(command string) bool {
	// Extract the base command name (first word)
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	// Handle command paths (e.g., /usr/bin/vim -> vim)
	if strings.Contains(cmdName, "/") {
		cmdParts := strings.Split(cmdName, "/")
		cmdName = cmdParts[len(cmdParts)-1]
	}

	return e.tuiPrograms[cmdName]
}

// ExecuteInteractive runs a command with PTY support for full terminal interaction
func (e *PTYExecutor) ExecuteInteractive(ctx context.Context, command string) (*PTYResult, error) {
	startTime := time.Now()

	// Prepare the command
	cmd, err := e.prepareCommand(ctx, command)
	if err != nil {
		return &PTYResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to prepare command: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Check if we're in a terminal environment
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Fallback to regular execution if not in a terminal
		log.Printf("Not in terminal environment, falling back to regular execution")
		result, err := e.Execute(ctx, command)
		if err != nil {
			return &PTYResult{
				Command:  command,
				ExitCode: result.ExitCode,
				Error:    err,
				Duration: result.Duration,
				Pid:      result.Pid,
			}, err
		}
		return &PTYResult{
			Command:  command,
			ExitCode: result.ExitCode,
			Duration: result.Duration,
			Pid:      result.Pid,
		}, nil
	}

	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return &PTYResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to make terminal raw: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Ensure terminal state is restored on exit
	defer func() {
		if restoreErr := term.Restore(int(os.Stdin.Fd()), oldState); restoreErr != nil {
			log.Printf("Warning: Failed to restore terminal state: %v", restoreErr)
		}
	}()

	// Create PTY and start command
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return &PTYResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to start command with PTY: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Ensure PTY is closed on exit
	defer func() {
		if closeErr := ptmx.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close PTY: %v", closeErr)
		}
	}()

	// Handle window size changes
	e.handleWindowResize(ptmx)

	// Handle input/output copying
	e.handleIO(ptmx)

	// Wait for command to complete
	execErr := cmd.Wait()
	duration := time.Since(startTime)

	// Determine exit code
	exitCode := 0
	if execErr != nil {
		if exitError, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	result := &PTYResult{
		Command:  command,
		ExitCode: exitCode,
		Duration: duration,
		Pid:      cmd.Process.Pid,
	}

	if execErr != nil && exitCode != 0 {
		result.Error = fmt.Errorf("command failed with exit code %d: %w", exitCode, execErr)
		return result, execErr
	}

	return result, nil
}

// handleWindowResize sets up window resize signal handling
func (e *PTYExecutor) handleWindowResize(ptmx *os.File) {
	// Create channel for window size change signals
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)

	// Handle resize in a goroutine
	go func() {
		defer signal.Stop(ch)
		defer close(ch)

		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("Warning: Failed to resize PTY: %v", err)
			}
		}
	}()

	// Set initial size
	ch <- syscall.SIGWINCH
}

// handleIO manages bidirectional I/O between terminal and PTY
func (e *PTYExecutor) handleIO(ptmx *os.File) {
	// Copy input from stdin to PTY (user input to program)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Warning: Input copy goroutine panic: %v", r)
			}
		}()

		if _, err := io.Copy(ptmx, os.Stdin); err != nil {
			log.Printf("Warning: Failed to copy stdin to PTY: %v", err)
		}
	}()

	// Copy output from PTY to stdout (program output to user)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Warning: Output copy goroutine panic: %v", r)
			}
		}()

		if _, err := io.Copy(os.Stdout, ptmx); err != nil {
			log.Printf("Warning: Failed to copy PTY to stdout: %v", err)
		}
	}()
}

// ExecuteWithAutoDetection automatically chooses between PTY and regular execution
func (e *PTYExecutor) ExecuteWithAutoDetection(ctx context.Context, command string) (*ExecutionResult, error) {
	// Check if command needs PTY
	if e.IsTUIProgram(command) {
		log.Printf("Detected TUI program, using PTY execution: %s", command)

		ptyResult, err := e.ExecuteInteractive(ctx, command)
		if err != nil {
			// Convert PTYResult to ExecutionResult for consistency
			return &ExecutionResult{
				Command:  ptyResult.Command,
				ExitCode: ptyResult.ExitCode,
				Duration: ptyResult.Duration,
				Error:    ptyResult.Error,
				Pid:      ptyResult.Pid,
				Stdout:   "", // PTY output goes directly to terminal
				Stderr:   "",
			}, err
		}

		return &ExecutionResult{
			Command:  ptyResult.Command,
			ExitCode: ptyResult.ExitCode,
			Duration: ptyResult.Duration,
			Pid:      ptyResult.Pid,
			Stdout:   "", // PTY output goes directly to terminal
			Stderr:   "",
		}, nil
	}

	// Use regular execution for non-TUI programs
	log.Printf("Using regular execution: %s", command)
	return e.Execute(ctx, command)
}

// AddTUIProgram adds a program to the TUI programs list
func (e *PTYExecutor) AddTUIProgram(program string) {
	e.tuiPrograms[program] = true
}

// RemoveTUIProgram removes a program from the TUI programs list
func (e *PTYExecutor) RemoveTUIProgram(program string) {
	delete(e.tuiPrograms, program)
}

// GetTUIPrograms returns a copy of the TUI programs map
func (e *PTYExecutor) GetTUIPrograms() map[string]bool {
	result := make(map[string]bool)
	for k, v := range e.tuiPrograms {
		result[k] = v
	}
	return result
}

// IsTerminalAvailable checks if we're running in a terminal environment
func IsTerminalAvailable() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd())) &&
		term.IsTerminal(int(os.Stderr.Fd()))
}

// GetTerminalSize returns the current terminal size
func GetTerminalSize() (width, height int, err error) {
	if !IsTerminalAvailable() {
		return 0, 0, fmt.Errorf("not running in a terminal")
	}

	width, height, err = term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get terminal size: %w", err)
	}

	return width, height, nil
}
