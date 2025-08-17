package output

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/yourusername/clia/core/agent"
	"github.com/yourusername/clia/core/executor"
)

// Formatter handles output formatting
type Formatter struct {
	colorEnabled bool
	verbose      bool
	successColor *color.Color
	errorColor   *color.Color
	warningColor *color.Color
	infoColor    *color.Color
	dimColor     *color.Color
}

// NewFormatter creates a new output formatter
func NewFormatter(colorEnabled, verbose bool) *Formatter {
	if !colorEnabled {
		color.NoColor = true
	}

	return &Formatter{
		colorEnabled: colorEnabled,
		verbose:      verbose,
		successColor: color.New(color.FgGreen, color.Bold),
		errorColor:   color.New(color.FgRed, color.Bold),
		warningColor: color.New(color.FgYellow, color.Bold),
		infoColor:    color.New(color.FgCyan),
		dimColor:     color.New(color.FgHiBlack),
	}
}

// Success prints a success message
func (f *Formatter) Success(message string) {
	f.successColor.Println("✓ " + message)
}

// Error prints an error message
func (f *Formatter) Error(message string) {
	f.errorColor.Println("✗ " + message)
}

// Warning prints a warning message
func (f *Formatter) Warning(message string) {
	f.warningColor.Println("⚠ " + message)
}

// Info prints an info message
func (f *Formatter) Info(message string) {
	f.infoColor.Println("ℹ " + message)
}

// Debug prints a debug message (only in verbose mode)
func (f *Formatter) Debug(message string) {
	if f.verbose {
		f.dimColor.Println("» " + message)
	}
}

// ShowCommandSuggestion displays a command suggestion
func (f *Formatter) ShowCommandSuggestion(suggestion *agent.CommandSuggestion) {
	fmt.Println()
	
	// Draw box around command
	f.drawBox("Suggested Command", func() {
		fmt.Println(suggestion.Command)
		
		if suggestion.Explanation != "" {
			fmt.Println()
			f.dimColor.Println(suggestion.Explanation)
		}
		
		if suggestion.Confidence > 0 {
			fmt.Printf("\nConfidence: %.0f%%\n", suggestion.Confidence*100)
		}
	})
	
	// Show alternatives if available
	if len(suggestion.Alternatives) > 0 {
		fmt.Println()
		f.infoColor.Println("Alternative commands:")
		for _, alt := range suggestion.Alternatives {
			fmt.Println("  • " + alt)
		}
	}
	
	fmt.Println()
}

// ShowRisks displays security risks
func (f *Formatter) ShowRisks(risks []agent.SecurityRisk) {
	if len(risks) == 0 {
		return
	}

	fmt.Println()
	f.warningColor.Println("⚠ SECURITY WARNINGS:")
	
	for _, risk := range risks {
		var levelColor *color.Color
		switch risk.Level {
		case agent.RiskCritical:
			levelColor = color.New(color.FgRed, color.Bold, color.BlinkSlow)
		case agent.RiskHigh:
			levelColor = color.New(color.FgRed, color.Bold)
		case agent.RiskMedium:
			levelColor = color.New(color.FgYellow)
		default:
			levelColor = color.New(color.FgWhite)
		}
		
		fmt.Printf("\n  ")
		levelColor.Printf("[%s]", strings.ToUpper(string(risk.Level)))
		fmt.Printf(" %s\n", risk.Description)
		
		if risk.Mitigation != "" {
			f.dimColor.Printf("    → %s\n", risk.Mitigation)
		}
	}
	fmt.Println()
}

// ShowExecutionResult displays command execution result
func (f *Formatter) ShowExecutionResult(result *executor.Result) {
	// Show output
	if result.Output != "" {
		fmt.Println(result.Output)
	}
	
	// Show error output if any
	if result.Error != nil {
		f.errorColor.Println(result.Error.Error())
	}
	
	// Show exit code if non-zero
	if result.ExitCode != 0 {
		f.errorColor.Printf("Exit code: %d\n", result.ExitCode)
	}
}

// ConfirmExecution asks for user confirmation
func (f *Formatter) ConfirmExecution(command string) (bool, error) {
	fmt.Print("Execute? (y/n/e[dit]) > ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	
	switch response {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "e", "edit":
		// TODO: Implement command editing
		f.Info("Command editing not yet implemented")
		return false, nil
	default:
		return false, nil
	}
}

// ShowTable displays data in a table format
func (f *Formatter) ShowTable(headers []string, data [][]string) {
	// For now, use simple formatting
	// TODO: Fix tablewriter API calls
	
	// Print headers
	fmt.Print("│")
	for _, h := range headers {
		fmt.Printf(" %-15s │", h)
	}
	fmt.Println()
	
	// Print separator
	fmt.Print("├")
	for range headers {
		fmt.Print("─────────────────┼")
	}
	fmt.Println()
	
	// Print data
	for _, row := range data {
		fmt.Print("│")
		for _, cell := range row {
			fmt.Printf(" %-15s │", cell)
		}
		fmt.Println()
	}
}

// drawBox draws a box around content
func (f *Formatter) drawBox(title string, content func()) {
	width := 50
	
	// Top border
	fmt.Print("╭─ ")
	f.infoColor.Print(title)
	fmt.Print(" ")
	remaining := width - len(title) - 2
	for i := 0; i < remaining; i++ {
		fmt.Print("─")
	}
	fmt.Println("╮")
	
	// Content with padding
	fmt.Print("│ ")
	// Capture content and add padding
	// For simplicity, we'll just call the content function
	// In a real implementation, we'd capture and format the output
	content()
	
	// Bottom border
	fmt.Print("╰")
	for i := 0; i < width+2; i++ {
		fmt.Print("─")
	}
	fmt.Println("╯")
}

// PrintJSON prints formatted JSON (for config display)
func (f *Formatter) PrintJSON(data interface{}) {
	// Simple JSON printing
	// In a real implementation, we'd use a proper JSON formatter
	fmt.Printf("%+v\n", data)
}

// Spinner starts a spinner for long-running operations
func (f *Formatter) Spinner(message string) func() {
	// Simple spinner implementation
	// In a real implementation, we'd use a proper spinner library
	fmt.Print(message + "...")
	return func() {
		fmt.Println(" done")
	}
}