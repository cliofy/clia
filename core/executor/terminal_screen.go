package executor

// TerminalScreen defines the interface for terminal screen implementations
// This interface is implemented by GovteTerminalScreen for terminal emulation and screen capture
type TerminalScreen interface {
	// ProcessOutput processes terminal output data
	ProcessOutput(data []byte)
	
	// CaptureFrame captures the current screen content as a string
	CaptureFrame() string
	
	// GetLastFrame returns the last captured frame (typically from alternate screen exit)
	GetLastFrame() string
	
	// DetectedAltScreenExit returns true if we just exited alternate screen mode
	DetectedAltScreenExit() bool
}

// Ensure govte implementation satisfies the interface
var _ TerminalScreen = (*GovteTerminalScreen)(nil)