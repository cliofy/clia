package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/clia/internal/ai"
	"github.com/yourusername/clia/internal/renderer"
)

// AnalyzerTUIState represents the state of the analysis TUI
type AnalyzerTUIState int

const (
	StateAnalyzing AnalyzerTUIState = iota // AI is analyzing the data
	StateDisplaying                        // Showing analysis results
	StateError                             // Error occurred
)

// AnalyzerTUIModel represents the data analysis TUI model
type AnalyzerTUIModel struct {
	// Core state
	state            AnalyzerTUIState
	inputData        string
	analysisCommand  string
	
	// AI service
	aiService        *ai.Service
	
	// Results
	analysisResult   *ai.AnalysisResponse
	renderedContent  string
	errorMessage     string
	
	// UI components
	viewport         viewport.Model
	markdownRenderer *renderer.MarkdownRenderer
	
	// Layout
	width            int
	height           int
	ready            bool
}

// AnalysisCompleteMsg represents completion of analysis
type AnalysisCompleteMsg struct {
	result *ai.AnalysisResponse
	error  error
}

// NewAnalyzerTUIModel creates a new analyzer TUI model
func NewAnalyzerTUIModel(inputData, analysisCommand string) (*AnalyzerTUIModel, error) {
	// Initialize AI service
	aiService := ai.NewService().SetFallbackMode(true)
	
	// Try to configure providers based on available API keys
	if err := configureAIProviders(aiService); err != nil {
		return nil, fmt.Errorf("failed to configure AI providers: %w", err)
	}
	
	// Initialize markdown renderer with default options
	rendererOpts := renderer.DefaultRendererOptions()
	rendererOpts.Width = 78 // Default width, will be updated on window size
	markdownRenderer, err := renderer.NewMarkdownRenderer(rendererOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}
	
	// Create viewport
	vp := viewport.New(78, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)
	
	return &AnalyzerTUIModel{
		state:            StateAnalyzing,
		inputData:        inputData,
		analysisCommand:  analysisCommand,
		aiService:        aiService,
		viewport:         vp,
		markdownRenderer: markdownRenderer,
	}, nil
}

// configureAIProviders attempts to configure available AI providers
func configureAIProviders(aiService *ai.Service) error {
	var configErrors []string
	
	// Try OpenRouter first
	if apiKey := getEnvVar("OPENROUTER_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenRouter)
		config.APIKey = apiKey
		config.Model = "z-ai/glm-4.5-air:free"
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenRouter, config); err != nil {
			configErrors = append(configErrors, fmt.Sprintf("OpenRouter: %v", err))
		} else {
			return nil // Successfully configured
		}
	}
	
	// Try OpenAI
	if apiKey := getEnvVar("OPENAI_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeOpenAI)
		config.APIKey = apiKey
		config.Model = "gpt-3.5-turbo"
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeOpenAI, config); err != nil {
			configErrors = append(configErrors, fmt.Sprintf("OpenAI: %v", err))
		} else {
			return nil // Successfully configured
		}
	}
	
	// Try Anthropic
	if apiKey := getEnvVar("ANTHROPIC_API_KEY"); apiKey != "" {
		config := ai.DefaultProviderConfig(ai.ProviderTypeAnthropic)
		config.APIKey = apiKey
		config.Model = "claude-3-haiku-20240307"
		
		if err := aiService.SetProviderByConfig(ai.ProviderTypeAnthropic, config); err != nil {
			configErrors = append(configErrors, fmt.Sprintf("Anthropic: %v", err))
		} else {
			return nil // Successfully configured
		}
	}
	
	if len(configErrors) > 0 {
		return fmt.Errorf("no AI providers could be configured: %v", strings.Join(configErrors, ", "))
	}
	
	return fmt.Errorf("no API keys found - set OPENROUTER_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY")
}

// getEnvVar safely gets environment variable
func getEnvVar(key string) string {
	return os.Getenv(key)
}

// Init initializes the analyzer TUI
func (m AnalyzerTUIModel) Init() tea.Cmd {
	return tea.Batch(
		m.startAnalysis(),
		m.viewport.Init(),
	)
}

// startAnalysis begins the AI analysis process
func (m AnalyzerTUIModel) startAnalysis() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		result, err := m.aiService.AnalyzeData(ctx, m.inputData, m.analysisCommand)
		
		return AnalysisCompleteMsg{
			result: result,
			error:  err,
		}
	})
}

// Update handles messages and updates the model
func (m AnalyzerTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		
		// Update viewport size
		headerHeight := 3 // Title and borders
		footerHeight := 2 // Help text
		verticalMargins := headerHeight + footerHeight
		
		m.viewport.Width = msg.Width - 4  // Account for borders
		m.viewport.Height = msg.Height - verticalMargins
		
		// Update markdown renderer width
		if m.markdownRenderer != nil {
			m.markdownRenderer.SetWidth(msg.Width - 8) // Account for borders and padding
		}
		
		// Re-render content if we have it
		if m.state == StateDisplaying && m.analysisResult != nil {
			if rendered, err := m.markdownRenderer.Render(m.analysisResult.Result); err == nil {
				m.renderedContent = rendered
				m.viewport.SetContent(rendered)
			}
		}
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		default:
			// Pass key events to viewport for scrolling
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		
	case AnalysisCompleteMsg:
		if msg.error != nil {
			m.state = StateError
			m.errorMessage = msg.error.Error()
		} else {
			m.state = StateDisplaying
			m.analysisResult = msg.result
			
			// Render the analysis result with markdown
			if rendered, err := m.markdownRenderer.Render(msg.result.Result); err == nil {
				m.renderedContent = rendered
				m.viewport.SetContent(rendered)
			} else {
				// Fallback to plain text if markdown rendering fails
				m.renderedContent = msg.result.Result
				m.viewport.SetContent(msg.result.Result)
			}
		}
		
	default:
		// Pass other messages to viewport
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	
	return m, nil
}

// View renders the analyzer TUI
func (m AnalyzerTUIModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	
	switch m.state {
	case StateAnalyzing:
		return m.viewAnalyzing()
	case StateDisplaying:
		return m.viewResults()
	case StateError:
		return m.viewError()
	default:
		return "Unknown state"
	}
}

// viewAnalyzing renders the analyzing state
func (m AnalyzerTUIModel) viewAnalyzing() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Render("🤖 Analyzing Data...")
	
	info := fmt.Sprintf("Command: %s\nData length: %d bytes", 
		m.analysisCommand, len(m.inputData))
	
	spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏" // Simple spinner chars
	spinnerChar := string(spinner[0]) // Would rotate in real implementation
	
	content := fmt.Sprintf("%s\n\n%s %s\n\n%s", 
		title, spinnerChar, "Processing your request...", info)
	
	return lipgloss.Place(m.width, m.height, 
		lipgloss.Center, lipgloss.Center, content)
}

// viewResults renders the analysis results
func (m AnalyzerTUIModel) viewResults() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Render(fmt.Sprintf("📊 Analysis Results (%s)", m.analysisResult.AnalysisType))
	
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("↑/↓: Navigate • q: Quit")
	
	return fmt.Sprintf("%s\n%s\n%s", title, m.viewport.View(), help)
}

// viewError renders the error state
func (m AnalyzerTUIModel) viewError() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("9")).
		Render("❌ Analysis Failed")
	
	errorMsg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Render(m.errorMessage)
	
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("q: Quit")
	
	suggestions := `
💡 Troubleshooting suggestions:
• Ensure you have an API key configured:
  export OPENROUTER_API_KEY="your-key"
  export OPENAI_API_KEY="your-key"
• Check your internet connection
• Verify the input data format is supported`
	
	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s", 
		title, errorMsg, suggestions, help)
	
	return lipgloss.Place(m.width, m.height, 
		lipgloss.Center, lipgloss.Center, content)
}

// runAnalyzerTUI starts the analyzer TUI
func runAnalyzerTUI(inputData, analysisCommand string) error {
	model, err := NewAnalyzerTUIModel(inputData, analysisCommand)
	if err != nil {
		return fmt.Errorf("failed to create analyzer TUI: %w", err)
	}
	
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	_, err = program.Run()
	return err
}