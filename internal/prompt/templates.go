package prompt

import (
	"fmt"
	"strings"
)

// SystemPrompt contains the base system prompt for command generation
const SystemPrompt = `You are a command line expert assistant. Based on the user's natural language description and the current environment context, suggest the most appropriate shell commands.

IMPORTANT INSTRUCTIONS:
1. Analyze the user's request and current environment context carefully
2. Suggest 1-3 most relevant commands, ordered by relevance and safety
3. Consider the operating system, current directory contents, and shell type
4. ALWAYS prioritize safe commands and warn about dangerous operations
5. Return your response as valid JSON only, no additional text

RESPONSE FORMAT (JSON only):
{
  "commands": [
    {
      "cmd": "exact command to execute",
      "description": "brief explanation of what this command does",
      "confidence": 0.95,
      "safe": true,
      "category": "file_management"
    }
  ]
}

SAFETY GUIDELINES:
- Mark commands as "safe": false if they modify/delete files or system settings
- Never suggest commands that could harm the system (rm -rf /, format, etc.)
- Be extra careful with commands involving sudo, rm, chmod, dd, etc.
- For potentially dangerous commands, explain the risk in the description

CATEGORIES:
- file_management: ls, cp, mv, mkdir, etc.
- text_processing: grep, sed, awk, sort, etc.
- system_info: ps, top, df, uname, etc.
- network: ping, curl, wget, ssh, etc.
- development: git, npm, go, python, etc.
- archive: tar, zip, unzip, etc.
- search: find, locate, which, etc.`

// CommandPromptTemplate builds a prompt for command suggestion
type CommandPromptTemplate struct {
	SystemPrompt string
	Context      *Context
	UserInput    string
}

// NewCommandPromptTemplate creates a new command prompt template
func NewCommandPromptTemplate(userInput string, ctx *Context) *CommandPromptTemplate {
	return &CommandPromptTemplate{
		SystemPrompt: SystemPrompt,
		Context:      ctx,
		UserInput:    userInput,
	}
}

// Build builds the complete prompt
func (t *CommandPromptTemplate) Build() string {
	var parts []string

	// Add system prompt
	parts = append(parts, t.SystemPrompt)

	// Add context information
	if t.Context != nil {
		parts = append(parts, "\nCURRENT ENVIRONMENT:")
		parts = append(parts, t.Context.FormatForPrompt())
	}

	// Add user request
	parts = append(parts, "\nUSER REQUEST:")
	parts = append(parts, t.UserInput)

	// Add final instruction
	parts = append(parts, "\nRespond with JSON only:")

	return strings.Join(parts, "\n")
}

// EnhanceWithExamples adds examples to the prompt based on context
func (t *CommandPromptTemplate) EnhanceWithExamples() *CommandPromptTemplate {
	if t.Context == nil {
		return t
	}

	examples := t.generateContextualExamples()
	if examples != "" {
		t.SystemPrompt = t.SystemPrompt + "\n\nEXAMPLES:\n" + examples
	}

	return t
}

// generateContextualExamples generates examples based on the current context
func (t *CommandPromptTemplate) generateContextualExamples() string {
	if t.Context == nil {
		return ""
	}

	var examples []string

	// File management examples
	if t.Context.FileCount > 0 {
		examples = append(examples, `User: "list all files with details"
Response: {"commands":[{"cmd":"ls -la","description":"List all files and directories with detailed information","confidence":0.95,"safe":true,"category":"file_management"}]}`)
	}

	// Python-specific examples
	if t.Context.HasFiles("py") {
		examples = append(examples, `User: "run the python script"
Response: {"commands":[{"cmd":"python script.py","description":"Execute the Python script found in current directory","confidence":0.9,"safe":true,"category":"development"}]}`)
	}

	// Git examples if in a git repository
	if t.Context.HasFiles("git") {
		examples = append(examples, `User: "show git status"
Response: {"commands":[{"cmd":"git status","description":"Show the current status of the Git repository","confidence":0.98,"safe":true,"category":"development"}]}`)
	}

	// Archive examples
	if t.Context.HasFiles("tar") || t.Context.HasFiles("zip") {
		examples = append(examples, `User: "extract the archive"
Response: {"commands":[{"cmd":"tar -xzf archive.tar.gz","description":"Extract gzipped tar archive","confidence":0.85,"safe":true,"category":"archive"}]}`)
	}

	// Limit examples to avoid making prompt too long
	if len(examples) > 3 {
		examples = examples[:3]
	}

	return strings.Join(examples, "\n\n")
}

// CustomPromptTemplate allows for custom prompt templates
type CustomPromptTemplate struct {
	Template  string
	Context   *Context
	UserInput string
	Variables map[string]string
}

// NewCustomPromptTemplate creates a new custom prompt template
func NewCustomPromptTemplate(template, userInput string, ctx *Context) *CustomPromptTemplate {
	return &CustomPromptTemplate{
		Template:  template,
		Context:   ctx,
		UserInput: userInput,
		Variables: make(map[string]string),
	}
}

// SetVariable sets a template variable
func (t *CustomPromptTemplate) SetVariable(key, value string) *CustomPromptTemplate {
	t.Variables[key] = value
	return t
}

// Build builds the custom prompt with variable substitution
func (t *CustomPromptTemplate) Build() string {
	result := t.Template

	// Replace standard variables
	result = strings.ReplaceAll(result, "{user_input}", t.UserInput)

	if t.Context != nil {
		result = strings.ReplaceAll(result, "{os}", t.Context.OS)
		result = strings.ReplaceAll(result, "{shell}", t.Context.Shell)
		result = strings.ReplaceAll(result, "{working_dir}", t.Context.WorkingDir)
		result = strings.ReplaceAll(result, "{context}", t.Context.FormatForPrompt())
	}

	// Replace custom variables
	for key, value := range t.Variables {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}

	return result
}

// QuickCommandPrompt creates a simple prompt for quick command suggestions
func QuickCommandPrompt(userInput string, os, shell string) string {
	return fmt.Sprintf(`Generate a shell command for: "%s"

Operating System: %s
Shell: %s

Respond with JSON format:
{"commands":[{"cmd":"command here","description":"what it does","confidence":0.9,"safe":true,"category":"category"}]}`,
		userInput, os, shell)
}
