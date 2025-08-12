package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/config"
	"github.com/yourusername/clia/internal/executor"
	"github.com/yourusername/clia/pkg/utils"
)

// CLIService holds the services needed for CLI mode
type CLIService struct {
	aiService    *ai.Service
	executor     *executor.Executor
	configManager *config.Manager
}

// runCLIMode processes a user request in CLI mode using TUI and exits
func runCLIMode(userRequest string) error {
	// Initialize services
	service, err := initializeCLIServices()
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}
	
	// Get AI suggestions (with fallback)
	suggestions, err := service.getAISuggestions(userRequest)
	if err != nil {
		// Try fallback suggestions if AI is not available
		if fallbackSuggestions := service.getFallbackSuggestions(userRequest); len(fallbackSuggestions) > 0 {
			suggestions = fallbackSuggestions
		} else {
			return fmt.Errorf("failed to get AI suggestions: %w", err)
		}
	}
	
	if len(suggestions) == 0 {
		fmt.Printf("❌ No command suggestions available for: %s\n", userRequest)
		return nil
	}
	
	// Start CLI TUI with suggestions
	return runCLITUI(userRequest, suggestions)
}

// initializeCLIServices initializes AI service and executor for CLI mode
func initializeCLIServices() (*CLIService, error) {
	// Initialize configuration manager
	configManager, err := config.NewManager()
	if err != nil {
		// Warning only, not fatal
		fmt.Printf("Warning: Failed to initialize config manager: %v\n", err)
	}
	
	// Initialize AI service
	aiService := ai.NewService().SetFallbackMode(true)
	
	// Initialize executor
	cmdExecutor := executor.New()
	
	// Try to configure providers based on available API keys
	var initErrors []string
	
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenRouter)
		config.APIKey = apiKey
		config.Model = "z-ai/glm-4.5-air:free"
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenRouter, config); err != nil {
			initErrors = append(initErrors, fmt.Sprintf("Failed to configure OpenRouter: %v", err))
		}
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenAI)
		config.APIKey = apiKey
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenAI, config); err != nil {
			initErrors = append(initErrors, fmt.Sprintf("Failed to configure OpenAI: %v", err))
		}
	} else {
		initErrors = append(initErrors, "No API keys found")
	}
	
	if len(initErrors) > 0 {
		fmt.Println("❌ Configuration Issues:")
		for _, err := range initErrors {
			fmt.Printf("  • %s\n", err)
		}
		fmt.Println("\n💡 To use AI features, set one of these environment variables:")
		fmt.Println("  export OPENROUTER_API_KEY=\"your-key-here\"")
		fmt.Println("  export OPENAI_API_KEY=\"your-key-here\"")
		fmt.Println()
	}
	
	return &CLIService{
		aiService:     aiService,
		executor:      cmdExecutor,
		configManager: configManager,
	}, nil
}

// getAISuggestions gets command suggestions from AI
func (s *CLIService) getAISuggestions(userRequest string) ([]ai.CommandSuggestion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	response, err := s.aiService.SuggestCommands(ctx, userRequest)
	if err != nil {
		return nil, err
	}
	
	return response.Suggestions, nil
}

// getFallbackSuggestions provides rule-based suggestions when AI is not available
func (s *CLIService) getFallbackSuggestions(userRequest string) []ai.CommandSuggestion {
	request := strings.ToLower(strings.TrimSpace(userRequest))
	
	var suggestions []ai.CommandSuggestion
	
	// Disk/space related commands
	if strings.Contains(request, "disk") || strings.Contains(request, "space") || 
	   strings.Contains(request, "storage") || strings.Contains(request, "剩余空间") {
		suggestions = append(suggestions, []ai.CommandSuggestion{
			{
				Command:     "df -h",
				Description: "Show filesystem disk space usage in human readable format",
				Safe:        true,
				Confidence:  0.9,
				Category:    "system",
			},
			{
				Command:     "du -sh *",
				Description: "Show directory sizes in current location",
				Safe:        true,
				Confidence:  0.85,
				Category:    "files",
			},
		}...)
	}
	
	// Directory/listing related commands
	if strings.Contains(request, "directory") || strings.Contains(request, "list") || 
	   strings.Contains(request, "files") || strings.Contains(request, "current") ||
	   strings.Contains(request, "目录") || strings.Contains(request, "文件") {
		suggestions = append(suggestions, []ai.CommandSuggestion{
			{
				Command:     "pwd",
				Description: "Print current working directory",
				Safe:        true,
				Confidence:  0.95,
				Category:    "navigation",
			},
			{
				Command:     "ls -la",
				Description: "List all files with detailed information",
				Safe:        true,
				Confidence:  0.9,
				Category:    "files",
			},
		}...)
	}
	
	// Process related commands
	if strings.Contains(request, "process") || strings.Contains(request, "running") || 
	   strings.Contains(request, "ps") || strings.Contains(request, "进程") {
		suggestions = append(suggestions, []ai.CommandSuggestion{
			{
				Command:     "ps aux",
				Description: "Show all running processes",
				Safe:        true,
				Confidence:  0.9,
				Category:    "system",
			},
			{
				Command:     "top",
				Description: "Display running processes in real time",
				Safe:        true,
				Confidence:  0.85,
				Category:    "system",
			},
		}...)
	}
	
	// Memory related commands
	if strings.Contains(request, "memory") || strings.Contains(request, "ram") || 
	   strings.Contains(request, "内存") {
		suggestions = append(suggestions, []ai.CommandSuggestion{
			{
				Command:     "free -h",
				Description: "Show memory usage in human readable format",
				Safe:        true,
				Confidence:  0.9,
				Category:    "system",
			},
		}...)
	}
	
	// Network related commands
	if strings.Contains(request, "network") || strings.Contains(request, "ip") || 
	   strings.Contains(request, "connection") || strings.Contains(request, "网络") {
		suggestions = append(suggestions, []ai.CommandSuggestion{
			{
				Command:     "ifconfig",
				Description: "Display network interface configuration",
				Safe:        true,
				Confidence:  0.9,
				Category:    "network",
			},
			{
				Command:     "ping -c 4 google.com",
				Description: "Test network connectivity",
				Safe:        true,
				Confidence:  0.85,
				Category:    "network",
			},
		}...)
	}
	
	// If no specific matches, provide some general useful commands
	if len(suggestions) == 0 {
		suggestions = []ai.CommandSuggestion{
			{
				Command:     "pwd",
				Description: "Print current working directory",
				Safe:        true,
				Confidence:  0.7,
				Category:    "navigation",
			},
			{
				Command:     "ls -la",
				Description: "List files in current directory",
				Safe:        true,
				Confidence:  0.7,
				Category:    "files",
			},
			{
				Command:     "df -h",
				Description: "Show disk space usage",
				Safe:        true,
				Confidence:  0.6,
				Category:    "system",
			},
		}
	}
	
	return suggestions
}

// displaySuggestionsAndGetChoice shows suggestions and gets user choice
func displaySuggestionsAndGetChoice(suggestions []ai.CommandSuggestion) (int, error) {
	fmt.Println("🤖 AI Suggestions (sorted by confidence):")
	fmt.Println()
	
	for i, suggestion := range suggestions {
		safetyIcon := "✅"
		if !suggestion.Safe {
			safetyIcon = "⚠️"
		}
		
		confidencePercent := int(suggestion.Confidence * 100)
		fmt.Printf("%d. %s %-30s - %s (%d%%)\n", 
			i+1, safetyIcon, suggestion.Command, suggestion.Description, confidencePercent)
	}
	
	fmt.Printf("\n🎯 Choose command (1-%d) or 'q' to quit: ", len(suggestions))
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "q" || input == "quit" {
		return -1, nil
	}
	
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(suggestions) {
		return -1, fmt.Errorf("invalid choice. Please enter 1-%d or 'q'", len(suggestions))
	}
	
	return choice - 1, nil // Convert to 0-based index
}

// executeCommand executes the selected command with safety checks
func (s *CLIService) executeCommand(suggestion ai.CommandSuggestion) error {
	fmt.Printf("\n🎯 Selected: %s\n", suggestion.Command)
	
	// Safety check
	isDangerous := utils.IsDangerousCommand(suggestion.Command)
	if isDangerous || !suggestion.Safe {
		fmt.Printf("⚠️  SAFETY WARNING: This command may be dangerous\n")
		fmt.Printf("🔍 Command: %s\n", suggestion.Command)
		
		if suggestion.Description != "" {
			fmt.Printf("📝 Description: %s\n", suggestion.Description)
		}
		
		confidencePercent := int(suggestion.Confidence * 100)
		fmt.Printf("🎯 AI Confidence: %d%%\n", confidencePercent)
		
		fmt.Printf("\n❓ Do you want to proceed? (y/N): ")
		
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("❌ Command execution cancelled")
			return nil
		}
		
		fmt.Println("✅ Command confirmed by user")
	}
	
	// Execute command
	fmt.Printf("\n🚀 Executing: %s\n", suggestion.Command)
	
	ctx := context.Background()
	result, err := s.executor.Execute(ctx, suggestion.Command)
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	
	// Display results
	fmt.Println("📤 Output:")
	if result.Stdout != "" {
		lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		for _, line := range lines {
			fmt.Printf("   %s\n", line)
		}
	}
	
	if result.Stderr != "" {
		fmt.Println("❌ Errors:")
		lines := strings.Split(strings.TrimSpace(result.Stderr), "\n")
		for _, line := range lines {
			fmt.Printf("   %s\n", line)
		}
	}
	
	// Display completion status
	if result.ExitCode == 0 {
		fmt.Printf("\n✅ Command completed successfully (%.2fs)\n", result.Duration.Seconds())
	} else {
		fmt.Printf("\n❌ Command failed with exit code %d (%.2fs)\n", result.ExitCode, result.Duration.Seconds())
	}
	
	return nil
}

// runCLITUI starts the CLI-style interactive selection with the given suggestions
func runCLITUI(userRequest string, suggestions []ai.CommandSuggestion) error {
	// Create the CLI TUI model
	model := NewCLITUIModel(userRequest, suggestions)
	
	// Create and run the TUI program WITHOUT alt screen (CLI-style)
	program := tea.NewProgram(
		model,
		// No WithAltScreen() - this keeps it in the CLI instead of full-screen TUI
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)
	
	// Run the program
	_, err := program.Run()
	return err
}