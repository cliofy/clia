package executor

import (
	"strings"

	"github.com/Azure/go-ansiterm"
)

// AnsiTermCell represents a single cell in the terminal screen
type AnsiTermCell struct {
	Char       rune   // The character at this position
	FgColor    string // Foreground color ANSI sequence
	BgColor    string // Background color ANSI sequence  
	Attributes string // Text attributes (bold, underline, etc.)
}

// AnsiTermScreen represents a virtual terminal screen using go-ansiterm
type AnsiTermScreen struct {
	buffer       [][]AnsiTermCell // 2D buffer for screen content
	altBuffer    [][]AnsiTermCell // Alternate screen buffer
	width   int // Screen width in columns
	height  int // Screen height in rows
	cursorX int // Current cursor X position
	cursorY int // Current cursor Y position
	
	// Screen state
	inAltScreen bool // Whether we're in alternate screen mode
	
	// Current text attributes
	currentFgColor    string // Current foreground color
	currentBgColor    string // Current background color
	currentAttributes string // Current text attributes
	
	// Scroll region
	scrollTop    int // Top line of scroll region (0-based)
	scrollBottom int // Bottom line of scroll region (0-based)
	
	// Mode flags
	cursorVisible bool
	autowrap      bool
	
	// go-ansiterm parser
	parser *ansiterm.AnsiParser
	
	// Frame capture
	lastFrame string // Last captured frame
}

// NewAnsiTermScreen creates a new terminal screen using go-ansiterm
func NewAnsiTermScreen(width, height int) *AnsiTermScreen {
	screen := &AnsiTermScreen{
		width:        width,
		height:       height,
		scrollTop:    0,
		scrollBottom: height - 1,
		cursorVisible: true,
		autowrap:     true,
	}
	
	screen.initializeBuffers()
	
	// Create the ANSI parser with this screen as event handler
	screen.parser = ansiterm.CreateParser("Ground", screen)
	
	return screen
}

// initializeBuffers initializes both main and alternate screen buffers
func (ts *AnsiTermScreen) initializeBuffers() {
	ts.buffer = make([][]AnsiTermCell, ts.height)
	for y := range ts.buffer {
		ts.buffer[y] = make([]AnsiTermCell, ts.width)
		for x := range ts.buffer[y] {
			ts.buffer[y][x] = AnsiTermCell{Char: 0}
		}
	}
	
	// Initialize alternate screen buffer
	ts.altBuffer = make([][]AnsiTermCell, ts.height)
	for y := range ts.altBuffer {
		ts.altBuffer[y] = make([]AnsiTermCell, ts.width)
		for x := range ts.altBuffer[y] {
			ts.altBuffer[y][x] = AnsiTermCell{Char: 0}
		}
	}
}

// ProcessOutput processes terminal output through go-ansiterm parser
func (ts *AnsiTermScreen) ProcessOutput(data []byte) {
	ts.parser.Parse(data)
}

// getCurrentBuffer returns the currently active buffer
func (ts *AnsiTermScreen) getCurrentBuffer() [][]AnsiTermCell {
	if ts.inAltScreen {
		return ts.altBuffer
	}
	return ts.buffer
}

// CaptureFrame captures the current screen content as a string
func (ts *AnsiTermScreen) CaptureFrame() string {
	currentBuffer := ts.getCurrentBuffer()
	return ts.captureBuffer(currentBuffer)
}

// captureBuffer captures the content of a specific buffer
func (ts *AnsiTermScreen) captureBuffer(buffer [][]AnsiTermCell) string {
	var result strings.Builder
	
	for y := 0; y < ts.height && y < len(buffer); y++ {
		var line strings.Builder
		hasContent := false
		
		for x := 0; x < ts.width && x < len(buffer[y]); x++ {
			cell := buffer[y][x]
			if cell.Char != 0 {
				// Add color codes if present
				if cell.FgColor != "" {
					line.WriteString(cell.FgColor)
				}
				if cell.BgColor != "" {
					line.WriteString(cell.BgColor)
				}
				if cell.Attributes != "" {
					line.WriteString(cell.Attributes)
				}
				
				line.WriteRune(cell.Char)
				hasContent = true
				
				// Reset colors after character if they were set
				if cell.FgColor != "" || cell.BgColor != "" || cell.Attributes != "" {
					line.WriteString("\033[0m")
				}
			} else {
				line.WriteRune(' ')
			}
		}
		
		if hasContent || y < ts.height-1 {
			result.WriteString(strings.TrimRight(line.String(), " "))
		}
		if y < ts.height-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// GetLastFrame returns the last captured frame
func (ts *AnsiTermScreen) GetLastFrame() string {
	return ts.lastFrame
}

// DetectedAltScreenExit returns true if we just exited alternate screen
func (ts *AnsiTermScreen) DetectedAltScreenExit() bool {
	// This will be implemented based on tracking alt screen state changes
	return !ts.inAltScreen && ts.lastFrame != ""
}


//
// AnsiEventHandler interface implementation
// This implements all required methods for go-ansiterm
//

// Print handles printable characters
func (ts *AnsiTermScreen) Print(b byte) error {
	char := rune(b)
	ts.putChar(char)
	return nil
}

// Execute handles C0 control characters (0x00-0x1F)
func (ts *AnsiTermScreen) Execute(b byte) error {
	switch b {
	case '\r': // Carriage return
		ts.cursorX = 0
	case '\n': // Line feed
		ts.newLine()
	case '\b': // Backspace
		if ts.cursorX > 0 {
			ts.cursorX--
		}
	case '\t': // Tab
		ts.tab()
	case 0x07: // Bell
		// Ignore bell for now
	}
	return nil
}

// CUU - Cursor Up
func (ts *AnsiTermScreen) CUU(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorY = max(0, ts.cursorY-param)
	return nil
}

// CUD - Cursor Down
func (ts *AnsiTermScreen) CUD(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorY = min(ts.height-1, ts.cursorY+param)
	return nil
}

// CUF - Cursor Forward
func (ts *AnsiTermScreen) CUF(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorX = min(ts.width-1, ts.cursorX+param)
	return nil
}

// CUB - Cursor Backward
func (ts *AnsiTermScreen) CUB(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorX = max(0, ts.cursorX-param)
	return nil
}

// CNL - Cursor to Next Line
func (ts *AnsiTermScreen) CNL(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorY = min(ts.height-1, ts.cursorY+param)
	ts.cursorX = 0
	return nil
}

// CPL - Cursor to Previous Line
func (ts *AnsiTermScreen) CPL(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorY = max(0, ts.cursorY-param)
	ts.cursorX = 0
	return nil
}

// CHA - Cursor Horizontal position Absolute
func (ts *AnsiTermScreen) CHA(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorX = max(0, min(ts.width-1, param-1))
	return nil
}

// VPA - Vertical line Position Absolute
func (ts *AnsiTermScreen) VPA(param int) error {
	if param == 0 {
		param = 1
	}
	ts.cursorY = max(0, min(ts.height-1, param-1))
	return nil
}

// CUP - Cursor Position
func (ts *AnsiTermScreen) CUP(row, col int) error {
	if row == 0 {
		row = 1
	}
	if col == 0 {
		col = 1
	}
	ts.cursorY = max(0, min(ts.height-1, row-1))
	ts.cursorX = max(0, min(ts.width-1, col-1))
	return nil
}

// HVP - Horizontal and Vertical Position (same as CUP)
func (ts *AnsiTermScreen) HVP(row, col int) error {
	return ts.CUP(row, col)
}

// DECTCEM - Text Cursor Enable Mode
func (ts *AnsiTermScreen) DECTCEM(visible bool) error {
	ts.cursorVisible = visible
	return nil
}

// DECOM - Origin Mode
func (ts *AnsiTermScreen) DECOM(enable bool) error {
	// Origin mode - cursor addressing relative to scroll region
	// For now, we'll implement basic behavior
	if enable {
		ts.cursorY = ts.scrollTop
		ts.cursorX = 0
	}
	return nil
}

// enterAltScreen enters alternate screen mode
func (ts *AnsiTermScreen) enterAltScreen() {
	if !ts.inAltScreen {
		ts.inAltScreen = true
		// Move cursor to origin but don't clear buffer
		ts.cursorX = 0
		ts.cursorY = 0
	}
}

// exitAltScreen exits alternate screen mode  
func (ts *AnsiTermScreen) exitAltScreen() {
	if ts.inAltScreen {
		// Capture the alternate screen buffer directly before switching back
		ts.lastFrame = ts.captureBuffer(ts.altBuffer)
		ts.inAltScreen = false
	}
}

// DECCOLM - 132 Column Mode
func (ts *AnsiTermScreen) DECCOLM(use132 bool) error {
	// 132 column mode - would resize terminal
	// For now, ignore this as our terminal size is fixed
	return nil
}

// ED - Erase in Display
func (ts *AnsiTermScreen) ED(param int) error {
	currentBuffer := ts.getCurrentBuffer()
	
	switch param {
	case 0: // Erase from cursor to end of screen
		// Clear from cursor to end of current line
		for x := ts.cursorX; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = AnsiTermCell{Char: ' '}
		}
		// Clear all lines below cursor
		for y := ts.cursorY + 1; y < ts.height; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = AnsiTermCell{Char: ' '}
			}
		}
	case 1: // Erase from beginning of screen to cursor
		// Clear all lines above cursor
		for y := 0; y < ts.cursorY; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = AnsiTermCell{Char: ' '}
			}
		}
		// Clear from beginning of current line to cursor
		for x := 0; x <= ts.cursorX && x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = AnsiTermCell{Char: ' '}
		}
	case 2: // Erase entire screen
		for y := 0; y < ts.height; y++ {
			for x := 0; x < ts.width; x++ {
				currentBuffer[y][x] = AnsiTermCell{Char: ' '}
			}
		}
	}
	return nil
}

// EL - Erase in Line
func (ts *AnsiTermScreen) EL(param int) error {
	currentBuffer := ts.getCurrentBuffer()
	
	switch param {
	case 0: // Erase from cursor to end of line
		for x := ts.cursorX; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = AnsiTermCell{Char: ' '}
		}
	case 1: // Erase from beginning of line to cursor
		for x := 0; x <= ts.cursorX && x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = AnsiTermCell{Char: ' '}
		}
	case 2: // Erase entire line
		for x := 0; x < ts.width; x++ {
			currentBuffer[ts.cursorY][x] = AnsiTermCell{Char: ' '}
		}
	}
	return nil
}

// IL - Insert Line
func (ts *AnsiTermScreen) IL(param int) error {
	if param == 0 {
		param = 1
	}
	ts.insertLines(param)
	return nil
}

// DL - Delete Line
func (ts *AnsiTermScreen) DL(param int) error {
	if param == 0 {
		param = 1
	}
	ts.deleteLines(param)
	return nil
}

// ICH - Insert Character
func (ts *AnsiTermScreen) ICH(param int) error {
	if param == 0 {
		param = 1
	}
	ts.insertChars(param)
	return nil
}

// DCH - Delete Character
func (ts *AnsiTermScreen) DCH(param int) error {
	if param == 0 {
		param = 1
	}
	ts.deleteChars(param)
	return nil
}

// SGR - Set Graphics Rendition (colors and attributes)
func (ts *AnsiTermScreen) SGR(params []int) error {
	if len(params) == 0 {
		params = []int{0}
	}
	
	for _, param := range params {
		ts.setGraphicsRendition(param)
	}
	return nil
}

// SU - Pan Down (Scroll Up)
func (ts *AnsiTermScreen) SU(param int) error {
	if param == 0 {
		param = 1
	}
	ts.scrollUp(param)
	return nil
}

// SD - Pan Up (Scroll Down) 
func (ts *AnsiTermScreen) SD(param int) error {
	if param == 0 {
		param = 1
	}
	ts.scrollDown(param)
	return nil
}

// DA - Device Attributes
func (ts *AnsiTermScreen) DA(params []string) error {
	// Device attributes query - usually ignored in terminal emulators
	return nil
}

// DECSTBM - Set Top and Bottom Margins
func (ts *AnsiTermScreen) DECSTBM(top, bottom int) error {
	if top == 0 {
		top = 1
	}
	if bottom == 0 {
		bottom = ts.height
	}
	
	ts.scrollTop = max(0, min(ts.height-1, top-1))
	ts.scrollBottom = max(ts.scrollTop, min(ts.height-1, bottom-1))
	
	// Move cursor to origin
	ts.cursorY = ts.scrollTop
	ts.cursorX = 0
	return nil
}

// IND - Index (move cursor down one line)
func (ts *AnsiTermScreen) IND() error {
	if ts.cursorY >= ts.scrollBottom {
		ts.scrollUp(1)
	} else {
		ts.cursorY++
	}
	return nil
}

// RI - Reverse Index (move cursor up one line)
func (ts *AnsiTermScreen) RI() error {
	if ts.cursorY <= ts.scrollTop {
		ts.scrollDown(1)
	} else {
		ts.cursorY--
	}
	return nil
}

// Flush - Flush updates from previous commands
func (ts *AnsiTermScreen) Flush() error {
	// Update last frame when we flush, but only if we don't already have a frame
	// from alternate screen exit (which is more complete)
	if !ts.inAltScreen && ts.lastFrame == "" {
		ts.lastFrame = ts.CaptureFrame()
	}
	return nil
}

// SetAltScreenMode handles alternate screen buffer mode changes
func (ts *AnsiTermScreen) SetAltScreenMode(enable bool) error {
	if enable {
		ts.enterAltScreen()
	} else {
		ts.exitAltScreen()
	}
	return nil
}

//
// Helper functions for terminal operations
//

// putChar puts a character at the current cursor position
func (ts *AnsiTermScreen) putChar(char rune) {
	if ts.cursorY >= 0 && ts.cursorY < ts.height && ts.cursorX >= 0 && ts.cursorX < ts.width {
		currentBuffer := ts.getCurrentBuffer()
		
		currentBuffer[ts.cursorY][ts.cursorX] = AnsiTermCell{
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

// newLine moves cursor to next line
func (ts *AnsiTermScreen) newLine() {
	ts.cursorX = 0
	if ts.cursorY >= ts.scrollBottom {
		ts.scrollUp(1)
	} else {
		ts.cursorY++
	}
}

// tab moves cursor to next tab stop
func (ts *AnsiTermScreen) tab() {
	ts.cursorX = (ts.cursorX + 8) &^ 7
	if ts.cursorX >= ts.width {
		ts.cursorX = ts.width - 1
	}
}

// insertLines inserts n blank lines at cursor position
func (ts *AnsiTermScreen) insertLines(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines down
	for i := ts.scrollBottom; i >= ts.cursorY+n; i-- {
		if i-n >= 0 {
			copy(currentBuffer[i], currentBuffer[i-n])
		}
	}
	
	// Clear the inserted lines
	for i := 0; i < n && ts.cursorY+i < ts.height; i++ {
		for x := 0; x < ts.width; x++ {
			currentBuffer[ts.cursorY+i][x] = AnsiTermCell{Char: ' '}
		}
	}
}

// deleteLines deletes n lines at cursor position
func (ts *AnsiTermScreen) deleteLines(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines up
	for i := ts.cursorY; i <= ts.scrollBottom-n; i++ {
		if i+n < ts.height {
			copy(currentBuffer[i], currentBuffer[i+n])
		}
	}
	
	// Clear the bottom lines
	for i := ts.scrollBottom - n + 1; i <= ts.scrollBottom && i < ts.height; i++ {
		if i >= 0 {
			for x := 0; x < ts.width; x++ {
				currentBuffer[i][x] = AnsiTermCell{Char: ' '}
			}
		}
	}
}

// insertChars inserts n blank characters at cursor position
func (ts *AnsiTermScreen) insertChars(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move characters right
	for x := ts.width - 1; x >= ts.cursorX+n; x-- {
		if x-n >= 0 {
			currentBuffer[ts.cursorY][x] = currentBuffer[ts.cursorY][x-n]
		}
	}
	
	// Clear the inserted characters
	for i := 0; i < n && ts.cursorX+i < ts.width; i++ {
		currentBuffer[ts.cursorY][ts.cursorX+i] = AnsiTermCell{Char: ' '}
	}
}

// deleteChars deletes n characters at cursor position
func (ts *AnsiTermScreen) deleteChars(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move characters left
	for x := ts.cursorX; x < ts.width-n; x++ {
		if x+n < ts.width {
			currentBuffer[ts.cursorY][x] = currentBuffer[ts.cursorY][x+n]
		}
	}
	
	// Clear the end characters
	for i := ts.width - n; i < ts.width; i++ {
		if i >= 0 {
			currentBuffer[ts.cursorY][i] = AnsiTermCell{Char: ' '}
		}
	}
}

// scrollUp scrolls the screen up by n lines
func (ts *AnsiTermScreen) scrollUp(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines up within scroll region
	for i := ts.scrollTop; i <= ts.scrollBottom-n; i++ {
		if i+n <= ts.scrollBottom {
			copy(currentBuffer[i], currentBuffer[i+n])
		}
	}
	
	// Clear bottom lines
	for i := ts.scrollBottom - n + 1; i <= ts.scrollBottom; i++ {
		if i >= 0 && i < ts.height {
			for x := 0; x < ts.width; x++ {
				currentBuffer[i][x] = AnsiTermCell{Char: ' '}
			}
		}
	}
}

// scrollDown scrolls the screen down by n lines
func (ts *AnsiTermScreen) scrollDown(n int) {
	currentBuffer := ts.getCurrentBuffer()
	
	// Move lines down within scroll region
	for i := ts.scrollBottom; i >= ts.scrollTop+n; i-- {
		if i-n >= ts.scrollTop {
			copy(currentBuffer[i], currentBuffer[i-n])
		}
	}
	
	// Clear top lines
	for i := ts.scrollTop; i < ts.scrollTop+n && i < ts.height; i++ {
		if i >= 0 {
			for x := 0; x < ts.width; x++ {
				currentBuffer[i][x] = AnsiTermCell{Char: ' '}
			}
		}
	}
}

// setGraphicsRendition processes SGR parameters for colors and attributes
func (ts *AnsiTermScreen) setGraphicsRendition(param int) {
	switch param {
	case 0: // Reset all attributes
		ts.currentFgColor = ""
		ts.currentBgColor = ""
		ts.currentAttributes = ""
	case 1: // Bold
		ts.currentAttributes = "\033[1m"
	case 22: // Normal intensity (not bold)
		ts.currentAttributes = ""
	case 4: // Underline
		ts.currentAttributes = "\033[4m"
	case 24: // Not underlined
		ts.currentAttributes = ""
	case 7: // Reverse video
		ts.currentAttributes = "\033[7m"
	case 27: // Not reverse video
		ts.currentAttributes = ""
		
	// Foreground colors (30-37)
	case 30: // Black
		ts.currentFgColor = "\033[30m"
	case 31: // Red
		ts.currentFgColor = "\033[31m"
	case 32: // Green
		ts.currentFgColor = "\033[32m"
	case 33: // Yellow
		ts.currentFgColor = "\033[33m"
	case 34: // Blue
		ts.currentFgColor = "\033[34m"
	case 35: // Magenta
		ts.currentFgColor = "\033[35m"
	case 36: // Cyan
		ts.currentFgColor = "\033[36m"
	case 37: // White
		ts.currentFgColor = "\033[37m"
	case 39: // Default foreground
		ts.currentFgColor = ""
		
	// Background colors (40-47)
	case 40: // Black background
		ts.currentBgColor = "\033[40m"
	case 41: // Red background
		ts.currentBgColor = "\033[41m"
	case 42: // Green background
		ts.currentBgColor = "\033[42m"
	case 43: // Yellow background
		ts.currentBgColor = "\033[43m"
	case 44: // Blue background
		ts.currentBgColor = "\033[44m"
	case 45: // Magenta background
		ts.currentBgColor = "\033[45m"
	case 46: // Cyan background
		ts.currentBgColor = "\033[46m"
	case 47: // White background
		ts.currentBgColor = "\033[47m"
	case 49: // Default background
		ts.currentBgColor = ""
	}
}