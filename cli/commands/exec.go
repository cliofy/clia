package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/clia/core/executor"
)

// NewExecCommand creates the exec command
func NewExecCommand(cli *CLI, ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command directly",
		Long:  `Execute a shell command directly without AI processing.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Join all args as the command
			command := strings.Join(args, " ")
			
			// Check if this is an interactive command
			if executor.IsInteractiveCommand(command) {
				// Use interactive executor if available
				if extExec, ok := cli.Executor.(executor.ExtendedExecutor); ok {
					cli.Output.Info("Starting interactive session: " + command)
					return extExec.ExecuteInteractive(command)
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
			
			// Save to history
			if err := saveToHistory(command); err != nil {
				if verbose {
					cli.Output.Warning("Failed to save to history: " + err.Error())
				}
			}
			
			return nil
		},
	}

	return cmd
}