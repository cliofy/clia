package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewProviderCommand creates the provider command
func NewProviderCommand(cli *CLI, ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage LLM providers",
		Long:  `Manage and configure LLM providers for CLIA.`,
	}

	// Subcommands
	cmd.AddCommand(
		newProviderListCommand(cli),
		newProviderActiveCommand(cli),
		newProviderSetCommand(cli),
		newProviderTestCommand(cli, ctx),
	)

	return cmd
}

// newProviderListCommand creates the provider list command
func newProviderListCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prepare table data
			headers := []string{"Provider", "Type", "Status", "Model"}
			var data [][]string

			for name, provider := range cli.Config.Providers {
				status := "configured"
				if name == cli.Config.ActiveProvider {
					status = "active"
				}
				
				model := ""
				if m, ok := provider.Config["model"].(string); ok {
					model = m
				}
				
				data = append(data, []string{
					name,
					provider.Type,
					status,
					model,
				})
			}

			cli.Output.ShowTable(headers, data)
			return nil
		},
	}
}

// newProviderActiveCommand creates the provider active command
func newProviderActiveCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "active",
		Short: "Show the active provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Active provider: %s\n", cli.Config.ActiveProvider)
			
			if provider, err := cli.Config.GetProvider(cli.Config.ActiveProvider); err == nil {
				fmt.Printf("  Type: %s\n", provider.Type)
				if model, ok := provider.Config["model"].(string); ok {
					fmt.Printf("  Model: %s\n", model)
				}
			}
			
			return nil
		},
	}
}

// newProviderSetCommand creates the provider set command
func newProviderSetCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "set [provider]",
		Short: "Set the active provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName := args[0]
			
			// Check if provider exists
			if _, err := cli.Config.GetProvider(providerName); err != nil {
				return fmt.Errorf("provider not found: %s", providerName)
			}
			
			// Set active provider
			cli.Config.ActiveProvider = providerName
			
			// Save configuration
			if err := cli.saveConfig(); err != nil {
				return err
			}
			
			cli.Output.Success(fmt.Sprintf("Switched to provider: %s", providerName))
			return nil
		},
	}
}

// newProviderTestCommand creates the provider test command
func newProviderTestCommand(cli *CLI, ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test the active provider connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.ProviderManager == nil {
				return fmt.Errorf("provider manager not initialized")
			}
			
			provider, err := cli.ProviderManager.GetActiveProvider()
			if err != nil {
				return fmt.Errorf("failed to get active provider: %w", err)
			}
			
			// Check if provider is available
			if provider.Available() {
				cli.Output.Success("Provider is available and working")
			} else {
				cli.Output.Error("Provider is not available")
			}
			
			// Try a simple query
			cli.Output.Info("Testing with simple query...")
			suggestion, err := cli.Agent.ProcessQuery(ctx, "echo test")
			if err != nil {
				return fmt.Errorf("test query failed: %w", err)
			}
			
			cli.Output.Success(fmt.Sprintf("Test successful. Suggested command: %s", suggestion.Command))
			
			return nil
		},
	}
}