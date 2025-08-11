package main

import (
	"fmt"
	"os"

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
	fmt.Println("  clia          Start the interactive TUI interface")
	fmt.Println("  clia version  Show version information")
	fmt.Println("  clia help     Show this help message")
	fmt.Println("\nINTERACTIVE MODE SHORTCUTS:")
	fmt.Println("  Ctrl+C        Quit the application")
	fmt.Println("  Ctrl+L        Clear message history")
	fmt.Println("  Enter         Submit your input")
	fmt.Println("\nFor more information, visit: https://github.com/yourusername/clia")
}