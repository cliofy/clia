package executor

import (
	"strings"

	govte "github.com/cliofy/govte"
	"github.com/cliofy/govte/terminal"
)

// GovteTerminalScreen implements terminal screen capture using govte
// This uses the Parser+Performer architecture instead of Processor+Handler
type GovteTerminalScreen struct {
	// Screen dimensions
	width  int
	height int

	// Dual buffer management for main and alternate screens
	mainBuffer    *terminal.TerminalBuffer
	altBuffer     *terminal.TerminalBuffer
	currentBuffer *terminal.TerminalBuffer

	// State tracking
	inAltScreen   bool
	lastFrame     string
	lastFrameRead bool  // Track if last frame has been read to avoid repeated triggers

	// govte parser for parsing ANSI sequences
	parser *govte.Parser
}

// NewGovteTerminalScreen creates a new terminal screen using govte
func NewGovteTerminalScreen(width, height int) *GovteTerminalScreen {
	screen := &GovteTerminalScreen{
		width:      width,
		height:     height,
		mainBuffer: terminal.NewTerminalBuffer(width, height),
		altBuffer:  terminal.NewTerminalBuffer(width, height),
		parser:     govte.NewParser(),
	}
	
	screen.currentBuffer = screen.mainBuffer
	
	return screen
}

// ProcessOutput processes terminal output through govte parser
func (g *GovteTerminalScreen) ProcessOutput(data []byte) {
	// Use parser with ourselves as the performer wrapper
	// We'll delegate to the appropriate buffer and handle alt screen switching
	g.parser.Advance(g, data)
}

// CaptureFrame captures the current screen content as a string
func (g *GovteTerminalScreen) CaptureFrame() string {
	if g.inAltScreen {
		return g.altBuffer.GetDisplayWithColors()
	}
	return g.mainBuffer.GetDisplayWithColors()
}

// GetLastFrame returns the last captured frame (typically from alternate screen exit)
func (g *GovteTerminalScreen) GetLastFrame() string {
	g.lastFrameRead = true  // Mark frame as read
	return g.lastFrame
}

// DetectedAltScreenExit returns true if we just exited alternate screen
func (g *GovteTerminalScreen) DetectedAltScreenExit() bool {
	return !g.inAltScreen && g.lastFrame != "" && !g.lastFrameRead
}

//
// Performer Interface Implementation
// We implement the Performer interface to wrap TerminalBuffer and add alternate screen support
//

// Print implements Performer.Print
func (g *GovteTerminalScreen) Print(c rune) {
	g.currentBuffer.Print(c)
}

// Execute implements Performer.Execute
func (g *GovteTerminalScreen) Execute(b byte) {
	g.currentBuffer.Execute(b)
}

// Hook implements Performer.Hook
func (g *GovteTerminalScreen) Hook(params *govte.Params, intermediates []byte, ignore bool, action rune) {
	g.currentBuffer.Hook(params, intermediates, ignore, action)
}

// Put implements Performer.Put
func (g *GovteTerminalScreen) Put(b byte) {
	g.currentBuffer.Put(b)
}

// Unhook implements Performer.Unhook
func (g *GovteTerminalScreen) Unhook() {
	g.currentBuffer.Unhook()
}

// OscDispatch implements Performer.OscDispatch
func (g *GovteTerminalScreen) OscDispatch(params [][]byte, bellTerminated bool) {
	g.currentBuffer.OscDispatch(params, bellTerminated)
}

// CsiDispatch implements Performer.CsiDispatch with alternate screen detection
func (g *GovteTerminalScreen) CsiDispatch(params *govte.Params, intermediates []byte, ignore bool, action rune) {
	// Check for alternate screen mode changes before delegating
	if action == 'h' || action == 'l' {
		g.handleModeChange(params, action == 'h')
	}
	
	// Delegate to current buffer
	g.currentBuffer.CsiDispatch(params, intermediates, ignore, action)
}

// EscDispatch implements Performer.EscDispatch
func (g *GovteTerminalScreen) EscDispatch(intermediates []byte, ignore bool, b byte) {
	g.currentBuffer.EscDispatch(intermediates, ignore, b)
}

// handleModeChange handles terminal mode changes (like alternate screen)
func (g *GovteTerminalScreen) handleModeChange(params *govte.Params, setMode bool) {
	if params == nil {
		return
	}
	
	// Iterate through parameter groups
	paramGroups := params.Iter()
	for _, group := range paramGroups {
		if len(group) > 0 {
			param := group[0]
			// Check for ?1049 (save cursor + alt screen) or ?47 (alt screen)
			if param == 1049 || param == 47 {
				if setMode {
					g.enterAltScreen()
				} else {
					g.exitAltScreen()
				}
			}
		}
	}
}

// enterAltScreen switches to alternate screen
func (g *GovteTerminalScreen) enterAltScreen() {
	if !g.inAltScreen {
		g.inAltScreen = true
		// Create fresh alternate buffer
		g.altBuffer = terminal.NewTerminalBuffer(g.width, g.height)
		g.currentBuffer = g.altBuffer
	}
}

// exitAltScreen switches back to main screen and captures alternate screen content
func (g *GovteTerminalScreen) exitAltScreen() {
	if g.inAltScreen {
		// Capture the alternate screen content before switching
		g.lastFrame = g.altBuffer.GetDisplayWithColors()
		g.lastFrameRead = false
		
		// Switch back to main screen
		g.inAltScreen = false
		g.currentBuffer = g.mainBuffer
	}
}

//
// Utility methods for debugging
//

// GetCurrentScreenContent returns current screen content (for debugging)
func (g *GovteTerminalScreen) GetCurrentScreenContent() string {
	var result strings.Builder
	result.WriteString("=== Terminal Screen State ===\n")
	result.WriteString("Mode: ")
	if g.inAltScreen {
		result.WriteString("Alternate Screen\n")
	} else {
		result.WriteString("Main Screen\n")
	}
	result.WriteString("Content:\n")
	result.WriteString(g.currentBuffer.GetDisplay())
	result.WriteString("\n")
	return result.String()
}