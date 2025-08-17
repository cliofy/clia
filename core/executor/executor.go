package executor

import (
	"time"
)

// Executor defines the interface for command execution
type Executor interface {
	// Execute runs a command and returns the result
	Execute(cmd string) (*Result, error)
	
	// ExecuteWithTimeout runs a command with a timeout
	ExecuteWithTimeout(cmd string, timeout time.Duration) (*Result, error)
}

// Result contains the output and status of an executed command
type Result struct {
	Output   string    // Combined stdout and stderr output
	ExitCode int       // Exit code of the command
	Error    error     // Any error that occurred during execution
	Duration time.Duration // How long the command took to execute
}