package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Executor handles command execution with timeout and platform support
type Executor struct {
	timeout    time.Duration
	workDir    string
	shell      string
	env        []string
	ptyEnabled bool // Enable PTY support for interactive programs
}

// ExecutionResult contains the result of a command execution
type ExecutionResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
	Pid      int           `json:"pid,omitempty"`
}

// OutputLine represents a single line of output from a command
type OutputLine struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsStderr  bool      `json:"is_stderr"`
}

// New creates a new Executor with default settings
func New() *Executor {
	return &Executor{
		timeout:    30 * time.Second,
		workDir:    getCurrentDir(),
		shell:      detectShell(),
		env:        os.Environ(),
		ptyEnabled: true, // Enable PTY support by default
	}
}

// WithTimeout sets the execution timeout
func (e *Executor) WithTimeout(timeout time.Duration) *Executor {
	e.timeout = timeout
	return e
}

// WithWorkDir sets the working directory
func (e *Executor) WithWorkDir(dir string) *Executor {
	e.workDir = dir
	return e
}

// WithShell sets the shell to use
func (e *Executor) WithShell(shell string) *Executor {
	e.shell = shell
	return e
}

// WithEnv sets environment variables
func (e *Executor) WithEnv(env []string) *Executor {
	e.env = env
	return e
}

// WithPTY enables or disables PTY support
func (e *Executor) WithPTY(enabled bool) *Executor {
	e.ptyEnabled = enabled
	return e
}

// IsPTYEnabled returns whether PTY support is enabled
func (e *Executor) IsPTYEnabled() bool {
	return e.ptyEnabled
}

// Execute runs a command synchronously and returns the complete result
func (e *Executor) Execute(ctx context.Context, command string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Prepare command
	cmd, err := e.prepareCommand(timeoutCtx, command)
	if err != nil {
		return &ExecutionResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to prepare command: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &ExecutionResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to create stdout pipe: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &ExecutionResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to create stderr pipe: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return &ExecutionResult{
			Command:  command,
			ExitCode: -1,
			Error:    fmt.Errorf("failed to start command: %w", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Read output
	stdoutData, err1 := readAll(stdout)
	stderrData, err2 := readAll(stderr)

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

	result := &ExecutionResult{
		Command:  command,
		ExitCode: exitCode,
		Stdout:   string(stdoutData),
		Stderr:   string(stderrData),
		Duration: duration,
		Pid:      cmd.Process.Pid,
	}

	// Handle errors
	if err1 != nil {
		result.Error = fmt.Errorf("failed to read stdout: %w", err1)
		return result, err1
	}
	if err2 != nil {
		result.Error = fmt.Errorf("failed to read stderr: %w", err2)
		return result, err2
	}
	if execErr != nil && exitCode != 0 {
		result.Error = fmt.Errorf("command failed with exit code %d: %w", exitCode, execErr)
		return result, execErr
	}

	return result, nil
}

// Stream runs a command and returns a channel of output lines
func (e *Executor) Stream(ctx context.Context, command string) (<-chan OutputLine, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)

	// Prepare command
	cmd, err := e.prepareCommand(timeoutCtx, command)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to prepare command: %w", err)
	}

	// Create output channel
	outputChan := make(chan OutputLine, 100) // Buffered channel

	// Setup pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Start goroutines to read output
	go e.streamReader(stdout, outputChan, false, cancel)
	go e.streamReader(stderr, outputChan, true, cancel)

	// Wait for command completion in background
	go func() {
		defer cancel()
		defer close(outputChan)

		err := cmd.Wait()
		if err != nil {
			outputChan <- OutputLine{
				Content:   fmt.Sprintf("Command failed: %v", err),
				Timestamp: time.Now(),
				IsStderr:  true,
			}
		}
	}()

	return outputChan, nil
}

// streamReader reads from a pipe and sends lines to the output channel
func (e *Executor) streamReader(pipe interface {
	Read([]byte) (int, error)
}, outputChan chan<- OutputLine, isStderr bool, cancel context.CancelFunc) {
	defer func() {
		if r := recover(); r != nil {
			outputChan <- OutputLine{
				Content:   fmt.Sprintf("Stream reader error: %v", r),
				Timestamp: time.Now(),
				IsStderr:  true,
			}
		}
	}()

	buf := make([]byte, 4096)
	leftover := ""

	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			data := leftover + string(buf[:n])
			lines := strings.Split(data, "\n")

			// Process all complete lines
			for i := 0; i < len(lines)-1; i++ {
				if lines[i] != "" || i == 0 { // Include empty lines except pure separators
					select {
					case outputChan <- OutputLine{
						Content:   lines[i],
						Timestamp: time.Now(),
						IsStderr:  isStderr,
					}:
					default:
						// Channel is full, skip this line to prevent blocking
					}
				}
			}

			// Keep the last incomplete line for next iteration
			leftover = lines[len(lines)-1]
		}

		if err != nil {
			// Send any remaining data
			if leftover != "" {
				select {
				case outputChan <- OutputLine{
					Content:   leftover,
					Timestamp: time.Now(),
					IsStderr:  isStderr,
				}:
				default:
					// Channel full
				}
			}

			if err.Error() != "EOF" {
				select {
				case outputChan <- OutputLine{
					Content:   fmt.Sprintf("Read error: %v", err),
					Timestamp: time.Now(),
					IsStderr:  true,
				}:
				default:
					// Channel full
				}
			}
			break
		}
	}
}

// prepareCommand creates and configures the exec.Cmd
func (e *Executor) prepareCommand(ctx context.Context, command string) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	// Platform-specific command preparation
	switch runtime.GOOS {
	case "windows":
		// Use cmd or powershell on Windows
		if strings.Contains(e.shell, "powershell") || strings.Contains(e.shell, "pwsh") {
			cmd = exec.CommandContext(ctx, e.shell, "-NoProfile", "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		}
	default:
		// Use shell on Unix-like systems
		shellPath := e.shell
		if shellPath == "" {
			shellPath = "/bin/sh"
		}
		cmd = exec.CommandContext(ctx, shellPath, "-c", command)
	}

	// Set working directory
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}

	// Set environment
	if len(e.env) > 0 {
		cmd.Env = e.env
	}

	return cmd, nil
}

// detectShell detects the appropriate shell for the current platform
func detectShell() string {
	switch runtime.GOOS {
	case "windows":
		// Check for PowerShell first, then fall back to cmd
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell.exe"
		}
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			return "pwsh.exe"
		}
		return "cmd.exe"
	default:
		// Unix-like systems: check SHELL environment variable
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell
		}

		// Check for common shells
		shells := []string{"/bin/bash", "/usr/bin/bash", "/bin/zsh", "/usr/bin/zsh", "/bin/sh"}
		for _, shell := range shells {
			if _, err := os.Stat(shell); err == nil {
				return shell
			}
		}

		return "/bin/sh" // fallback
	}
}

// getCurrentDir gets the current working directory
func getCurrentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}

// readAll reads all data from a reader
func readAll(r interface {
	Read([]byte) (int, error)
}) ([]byte, error) {
	const maxSize = 10 * 1024 * 1024 // 10MB limit
	buf := make([]byte, 0, 512)

	for {
		if len(buf) >= maxSize {
			return buf, fmt.Errorf("output exceeds maximum size limit of %d bytes", maxSize)
		}

		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]

		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return buf, err
		}

		if len(buf) == cap(buf) {
			// Need to grow buffer
			newBuf := make([]byte, len(buf), 2*cap(buf)+512)
			copy(newBuf, buf)
			buf = newBuf
		}
	}

	return buf, nil
}

// ExecuteWithPTY executes a command with PTY support if enabled and needed
func (e *Executor) ExecuteWithPTY(ctx context.Context, command string) (*ExecutionResult, error) {
	if !e.ptyEnabled {
		// PTY disabled, use regular execution
		return e.Execute(ctx, command)
	}

	// Create PTY executor and use it
	ptyExecutor := NewPTYExecutor()
	ptyExecutor.Executor = e // Use current executor settings

	return ptyExecutor.ExecuteWithAutoDetection(ctx, command)
}

// IsCommandSafe checks if a command is considered safe to execute
func IsCommandSafe(command string) bool {
	command = strings.TrimSpace(strings.ToLower(command))

	// List of safe read-only commands
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

// IsDangerousCommand checks if a command contains dangerous operations
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
		"sudo rm",
		"rm -rf",
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(command, dangerous) {
			return true
		}
	}

	return false
}
