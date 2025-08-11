package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerType represents different types of spinners
type SpinnerType int

const (
	SpinnerClassic SpinnerType = iota
	SpinnerSimple
	SpinnerDots
	SpinnerPulse
)

// Spinner represents an animated loading indicator
type Spinner struct {
	Type     SpinnerType
	Frame    int
	Interval time.Duration
	Style    lipgloss.Style
}

// NewSpinner creates a new spinner with default settings
func NewSpinner() Spinner {
	return Spinner{
		Type:     SpinnerClassic,
		Frame:    0,
		Interval: 100 * time.Millisecond,
		Style:    lipgloss.NewStyle().Foreground(lipgloss.Color("69")), // Blue color
	}
}

// WithType sets the spinner type
func (s Spinner) WithType(t SpinnerType) Spinner {
	s.Type = t
	return s
}

// WithStyle sets the spinner style
func (s Spinner) WithStyle(style lipgloss.Style) Spinner {
	s.Style = style
	return s
}

// WithInterval sets the animation interval
func (s Spinner) WithInterval(interval time.Duration) Spinner {
	s.Interval = interval
	return s
}

// getFrames returns the animation frames for the spinner type
func (s Spinner) getFrames() []string {
	switch s.Type {
	case SpinnerClassic:
		return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	case SpinnerSimple:
		return []string{"-", "\\", "|", "/"}
	case SpinnerDots:
		return []string{"⠄", "⠆", "⠇", "⠋", "⠙", "⠸", "⠰", "⠠", "⠰", "⠸", "⠙", "⠋", "⠇", "⠆"}
	case SpinnerPulse:
		return []string{"○", "◎", "●", "◎"}
	default:
		return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
}

// View renders the current spinner frame
func (s Spinner) View() string {
	frames := s.getFrames()
	if len(frames) == 0 {
		return ""
	}
	
	currentFrame := frames[s.Frame%len(frames)]
	return s.Style.Render(currentFrame)
}

// NextFrame advances the spinner to the next frame
func (s Spinner) NextFrame() Spinner {
	frames := s.getFrames()
	s.Frame = (s.Frame + 1) % len(frames)
	return s
}

// Reset resets the spinner to the first frame
func (s Spinner) Reset() Spinner {
	s.Frame = 0
	return s
}

// TickCmd returns a command that will advance the spinner
func (s Spinner) TickCmd() tea.Cmd {
	return tea.Tick(s.Interval, func(t time.Time) tea.Msg {
		return SpinnerTickMsg{Time: t}
	})
}

// SpinnerTickMsg is sent when the spinner should advance
type SpinnerTickMsg struct {
	Time time.Time
}

// Animation-related message types for the TUI system

// startAnimationMsg indicates animation should start
type startAnimationMsg struct{}

// StartAnimationCmd returns a command to start animation
func StartAnimationCmd() tea.Cmd {
	return func() tea.Msg {
		return startAnimationMsg{}
	}
}

// stopAnimationMsg indicates animation should stop
type stopAnimationMsg struct{}

// StopAnimationCmd returns a command to stop animation
func StopAnimationCmd() tea.Cmd {
	return func() tea.Msg {
		return stopAnimationMsg{}
	}
}

// Predefined spinner styles for different contexts
var (
	// ProcessingSpinner for LLM processing
	ProcessingSpinner = NewSpinner().
				WithType(SpinnerClassic).
				WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("69"))) // Blue

	// LoadingSpinner for general loading
	LoadingSpinner = NewSpinner().
			WithType(SpinnerDots).
			WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("86"))) // Green

	// ErrorSpinner for error states
	ErrorSpinner = NewSpinner().
			WithType(SpinnerPulse).
			WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("204"))) // Red

	// SimpleSpinner for minimal UI
	SimpleSpinner = NewSpinner().
			WithType(SpinnerSimple).
			WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("243"))) // Gray
)

// SpinnerManager manages multiple spinners
type SpinnerManager struct {
	spinners map[string]Spinner
	active   map[string]bool
}

// NewSpinnerManager creates a new spinner manager
func NewSpinnerManager() *SpinnerManager {
	return &SpinnerManager{
		spinners: make(map[string]Spinner),
		active:   make(map[string]bool),
	}
}

// Add adds a spinner with a given name
func (sm *SpinnerManager) Add(name string, spinner Spinner) {
	sm.spinners[name] = spinner
	sm.active[name] = false
}

// Start starts a named spinner
func (sm *SpinnerManager) Start(name string) {
	if _, exists := sm.spinners[name]; exists {
		sm.active[name] = true
	}
}

// Stop stops a named spinner
func (sm *SpinnerManager) Stop(name string) {
	sm.active[name] = false
}

// IsActive returns whether a spinner is active
func (sm *SpinnerManager) IsActive(name string) bool {
	return sm.active[name]
}

// Tick advances all active spinners
func (sm *SpinnerManager) Tick() {
	for name, active := range sm.active {
		if active {
			spinner := sm.spinners[name]
			sm.spinners[name] = spinner.NextFrame()
		}
	}
}

// View returns the view for a named spinner
func (sm *SpinnerManager) View(name string) string {
	if spinner, exists := sm.spinners[name]; exists && sm.active[name] {
		return spinner.View()
	}
	return ""
}