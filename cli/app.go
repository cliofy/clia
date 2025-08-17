package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/yourusername/clia/cli/commands"
)

// NewRootCommand creates a new root command for testing
func NewRootCommand(ctx context.Context, testMode bool) *cobra.Command {
	return commands.NewRootCommand(ctx, testMode)
}