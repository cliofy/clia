package commands

import (
	"context"

	"github.com/spf13/cobra"
)

// NewRunCommand creates the run command
func NewRunCommand(cli *CLI, ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [query]",
		Short: "Execute a natural language query",
		Long:  `Process a natural language query and execute the suggested command.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cli, ctx, args)
		},
	}

	return cmd
}