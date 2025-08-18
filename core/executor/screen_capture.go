package executor

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TerminalCell represents a single cell in the terminal screen
type TerminalCell struct {
	Char       rune   // The character at this position
	FgColor    string // Foreground color ANSI sequence
	BgColor    string // Background color ANSI sequence
	Attributes string // Text attributes (bold, underline, etc.)
}

// TerminalScreen represents a virtual terminal screen for capturing TUI output
type TerminalScreen struct {
	buffer       [][]TerminalCell // 2D buffer for screen content
	width        int              // Screen width in columns
	height       int              // Screen height in rows
	cursorX      int              // Current cursor X position
	cursorY      int              // Current cursor Y position
	savedCursorX int              // Saved cursor X position
	savedCursorY int              // Saved cursor Y position
	
	// Current text attributes
	currentFgColor    string
	currentBgColor    string
	currentAttributes string
	
	// Screen state
	inAltScreen     bool   // Whether we're in alternate screen mode
	lastFrame       string // Last captured frame content
	altScreenBuffer [][]TerminalCell // Buffer for alternate screen
	
	// Scroll region
	scrollTop    int // Top line of scroll region
	scrollBottom int // Bottom line of scroll region
	
	// Mode flags
	cursorVisible bool
	autowrap      bool
}

// NewTerminalScreen creates a new terminal screen with specified dimensions
func NewTerminalScreen(width, height int) *TerminalScreen {
	screen := &TerminalScreen{
		width:         width,
		height:        height,
		cursorX:       0,
		cursorY:       0,
		savedCursorX:  0,
		savedCursorY:  0,
		scrollTop:     0,
		scrollBottom:  height - 1,
		cursorVisible: true,
		autowrap:      true,
	}
	
	screen.initializeBuffer()
	return screen
}

// initializeBuffer initializes the screen buffer with empty cells
func (ts *TerminalScreen) initializeBuffer() {
	ts.buffer = make([][]TerminalCell, ts.height)
	for y := range ts.buffer {
		ts.buffer[y] = make([]TerminalCell, ts.width)
		for x := range ts.buffer[y] {
			ts.buffer[y][x] = TerminalCell{Char: ' '}
		}
	}
	
	// Initialize alternate screen buffer
	ts.altScreenBuffer = make([][]TerminalCell, ts.height)
	for y := range ts.altScreenBuffer {
		ts.altScreenBuffer[y] = make([]TerminalCell, ts.width)
		for x := range ts.altScreenBuffer[y] {
			ts.altScreenBuffer[y][x] = TerminalCell{Char: ' '}
		}
	}
}

// ProcessOutput processes terminal output and updates the virtual screen
func (ts *TerminalScreen) ProcessOutput(data []byte) {
	input := string(data) // Go automatically handles UTF-8 decoding
	
	// Use byte index to handle escape sequences correctly
	byteIndex := 0
	for byteIndex < len(input) {
		// Decode the next UTF-8 character
		char, size := utf8.DecodeRuneInString(input[byteIndex:])
		if size == 0 {
			// Invalid UTF-8, skip this byte
			byteIndex++
			continue
		}
		
		// Handle escape sequences
		if char == '\033' {
			// Find the remaining string from this position
			remaining := input[byteIndex:]
			seqLen, processed := ts.processEscapeSequence(remaining)
			if processed {
				byteIndex += seqLen
				continue
			}
		}
		
		// Handle control characters
		if ts.processControlChar(char) {
			byteIndex += size
			continue
		}
		
		// Handle printable characters
		if unicode.IsPrint(char) {
			ts.putChar(char)
		}
		
		byteIndex += size
	}
}

// processEscapeSequence processes ANSI escape sequences
func (ts *TerminalScreen) processEscapeSequence(input string) (int, bool) {
	if len(input) < 2 {
		return 0, false
	}
	
	// ESC [
	if input[1] == '[' {
		return ts.processCSI(input)
	}
	
	// ESC ]
	if input[1] == ']' {
		return ts.processOSC(input)
	}
	
	// ESC (
	if input[1] == '(' {
		return ts.processCharset(input)
	}
	
	// ESC =, ESC >, etc.
	switch input[1] {
	case '=': // Application keypad mode
		return 2, true
	case '>': // Normal keypad mode
		return 2, true
	case 'c': // Reset terminal
		ts.reset()
		return 2, true
	case 'D': // Index (move cursor down one line)
		ts.index()
		return 2, true
	case 'E': // Next line
		ts.nextLine()
		return 2, true
	case 'H': // Set tab stop
		return 2, true
	case 'M': // Reverse index (move cursor up one line)
		ts.reverseIndex()
		return 2, true
	}
	
	return 0, false
}

// processCSI processes Control Sequence Introducer (CSI) sequences
func (ts *TerminalScreen) processCSI(input string) (int, bool) {
	if len(input) < 3 {
		return 0, false
	}
	
	// Find the end of the CSI sequence
	i := 2
	for i < len(input) && (unicode.IsDigit(rune(input[i])) || input[i] == ';' || input[i] == '?' || input[i] == '!' || input[i] == '$' || input[i] == '"' || input[i] == '\'' || input[i] == ' ') {
		i++
	}
	
	if i >= len(input) {
		return 0, false
	}
	
	// Get the final character
	finalChar := input[i]
	sequence := input[2:i]
	
	// Check for private mode sequences (starting with '?')
	isPrivate := false
	if len(sequence) > 0 && sequence[0] == '?' {
		isPrivate = true
		sequence = sequence[1:] // Remove the '?' prefix
	}
	
	switch finalChar {
	case 'A': // Cursor up
		ts.cursorUp(ts.parseParams(sequence, 1)[0])
	case 'B': // Cursor down
		ts.cursorDown(ts.parseParams(sequence, 1)[0])
	case 'C': // Cursor forward
		ts.cursorForward(ts.parseParams(sequence, 1)[0])
	case 'D': // Cursor backward
		ts.cursorBackward(ts.parseParams(sequence, 1)[0])
	case 'E': // Cursor next line
		ts.cursorNextLine(ts.parseParams(sequence, 1)[0])
	case 'F': // Cursor previous line
		ts.cursorPrevLine(ts.parseParams(sequence, 1)[0])
	case 'G': // Cursor horizontal absolute
		ts.setCursorColumn(ts.parseParams(sequence, 1)[0])
	case 'H', 'f': // Cursor position
		params := ts.parseParams(sequence, 2)
		ts.setCursorPosition(params[0], params[1])
	case 'J': // Erase in display
		ts.eraseInDisplay(ts.parseParams(sequence, 1)[0])
	case 'K': // Erase in line
		ts.eraseInLine(ts.parseParams(sequence, 1)[0])
	case 'L': // Insert lines
		ts.insertLines(ts.parseParams(sequence, 1)[0])
	case 'M': // Delete lines
		ts.deleteLines(ts.parseParams(sequence, 1)[0])
	case 'P': // Delete characters
		ts.deleteChars(ts.parseParams(sequence, 1)[0])
	case 'S': // Scroll up
		ts.scrollUp(ts.parseParams(sequence, 1)[0])
	case 'T': // Scroll down
		ts.scrollDown(ts.parseParams(sequence, 1)[0])
	case 'X': // Erase characters
		ts.eraseChars(ts.parseParams(sequence, 1)[0])
	case 'm': // Set graphics rendition (colors, attributes)
		ts.setGraphicsRendition(sequence)
	case 'n': // Device status report
		// Ignore for now
	case 'r': // Set scroll region
		params := ts.parseParams(sequence, 2)
		ts.setScrollRegion(params[0], params[1])
	case 's': // Save cursor position
		ts.saveCursor()
	case 'u': // Restore cursor position
		ts.restoreCursor()
	case 'h': // Set mode
		ts.setMode(sequence, isPrivate)
	case 'l': // Reset mode
		ts.resetMode(sequence, isPrivate)
	}
	
	return i + 1, true
}

// processOSC processes Operating System Command sequences
func (ts *TerminalScreen) processOSC(input string) (int, bool) {
	// Find the terminator (BEL or ST)
	for i := 2; i < len(input); i++ {
		if input[i] == '\007' || (i+1 < len(input) && input[i] == '\033' && input[i+1] == '\\') {
			// Process OSC command (mostly ignored for screen capture)
			return i + 1, true
		}
	}
	return 0, false
}

// processCharset processes charset selection sequences
func (ts *TerminalScreen) processCharset(input string) (int, bool) {
	if len(input) >= 3 {
		// Ignore charset selection for now
		return 3, true
	}
	return 0, false
}

// processControlChar processes control characters
func (ts *TerminalScreen) processControlChar(char rune) bool {
	switch char {
	case '\r': // Carriage return
		ts.cursorX = 0
		return true
	case '\n': // Line feed
		ts.newLine()
		return true
	case '\t': // Tab
		ts.tab()
		return true
	case '\b': // Backspace
		if ts.cursorX > 0 {
			ts.cursorX--
		}
		return true
	case '\a': // Bell
		return true
	default:
		return false
	}
}

// putChar puts a character at the current cursor position
func (ts *TerminalScreen) putChar(char rune) {
	if ts.cursorY >= 0 && ts.cursorY < ts.height && ts.cursorX >= 0 && ts.cursorX < ts.width {
		currentBuffer := ts.getCurrentBuffer()
		currentBuffer[ts.cursorY][ts.cursorX] = TerminalCell{
			Char:       char,
			FgColor:    ts.currentFgColor,
			BgColor:    ts.currentBgColor,
			Attributes: ts.currentAttributes,
		}
	}
	
	// Advance cursor
	ts.cursorX++
	if ts.autowrap && ts.cursorX >= ts.width {
		ts.newLine()
	}
}

// getCurrentBuffer returns the current screen buffer
func (ts *TerminalScreen) getCurrentBuffer() [][]TerminalCell {
	if ts.inAltScreen {
		return ts.altScreenBuffer
	}
	return ts.buffer
}

// Movement and positioning methods
func (ts *TerminalScreen) cursorUp(n int) {
	ts.cursorY = max(ts.scrollTop, ts.cursorY-n)
}

func (ts *TerminalScreen) cursorDown(n int) {
	ts.cursorY = min(ts.scrollBottom, ts.cursorY+n)
}

func (ts *TerminalScreen) cursorForward(n int) {
	ts.cursorX = min(ts.width-1, ts.cursorX+n)
}

func (ts *TerminalScreen) cursorBackward(n int) {
	ts.cursorX = max(0, ts.cursorX-n)
}

func (ts *TerminalScreen) cursorNextLine(n int) {
	ts.cursorY = min(ts.scrollBottom, ts.cursorY+n)
	ts.cursorX = 0
}

func (ts *TerminalScreen) cursorPrevLine(n int) {
	ts.cursorY = max(ts.scrollTop, ts.cursorY-n)
	ts.cursorX = 0
}

func (ts *TerminalScreen) setCursorColumn(col int) {
	ts.cursorX = max(0, min(ts.width-1, col-1))
}

func (ts *TerminalScreen) setCursorPosition(row, col int) {
	ts.cursorY = max(0, min(ts.height-1, row-1))
	ts.cursorX = max(0, min(ts.width-1, col-1))
}

func (ts *TerminalScreen) newLine() {
	ts.cursorX = 0
	if ts.cursorY >= ts.scrollBottom {
		ts.scrollUp(1)
	} else {
		ts.cursorY++
	}
}

func (ts *TerminalScreen) tab() {
	// Move to next tab stop (every 8 characters)
	ts.cursorX = (ts.cursorX + 8) &^ 7
	if ts.cursorX >= ts.width {
		ts.cursorX = ts.width - 1
	}
}

func (ts *TerminalScreen) index() {
	if ts.cursorY >= ts.scrollBottom {
		ts.scrollUp(1)
	} else {
		ts.cursorY++
	}
}

func (ts *TerminalScreen) nextLine() {
	ts.index()
	ts.cursorX = 0
}

func (ts *TerminalScreen) reverseIndex() {
	if ts.cursorY <= ts.scrollTop {
		ts.scrollDown(1)
	} else {
		ts.cursorY--
	}
}

// Erase and clear methods
func (ts *TerminalScreen) eraseInDisplay(mode int) {
	currentBuffer := ts.getCurrentBuffer()
	
	switch mode {
	case 0: // Erase from cursor to end of screen
		// Clear from cursor to end of current line
		for x := ts.cursorX; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
		}
		// Clear all lines below cursor
		for y := ts.cursorY + 1; y < ts.height; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = TerminalCell{Char: ' '}
			}
		}
	case 1: // Erase from beginning of screen to cursor
		// Clear all lines above cursor
		for y := 0; y < ts.cursorY; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = TerminalCell{Char: ' '}
			}
		}
		// Clear from beginning of current line to cursor
		for x := 0; x <= ts.cursorX; x++ {
			currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
		}
	case 2, 3: // Erase entire screen
		for y := 0; y < ts.height; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = TerminalCell{Char: ' '}
			}
		}
	}
}

func (ts *TerminalScreen) eraseInLine(mode int) {
	currentBuffer := ts.getCurrentBuffer()
	
	switch mode {
	case 0: // Erase from cursor to end of line
		for x := ts.cursorX; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
		}
	case 1: // Erase from beginning of line to cursor
		for x := 0; x <= ts.cursorX; x++ {
			currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
		}
	case 2: // Erase entire line
		for x := 0; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
		}
	}
}

func (ts *TerminalScreen) eraseChars(n int) {
	currentBuffer := ts.getCurrentBuffer()
	for i := 0; i < n && ts.cursorX+i < ts.width; i++ {
		currentBuffer[ts.cursorY][ts.cursorX+i] = TerminalCell{Char: ' '}
	}
}

// Scrolling methods
func (ts *TerminalScreen) scrollUp(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines up
	for i := ts.scrollTop; i <= ts.scrollBottom-n; i++ {
		copy(currentBuffer[i], currentBuffer[i+n])
	}
	
	// Clear bottom lines
	for i := ts.scrollBottom - n + 1; i <= ts.scrollBottom; i++ {
		for x := 0; x < ts.width; x++ {
			currentBuffer[i][x] = TerminalCell{Char: ' '}
		}
	}
}

func (ts *TerminalScreen) scrollDown(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines down
	for i := ts.scrollBottom; i >= ts.scrollTop+n; i-- {
		copy(currentBuffer[i], currentBuffer[i-n])
	}
	
	// Clear top lines
	for i := ts.scrollTop; i < ts.scrollTop+n; i++ {
		for x := 0; x < ts.width; x++ {
			currentBuffer[i][x] = TerminalCell{Char: ' '}
		}
	}
}

// Insert and delete methods
func (ts *TerminalScreen) insertLines(n int) {
	// Similar to scroll down in current region
	for i := 0; i < n; i++ {
		ts.scrollDown(1)
	}
}

func (ts *TerminalScreen) deleteLines(n int) {
	// Similar to scroll up in current region
	for i := 0; i < n; i++ {
		ts.scrollUp(1)
	}
}

func (ts *TerminalScreen) deleteChars(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Shift characters left
	for x := ts.cursorX; x < ts.width-n; x++ {
		currentBuffer[ts.cursorY][x] = currentBuffer[ts.cursorY][x+n]
	}
	
	// Clear rightmost characters
	for x := ts.width - n; x < ts.width; x++ {
		currentBuffer[ts.cursorY][x] = TerminalCell{Char: ' '}
	}
}

// Graphics rendition (colors and attributes)
func (ts *TerminalScreen) setGraphicsRendition(params string) {
	if params == "" {
		// Reset all attributes
		ts.currentFgColor = ""
		ts.currentBgColor = ""
		ts.currentAttributes = ""
		return
	}
	
	paramList := ts.parseParams(params, -1)
	
	for i := 0; i < len(paramList); i++ {
		param := paramList[i]
		
		switch param {
		case 0: // Reset
			ts.currentFgColor = ""
			ts.currentBgColor = ""
			ts.currentAttributes = ""
		case 1: // Bold
			ts.currentAttributes += "\033[1m"
		case 2: // Dim
			ts.currentAttributes += "\033[2m"
		case 3: // Italic
			ts.currentAttributes += "\033[3m"
		case 4: // Underline
			ts.currentAttributes += "\033[4m"
		case 5, 6: // Blink
			ts.currentAttributes += "\033[5m"
		case 7: // Reverse
			ts.currentAttributes += "\033[7m"
		case 8: // Hidden
			ts.currentAttributes += "\033[8m"
		case 9: // Strikethrough
			ts.currentAttributes += "\033[9m"
		case 22: // Normal intensity
			// Remove bold/dim from attributes - simplified for now
		case 24: // No underline
			// Remove underline from attributes - simplified for now
		case 25: // No blink
			// Remove blink from attributes - simplified for now
		case 27: // No reverse
			// Remove reverse from attributes - simplified for now
		case 28: // No hidden
			// Remove hidden from attributes - simplified for now
		case 29: // No strikethrough
			// Remove strikethrough from attributes - simplified for now
		case 38: // Extended foreground color
			if i+1 < len(paramList) {
				switch paramList[i+1] {
				case 5: // 256 color mode
					if i+2 < len(paramList) {
						color := paramList[i+2]
						ts.currentFgColor = fmt.Sprintf("\033[38;5;%dm", color)
						i += 2
					}
				case 2: // 24-bit RGB color mode
					if i+4 < len(paramList) {
						r, g, b := paramList[i+2], paramList[i+3], paramList[i+4]
						ts.currentFgColor = fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
						i += 4
					}
				}
			}
		case 48: // Extended background color
			if i+1 < len(paramList) {
				switch paramList[i+1] {
				case 5: // 256 color mode
					if i+2 < len(paramList) {
						color := paramList[i+2]
						ts.currentBgColor = fmt.Sprintf("\033[48;5;%dm", color)
						i += 2
					}
				case 2: // 24-bit RGB color mode
					if i+4 < len(paramList) {
						r, g, b := paramList[i+2], paramList[i+3], paramList[i+4]
						ts.currentBgColor = fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
						i += 4
					}
				}
			}
		case 39: // Default foreground color
			ts.currentFgColor = ""
		case 49: // Default background color
			ts.currentBgColor = ""
		default:
			if param >= 30 && param <= 37 {
				// Standard foreground colors
				ts.currentFgColor = fmt.Sprintf("\033[%dm", param)
			} else if param >= 40 && param <= 47 {
				// Standard background colors
				ts.currentBgColor = fmt.Sprintf("\033[%dm", param)
			} else if param >= 90 && param <= 97 {
				// Bright foreground colors
				ts.currentFgColor = fmt.Sprintf("\033[%dm", param)
			} else if param >= 100 && param <= 107 {
				// Bright background colors
				ts.currentBgColor = fmt.Sprintf("\033[%dm", param)
			}
		}
	}
}

// Mode setting
func (ts *TerminalScreen) setMode(params string, isPrivate bool) {
	paramList := ts.parseParams(params, -1)
	
	for _, param := range paramList {
		if isPrivate {
			// Private mode sequences (DEC modes)
			switch param {
			case 1049: // Alternate screen buffer with cursor save
				ts.saveCursor()
				ts.enterAltScreen()
			case 1048: // Save cursor
				ts.saveCursor()
			case 1047: // Alternate screen buffer
				ts.enterAltScreen()
			case 25: // Show cursor
				ts.cursorVisible = true
			case 7: // Autowrap
				ts.autowrap = true
			case 1: // Application cursor keys
				// Not implemented
			case 6: // Origin mode
				// Not implemented
			}
		} else {
			// Standard ANSI modes
			switch param {
			case 4: // Insert mode
				// Not implemented
			case 20: // Automatic newline
				// Not implemented
			}
		}
	}
}

func (ts *TerminalScreen) resetMode(params string, isPrivate bool) {
	paramList := ts.parseParams(params, -1)
	
	for _, param := range paramList {
		if isPrivate {
			// Private mode sequences (DEC modes)
			switch param {
			case 1049: // Exit alternate screen buffer with cursor restore
				ts.exitAltScreen()
				ts.restoreCursor()
			case 1048: // Restore cursor
				ts.restoreCursor()
			case 1047: // Exit alternate screen buffer
				ts.exitAltScreen()
			case 25: // Hide cursor
				ts.cursorVisible = false
			case 7: // No autowrap
				ts.autowrap = false
			case 1: // Normal cursor keys
				// Not implemented
			case 6: // Normal cursor addressing
				// Not implemented
			}
		} else {
			// Standard ANSI modes
			switch param {
			case 4: // Replace mode
				// Not implemented
			case 20: // No automatic newline
				// Not implemented
			}
		}
	}
}

// Alternate screen methods
func (ts *TerminalScreen) enterAltScreen() {
	if !ts.inAltScreen {
		ts.inAltScreen = true
		// Don't clear the alternate screen buffer automatically
		// Only clear when explicitly commanded via escape sequences (like \033[2J)
		// This preserves content that was drawn before entering alternate screen
		
		// Reset cursor to top-left corner
		ts.cursorX = 0
		ts.cursorY = 0
	}
}

func (ts *TerminalScreen) exitAltScreen() {
	if ts.inAltScreen {
		// Capture the last frame before exiting
		ts.lastFrame = ts.CaptureFrame()
		ts.inAltScreen = false
	}
}

// Cursor save/restore
func (ts *TerminalScreen) saveCursor() {
	ts.savedCursorX = ts.cursorX
	ts.savedCursorY = ts.cursorY
}

func (ts *TerminalScreen) restoreCursor() {
	ts.cursorX = ts.savedCursorX
	ts.cursorY = ts.savedCursorY
}

// Scroll region
func (ts *TerminalScreen) setScrollRegion(top, bottom int) {
	ts.scrollTop = max(0, top-1)
	ts.scrollBottom = min(ts.height-1, bottom-1)
	if ts.scrollBottom <= ts.scrollTop {
		ts.scrollBottom = ts.height - 1
	}
}

// Reset terminal
func (ts *TerminalScreen) reset() {
	ts.cursorX = 0
	ts.cursorY = 0
	ts.savedCursorX = 0
	ts.savedCursorY = 0
	ts.currentFgColor = ""
	ts.currentBgColor = ""
	ts.currentAttributes = ""
	ts.inAltScreen = false
	ts.scrollTop = 0
	ts.scrollBottom = ts.height - 1
	ts.cursorVisible = true
	ts.autowrap = true
	ts.initializeBuffer()
}

// DetectedAltScreenExit returns true if we just exited alternate screen
func (ts *TerminalScreen) DetectedAltScreenExit() bool {
	return ts.lastFrame != ""
}

// CaptureFrame captures the current screen content as a string
func (ts *TerminalScreen) CaptureFrame() string {
	var result strings.Builder
	currentBuffer := ts.getCurrentBuffer()
	
	// Track current attributes to avoid redundant escape sequences
	lastFgColor := ""
	lastBgColor := ""
	lastAttributes := ""
	
	for y := 0; y < ts.height; y++ {
		lineHasContent := false
		
		// Check if line has any non-space content
		for x := 0; x < ts.width; x++ {
			if currentBuffer[y][x].Char != ' ' || 
			   currentBuffer[y][x].FgColor != "" || 
			   currentBuffer[y][x].BgColor != "" ||
			   currentBuffer[y][x].Attributes != "" {
				lineHasContent = true
				break
			}
		}
		
		if !lineHasContent && y < ts.height-1 {
			// Skip empty lines except the last one
			result.WriteString("\n")
			continue
		}
		
		for x := 0; x < ts.width; x++ {
			cell := currentBuffer[y][x]
			
			// Apply color and attribute changes
			if cell.Attributes != lastAttributes {
				if lastAttributes != "" {
					result.WriteString("\033[0m") // Reset
				}
				if cell.Attributes != "" {
					result.WriteString(cell.Attributes)
				}
				lastAttributes = cell.Attributes
				lastFgColor = "" // Reset color tracking after attribute change
				lastBgColor = ""
			}
			
			if cell.FgColor != lastFgColor {
				if cell.FgColor != "" {
					result.WriteString(cell.FgColor)
				}
				lastFgColor = cell.FgColor
			}
			
			if cell.BgColor != lastBgColor {
				if cell.BgColor != "" {
					result.WriteString(cell.BgColor)
				}
				lastBgColor = cell.BgColor
			}
			
			// Write the character
			if cell.Char != 0 {
				result.WriteRune(cell.Char)
			} else {
				result.WriteRune(' ')
			}
		}
		
		// Reset colors at end of line and add newline
		if lastFgColor != "" || lastBgColor != "" || lastAttributes != "" {
			result.WriteString("\033[0m")
			lastFgColor = ""
			lastBgColor = ""
			lastAttributes = ""
		}
		
		if y < ts.height-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// GetLastFrame returns the last captured frame (when exiting alt screen)
func (ts *TerminalScreen) GetLastFrame() string {
	frame := ts.lastFrame
	ts.lastFrame = "" // Clear after retrieval
	return frame
}

// parseParams parses parameter string into integers
func (ts *TerminalScreen) parseParams(params string, defaultCount int) []int {
	if params == "" {
		if defaultCount > 0 {
			result := make([]int, defaultCount)
			for i := range result {
				result[i] = 1
			}
			return result
		}
		return []int{0}
	}
	
	parts := strings.Split(params, ";")
	var result []int
	
	for _, part := range parts {
		if part == "" {
			result = append(result, 1)
		} else {
			val, err := strconv.Atoi(part)
			if err != nil {
				result = append(result, 1)
			} else {
				result = append(result, val)
			}
		}
	}
	
	// Pad with default values if needed
	for len(result) < defaultCount {
		result = append(result, 1)
	}
	
	return result
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}