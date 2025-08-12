package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/yourusername/clia/internal/tui"
	"github.com/yourusername/clia/internal/version"
)

func main() {
	// Handle command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("clia version %s (built with %s)\n", version.Version, version.GoVersion)
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			// If we have arguments that aren't special commands, run in CLI mode
			userRequest := strings.Join(os.Args[1:], " ")
			if err := runCLIMode(userRequest); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// Start TUI application
	model := tui.New()
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternative screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error starting TUI: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("clia - Command Line Intelligent Assistant v%s\n\n", version.Version)
	fmt.Println("USAGE:")
	fmt.Println("  clia                    Start the interactive TUI interface")
	fmt.Println("  clia <request>          Process request in CLI mode and exit")
	fmt.Println("  clia version            Show version information")
	fmt.Println("  clia help               Show this help message")
	fmt.Println("\nCLI MODE EXAMPLES:")
	fmt.Println("  clia show disk space    Get AI suggestions for disk usage commands")
	fmt.Println("  clia list large files   Find commands to list large files")
	fmt.Println("  clia current directory  Show current directory commands")
	fmt.Println("\nINTERACTIVE MODE SHORTCUTS:")
	fmt.Println("  Ctrl+C        Quit the application")
	fmt.Println("  Ctrl+L        Clear message history")
	fmt.Println("  Enter         Submit your input")
	fmt.Println("  !<command>    Execute command directly (no safety checks)")
	fmt.Println("\nCONFIGURATION:")
	fmt.Println("  Set OPENROUTER_API_KEY or OPENAI_API_KEY environment variable")
	fmt.Println("  to enable AI-powered command suggestions")
	fmt.Println("\nFor more information, visit: https://github.com/yourusername/clia")
}