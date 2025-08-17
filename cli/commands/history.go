package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var historyLimit int

// NewHistoryCommand creates the history command
func NewHistoryCommand(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show command history",
		Long:  `Display the history of executed commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showHistory(cli, historyLimit)
		},
	}

	// Flags
	cmd.Flags().IntVar(&historyLimit, "limit", 20, "number of history items to show")

	// Subcommands
	cmd.AddCommand(newHistoryClearCommand(cli))

	return cmd
}

// newHistoryClearCommand creates the history clear command
func newHistoryClearCommand(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear command history",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := os.Getenv("CLIA_CONFIG_DIR")
			if configDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				configDir = filepath.Join(home, ".clia")
			}
			
			historyFile := filepath.Join(configDir, "history.txt")
			
			// Clear the file
			if err := os.WriteFile(historyFile, []byte{}, 0644); err != nil {
				return err
			}
			
			cli.Output.Success("History cleared")
			return nil
		},
	}
}

// showHistory displays the command history
func showHistory(cli *CLI, limit int) error {
	configDir := os.Getenv("CLIA_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".clia")
	}
	
	historyFile := filepath.Join(configDir, "history.txt")
	
	// Check if history file exists
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		cli.Output.Info("No history found")
		return nil
	}
	
	// Read history file
	file, err := os.Open(historyFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	// Display history (most recent first if limit is set)
	start := 0
	if limit > 0 && len(lines) > limit {
		start = len(lines) - limit
	}
	
	for i := start; i < len(lines); i++ {
		fmt.Printf("%4d  %s\n", i+1, lines[i])
	}
	
	return nil
}