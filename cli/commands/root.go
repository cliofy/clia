package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/clia/cli/output"
	"github.com/yourusername/clia/core/agent"
	"github.com/yourusername/clia/core/config"
	"github.com/yourusername/clia/core/executor"
	"github.com/yourusername/clia/core/provider"
)

// CLI holds the application state
type CLI struct {
	Config         *config.Config
	Agent          agent.Agent
	Executor       executor.Executor
	ProviderManager *provider.Manager
	Output         *output.Formatter
	TestMode       bool
}

// Global flags
var (
	cfgFile    string
	noColor    bool
	verbose    bool
	dryRun     bool
	autoConfirm bool
)

// NewRootCommand creates the root command
func NewRootCommand(ctx context.Context, testMode bool) *cobra.Command {
	cli := &CLI{
		TestMode: testMode,
	}

	rootCmd := &cobra.Command{
		Use:   "clia [query]",
		Short: "CLIA - Command Line Intelligence Assistant",
		Long: color.HiCyanString(`
   _____ _      _____          
  / ____| |    |_   _|   /\    
 | |    | |      | |    /  \   
 | |    | |      | |   / /\ \  
 | |____| |____ _| |_ / ____ \ 
  \_____|______|_____/_/    \_\
                                
`) + `CLIA helps you execute commands using natural language.

Examples:
  clia "show disk usage"
  clia "find large files"
  clia exec "docker ps -a"
  clia config set provider openai
  clia session`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration
			if err := cli.initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			// Initialize output formatter
			cli.Output = output.NewFormatter(!noColor, verbose)

			// Initialize components (only for commands that need them)
			needsComponents := false
			switch cmd.Name() {
			case "run", "exec", "session":
				needsComponents = true
			case "provider":
				// Provider test command needs components
				if len(args) > 0 && args[0] == "test" {
					needsComponents = true
				}
			}
			
			if needsComponents {
				if err := cli.initComponents(ctx); err != nil {
					// Non-fatal for some commands
					if verbose {
						color.Yellow("Warning: %v", err)
					}
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: run query if provided
			if len(args) > 0 {
				return runQuery(cli, ctx, args)
			}
			// Otherwise show help
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.clia/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show command without executing")
	rootCmd.PersistentFlags().BoolVarP(&autoConfirm, "yes", "y", false, "skip confirmation prompts")

	// Add subcommands
	rootCmd.AddCommand(
		NewRunCommand(cli, ctx),
		NewExecCommand(cli, ctx),
		NewConfigCommand(cli),
		NewProviderCommand(cli, ctx),
		NewHistoryCommand(cli),
		NewSessionCommand(cli, ctx),
		NewVersionCommand(),
	)

	return rootCmd
}

// initConfig initializes the configuration
func (cli *CLI) initConfig() error {
	if cli.TestMode {
		// Use test configuration
		cli.Config = config.DefaultConfig()
		return nil
	}

	// Determine config directory
	configDir := os.Getenv("CLIA_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".clia")
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Set config file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("CLIA")
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create default
			cli.Config = config.DefaultConfig()
			if err := cli.saveConfig(); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Load config from file
		cfg, err := config.Load(viper.ConfigFileUsed())
		if err != nil {
			return err
		}
		cli.Config = cfg
	}

	return nil
}

// initComponents initializes the core components
func (cli *CLI) initComponents(ctx context.Context) error {
	if cli.TestMode {
		// Use mock components for testing
		cli.initTestComponents()
		return nil
	}

	// Initialize executor with interactive support
	cli.Executor = executor.NewExtendedExecutor()

	// Initialize provider manager
	cli.ProviderManager = provider.NewManager(cli.Config)

	// Get active provider
	activeProvider, err := cli.ProviderManager.GetActiveProvider()
	if err != nil {
		return fmt.Errorf("failed to get active provider: %w", err)
	}

	// Initialize agent
	agentConfig := &agent.Config{
		Provider:      cli.Config.ActiveProvider,
		Model:         getModelFromConfig(cli.Config),
		Temperature:   0.7,
		MaxTokens:     1000,
		SafetyEnabled: true,
		ContextSize:   10,
	}
	cli.Agent = agent.NewAgentImpl(agentConfig, activeProvider)

	return nil
}

// initTestComponents initializes mock components for testing
func (cli *CLI) initTestComponents() {
	// Use mock implementations for testing
	// These would be created in the test package
	cli.Config = config.DefaultConfig()
	// Mock executor, agent, and provider would be set here
}

// saveConfig saves the configuration to file
func (cli *CLI) saveConfig() error {
	configDir := os.Getenv("CLIA_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".clia")
	}

	configFile := filepath.Join(configDir, "config.yaml")
	return config.SaveToFile(cli.Config, configFile)
}

// getModelFromConfig extracts the model from the active provider config
func getModelFromConfig(cfg *config.Config) string {
	if provider, err := cfg.GetProvider(cfg.ActiveProvider); err == nil {
		if model, ok := provider.Config["model"].(string); ok {
			return model
		}
	}
	return "gpt-3.5-turbo" // Default model
}

// runQuery handles the default behavior of running a natural language query
func runQuery(cli *CLI, ctx context.Context, args []string) error {
	query := args[0]
	
	// Process query with agent
	suggestion, err := cli.Agent.ProcessQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	// Display suggestion
	cli.Output.ShowCommandSuggestion(suggestion)

	// Check for risks
	if len(suggestion.Risks) > 0 {
		cli.Output.ShowRisks(suggestion.Risks)
	}

	// Handle dry-run
	if dryRun {
		cli.Output.Info("[DRY RUN] Command not executed")
		return nil
	}

	// Confirm execution
	if !autoConfirm && !cli.TestMode {
		confirmed, err := cli.Output.ConfirmExecution(suggestion.Command)
		if err != nil {
			return err
		}
		if !confirmed {
			cli.Output.Info("Command cancelled")
			return nil
		}
	}

	// Execute command
	// Check if this is an interactive command
	if executor.IsInteractiveCommand(suggestion.Command) {
		// Use interactive executor if available
		if extExec, ok := cli.Executor.(executor.ExtendedExecutor); ok {
			cli.Output.Info("Starting interactive session: " + suggestion.Command)
			if err := extExec.ExecuteInteractive(suggestion.Command); err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}
			// For interactive commands, we can't capture output, so add a placeholder
			cli.Agent.AddExecutionResult(suggestion.Command, "[Interactive session completed]", 0)
			saveToHistory(suggestion.Command)
			return nil
		}
	}
	
	cli.Output.Info("Executing: " + suggestion.Command)
	result, err := cli.Executor.Execute(suggestion.Command)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Display result
	cli.Output.ShowExecutionResult(result)

	// Add to agent context
	cli.Agent.AddExecutionResult(suggestion.Command, result.Output, result.ExitCode)

	// Save to history
	if err := saveToHistory(suggestion.Command); err != nil {
		// Non-fatal error
		if verbose {
			cli.Output.Warning("Failed to save to history: " + err.Error())
		}
	}

	return nil
}

// saveToHistory saves a command to the history file
func saveToHistory(command string) error {
	configDir := os.Getenv("CLIA_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".clia")
	}

	historyFile := filepath.Join(configDir, "history.txt")
	
	// Append to history file
	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, command)
	return err
}