package executor

import (
	"strings"
	"testing"
)

func TestNewTerminalScreen(t *testing.T) {
	screen := NewTerminalScreen(80, 24)
	
	if screen.width != 80 {
		t.Errorf("Expected width 80, got %d", screen.width)
	}
	if screen.height != 24 {
		t.Errorf("Expected height 24, got %d", screen.height)
	}
	if screen.cursorX != 0 {
		t.Errorf("Expected cursorX 0, got %d", screen.cursorX)
	}
	if screen.cursorY != 0 {
		t.Errorf("Expected cursorY 0, got %d", screen.cursorY)
	}
	if !screen.cursorVisible {
		t.Error("Expected cursor to be visible by default")
	}
	if !screen.autowrap {
		t.Error("Expected autowrap to be enabled by default")
	}
}

func TestPutChar(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test putting a character
	screen.putChar('A')
	if screen.buffer[0][0].Char != 'A' {
		t.Errorf("Expected 'A' at position (0,0), got %c", screen.buffer[0][0].Char)
	}
	if screen.cursorX != 1 {
		t.Errorf("Expected cursor to advance to position 1, got %d", screen.cursorX)
	}
	
	// Test cursor advancement
	screen.putChar('B')
	if screen.buffer[0][1].Char != 'B' {
		t.Errorf("Expected 'B' at position (0,1), got %c", screen.buffer[0][1].Char)
	}
	if screen.cursorX != 2 {
		t.Errorf("Expected cursor at position 2, got %d", screen.cursorX)
	}
}

func TestCursorMovement(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test cursor up
	screen.cursorY = 2
	screen.cursorUp(1)
	if screen.cursorY != 1 {
		t.Errorf("Expected cursorY 1, got %d", screen.cursorY)
	}
	
	// Test cursor down
	screen.cursorDown(2)
	if screen.cursorY != 3 {
		t.Errorf("Expected cursorY 3, got %d", screen.cursorY)
	}
	
	// Test cursor forward
	screen.cursorX = 2
	screen.cursorForward(3)
	if screen.cursorX != 5 {
		t.Errorf("Expected cursorX 5, got %d", screen.cursorX)
	}
	
	// Test cursor backward
	screen.cursorBackward(2)
	if screen.cursorX != 3 {
		t.Errorf("Expected cursorX 3, got %d", screen.cursorX)
	}
}

func TestSetCursorPosition(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test setting cursor position (1-based input)
	screen.setCursorPosition(3, 5)
	if screen.cursorY != 2 { // Should be 0-based
		t.Errorf("Expected cursorY 2, got %d", screen.cursorY)
	}
	if screen.cursorX != 4 { // Should be 0-based
		t.Errorf("Expected cursorX 4, got %d", screen.cursorX)
	}
}

func TestEraseInDisplay(t *testing.T) {
	screen := NewTerminalScreen(5, 3)
	
	// Fill screen with characters
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			screen.buffer[y][x].Char = 'X'
		}
	}
	
	// Position cursor at (1, 1)
	screen.cursorX = 1
	screen.cursorY = 1
	
	// Test erase from cursor to end of screen (mode 0)
	screen.eraseInDisplay(0)
	
	// Check that positions before cursor are unchanged
	if screen.buffer[0][0].Char != 'X' {
		t.Error("Expected character before cursor position to remain unchanged")
	}
	if screen.buffer[1][0].Char != 'X' {
		t.Error("Expected character before cursor on same line to remain unchanged")
	}
	
	// Check that cursor position and after are cleared
	if screen.buffer[1][1].Char != ' ' {
		t.Error("Expected cursor position to be cleared")
	}
	if screen.buffer[2][0].Char != ' ' {
		t.Error("Expected lines after cursor to be cleared")
	}
}

func TestEraseInLine(t *testing.T) {
	screen := NewTerminalScreen(5, 3)
	
	// Fill current line with characters
	for x := 0; x < 5; x++ {
		screen.buffer[1][x].Char = 'X'
	}
	
	// Position cursor at (2, 1)
	screen.cursorX = 2
	screen.cursorY = 1
	
	// Test erase from cursor to end of line (mode 0)
	screen.eraseInLine(0)
	
	// Check that positions before cursor are unchanged
	if screen.buffer[1][0].Char != 'X' {
		t.Error("Expected character before cursor to remain unchanged")
	}
	if screen.buffer[1][1].Char != 'X' {
		t.Error("Expected character before cursor to remain unchanged")
	}
	
	// Check that cursor position and after are cleared
	if screen.buffer[1][2].Char != ' ' {
		t.Error("Expected cursor position to be cleared")
	}
	if screen.buffer[1][3].Char != ' ' {
		t.Error("Expected characters after cursor to be cleared")
	}
}

func TestProcessControlChar(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test carriage return
	screen.cursorX = 5
	processed := screen.processControlChar('\r')
	if !processed {
		t.Error("Expected carriage return to be processed")
	}
	if screen.cursorX != 0 {
		t.Errorf("Expected cursor to move to column 0, got %d", screen.cursorX)
	}
	
	// Test line feed
	screen.cursorY = 2
	processed = screen.processControlChar('\n')
	if !processed {
		t.Error("Expected line feed to be processed")
	}
	if screen.cursorY != 3 {
		t.Errorf("Expected cursor to move to next line, got %d", screen.cursorY)
	}
	if screen.cursorX != 0 {
		t.Errorf("Expected cursor to move to column 0, got %d", screen.cursorX)
	}
	
	// Test tab
	screen.cursorX = 3
	processed = screen.processControlChar('\t')
	if !processed {
		t.Error("Expected tab to be processed")
	}
	if screen.cursorX != 8 {
		t.Errorf("Expected cursor to move to next tab stop (8), got %d", screen.cursorX)
	}
}

func TestProcessCSI(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	tests := []struct {
		name     string
		input    string
		expectX  int
		expectY  int
		processed bool
	}{
		{"Cursor up", "\033[A", 0, 0, true}, // At top, should stay at 0
		{"Cursor down", "\033[B", 0, 1, true},
		{"Cursor forward", "\033[C", 1, 0, true}, // X changes, Y stays same
		{"Cursor backward", "\033[D", 0, 0, true}, // X changes, Y stays same  
		{"Cursor position", "\033[3;5H", 4, 2, true}, // 1-based input becomes 0-based
		{"Cursor position alt", "\033[2;3f", 2, 1, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cursor
			screen.cursorX = 0
			screen.cursorY = 0
			
			length, processed := screen.processCSI(tt.input)
			if processed != tt.processed {
				t.Errorf("Expected processed=%v, got %v", tt.processed, processed)
			}
			if processed {
				if length != len(tt.input) {
					t.Errorf("Expected length %d, got %d", len(tt.input), length)
				}
				if screen.cursorX != tt.expectX {
					t.Errorf("Expected cursorX %d, got %d", tt.expectX, screen.cursorX)
				}
				if screen.cursorY != tt.expectY {
					t.Errorf("Expected cursorY %d, got %d", tt.expectY, screen.cursorY)
				}
			}
		})
	}
}

func TestSetGraphicsRendition(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test setting foreground color
	screen.setGraphicsRendition("31") // Red foreground
	if screen.currentFgColor != "\033[31m" {
		t.Errorf("Expected red foreground color, got %q", screen.currentFgColor)
	}
	
	// Test setting background color
	screen.setGraphicsRendition("42") // Green background
	if screen.currentBgColor != "\033[42m" {
		t.Errorf("Expected green background color, got %q", screen.currentBgColor)
	}
	
	// Test setting multiple attributes
	screen.setGraphicsRendition("1;31;42") // Bold, red fg, green bg
	if screen.currentAttributes != "\033[1m" {
		t.Errorf("Expected bold attribute, got %q", screen.currentAttributes)
	}
	if screen.currentFgColor != "\033[31m" {
		t.Errorf("Expected red foreground, got %q", screen.currentFgColor)
	}
	if screen.currentBgColor != "\033[42m" {
		t.Errorf("Expected green background, got %q", screen.currentBgColor)
	}
	
	// Test reset
	screen.setGraphicsRendition("0")
	if screen.currentFgColor != "" {
		t.Error("Expected foreground color to be reset")
	}
	if screen.currentBgColor != "" {
		t.Error("Expected background color to be reset")
	}
	if screen.currentAttributes != "" {
		t.Error("Expected attributes to be reset")
	}
}

func TestAlternateScreen(t *testing.T) {
	screen := NewTerminalScreen(5, 3)
	
	// Fill main screen
	screen.putChar('M')
	
	// Enter alternate screen
	screen.enterAltScreen()
	if !screen.inAltScreen {
		t.Error("Expected to be in alternate screen")
	}
	
	// Check that alternate screen is clear
	if screen.altScreenBuffer[0][0].Char != ' ' {
		t.Error("Expected alternate screen to be clear")
	}
	
	// Add content to alternate screen
	screen.putChar('A')
	if screen.altScreenBuffer[0][0].Char != 'A' {
		t.Error("Expected character to be in alternate screen buffer")
	}
	
	// Exit alternate screen
	screen.exitAltScreen()
	if screen.inAltScreen {
		t.Error("Expected to exit alternate screen")
	}
	
	// Check that we captured the frame
	lastFrame := screen.GetLastFrame()
	if !strings.Contains(lastFrame, "A") {
		t.Errorf("Expected captured frame to contain 'A', got: %q", lastFrame)
	}
}

func TestProcessOutput(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test simple text
	screen.ProcessOutput([]byte("Hello"))
	captured := screen.CaptureFrame()
	if !strings.Contains(captured, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %q", captured)
	}
	
	// Test with ANSI sequences
	screen = NewTerminalScreen(10, 5)
	screen.ProcessOutput([]byte("\033[31mRed\033[0m"))
	captured = screen.CaptureFrame()
	if !strings.Contains(captured, "Red") {
		t.Errorf("Expected output to contain 'Red', got: %q", captured)
	}
	
	// Test cursor positioning
	screen = NewTerminalScreen(10, 5)
	screen.ProcessOutput([]byte("\033[2;3HPos"))
	captured = screen.CaptureFrame()
	lines := strings.Split(captured, "\n")
	if len(lines) < 2 || !strings.Contains(lines[1], "Pos") {
		t.Errorf("Expected 'Pos' at line 2, got: %q", captured)
	}
}

func TestCaptureFrame(t *testing.T) {
	screen := NewTerminalScreen(5, 3)
	
	// Add some content with colors
	screen.currentFgColor = "\033[31m" // Red
	screen.putChar('R')
	screen.currentFgColor = ""
	screen.putChar('e')
	screen.putChar('d')
	
	// Position cursor on second line
	screen.cursorX = 0
	screen.cursorY = 1
	screen.putChar('L')
	screen.putChar('i')
	screen.putChar('n')
	screen.putChar('e')
	
	frame := screen.CaptureFrame()
	
	// Check that frame contains our content
	if !strings.Contains(frame, "Red") {
		t.Errorf("Expected frame to contain 'Red', got: %q", frame)
	}
	if !strings.Contains(frame, "Line") {
		t.Errorf("Expected frame to contain 'Line', got: %q", frame)
	}
	
	// Check that frame has correct number of lines
	lines := strings.Split(frame, "\n")
	expectedLines := 3 // height of screen
	if len(lines) != expectedLines {
		t.Errorf("Expected %d lines in frame, got %d", expectedLines, len(lines))
	}
}

func TestParseParams(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	tests := []struct {
		input    string
		count    int
		expected []int
	}{
		{"", 1, []int{1}},
		{"", 3, []int{1, 1, 1}},
		{"5", 1, []int{5}},
		{"2;3", 2, []int{2, 3}},
		{"1;2;3", 3, []int{1, 2, 3}},
		{";;", 3, []int{1, 1, 1}},
		{"5;;7", 3, []int{5, 1, 7}},
		{"1;2", 3, []int{1, 2, 1}}, // Pad with default
		{"0", 1, []int{0}},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := screen.parseParams(tt.input, tt.count)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d params, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected param[%d]=%d, got %d", i, expected, result[i])
				}
			}
		})
	}
}

func TestScrolling(t *testing.T) {
	screen := NewTerminalScreen(5, 3)
	
	// Fill screen with pattern
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			screen.buffer[y][x].Char = rune('0' + y)
		}
	}
	
	// Scroll up by 1
	screen.scrollUp(1)
	
	// Check that content moved up
	if screen.buffer[0][0].Char != '1' {
		t.Errorf("Expected first line to contain '1', got %c", screen.buffer[0][0].Char)
	}
	if screen.buffer[1][0].Char != '2' {
		t.Errorf("Expected second line to contain '2', got %c", screen.buffer[1][0].Char)
	}
	if screen.buffer[2][0].Char != ' ' {
		t.Errorf("Expected last line to be cleared, got %c", screen.buffer[2][0].Char)
	}
}

func TestCursorSaveRestore(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Move cursor to specific position
	screen.cursorX = 5
	screen.cursorY = 3
	
	// Save cursor
	screen.saveCursor()
	
	// Move cursor elsewhere
	screen.cursorX = 1
	screen.cursorY = 1
	
	// Restore cursor
	screen.restoreCursor()
	
	// Check that cursor is restored
	if screen.cursorX != 5 {
		t.Errorf("Expected cursorX 5, got %d", screen.cursorX)
	}
	if screen.cursorY != 3 {
		t.Errorf("Expected cursorY 3, got %d", screen.cursorY)
	}
}

func TestScrollRegion(t *testing.T) {
	screen := NewTerminalScreen(5, 5)
	
	// Set scroll region (1-based input)
	screen.setScrollRegion(2, 4) // Lines 2-4 (0-based: 1-3)
	
	if screen.scrollTop != 1 {
		t.Errorf("Expected scrollTop 1, got %d", screen.scrollTop)
	}
	if screen.scrollBottom != 3 {
		t.Errorf("Expected scrollBottom 3, got %d", screen.scrollBottom)
	}
	
	// Test cursor movement respects scroll region
	screen.cursorY = 2 // In scroll region
	screen.cursorDown(5) // Try to move way down
	
	// Should be limited to scroll region
	if screen.cursorY != 3 {
		t.Errorf("Expected cursor to be limited to scroll region bottom, got %d", screen.cursorY)
	}
}

// Test UTF-8 character handling
func TestUTF8CharacterHandling(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test various UTF-8 characters
	testChars := []string{
		"╔═══╗",     // Box drawing characters
		"中文",        // Chinese characters
		"🌟✨",        // Emoji
		"αβγδε",      // Greek letters
	}
	
	for _, chars := range testChars {
		t.Run("UTF8_"+chars, func(t *testing.T) {
			// Reset screen
			screen = NewTerminalScreen(10, 5)
			
			// Process the UTF-8 string
			screen.ProcessOutput([]byte(chars))
			
			// Capture and verify
			captured := screen.CaptureFrame()
			
			// The captured output should contain the original characters
			if !strings.Contains(captured, chars) {
				t.Errorf("Expected captured frame to contain %q, but got: %q", chars, captured)
			}
		})
	}
}

// Test 256 color support
func Test256ColorSupport(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test 256 color foreground
	screen.ProcessOutput([]byte("\033[38;5;196mRed256"))
	if screen.currentFgColor != "\033[38;5;196m" {
		t.Errorf("Expected 256 color foreground, got: %q", screen.currentFgColor)
	}
	
	// Test 256 color background
	screen.ProcessOutput([]byte("\033[48;5;46mGreen256"))
	if screen.currentBgColor != "\033[48;5;46m" {
		t.Errorf("Expected 256 color background, got: %q", screen.currentBgColor)
	}
}

// Test 24-bit RGB color support
func TestRGBColorSupport(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test RGB foreground color
	screen.ProcessOutput([]byte("\033[38;2;255;128;0mOrange"))
	if screen.currentFgColor != "\033[38;2;255;128;0m" {
		t.Errorf("Expected RGB foreground color, got: %q", screen.currentFgColor)
	}
	
	// Test RGB background color
	screen.ProcessOutput([]byte("\033[48;2;0;255;255mCyan"))
	if screen.currentBgColor != "\033[48;2;0;255;255m" {
		t.Errorf("Expected RGB background color, got: %q", screen.currentBgColor)
	}
}

// Test private mode sequences
func TestPrivateModeSequences(t *testing.T) {
	screen := NewTerminalScreen(10, 5)
	
	// Test alternate screen buffer with private mode
	screen.ProcessOutput([]byte("\033[?1049h")) // Enter alt screen
	if !screen.inAltScreen {
		t.Error("Expected to be in alternate screen mode")
	}
	
	screen.ProcessOutput([]byte("\033[?1049l")) // Exit alt screen
	if screen.inAltScreen {
		t.Error("Expected to exit alternate screen mode")
	}
	
	// Test cursor visibility
	screen.ProcessOutput([]byte("\033[?25l")) // Hide cursor
	if screen.cursorVisible {
		t.Error("Expected cursor to be hidden")
	}
	
	screen.ProcessOutput([]byte("\033[?25h")) // Show cursor
	if !screen.cursorVisible {
		t.Error("Expected cursor to be visible")
	}
}

// Test complex ANSI sequences mixed with UTF-8
func TestComplexSequencesWithUTF8(t *testing.T) {
	screen := NewTerminalScreen(20, 5)
	
	// Simulate complex output similar to btm
	complexOutput := "\033[38;5;12m╔═══════════════╗\033[0m\n" +
		"\033[38;5;12m║\033[0m \033[31mCPU: 25%\033[0m     \033[38;5;12m║\033[0m\n" +
		"\033[38;5;12m║\033[0m \033[32mRAM: 70%\033[0m     \033[38;5;12m║\033[0m\n" +
		"\033[38;5;12m╚═══════════════╝\033[0m"
	
	screen.ProcessOutput([]byte(complexOutput))
	captured := screen.CaptureFrame()
	
	// Should contain box-drawing characters
	if !strings.Contains(captured, "╔") || !strings.Contains(captured, "═") || !strings.Contains(captured, "╗") {
		t.Errorf("Box-drawing characters not preserved. Got: %q", captured)
	}
	
	// Should contain text content
	if !strings.Contains(captured, "CPU") || !strings.Contains(captured, "RAM") {
		t.Errorf("Text content not preserved. Got: %q", captured)
	}
}

// Benchmark tests
func BenchmarkPutChar(b *testing.B) {
	screen := NewTerminalScreen(80, 24)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.putChar('A')
		if screen.cursorX >= screen.width {
			screen.cursorX = 0
			screen.cursorY = (screen.cursorY + 1) % screen.height
		}
	}
}

func BenchmarkProcessOutput(b *testing.B) {
	screen := NewTerminalScreen(80, 24)
	data := []byte("Hello, World! This is a test string with some content.")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.ProcessOutput(data)
	}
}

func BenchmarkCaptureFrame(b *testing.B) {
	screen := NewTerminalScreen(80, 24)
	
	// Fill screen with content
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			screen.buffer[y][x].Char = 'X'
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = screen.CaptureFrame()
	}
}