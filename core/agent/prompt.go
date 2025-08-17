package agent

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// PromptBuilder builds prompts for the LLM
type PromptBuilder struct {
	config         *Config
	systemTemplate *template.Template
	userTemplate   *template.Template
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(config *Config) *PromptBuilder {
	pb := &PromptBuilder{
		config: config,
	}

	// Initialize templates
	pb.initTemplates()

	return pb
}

// initTemplates initializes the prompt templates
func (pb *PromptBuilder) initTemplates() {
	// System prompt template
	systemPrompt := `You are CLIA (Command Line Intelligence Assistant), an expert at converting natural language queries to shell commands.

IMPORTANT RULES:
1. Always provide commands that are safe and appropriate for the user's system
2. Include explanations for complex commands
3. Consider the user's operating system ({{.OS}}) and shell ({{.Shell}})
4. Be aware of the current working directory: {{.WorkingDir}}
5. Provide alternatives when multiple approaches exist
6. Identify and warn about potentially dangerous operations

OUTPUT FORMAT:
Respond with a JSON object containing:
{
  "command": "the shell command to execute",
  "explanation": "brief explanation of what the command does",
  "confidence": 0.0-1.0,
  "risks": [{"level": "low|medium|high|critical", "description": "risk description", "mitigation": "how to mitigate"}],
  "alternatives": ["alternative command 1", "alternative command 2"]
}`

	// User prompt template
	userPrompt := `System Information:
- OS: {{.SystemInfo.OS}}
- Shell: {{.SystemInfo.Shell}}
- Working Directory: {{.SystemInfo.WorkingDir}}
- User: {{.SystemInfo.Username}}
- Hostname: {{.SystemInfo.Hostname}}

{{if .RecentCommands}}Recent Commands:
{{range .RecentCommands}}- {{.Command}} (exit code: {{.ExitCode}})
  Output: {{.Output | truncate 100}}
{{end}}
{{end}}

{{if .RecentQueries}}Recent Queries:
{{range .RecentQueries}}- Query: {{.Query}}
  Command: {{.Response.Command}}
{{end}}
{{end}}

User Query: {{.Query}}

Please provide a shell command to accomplish this task.`

	var err error
	pb.systemTemplate, err = template.New("system").Parse(systemPrompt)
	if err != nil {
		// Fallback to simple template
		pb.systemTemplate, _ = template.New("system").Parse("You are a command line assistant.")
	}

	// Add template functions
	funcMap := template.FuncMap{
		"truncate": truncateString,
	}

	pb.userTemplate, err = template.New("user").Funcs(funcMap).Parse(userPrompt)
	if err != nil {
		// Fallback to simple template
		pb.userTemplate, _ = template.New("user").Parse("Convert to command: {{.Query}}")
	}
}

// Build builds a prompt from the query and context
func (pb *PromptBuilder) Build(query string, ctx *Context) (string, error) {
	// Prepare template data
	data := struct {
		Query          string
		SystemInfo     SystemInfo
		RecentCommands []ExecutionRecord
		RecentQueries  []ConversationTurn
	}{
		Query:      query,
		SystemInfo: ctx.SystemInfo,
	}

	// Add recent commands (last 5)
	if len(ctx.Executions) > 0 {
		start := 0
		if len(ctx.Executions) > 5 {
			start = len(ctx.Executions) - 5
		}
		data.RecentCommands = ctx.Executions[start:]
	}

	// Add recent queries (last 3)
	if len(ctx.History) > 0 {
		start := 0
		if len(ctx.History) > 3 {
			start = len(ctx.History) - 3
		}
		data.RecentQueries = ctx.History[start:]
	}

	// Execute template
	var buf bytes.Buffer
	if err := pb.userTemplate.Execute(&buf, data); err != nil {
		// Fallback to simple prompt
		return fmt.Sprintf("Convert this to a shell command: %s", query), nil
	}

	return buf.String(), nil
}

// GetSystemPrompt returns the system prompt
func (pb *PromptBuilder) GetSystemPrompt() string {
	if pb.config.SystemPrompt != "" {
		return pb.config.SystemPrompt
	}

	// Build default system prompt
	data := struct {
		OS         string
		Shell      string
		WorkingDir string
	}{
		OS:         "unix",
		Shell:      "/bin/bash",
		WorkingDir: ".",
	}

	var buf bytes.Buffer
	if pb.systemTemplate != nil {
		pb.systemTemplate.Execute(&buf, data)
		return buf.String()
	}

	return "You are a command line assistant."
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// AddExample adds an example to the prompt builder
func (pb *PromptBuilder) AddExample(query, command, explanation string) {
	// This can be used to add few-shot examples
	// Implementation depends on how we want to manage examples
}

// OptimizeForModel optimizes the prompt for specific models
func (pb *PromptBuilder) OptimizeForModel(model string) {
	// Different models may need different prompt formats
	// For example, Claude prefers XML tags, GPT prefers JSON
	if strings.Contains(model, "claude") {
		// Optimize for Claude
		pb.config.SystemPrompt = strings.ReplaceAll(pb.config.SystemPrompt, "```json", "<json>")
		pb.config.SystemPrompt = strings.ReplaceAll(pb.config.SystemPrompt, "```", "</json>")
	}
}