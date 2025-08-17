package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewConfigCommand creates the config command
func NewConfigCommand(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage CLIA configuration settings.`,
	}

	// Subcommands
	cmd.AddCommand(
		newConfigSetCommand(cli),
		newConfigGetCommand(cli),
		newConfigListCommand(cli),
		newConfigPathCommand(cli),
	)

	return cmd
}

// newConfigSetCommand creates the config set command
func newConfigSetCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// Special handling for provider setting
			if key == "provider" {
				cli.Config.ActiveProvider = value
				cli.Output.Success(fmt.Sprintf("Provider set to: %s", value))
			} else {
				// Use viper for other settings
				viper.Set(key, value)
				cli.Output.Success(fmt.Sprintf("Set %s = %s", key, value))
			}

			// Save configuration
			return cli.saveConfig()
		},
	}
}

// newConfigGetCommand creates the config get command
func newConfigGetCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// Special handling for provider
			if key == "provider" {
				fmt.Println(cli.Config.ActiveProvider)
			} else {
				value := viper.Get(key)
				if value != nil {
					fmt.Println(value)
				} else {
					return fmt.Errorf("key not found: %s", key)
				}
			}

			return nil
		},
	}
}

// newConfigListCommand creates the config list command
func newConfigListCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.Output.Info("Current Configuration:")
			fmt.Println()
			
			// Show active provider
			fmt.Printf("provider: %s\n", cli.Config.ActiveProvider)
			
			// Show all providers
			fmt.Println("\nProviders:")
			for name, provider := range cli.Config.Providers {
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    type: %s\n", provider.Type)
				if model, ok := provider.Config["model"].(string); ok {
					fmt.Printf("    model: %s\n", model)
				}
			}
			
			return nil
		},
	}
}

// newConfigPathCommand creates the config path command
func newConfigPathCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := os.Getenv("CLIA_CONFIG_DIR")
			if configDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				configDir = filepath.Join(home, ".clia")
			}
			
			configFile := filepath.Join(configDir, "config.yaml")
			fmt.Println(configFile)
			
			return nil
		},
	}
}