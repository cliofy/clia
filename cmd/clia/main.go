package main

import (
	"fmt"
	"os"

	"github.com/yourusername/clia/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("clia version %s (built with %s)\n", version.Version, version.GoVersion)
		return
	}

	fmt.Println("🚀 Welcome to clia - Command Line Intelligent Assistant")
	fmt.Println("This is Phase 0 - Project Initialization")
	fmt.Println("\nNext steps:")
	fmt.Println("  • Phase 1: TUI Framework Integration")
	fmt.Println("  • Phase 2: LLM Integration")
	fmt.Println("  • Phase 3: Command Execution")
	fmt.Println("\nRun 'clia version' to see version info")
}