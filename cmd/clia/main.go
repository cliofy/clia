package main

import (
	"context"
	"os"

	"github.com/fatih/color"
	"github.com/yourusername/clia/cli/commands"
)

func main() {
	ctx := context.Background()
	
	// Create and execute root command
	rootCmd := commands.NewRootCommand(ctx, false)
	
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}