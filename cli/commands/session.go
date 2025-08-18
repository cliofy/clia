package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/clia/core/executor"
)

// NewSessionCommand creates the session command
func NewSessionCommand(cli *CLI, ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Start an interactive session",
		Long:  `Start an interactive CLIA session for continuous command assistance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveSession(cli, ctx)
		},
	}

	return cmd
}

// runInteractiveSession runs the interactive session
func runInteractiveSession(cli *CLI, ctx context.Context) error {
	// Welcome message
	color.HiCyan("Welcome to CLIA Interactive Session")
	color.HiBlack("Type 'help' for commands, 'exit' to quit")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	
	for {
		// Show prompt
		prompt := fmt.Sprintf("CLIA [%s] %s> ", 
			color.CyanString(cli.Config.ActiveProvider),
			color.HiBlackString(getCurrentDir()))
		fmt.Print(prompt)
		
		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				// Ctrl+D pressed
				fmt.Println("\nGoodbye!")
				return nil
			}
			return err
		}
		
		input = strings.TrimSpace(input)
		
		// Handle special commands
		switch input {
		case "":
			continue
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return nil
		case "help", "?":
			showSessionHelp()
			continue
		case "clear", "cls":
			clearScreen()
			continue
		case "provider":
			fmt.Printf("Active provider: %s\n", cli.Config.ActiveProvider)
			continue
		}
		
		// Handle direct execution (commands starting with !)
		if strings.HasPrefix(input, "!") {
			command := strings.TrimPrefix(input, "!")
			
			// Use intelligent detection with config
			decision := executor.IsInteractiveCommandWithConfig(command, cli.Config)
			
			if decision.IsInteractive {
				// Use interactive executor if available
				if extExec, ok := cli.Executor.(executor.ExtendedExecutor); ok {
					if verbose {
						cli.Output.Info(fmt.Sprintf("Interactive detection: %s (confidence: %.2f, method: %s)", 
							decision.Reason, decision.Confidence, decision.Method))
					}
					cli.Output.Info("Starting interactive session: " + command)
					
					// Check if we should capture the last frame
					shouldCapture := cli.Config.ShouldCaptureLastFrame()
					
					// Execute with optional capture
					lastFrame, err := extExec.ExecuteInteractiveWithCapture(command, shouldCapture)
					if err != nil {
						cli.Output.Error("Execution failed: " + err.Error())
					} else {
						// Display captured frame if available
						if lastFrame != "" {
							fmt.Println("\n" + strings.Repeat("=", 60))
							fmt.Println("Last Frame Captured:")
							fmt.Println(strings.Repeat("=", 60))
							fmt.Println(lastFrame)
							fmt.Println(strings.Repeat("=", 60))
						}
						// Learn from this execution if confidence is low
						if decision.Confidence < 0.8 {
							if learningErr := executor.LearnInteractiveCommand(command, true); learningErr != nil && verbose {
								cli.Output.Warning("Failed to save learning: " + learningErr.Error())
							}
						}
					}
					saveToHistory(command)
					continue
				}
			}
			
			cli.Output.Info("Executing: " + command)
			result, err := cli.Executor.Execute(command)
			if err != nil {
				cli.Output.Error("Execution failed: " + err.Error())
				continue
			}
			
			cli.Output.ShowExecutionResult(result)
			
			// Learn from this execution if detection was uncertain
			if decision.Confidence < 0.8 {
				if learningErr := executor.LearnInteractiveCommand(command, false); learningErr != nil && verbose {
					cli.Output.Warning("Failed to save learning: " + learningErr.Error())
				}
			}
			
			saveToHistory(command)
			continue
		}
		
		// Process as natural language query
		suggestion, err := cli.Agent.ProcessQuery(ctx, input)
		if err != nil {
			cli.Output.Error("Failed to process query: " + err.Error())
			continue
		}
		
		// Display suggestion
		cli.Output.ShowCommandSuggestion(suggestion)
		
		// Show risks if any
		if len(suggestion.Risks) > 0 {
			cli.Output.ShowRisks(suggestion.Risks)
		}
		
		// Confirm execution
		confirmed, err := cli.Output.ConfirmExecution(suggestion.Command)
		if err != nil {
			cli.Output.Error("Failed to read confirmation: " + err.Error())
			continue
		}
		
		if !confirmed {
			cli.Output.Info("Command cancelled")
			continue
		}
		
		// Execute command
		// Use intelligent detection with config
		decision := executor.IsInteractiveCommandWithConfig(suggestion.Command, cli.Config)
		
		if decision.IsInteractive {
			// Use interactive executor if available
			if extExec, ok := cli.Executor.(executor.ExtendedExecutor); ok {
				if verbose {
					cli.Output.Info(fmt.Sprintf("Interactive detection: %s (confidence: %.2f, method: %s)", 
						decision.Reason, decision.Confidence, decision.Method))
				}
				cli.Output.Info("Starting interactive session: " + suggestion.Command)
				
				// Check if we should capture the last frame
				shouldCapture := cli.Config.ShouldCaptureLastFrame()
				
				// Execute with optional capture
				lastFrame, err := extExec.ExecuteInteractiveWithCapture(suggestion.Command, shouldCapture)
				if err != nil {
					cli.Output.Error("Execution failed: " + err.Error())
				} else {
					// Display captured frame if available
					if lastFrame != "" {
						fmt.Println("\n" + strings.Repeat("=", 60))
						fmt.Println("Last Frame Captured:")
						fmt.Println(strings.Repeat("=", 60))
						fmt.Println(lastFrame)
						fmt.Println(strings.Repeat("=", 60))
					}
					
					// For interactive commands, we can't capture output, so add a placeholder
					cli.Agent.AddExecutionResult(suggestion.Command, "[Interactive session completed]", 0)
					
					// Learn from this execution if confidence is low
					if decision.Confidence < 0.8 {
						if learningErr := executor.LearnInteractiveCommand(suggestion.Command, true); learningErr != nil && verbose {
							cli.Output.Warning("Failed to save learning: " + learningErr.Error())
						}
					}
				}
				saveToHistory(suggestion.Command)
				continue
			}
		}
		
		cli.Output.Info("Executing: " + suggestion.Command)
		result, err := cli.Executor.Execute(suggestion.Command)
		if err != nil {
			cli.Output.Error("Execution failed: " + err.Error())
			continue
		}
		
		// Display result
		cli.Output.ShowExecutionResult(result)
		
		// Update agent context
		cli.Agent.AddExecutionResult(suggestion.Command, result.Output, result.ExitCode)
		
		// Learn from this execution if detection was uncertain
		if decision.Confidence < 0.8 {
			if learningErr := executor.LearnInteractiveCommand(suggestion.Command, false); learningErr != nil && verbose {
				cli.Output.Warning("Failed to save learning: " + learningErr.Error())
			}
		}
		
		// Save to history
		saveToHistory(suggestion.Command)
	}
}

// showSessionHelp displays help for the interactive session
func showSessionHelp() {
	fmt.Println()
	color.HiCyan("Interactive Session Commands:")
	fmt.Println()
	fmt.Println("  help, ?        Show this help")
	fmt.Println("  exit, quit, q  Exit the session")
	fmt.Println("  clear, cls     Clear the screen")
	fmt.Println("  provider       Show active provider")
	fmt.Println("  !<command>     Execute command directly (bypass AI)")
	fmt.Println()
	fmt.Println("  Ctrl+C         Cancel current input")
	fmt.Println("  Ctrl+D         Exit the session")
	fmt.Println()
}

// clearScreen clears the terminal screen
func clearScreen() {
	// ANSI escape code to clear screen
	fmt.Print("\033[2J\033[H")
}

// getCurrentDir returns the current directory name
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return filepath.Base(dir)
}