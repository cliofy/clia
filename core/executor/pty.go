package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

// ptyExecutor implements Executor using pseudo-terminal
type ptyExecutor struct {
	shell string
}

// NewExecutor creates a new PTY-based executor
func NewExecutor() Executor {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return &ptyExecutor{
		shell: shell,
	}
}

// Execute runs a command in a PTY and captures its output
func (e *ptyExecutor) Execute(cmd string) (*Result, error) {
	return e.ExecuteWithTimeout(cmd, 30*time.Second) // Default 30 second timeout
}

// ExecuteWithTimeout runs a command with a specified timeout
func (e *ptyExecutor) ExecuteWithTimeout(cmd string, timeout time.Duration) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create command with context
	command := exec.CommandContext(ctx, e.shell, "-c", cmd)
	
	// Start command with PTY
	ptmx, err := pty.Start(command)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	defer ptmx.Close()

	// Set PTY size (standard terminal size)
	if err := pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
		X:    0,
		Y:    0,
	}); err != nil {
		// Non-fatal error, continue
		_ = err
	}

	// Capture output
	var output bytes.Buffer
	outputDone := make(chan error, 1)

	go func() {
		_, err := io.Copy(&output, ptmx)
		outputDone <- err
	}()

	startTime := time.Now()
	
	// Wait for command to complete
	cmdErr := command.Wait()
	duration := time.Since(startTime)
	
	// Wait for output capture to complete (with small timeout)
	select {
	case <-outputDone:
	case <-time.After(100 * time.Millisecond):
		// Output capture timeout, continue
	}

	// Determine exit code
	exitCode := 0
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			// Command was killed due to timeout
			exitCode = -1
			cmdErr = fmt.Errorf("command timed out after %v", timeout)
		} else {
			exitCode = -1
		}
	}

	result := &Result{
		Output:   output.String(),
		ExitCode: exitCode,
		Error:    cmdErr,
		Duration: duration,
	}

	return result, nil
}