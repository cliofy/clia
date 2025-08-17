package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ResponseParser parses LLM responses into CommandSuggestion
type ResponseParser struct {
	jsonRegex    *regexp.Regexp
	commandRegex *regexp.Regexp
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		// Match JSON blocks in response
		jsonRegex: regexp.MustCompile(`(?s)\{.*"command".*\}`),
		// Match command patterns
		commandRegex: regexp.MustCompile("`([^`]+)`|```(?:bash|sh|shell)?\n?([^`]+)```"),
	}
}

// Parse parses an LLM response into a CommandSuggestion
func (rp *ResponseParser) Parse(response string) (*CommandSuggestion, error) {
	// Try JSON parsing first
	if suggestion := rp.parseJSON(response); suggestion != nil {
		return suggestion, nil
	}

	// Try structured text parsing
	if suggestion := rp.parseStructuredText(response); suggestion != nil {
		return suggestion, nil
	}

	// Fallback to simple parsing
	return rp.parseSimple(response)
}

// parseJSON attempts to parse JSON from the response
func (rp *ResponseParser) parseJSON(response string) *CommandSuggestion {
	// Find JSON block in response
	matches := rp.jsonRegex.FindStringSubmatch(response)
	if len(matches) == 0 {
		// Try to parse entire response as JSON
		matches = []string{response}
	}

	for _, match := range matches {
		var result struct {
			Command      string   `json:"command"`
			Explanation  string   `json:"explanation"`
			Confidence   float64  `json:"confidence"`
			Risks        []struct {
				Level       string `json:"level"`
				Description string `json:"description"`
				Mitigation  string `json:"mitigation"`
			} `json:"risks"`
			Alternatives []string `json:"alternatives"`
		}

		if err := json.Unmarshal([]byte(match), &result); err == nil {
			// Successfully parsed JSON
			suggestion := &CommandSuggestion{
				Command:      result.Command,
				Explanation:  result.Explanation,
				Confidence:   result.Confidence,
				Alternatives: result.Alternatives,
			}

			// Convert risks
			for _, risk := range result.Risks {
				suggestion.Risks = append(suggestion.Risks, SecurityRisk{
					Level:       RiskLevel(risk.Level),
					Description: risk.Description,
					Mitigation:  risk.Mitigation,
				})
			}

			// Validate and set defaults
			if suggestion.Confidence == 0 {
				suggestion.Confidence = 0.7
			}
			if suggestion.Explanation == "" {
				suggestion.Explanation = "Command suggested by AI"
			}

			return suggestion
		}
	}

	return nil
}

// parseStructuredText attempts to parse structured text response
func (rp *ResponseParser) parseStructuredText(response string) *CommandSuggestion {
	suggestion := &CommandSuggestion{
		Confidence: 0.7,
		Risks:      []SecurityRisk{},
	}

	// Look for command in code blocks or backticks
	if matches := rp.commandRegex.FindStringSubmatch(response); len(matches) > 0 {
		// Get the last non-empty match
		for i := len(matches) - 1; i >= 1; i-- {
			if matches[i] != "" {
				suggestion.Command = strings.TrimSpace(matches[i])
				break
			}
		}
	}

	if suggestion.Command == "" {
		// Look for lines starting with $ or #
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "$") || strings.HasPrefix(line, "#") {
				suggestion.Command = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "$"), "#"))
				if suggestion.Command != "" {
					break
				}
			}
		}
	}

	if suggestion.Command == "" {
		return nil
	}

	// Extract explanation
	suggestion.Explanation = rp.extractExplanation(response, suggestion.Command)

	// Extract alternatives
	suggestion.Alternatives = rp.extractAlternatives(response, suggestion.Command)

	// Extract confidence if mentioned
	if conf := rp.extractConfidence(response); conf > 0 {
		suggestion.Confidence = conf
	}

	return suggestion
}

// parseSimple performs simple parsing as last resort
func (rp *ResponseParser) parseSimple(response string) (*CommandSuggestion, error) {
	// Clean the response
	response = strings.TrimSpace(response)
	
	if response == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Find first line that looks like a command
	lines := strings.Split(response, "\n")
	command := ""
	explanation := ""

	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and obvious non-commands
		if line == "" || strings.HasPrefix(line, "The command") || strings.HasPrefix(line, "To ") {
			continue
		}

		// This might be a command
		if command == "" && rp.looksLikeCommand(line) {
			command = line
			// Rest is explanation
			if i+1 < len(lines) {
				explanation = strings.Join(lines[i+1:], " ")
			}
			break
		}
	}

	if command == "" {
		// Take first non-empty line as command
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				command = line
				break
			}
		}
	}

	if command == "" {
		return nil, fmt.Errorf("no command found in response")
	}

	return &CommandSuggestion{
		Command:     command,
		Explanation: explanation,
		Confidence:  0.5, // Lower confidence for simple parsing
	}, nil
}

// looksLikeCommand checks if a string looks like a shell command
func (rp *ResponseParser) looksLikeCommand(s string) bool {
	// Common command patterns
	commandStarts := []string{
		"ls", "cd", "pwd", "mkdir", "rm", "cp", "mv", "cat", "echo", "grep", "find",
		"sed", "awk", "sort", "uniq", "head", "tail", "ps", "kill", "df", "du",
		"tar", "zip", "unzip", "chmod", "chown", "sudo", "apt", "yum", "brew",
		"git", "docker", "kubectl", "npm", "pip", "go", "make", "curl", "wget",
	}

	s = strings.ToLower(s)
	for _, cmd := range commandStarts {
		if strings.HasPrefix(s, cmd+" ") || s == cmd {
			return true
		}
	}

	// Check for common command patterns
	if strings.Contains(s, "|") || strings.Contains(s, ">") || strings.Contains(s, "<") {
		return true
	}

	// Check for flags
	if strings.Contains(s, " -") || strings.HasPrefix(s, "-") {
		return true
	}

	return false
}

// extractExplanation extracts explanation from response
func (rp *ResponseParser) extractExplanation(response, command string) string {
	// Look for explanation patterns
	patterns := []string{
		"This command ",
		"This will ",
		"Explanation:",
		"Description:",
		"What it does:",
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				return strings.TrimSpace(strings.SplitN(line, pattern, 2)[1])
			}
		}
	}

	// If command is in response, take the next sentence as explanation
	if idx := strings.Index(response, command); idx >= 0 {
		after := response[idx+len(command):]
		if sentences := strings.Split(after, "."); len(sentences) > 0 {
			explanation := strings.TrimSpace(sentences[0])
			if explanation != "" && !strings.Contains(explanation, command) {
				return explanation
			}
		}
	}

	return "Command suggested by AI assistant"
}

// extractAlternatives extracts alternative commands from response
func (rp *ResponseParser) extractAlternatives(response, mainCommand string) []string {
	var alternatives []string

	// Look for alternative patterns
	if strings.Contains(response, "Alternative") || strings.Contains(response, "You can also") || strings.Contains(response, "Or ") {
		// Find code blocks or commands after these keywords
		lines := strings.Split(response, "\n")
		foundAlt := false
		for _, line := range lines {
			if strings.Contains(line, "Alternative") || strings.Contains(line, "You can also") {
				foundAlt = true
				continue
			}
			if foundAlt && rp.looksLikeCommand(strings.TrimSpace(line)) {
				alt := strings.TrimSpace(line)
				if alt != mainCommand {
					alternatives = append(alternatives, alt)
				}
			}
		}
	}

	return alternatives
}

// extractConfidence extracts confidence score from response
func (rp *ResponseParser) extractConfidence(response string) float64 {
	// Look for confidence patterns
	patterns := []regexp.Regexp{
		*regexp.MustCompile(`confidence[:\s]+([0-9.]+)`),
		*regexp.MustCompile(`([0-9]+)%\s+confident`),
		*regexp.MustCompile(`confidence.*?([0-9.]+)`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(strings.ToLower(response)); len(matches) > 1 {
			var conf float64
			if strings.Contains(matches[1], "%") {
				fmt.Sscanf(matches[1], "%f%%", &conf)
				conf /= 100
			} else {
				fmt.Sscanf(matches[1], "%f", &conf)
			}
			if conf > 0 && conf <= 1 {
				return conf
			}
		}
	}

	return 0
}