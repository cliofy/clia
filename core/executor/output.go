package executor

import (
	"bytes"
	"regexp"
)

// ANSIParser handles ANSI escape sequence parsing
type ANSIParser struct {
	preserveColors bool
}

// NewANSIParser creates a new ANSI parser
func NewANSIParser(preserveColors bool) *ANSIParser {
	return &ANSIParser{
		preserveColors: preserveColors,
	}
}

// Parse processes input containing ANSI escape sequences
func (p *ANSIParser) Parse(input []byte) string {
	if p.preserveColors {
		// Keep ANSI sequences intact
		return string(input)
	}
	
	// Strip ANSI escape sequences
	return p.stripANSI(string(input))
}

// stripANSI removes ANSI escape sequences from text
func (p *ANSIParser) stripANSI(text string) string {
	// Common ANSI escape sequence patterns
	patterns := []string{
		`\x1b\[[0-9;]*m`,        // Color codes
		`\x1b\[[0-9]*[A-Z]`,     // Cursor movement
		`\x1b\[([0-9]+;)*[0-9]*[HJK]`, // Clear screen, cursor position
		`\x1b\][0-9];[^\x07]*\x07`,    // Terminal title
		`\x1b[>=?]?[0-9;]*[a-zA-Z]`,   // Other escape sequences
	}
	
	result := text
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}
	
	return result
}

// ScreenBuffer captures the state of a terminal screen
type ScreenBuffer struct {
	lines  []string
	width  int
	height int
	cursor struct {
		row int
		col int
	}
}

// NewScreenBuffer creates a new screen buffer
func NewScreenBuffer(width, height int) *ScreenBuffer {
	sb := &ScreenBuffer{
		width:  width,
		height: height,
		lines:  make([]string, height),
	}
	// Initialize with empty lines
	for i := range sb.lines {
		sb.lines[i] = ""
	}
	return sb
}

// ProcessOutput processes output that may contain terminal control sequences
func (sb *ScreenBuffer) ProcessOutput(output []byte) {
	// This is a simplified implementation
	// In a real implementation, we would parse ANSI sequences more thoroughly
	
	// Split by newlines
	lines := bytes.Split(output, []byte("\n"))
	
	for _, line := range lines {
		// Check for clear screen sequence
		if bytes.Contains(line, []byte("\x1b[2J")) {
			// Clear screen
			for i := range sb.lines {
				sb.lines[i] = ""
			}
			sb.cursor.row = 0
			sb.cursor.col = 0
			continue
		}
		
		// Check for cursor home sequence
		if bytes.Contains(line, []byte("\x1b[H")) {
			sb.cursor.row = 0
			sb.cursor.col = 0
			continue
		}
		
		// Add line to buffer (simplified)
		if sb.cursor.row < len(sb.lines) {
			sb.lines[sb.cursor.row] = string(line)
			sb.cursor.row++
		}
	}
}

// GetContent returns the current screen content as a string
func (sb *ScreenBuffer) GetContent() string {
	var result bytes.Buffer
	for i, line := range sb.lines {
		if i > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(line)
	}
	return result.String()
}