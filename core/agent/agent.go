package agent

import (
	"context"
	"time"
)

// Agent is the interface for the AI agent that processes natural language queries
type Agent interface {
	// ProcessQuery processes a natural language query and returns command suggestions
	ProcessQuery(ctx context.Context, query string) (*CommandSuggestion, error)
	
	// AddExecutionResult adds a command execution result to the context
	AddExecutionResult(cmd string, output string, exitCode int)
	
	// ClearContext clears the conversation context
	ClearContext()
	
	// GetContext returns the current context
	GetContext() *Context
}

// CommandSuggestion represents a command suggestion from the AI
type CommandSuggestion struct {
	Command      string          `json:"command"`       // The suggested command
	Explanation  string          `json:"explanation"`   // Explanation of what the command does
	Confidence   float64         `json:"confidence"`    // Confidence score (0-1)
	Risks        []SecurityRisk  `json:"risks"`         // Potential security risks
	Alternatives []string        `json:"alternatives"`  // Alternative commands
}

// SecurityRisk represents a potential security risk in a command
type SecurityRisk struct {
	Level       RiskLevel `json:"level"`       // Risk level
	Description string    `json:"description"` // Description of the risk
	Mitigation  string    `json:"mitigation"`  // How to mitigate the risk
}

// RiskLevel represents the severity of a security risk
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
	Query     string            `json:"query"`      // User query
	Response  CommandSuggestion `json:"response"`   // Agent response
	Timestamp time.Time         `json:"timestamp"`  // When this turn occurred
}

// ExecutionRecord represents a record of command execution
type ExecutionRecord struct {
	Command   string    `json:"command"`    // The executed command
	Output    string    `json:"output"`     // Command output
	ExitCode  int       `json:"exit_code"`  // Exit code
	Timestamp time.Time `json:"timestamp"`  // When executed
}

// SystemInfo contains system information for context
type SystemInfo struct {
	OS             string            `json:"os"`               // Operating system
	Shell          string            `json:"shell"`            // Current shell
	WorkingDir     string            `json:"working_dir"`      // Current working directory
	EnvironmentVars map[string]string `json:"environment_vars"` // Relevant environment variables
	Username       string            `json:"username"`         // Current user
	Hostname       string            `json:"hostname"`         // System hostname
}

// Context represents the conversation and execution context
type Context struct {
	History      []ConversationTurn `json:"history"`       // Conversation history
	Executions   []ExecutionRecord  `json:"executions"`    // Command execution history
	SystemInfo   SystemInfo         `json:"system_info"`   // System information
	MaxHistorySize int              `json:"max_history"`   // Maximum history size
	MaxTokens    int                `json:"max_tokens"`    // Maximum context tokens
}

// Config represents the agent configuration
type Config struct {
	Provider       string            `json:"provider"`        // LLM provider to use
	Model          string            `json:"model"`           // Model to use
	Temperature    float64           `json:"temperature"`     // Temperature for generation
	MaxTokens      int               `json:"max_tokens"`      // Max tokens in response
	SystemPrompt   string            `json:"system_prompt"`   // System prompt
	SafetyEnabled  bool              `json:"safety_enabled"`  // Enable safety checks
	ContextSize    int               `json:"context_size"`    // Context window size
	CustomPrompts  map[string]string `json:"custom_prompts"`  // Custom prompts
}