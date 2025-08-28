package executor

// TerminalScreen defines the common interface for terminal screen implementations
// This interface abstracts the differences between AnsiTermScreen and GovteTerminalScreen
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

// Ensure both implementations satisfy the interface
var _ TerminalScreen = (*AnsiTermScreen)(nil)
var _ TerminalScreen = (*GovteTerminalScreen)(nil)