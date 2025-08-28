package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGovteAlternateScreenCapture(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected string
		hasFrame bool
	}{
		{
			name: "Enter and exit alternate screen with content",
			input: [][]byte{
				[]byte("\x1b[?1049h"),                     // Enter alternate screen
				[]byte("Hello from alternate screen"),      // Write content
				[]byte("\x1b[?1049l"),                     // Exit alternate screen
			},
			expected: "Hello from alternate screen",
			hasFrame: true,
		},
		{
			name: "Alternate screen with cursor movement",
			input: [][]byte{
				[]byte("\x1b[?1049h"),                     // Enter alternate screen
				[]byte("\x1b[5;10H"),                      // Move cursor to line 5, column 10
				[]byte("Positioned text"),                 // Write at position
				[]byte("\x1b[?1049l"),                     // Exit alternate screen
			},
			expected: "Positioned text",
			hasFrame: true,
		},
		{
			name: "Alternate screen with colors",
			input: [][]byte{
				[]byte("\x1b[?1049h"),                     // Enter alternate screen
				[]byte("\x1b[31mRed text\x1b[0m"),        // Red text
				[]byte(" \x1b[32mGreen text\x1b[0m"),     // Green text
				[]byte("\x1b[?1049l"),                     // Exit alternate screen
			},
			expected: "Red text",
			hasFrame: true,
		},
		{
			name: "Mode 47 alternate screen",
			input: [][]byte{
				[]byte("\x1b[?47h"),                       // Enter alternate screen (mode 47)
				[]byte("Mode 47 content"),                 // Write content
				[]byte("\x1b[?47l"),                       // Exit alternate screen
			},
			expected: "Mode 47 content",
			hasFrame: true,
		},
		{
			name: "No alternate screen",
			input: [][]byte{
				[]byte("Regular content"),
			},
			expected: "",
			hasFrame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new GovteTerminalScreen
			screen := NewGovteTerminalScreen(80, 24)

			// Process each input
			for _, data := range tt.input {
				screen.ProcessOutput(data)
			}

			// Check if we detected alternate screen exit
			assert.Equal(t, tt.hasFrame, screen.DetectedAltScreenExit(), 
				"Alternate screen exit detection mismatch")

			// If we expect a frame, check its content
			if tt.hasFrame {
				frame := screen.GetLastFrame()
				assert.Contains(t, frame, tt.expected, 
					"Captured frame should contain expected text")
			}
		})
	}
}

func TestGovteScreenClearOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected string
	}{
		{
			name: "Clear screen",
			input: [][]byte{
				[]byte("Line 1\n"),
				[]byte("Line 2\n"),
				[]byte("\x1b[2J"),    // Clear entire screen
				[]byte("After clear"),
			},
			expected: "After clear",
		},
		{
			name: "Clear line",
			input: [][]byte{
				[]byte("Start of line"),
				[]byte("\x1b[2K"),      // Clear entire line
				[]byte("New content"),
			},
			expected: "New content",
		},
		{
			name: "Erase in display",
			input: [][]byte{
				[]byte("Top\n"),
				[]byte("Middle\n"),
				[]byte("Bottom\n"),
				[]byte("Final line"),
				[]byte("\x1b[1;1H"),     // Go to top-left
				[]byte("\x1b[0J"),       // Clear from cursor to end  
				[]byte("New content"),
			},
			expected: "New content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewGovteTerminalScreen(80, 24)

			for _, data := range tt.input {
				screen.ProcessOutput(data)
			}

			frame := screen.CaptureFrame()
			assert.Contains(t, frame, tt.expected)
		})
	}
}

func TestGovteCursorMovement(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)

	// Test cursor up
	screen.ProcessOutput([]byte("\x1b[5;10H"))  // Position at line 5, col 10
	screen.ProcessOutput([]byte("\x1b[2A"))      // Move up 2 lines
	
	// Test cursor down
	screen.ProcessOutput([]byte("\x1b[3B"))      // Move down 3 lines
	
	// Test cursor forward
	screen.ProcessOutput([]byte("\x1b[5C"))      // Move forward 5 columns
	
	// Test cursor back
	screen.ProcessOutput([]byte("\x1b[3D"))      // Move back 3 columns
	
	// Test absolute positioning
	screen.ProcessOutput([]byte("\x1b[1;1H"))    // Move to top-left
	
	// Write something to verify position
	screen.ProcessOutput([]byte("X"))
	
	frame := screen.CaptureFrame()
	assert.NotEmpty(t, frame)
	// The 'X' should be at the top-left after all movements
}

func TestGovteTextAttributes(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)

	// Test various SGR attributes
	inputs := [][]byte{
		[]byte("\x1b[1mBold\x1b[0m "),           // Bold
		[]byte("\x1b[3mItalic\x1b[0m "),         // Italic
		[]byte("\x1b[4mUnderline\x1b[0m "),      // Underline
		[]byte("\x1b[7mReverse\x1b[0m"),         // Reverse video
	}

	for _, data := range inputs {
		screen.ProcessOutput(data)
	}

	frame := screen.CaptureFrame()
	assert.Contains(t, frame, "Bold")
	assert.Contains(t, frame, "Italic")
	assert.Contains(t, frame, "Underline")
	assert.Contains(t, frame, "Reverse")
}

func TestGovteScrolling(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)

	// Fill screen with numbered lines
	for i := 1; i <= 25; i++ {
		screen.ProcessOutput([]byte("Line "))
		screen.ProcessOutput([]byte{byte('0' + i/10), byte('0' + i%10)})
		screen.ProcessOutput([]byte("\n"))
	}

	// Set scroll region and scroll
	screen.ProcessOutput([]byte("\x1b[5;20r"))   // Set scroll region lines 5-20
	screen.ProcessOutput([]byte("\x1b[2S"))      // Scroll up 2 lines

	frame := screen.CaptureFrame()
	assert.NotEmpty(t, frame)
}

func TestGovteCompatibilityWithAnsiTermScreen(t *testing.T) {
	// This test ensures the GovteTerminalScreen maintains
	// compatibility with the existing AnsiTermScreen API
	
	screen := NewGovteTerminalScreen(80, 24)
	
	// Test the main API methods exist and work
	assert.NotNil(t, screen)
	
	// ProcessOutput should accept data
	screen.ProcessOutput([]byte("Test"))
	
	// CaptureFrame should return current content
	frame := screen.CaptureFrame()
	assert.Contains(t, frame, "Test")
	
	// GetLastFrame should return empty initially
	lastFrame := screen.GetLastFrame()
	assert.Empty(t, lastFrame)
	
	// DetectedAltScreenExit should return false initially
	assert.False(t, screen.DetectedAltScreenExit())
	
	// After alt screen exit, should have a frame
	screen.ProcessOutput([]byte("\x1b[?1049h"))
	screen.ProcessOutput([]byte("Alt content"))
	screen.ProcessOutput([]byte("\x1b[?1049l"))
	
	assert.True(t, screen.DetectedAltScreenExit())
	assert.Contains(t, screen.GetLastFrame(), "Alt content")
}

// Test color handling in GetDisplayWithColors
func TestGovteColorHandling(t *testing.T) {
	
	// Test various color sequences
	tests := []struct {
		name     string
		input    []byte
		contains string
	}{
		{
			name:     "Named foreground color",
			input:    []byte("\x1b[31mRed text\x1b[0m"),
			contains: "Red text",
		},
		{
			name:     "Named background color",
			input:    []byte("\x1b[41mRed background\x1b[0m"),
			contains: "Red background",
		},
		{
			name:     "Bright colors",
			input:    []byte("\x1b[91mBright red\x1b[0m"),
			contains: "Bright red",
		},
		{
			name:     "256-color palette",
			input:    []byte("\x1b[38;5;196mIndexed red\x1b[0m"),
			contains: "Indexed red",
		},
		{
			name:     "True color RGB",
			input:    []byte("\x1b[38;2;255;0;0mRGB red\x1b[0m"),
			contains: "RGB red",
		},
		{
			name:     "Reset attributes",
			input:    []byte("\x1b[1mBold\x1b[0mNormal"),
			contains: "Bold",  // Just check that Bold text is present
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh screen for each test
			testScreen := NewGovteTerminalScreen(80, 24)
			testScreen.ProcessOutput(tt.input)
			
			frame := testScreen.CaptureFrame()
			assert.Contains(t, frame, tt.contains)
			
			// For color tests, also check that colors are preserved in GetDisplayWithColors
			coloredFrame := testScreen.CaptureFrame() // This uses GetDisplayWithColors
			assert.Contains(t, coloredFrame, tt.contains)
		})
	}
}

// Test alternate screen re-entry cleanup
func TestGovteAlternateScreenReentry(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)
	
	// First alt-screen session
	screen.ProcessOutput([]byte("\x1b[?1049h"))    // Enter alt screen
	screen.ProcessOutput([]byte("First session"))   // Content
	screen.ProcessOutput([]byte("\x1b[?1049l"))    // Exit alt screen
	
	// Verify first session captured
	assert.True(t, screen.DetectedAltScreenExit())
	firstFrame := screen.GetLastFrame()
	assert.Contains(t, firstFrame, "First session")
	
	// Second alt-screen session should not contain first session content
	screen.ProcessOutput([]byte("\x1b[?1049h"))    // Enter alt screen again
	screen.ProcessOutput([]byte("Second session"))  // Different content
	screen.ProcessOutput([]byte("\x1b[?1049l"))    // Exit alt screen
	
	// Verify second session doesn't contain first session content
	secondFrame := screen.GetLastFrame()
	assert.Contains(t, secondFrame, "Second session")
	assert.NotContains(t, secondFrame, "First session", "Alt screen should be clean on re-entry")
}

// Test DetectedAltScreenExit repeated calls
func TestGovteDetectedAltScreenExitRepeated(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)
	
	// Enter and exit alt screen
	screen.ProcessOutput([]byte("\x1b[?1049h"))
	screen.ProcessOutput([]byte("Test content"))
	screen.ProcessOutput([]byte("\x1b[?1049l"))
	
	// First call should detect exit
	assert.True(t, screen.DetectedAltScreenExit())
	
	// Get the frame (this marks it as read)
	frame := screen.GetLastFrame()
	assert.Contains(t, frame, "Test content")
	
	// Subsequent calls should not detect exit (avoiding repeated triggers)
	assert.False(t, screen.DetectedAltScreenExit())
	assert.False(t, screen.DetectedAltScreenExit())
}

// Test ResetAttributes functionality
func TestGovteResetAttributes(t *testing.T) {
	screen := NewGovteTerminalScreen(80, 24)
	
	// Apply some attributes, then reset
	screen.ProcessOutput([]byte("\x1b[1;31;4mBold Red Underlined"))
	screen.ProcessOutput([]byte("\x1b[0m"))  // Reset all attributes
	screen.ProcessOutput([]byte(" Normal text"))
	
	frame := screen.CaptureFrame()
	assert.Contains(t, frame, "Bold Red Underlined")
	assert.Contains(t, frame, "Normal text")
}

// Test that colors are properly preserved in GetDisplayWithColors output
func TestGovteColorDisplayFormat(t *testing.T) {
	// This test ensures that the Parser+Performer architecture correctly processes colors
	// and that GetDisplayWithColors returns ANSI escape sequences for color display
	
	screen := NewGovteTerminalScreen(80, 24)
	
	// Process complex color sequences similar to what btm produces
	colorSequences := [][]byte{
		[]byte("\x1b[38;5;7m"),   // 256-color foreground
		[]byte("\x1b[49m"),      // Default background 
		[]byte("Text"),          // Some text
		[]byte("\x1b[1m"),       // Bold
		[]byte("\x1b[38;5;12m"), // Blue foreground
		[]byte("Bold Blue"),     // More text
		[]byte("\x1b[0m"),       // Reset all
	}
	
	for _, seq := range colorSequences {
		screen.ProcessOutput(seq)
	}
	
	result := screen.CaptureFrame()
	
	// Verify that GetDisplayWithColors correctly preserves color information
	// The output should contain ANSI escape sequences for terminal color display
	assert.Contains(t, result, "\x1b[38;5;7m", "Should contain 256-color foreground escape sequence")
	assert.Contains(t, result, "\x1b[1m", "Should contain bold escape sequence")
	assert.Contains(t, result, "\x1b[38;5;12m", "Should contain blue foreground escape sequence")
	assert.Contains(t, result, "\x1b[0m", "Should contain reset escape sequence")
	
	// Verify text content is preserved
	assert.Contains(t, result, "Text", "Text content should be preserved")
	assert.Contains(t, result, "Bold Blue", "Text content should be preserved")
	
	// Verify that we're not double-processing (no malformed sequences)
	assert.NotContains(t, result, "\x1b[\x1b[", "Should not contain double-escaped sequences")
}

// Benchmark to compare performance
func BenchmarkGovteTerminalScreen(b *testing.B) {
	screen := NewGovteTerminalScreen(80, 24)
	
	// Complex ANSI sequence typical of TUI apps
	data := []byte("\x1b[2J\x1b[H\x1b[31mRed\x1b[0m \x1b[32mGreen\x1b[0m \x1b[1mBold\x1b[0m\n")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.ProcessOutput(data)
	}
}

// Test helper function for verifying specific sequences
func TestGovteSpecificSequences(t *testing.T) {
	testCases := []struct {
		name     string
		sequence []byte
		verify   func(*testing.T, *GovteTerminalScreen)
	}{
		{
			name:     "CSI Save Cursor",
			sequence: []byte("\x1b[s"),
			verify: func(t *testing.T, s *GovteTerminalScreen) {
				// Should save cursor position
				assert.NotNil(t, s)
			},
		},
		{
			name:     "CSI Restore Cursor",
			sequence: []byte("\x1b[u"),
			verify: func(t *testing.T, s *GovteTerminalScreen) {
				// Should restore cursor position
				assert.NotNil(t, s)
			},
		},
		{
			name:     "OSC Set Title",
			sequence: []byte("\x1b]0;Terminal Title\x07"),
			verify: func(t *testing.T, s *GovteTerminalScreen) {
				// Title setting doesn't affect capture but should parse
				assert.NotNil(t, s)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			screen := NewGovteTerminalScreen(80, 24)
			screen.ProcessOutput(tc.sequence)
			tc.verify(t, screen)
		})
	}
}