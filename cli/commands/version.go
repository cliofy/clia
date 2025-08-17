package commands

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Version information (set during build)
	Version   = "0.1.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version information about CLIA.`,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}

	return cmd
}

// showVersion displays version information
func showVersion() {
	color.HiCyan("CLIA - Command Line Intelligence Assistant")
	fmt.Println()
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}