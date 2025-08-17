package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

// ProbeResult contains the result of interactive probing
type ProbeResult struct {
	IsInteractive bool
	Confidence    float64 // 0.0 to 1.0
	Reason        string
}

// ProbeInteractive dynamically detects if a command needs interactive mode
func ProbeInteractive(cmd string) (*ProbeResult, error) {
	// Quick checks first
	if result := quickProbeChecks(cmd); result != nil {
		return result, nil
	}

	// Try to run the command briefly to detect interactive behavior
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Extract the actual command (first word)
	cmdParts := strings.Fields(cmd)
	if len(cmdParts) == 0 {
		return &ProbeResult{IsInteractive: false, Confidence: 1.0, Reason: "empty command"}, nil
	}

	// Create command with context
	command := exec.CommandContext(ctx, "sh", "-c", cmd)
	
	// Start with PTY to see if the program expects it
	ptmx, err := pty.Start(command)
	if err != nil {
		// If PTY fails, it might be a non-existent command
		return &ProbeResult{IsInteractive: false, Confidence: 0.5, Reason: "failed to start with PTY"}, nil
	}
	defer ptmx.Close()

	// Capture initial output
	outputBuf := make([]byte, 4096)
	outputChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	go func() {
		n, err := ptmx.Read(outputBuf)
		if err != nil {
			errorChan <- err
			return
		}
		outputChan <- outputBuf[:n]
	}()

	// Wait for either output or timeout
	select {
	case output := <-outputChan:
		result := analyzeOutput(output, cmd)
		return result, nil
	case <-ctx.Done():
		// Command didn't produce output quickly - might be waiting for input
		command.Process.Kill()
		return &ProbeResult{
			IsInteractive: true,
			Confidence:    0.7,
			Reason:        "no immediate output, likely waiting for input",
		}, nil
	case err := <-errorChan:
		// Read error might indicate the program terminated
		if err.Error() == "EOF" {
			return &ProbeResult{IsInteractive: false, Confidence: 0.8, Reason: "program terminated quickly"}, nil
		}
		return &ProbeResult{IsInteractive: false, Confidence: 0.5, Reason: fmt.Sprintf("read error: %v", err)}, nil
	}
}

// quickProbeChecks performs fast heuristic checks without running the command
func quickProbeChecks(cmd string) *ProbeResult {
	cmdLower := strings.ToLower(cmd)
	
	// Check for terminal multiplexers and known TUI patterns
	tuiPatterns := []string{
		"zellij", "wezterm", "alacritty", "kitty", "iterm",
		"-tui", "_tui", "-cli", "-repl", "-interactive",
		"--interactive", "--tty",
	}
	
	for _, pattern := range tuiPatterns {
		if strings.Contains(cmdLower, pattern) {
			return &ProbeResult{
				IsInteractive: true,
				Confidence:    0.9,
				Reason:        fmt.Sprintf("contains TUI pattern: %s", pattern),
			}
		}
	}

	// Check if piped or redirected (non-interactive)
	if strings.Contains(cmd, "|") || strings.Contains(cmd, ">") || strings.Contains(cmd, "<") {
		return &ProbeResult{
			IsInteractive: false,
			Confidence:    0.95,
			Reason:        "contains pipe or redirection",
		}
	}

	// Check for background execution
	if strings.HasSuffix(strings.TrimSpace(cmd), "&") {
		return &ProbeResult{
			IsInteractive: false,
			Confidence:    1.0,
			Reason:        "background execution",
		}
	}

	return nil
}

// analyzeOutput analyzes the initial output to determine if it's interactive
func analyzeOutput(output []byte, cmd string) *ProbeResult {
	outputStr := string(output)
	
	// Strong indicators of interactive programs
	interactiveIndicators := []struct {
		pattern    string
		confidence float64
		reason     string
	}{
		{"\033[2J", 0.95, "clear screen sequence"},           // Clear screen
		{"\033[H", 0.9, "cursor home sequence"},              // Cursor home
		{"\033[?1049h", 0.95, "alternate screen buffer"},     // Alt screen buffer
		{"\033[?25l", 0.85, "hide cursor"},                   // Hide cursor
		{"\033[?1000h", 0.9, "mouse tracking enabled"},       // Mouse tracking
		{"\033]0;", 0.8, "terminal title setting"},           // Set terminal title
		{"\033[?47h", 0.9, "alternate screen"},               // Alternate screen
		{"\033[m", 0.6, "color/style reset"},                 // Reset attributes
		{"\r\033[K", 0.7, "line clearing pattern"},           // Clear line pattern
	}

	// Count ANSI sequences
	ansiCount := strings.Count(outputStr, "\033[")
	
	// High density of ANSI sequences indicates TUI
	if ansiCount > 5 {
		return &ProbeResult{
			IsInteractive: true,
			Confidence:    0.85,
			Reason:        fmt.Sprintf("high ANSI sequence density (%d sequences)", ansiCount),
		}
	}

	// Check for specific interactive indicators
	for _, indicator := range interactiveIndicators {
		if strings.Contains(outputStr, indicator.pattern) {
			return &ProbeResult{
				IsInteractive: true,
				Confidence:    indicator.confidence,
				Reason:        indicator.reason,
			}
		}
	}

	// Check for prompts
	promptPatterns := []string{
		">>> ", ">> ", "> ",  // Python, R, etc.
		"$ ", "# ",           // Shell prompts
		": ", "=> ",          // Ruby, other REPLs
		"mysql> ", "postgres=#", // Database clients
	}

	for _, prompt := range promptPatterns {
		if strings.HasSuffix(strings.TrimSpace(outputStr), prompt) {
			return &ProbeResult{
				IsInteractive: true,
				Confidence:    0.8,
				Reason:        fmt.Sprintf("detected prompt: %s", prompt),
			}
		}
	}

	// If output is very short or empty, might be waiting for input
	if len(output) < 10 {
		return &ProbeResult{
			IsInteractive: true,
			Confidence:    0.6,
			Reason:        "minimal output, possibly waiting for input",
		}
	}

	// Default: assume non-interactive
	return &ProbeResult{
		IsInteractive: false,
		Confidence:    0.7,
		Reason:        "no interactive indicators detected",
	}
}

// CheckBinaryType uses the file command to check if a binary is likely interactive
func CheckBinaryType(cmdPath string) (*ProbeResult, error) {
	// This is optional and can be called for additional validation
	cmd := exec.Command("file", "-L", cmdPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	outputStr := strings.ToLower(string(output))
	
	// Check for libraries that indicate TUI programs
	if strings.Contains(outputStr, "ncurses") ||
		strings.Contains(outputStr, "terminfo") ||
		strings.Contains(outputStr, "readline") {
		return &ProbeResult{
			IsInteractive: true,
			Confidence:    0.75,
			Reason:        "linked with terminal libraries",
		}, nil
	}

	return &ProbeResult{
		IsInteractive: false,
		Confidence:    0.5,
		Reason:        "no terminal libraries detected",
	}, nil
}

// LearnInteractiveCommand records a command as interactive for future reference
func LearnInteractiveCommand(cmd string, isInteractive bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cliaDir := fmt.Sprintf("%s/.clia", home)
	if err := os.MkdirAll(cliaDir, 0755); err != nil {
		return err
	}

	// Store learned commands in a simple text file
	learnedFile := fmt.Sprintf("%s/learned_interactive.txt", cliaDir)
	
	// Read existing entries
	existing := make(map[string]bool)
	if data, err := os.ReadFile(learnedFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				existing[parts[0]] = parts[1] == "true"
			}
		}
	}

	// Update with new knowledge
	cmdBase := strings.Fields(cmd)[0] // Just the command name
	existing[cmdBase] = isInteractive

	// Write back
	var buffer bytes.Buffer
	for cmd, interactive := range existing {
		fmt.Fprintf(&buffer, "%s:%v\n", cmd, interactive)
	}

	return os.WriteFile(learnedFile, buffer.Bytes(), 0644)
}

// CheckLearnedCommands checks if we've learned about this command before
func CheckLearnedCommands(cmd string) *ProbeResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	learnedFile := fmt.Sprintf("%s/.clia/learned_interactive.txt", home)
	data, err := os.ReadFile(learnedFile)
	if err != nil {
		return nil
	}

	cmdBase := strings.Fields(cmd)[0] // Just the command name
	lines := strings.Split(string(data), "\n")
	
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == cmdBase {
			isInteractive := parts[1] == "true"
			return &ProbeResult{
				IsInteractive: isInteractive,
				Confidence:    1.0,
				Reason:        "learned from previous usage",
			}
		}
	}

	return nil
}