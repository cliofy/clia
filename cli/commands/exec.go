package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/clia/core/executor"
)

// NewExecCommand creates the exec command
func NewExecCommand(cli *CLI, ctx context.Context) *cobra.Command {
	var forceInteractive, forceNonInteractive, captureLastFrame bool
	var timeout time.Duration
	
	cmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command directly",
		Long:  `Execute a shell command directly without AI processing.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Join all args as the command
			command := strings.Join(args, " ")
			
			// Check for conflicting flags
			if forceInteractive && forceNonInteractive {
				return fmt.Errorf("cannot use both --interactive and --no-interactive flags")
			}
			
			var shouldUseInteractive bool
			var decision *executor.InteractiveDecision
			
			if forceInteractive {
				shouldUseInteractive = true
				decision = &executor.InteractiveDecision{
					IsInteractive: true,
					Confidence:    1.0,
					Reason:        "forced by --interactive flag",
					Method:        "flag",
				}
			} else if forceNonInteractive {
				shouldUseInteractive = false
				decision = &executor.InteractiveDecision{
					IsInteractive: false,
					Confidence:    1.0,
					Reason:        "forced by --no-interactive flag",
					Method:        "flag",
				}
			} else {
				// Use intelligent detection with config
				decision = executor.IsInteractiveCommandWithConfig(command, cli.Config)
				shouldUseInteractive = decision.IsInteractive
			}
			
			// Show detection info if verbose
			if verbose {
				cli.Output.Info(fmt.Sprintf("Interactive detection: %s (confidence: %.2f, method: %s)", 
					decision.Reason, decision.Confidence, decision.Method))
			}
			
			// Use interactive mode if needed and available
			if shouldUseInteractive {
				if extExec, ok := cli.Executor.(executor.ExtendedExecutor); ok {
					cli.Output.Info("Starting interactive session: " + command)
					
					// Check if we should capture the last frame
					shouldCapture := captureLastFrame
					if !shouldCapture {
						// Check config for global capture setting
						shouldCapture = cli.Config.ShouldCaptureLastFrame()
					}
					
					// Auto-enable capture when timeout is specified
					if timeout > 0 {
						shouldCapture = true
					}
					
					// Execute with optional capture and timeout
					lastFrame, err := extExec.ExecuteInteractiveWithCaptureAndTimeout(command, shouldCapture, timeout)
					
					// Display captured frame if available
					if lastFrame != "" {
						fmt.Println("\n" + strings.Repeat("=", 60))
						fmt.Println("Last Frame Captured:")
						fmt.Println(strings.Repeat("=", 60))
						fmt.Println(lastFrame)
						fmt.Println(strings.Repeat("=", 60))
					}
					
					// Learn from this execution if confidence is low
					if decision.Confidence < 0.8 && err == nil {
						if learningErr := executor.LearnInteractiveCommand(command, true); learningErr != nil && verbose {
							cli.Output.Warning("Failed to save learning: " + learningErr.Error())
						}
					}
					
					saveToHistory(command)
					return err
				}
				// Fall back to regular execution with a warning
				cli.Output.Warning("Interactive mode not available, using standard execution")
			}
			
			cli.Output.Info("Executing: " + command)
			
			// Execute command normally
			result, err := cli.Executor.Execute(command)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}
			
			// Display result
			cli.Output.ShowExecutionResult(result)
			
			// Learn from this execution if detection was uncertain
			if decision.Confidence < 0.8 {
				if learningErr := executor.LearnInteractiveCommand(command, false); learningErr != nil && verbose {
					cli.Output.Warning("Failed to save learning: " + learningErr.Error())
				}
			}
			
			// Save to history
			if err := saveToHistory(command); err != nil {
				if verbose {
					cli.Output.Warning("Failed to save to history: " + err.Error())
				}
			}
			
			return nil
		},
	}

	// Add command-specific flags
	cmd.Flags().BoolVar(&forceInteractive, "interactive", false, "force interactive mode")
	cmd.Flags().BoolVar(&forceNonInteractive, "no-interactive", false, "force non-interactive mode")
	cmd.Flags().BoolVar(&captureLastFrame, "capture-frame", false, "capture and display the last frame of TUI programs")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "run TUI program for specified duration and then exit automatically (e.g., 5s, 2m)")

	return cmd
}