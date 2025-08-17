package agent

import (
	"context"
	"os"
	"os/user"
	"runtime"
	"sync"
	"time"

	"github.com/yourusername/clia/core/provider"
)

// defaultAgent is the default implementation of the Agent interface
type defaultAgent struct {
	config   *Config
	context  *Context
	provider provider.Provider
	prompt   *PromptBuilder
	safety   *SafetyChecker
	parser   *ResponseParser
	mu       sync.RWMutex
}

// NewAgent creates a new agent with the given configuration
func NewAgentImpl(config *Config, provider provider.Provider) Agent {
	return &defaultAgent{
		config:   config,
		context:  newContext(config),
		provider: provider,
		prompt:   NewPromptBuilder(config),
		safety:   NewSafetyChecker(config),
		parser:   NewResponseParser(),
	}
}

// ProcessQuery processes a natural language query and returns command suggestions
func (a *defaultAgent) ProcessQuery(ctx context.Context, query string) (*CommandSuggestion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Build prompt with context
	prompt, err := a.prompt.Build(query, a.context)
	if err != nil {
		return nil, err
	}

	// Call LLM provider
	req := &provider.ChatRequest{
		Model: a.config.Model,
		Messages: []provider.Message{
			{Role: "system", Content: a.config.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Options: &provider.ChatOptions{
			Temperature: &a.config.Temperature,
			MaxTokens:   &a.config.MaxTokens,
		},
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse response
	suggestion, err := a.parser.Parse(resp.Content)
	if err != nil {
		// Fallback to basic parsing
		suggestion = &CommandSuggestion{
			Command:     resp.Content,
			Explanation: "Command suggested by AI",
			Confidence:  0.5,
		}
	}

	// Perform safety checks if enabled
	if a.config.SafetyEnabled {
		risks := a.safety.CheckCommand(suggestion.Command)
		suggestion.Risks = risks
		
		// Adjust confidence based on risks
		for _, risk := range risks {
			if risk.Level == RiskCritical {
				suggestion.Confidence *= 0.3
			} else if risk.Level == RiskHigh {
				suggestion.Confidence *= 0.5
			}
		}
	}

	// Add to history
	turn := ConversationTurn{
		Query:     query,
		Response:  *suggestion,
		Timestamp: time.Now(),
	}
	a.context.AddTurn(turn)

	return suggestion, nil
}

// AddExecutionResult adds a command execution result to the context
func (a *defaultAgent) AddExecutionResult(cmd string, output string, exitCode int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	record := ExecutionRecord{
		Command:   cmd,
		Output:    output,
		ExitCode:  exitCode,
		Timestamp: time.Now(),
	}
	a.context.AddExecution(record)
}

// ClearContext clears the conversation context
func (a *defaultAgent) ClearContext() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.context.Clear()
}

// GetContext returns the current context
func (a *defaultAgent) GetContext() *Context {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.context
}

// newContext creates a new context with system information
func newContext(config *Config) *Context {
	ctx := &Context{
		History:        []ConversationTurn{},
		Executions:     []ExecutionRecord{},
		MaxHistorySize: config.ContextSize,
		MaxTokens:      config.MaxTokens,
	}

	// Populate system info
	ctx.SystemInfo = getSystemInfo()

	return ctx
}

// getSystemInfo gathers system information
func getSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:              runtime.GOOS,
		EnvironmentVars: make(map[string]string),
	}

	// Get shell
	if shell := os.Getenv("SHELL"); shell != "" {
		info.Shell = shell
	} else {
		info.Shell = "/bin/sh"
	}

	// Get working directory
	if wd, err := os.Getwd(); err == nil {
		info.WorkingDir = wd
	}

	// Get username
	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Get relevant environment variables
	relevantVars := []string{"PATH", "HOME", "USER", "LANG", "TERM"}
	for _, v := range relevantVars {
		if val := os.Getenv(v); val != "" {
			info.EnvironmentVars[v] = val
		}
	}

	return info
}

// Context methods

// AddTurn adds a conversation turn to the history
func (c *Context) AddTurn(turn ConversationTurn) {
	c.History = append(c.History, turn)
	
	// Trim history if exceeds limit
	if c.MaxHistorySize > 0 && len(c.History) > c.MaxHistorySize {
		c.History = c.History[len(c.History)-c.MaxHistorySize:]
	}
}

// AddExecution adds an execution record
func (c *Context) AddExecution(record ExecutionRecord) {
	c.Executions = append(c.Executions, record)
	
	// Keep only recent executions (same limit as history)
	if c.MaxHistorySize > 0 && len(c.Executions) > c.MaxHistorySize {
		c.Executions = c.Executions[len(c.Executions)-c.MaxHistorySize:]
	}
}

// Clear clears the context
func (c *Context) Clear() {
	c.History = []ConversationTurn{}
	c.Executions = []ExecutionRecord{}
}