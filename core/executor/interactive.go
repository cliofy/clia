package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// InteractiveDecision represents the result of interactive command detection
type InteractiveDecision struct {
	IsInteractive bool
	Confidence    float64 // 0.0 to 1.0
	Reason        string
	Method        string // "config", "hardcoded", "learned", "probe"
}

// InteractiveExecutor handles interactive/TUI commands
type InteractiveExecutor interface {
	// ExecuteInteractive runs an interactive command with full terminal control
	ExecuteInteractive(cmd string) error
	// ExecuteInteractiveWithCapture runs an interactive command and optionally captures the last frame
	ExecuteInteractiveWithCapture(cmd string, captureLastFrame bool) (string, error)
	// ExecuteInteractiveWithCaptureAndTimeout runs an interactive command with optional capture and timeout
	ExecuteInteractiveWithCaptureAndTimeout(cmd string, captureLastFrame bool, timeout time.Duration) (string, error)
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

// ExecuteInteractiveWithCapture runs a command with full PTY pass-through and optional screen capture
func (e *interactiveExecutor) ExecuteInteractiveWithCapture(cmd string, captureLastFrame bool) (string, error) {
	// Create command
	command := exec.Command(e.shell, "-c", cmd)

	// Start the command with a PTY
	ptmx, err := pty.Start(command)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
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
		return "", fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// Create a context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize screen capture if requested
	var screen *AnsiTermScreen
	var lastFrame string

	if captureLastFrame {
		// Get terminal size
		cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			// Fallback to default size
			cols, rows = 80, 24
		}
		screen = NewAnsiTermScreen(cols, rows)
	}

	// Copy stdin to PTY
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
		cancel()
	}()

	// Copy PTY to stdout with optional screen capture
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}

			// Write to stdout
			_, _ = os.Stdout.Write(buf[:n])

			// Update virtual screen if capturing
			if screen != nil {
				screen.ProcessOutput(buf[:n])

				// Check if we just exited alternate screen
				if screen.DetectedAltScreenExit() {
					lastFrame = screen.GetLastFrame()
				}
			}
		}
		cancel()
	}()

	// Wait for the command to complete or context to be done
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	var cmdErr error
	select {
	case cmdErr = <-done:
		// Command completed
	case <-ctx.Done():
		// Context cancelled
	}

	// Capture final frame if we haven't captured one yet and capturing is enabled
	if captureLastFrame && lastFrame == "" && screen != nil {
		lastFrame = screen.CaptureFrame()
	}

	return lastFrame, cmdErr
}

// ExecuteInteractiveWithCaptureAndTimeout runs a command with optional timeout and screen capture
func (e *interactiveExecutor) ExecuteInteractiveWithCaptureAndTimeout(cmd string, captureLastFrame bool, timeout time.Duration) (string, error) {
	// If no timeout is specified, use the regular method
	if timeout <= 0 {
		return e.ExecuteInteractiveWithCapture(cmd, captureLastFrame)
	}

	// For timeout mode, run in non-interactive capture mode
	return e.executeWithTimeout(cmd, captureLastFrame, timeout)
}

// executeWithTimeout implements the timeout-based execution
func (e *interactiveExecutor) executeWithTimeout(cmd string, captureLastFrame bool, timeout time.Duration) (string, error) {
	// Create command
	command := exec.Command(e.shell, "-c", cmd)

	// Start the command with a PTY
	ptmx, err := pty.Start(command)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	// Initialize screen capture if requested
	var screen *AnsiTermScreen
	var lastFrame string

	if captureLastFrame {
		// Get actual terminal size
		cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			// Fallback to default size
			cols, rows = 80, 24
		}

		// Set PTY size to match our virtual screen
		err = pty.Setsize(ptmx, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		})
		if err != nil {
			return "", fmt.Errorf("failed to set PTY size: %w", err)
		}

		screen = NewAnsiTermScreen(cols, rows)
	}

	// Create timeout timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Create context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture output
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					return
				}

				// Update virtual screen if capturing
				if screen != nil {
					screen.ProcessOutput(buf[:n])

					// Check if we just exited alternate screen
					if screen.DetectedAltScreenExit() {
						lastFrame = screen.GetLastFrame()
					}
				}
			}
		}
	}()

	// Wait for either timeout or command completion
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	var cmdErr error
	select {
	case cmdErr = <-done:
		// Command completed before timeout
	case <-timer.C:
		// Timeout reached - send quit signals
		if err := e.sendQuitSignals(ptmx, cmd); err != nil {
			// If graceful quit fails, kill the process
			if command.Process != nil {
				command.Process.Kill()
			}
		}

		// Wait a bit for graceful termination
		select {
		case cmdErr = <-done:
			// Command terminated gracefully
		case <-time.After(1 * time.Second):
			// Force kill if still running
			if command.Process != nil {
				command.Process.Kill()
				cmdErr = fmt.Errorf("command terminated due to timeout")
			}
		}
	}

	// Cancel context to stop output goroutine
	cancel()

	// Wait for output goroutine to finish
	<-outputDone

	// Capture final frame if we haven't captured one yet and capturing is enabled
	if captureLastFrame && screen != nil {
		if lastFrame == "" {
			// Get the final frame - AnsiTermScreen handles this automatically
			if altExitFrame := screen.GetLastFrame(); altExitFrame != "" {
				lastFrame = altExitFrame
			} else {
				lastFrame = screen.CaptureFrame()
			}
		}
	}

	return lastFrame, cmdErr
}

// sendQuitSignals sends appropriate quit signals to terminate TUI programs
func (e *interactiveExecutor) sendQuitSignals(ptmx *os.File, cmd string) error {
	// Determine the appropriate quit signal based on the command
	quitSignals := e.getQuitSignalsForCommand(cmd)

	for _, signal := range quitSignals {
		if _, err := ptmx.Write([]byte(signal)); err == nil {
			// Wait a bit for the command to process the signal
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// getQuitSignalsForCommand returns appropriate quit signals for different commands
func (e *interactiveExecutor) getQuitSignalsForCommand(cmd string) []string {
	cmdLower := strings.ToLower(cmd)

	// Extract the base command
	cmdFields := strings.Fields(cmd)
	if len(cmdFields) == 0 {
		return []string{"\x03"} // Ctrl+C as fallback
	}

	baseCmd := strings.ToLower(cmdFields[0])

	switch baseCmd {
	case "top", "htop", "btop":
		return []string{"q", "\x03"} // 'q' then Ctrl+C
	case "less", "more":
		return []string{"q", "\x03"} // 'q' then Ctrl+C
	case "vim", "vi", "nvim":
		return []string{"\x1b", ":q!\r", "\x03"} // ESC, :q!, then Ctrl+C
	case "nano":
		return []string{"\x03", "\x18"} // Ctrl+C, then Ctrl+X
	case "emacs":
		return []string{"\x03\x03"} // Ctrl+C Ctrl+C (keyboard-quit)
	case "watch":
		return []string{"\x03"} // Ctrl+C
	case "tail":
		if strings.Contains(cmdLower, "-f") {
			return []string{"\x03"} // Ctrl+C for tail -f
		}
		return []string{"\x03"}
	case "ssh", "telnet":
		return []string{"~.", "\x03"} // SSH escape, then Ctrl+C
	case "docker":
		if strings.Contains(cmdLower, "attach") {
			return []string{"\x10\x11", "\x03"} // Ctrl+P Ctrl+Q, then Ctrl+C
		}
		return []string{"\x03"}
	case "mysql", "psql", "redis-cli", "mongo":
		return []string{"exit\r", "quit\r", "\x03"} // Database exit commands, then Ctrl+C
	case "python", "python3", "node", "ruby", "irb":
		return []string{"exit()\r", "quit()\r", "\x04", "\x03"} // REPL exit, Ctrl+D, Ctrl+C
	case "btm":
		return []string{"q", "\x03"} // 'q' then Ctrl+C
	default:
		// Generic quit signals
		return []string{"q", "\x1b", "\x03"} // 'q', ESC, then Ctrl+C
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
	result := IsInteractiveCommandWithConfig(cmd, nil)
	return result.IsInteractive
}

// IsInteractiveCommandWithConfig checks if a command requires interactive mode using config
func IsInteractiveCommandWithConfig(cmd string, cfg interface{}) *InteractiveDecision {
	// Priority order for detection:
	// 1. User config (always/never lists and patterns)
	// 2. Learned commands
	// 3. Hardcoded known commands
	// 4. Dynamic probing

	// 1. Check user configuration first
	if config, ok := cfg.(interface{ IsInteractiveCommand(string) (bool, bool) }); ok {
		if isInteractive, hasAnswer := config.IsInteractiveCommand(cmd); hasAnswer {
			return &InteractiveDecision{
				IsInteractive: isInteractive,
				Confidence:    1.0,
				Reason:        "defined in user configuration",
				Method:        "config",
			}
		}
	}

	// 2. Check learned commands (only use if high confidence)
	if result := CheckLearnedCommands(cmd); result != nil && result.Confidence >= 0.9 {
		return &InteractiveDecision{
			IsInteractive: result.IsInteractive,
			Confidence:    result.Confidence,
			Reason:        result.Reason,
			Method:        "learned",
		}
	}

	// 3. Check hardcoded known commands
	if result := checkHardcodedCommands(cmd); result != nil {
		return result
	}

	// 4. Try dynamic probing as last resort with high confidence threshold
	if result, err := ProbeInteractive(cmd); err == nil && result.Confidence > 0.85 {
		return &InteractiveDecision{
			IsInteractive: result.IsInteractive,
			Confidence:    result.Confidence,
			Reason:        result.Reason,
			Method:        "probe",
		}
	}

	// Default: assume non-interactive
	return &InteractiveDecision{
		IsInteractive: false,
		Confidence:    0.5,
		Reason:        "no interactive indicators found",
		Method:        "default",
	}
}

// checkHardcodedCommands checks against the built-in list of known commands
func checkHardcodedCommands(cmd string) *InteractiveDecision {
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

	// Check if the command is an exact match for REPL commands (no arguments)
	cmdFields := strings.Fields(cmd)
	if len(cmdFields) == 1 {
		cmdBase := cmdFields[0]
		for _, repl := range replCommands {
			if cmdBase == repl {
				return &InteractiveDecision{
					IsInteractive: true,
					Confidence:    0.95,
					Reason:        fmt.Sprintf("REPL command: %s", repl),
					Method:        "hardcoded",
				}
			}
		}
	}

	// Check if the command starts with any interactive command
	for _, ic := range interactiveCommands {
		if cmd == ic {
			return &InteractiveDecision{
				IsInteractive: true,
				Confidence:    0.95,
				Reason:        fmt.Sprintf("known interactive command: %s", ic),
				Method:        "hardcoded",
			}
		}
		// Check if command starts with the interactive command
		if len(cmd) > len(ic) && cmd[:len(ic)] == ic && (cmd[len(ic)] == ' ' || cmd[len(ic)] == '\t') {
			return &InteractiveDecision{
				IsInteractive: true,
				Confidence:    0.9,
				Reason:        fmt.Sprintf("starts with interactive command: %s", ic),
				Method:        "hardcoded",
			}
		}
	}

	// Check for docker exec with -it flag
	if contains(cmd, "docker exec") && (contains(cmd, " -it ") || contains(cmd, " -ti ")) {
		return &InteractiveDecision{
			IsInteractive: true,
			Confidence:    0.95,
			Reason:        "docker exec with -it flag",
			Method:        "hardcoded",
		}
	}

	// Check for common interactive flags in other commands
	if contains(cmd, " -it ") || (contains(cmd, " -i ") && contains(cmd, " -t ")) {
		return &InteractiveDecision{
			IsInteractive: true,
			Confidence:    0.85,
			Reason:        "command with interactive flags",
			Method:        "hardcoded",
		}
	}

	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
