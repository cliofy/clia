package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer handles markdown rendering with glamour
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
	style    string
}

// RendererOptions configures the markdown renderer
type RendererOptions struct {
	Width      int    // Terminal width for rendering
	Style      string // Glamour style ("auto", "dark", "light", "notty")
	WordWrap   bool   // Enable word wrapping
	ColorStyle string // Color style override
}

// DefaultRendererOptions returns sensible default options
func DefaultRendererOptions() *RendererOptions {
	return &RendererOptions{
		Width:      80,
		Style:      "auto",
		WordWrap:   true,
		ColorStyle: "",
	}
}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(opts *RendererOptions) (*MarkdownRenderer, error) {
	if opts == nil {
		opts = DefaultRendererOptions()
	}

	// Calculate glamour render width accounting for margins/padding
	const glamourGutter = 2
	glamourWidth := opts.Width - glamourGutter
	if glamourWidth < 20 {
		glamourWidth = 20 // Minimum readable width
	}

	// Configure glamour renderer options
	rendererOpts := []glamour.TermRendererOption{
		glamour.WithWordWrap(glamourWidth),
	}

	// Set style
	switch opts.Style {
	case "auto":
		rendererOpts = append(rendererOpts, glamour.WithAutoStyle())
	case "dark":
		rendererOpts = append(rendererOpts, glamour.WithStandardStyle("dark"))
	case "light":
		rendererOpts = append(rendererOpts, glamour.WithStandardStyle("base16"))
	case "notty":
		rendererOpts = append(rendererOpts, glamour.WithStandardStyle("notty"))
	default:
		// Try to use as custom style, fallback to auto
		if opts.Style != "" {
			rendererOpts = append(rendererOpts, glamour.WithStandardStyle(opts.Style))
		} else {
			rendererOpts = append(rendererOpts, glamour.WithAutoStyle())
		}
	}

	// Create renderer
	renderer, err := glamour.NewTermRenderer(rendererOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	return &MarkdownRenderer{
		renderer: renderer,
		width:    opts.Width,
		style:    opts.Style,
	}, nil
}

// Render renders markdown content to terminal-formatted string
func (r *MarkdownRenderer) Render(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}

	// Clean up markdown (remove excessive whitespace)
	markdown = strings.TrimSpace(markdown)

	// Render with glamour
	result, err := r.renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return result, nil
}

// RenderWithTitle renders markdown with a title header
func (r *MarkdownRenderer) RenderWithTitle(title, markdown string) (string, error) {
	if title != "" {
		// Add title as H1 if not already present
		if !strings.HasPrefix(strings.TrimSpace(markdown), "#") {
			markdown = fmt.Sprintf("# %s\n\n%s", title, markdown)
		}
	}

	return r.Render(markdown)
}

// SetWidth updates the renderer width
func (r *MarkdownRenderer) SetWidth(width int) error {
	// Recreate renderer with new width
	opts := &RendererOptions{
		Width:    width,
		Style:    r.style,
		WordWrap: true,
	}

	newRenderer, err := NewMarkdownRenderer(opts)
	if err != nil {
		return err
	}

	r.renderer = newRenderer.renderer
	r.width = width
	return nil
}

// GetWidth returns the current rendering width
func (r *MarkdownRenderer) GetWidth() int {
	return r.width
}

// PreviewMarkdown creates a short preview of markdown content
func (r *MarkdownRenderer) PreviewMarkdown(markdown string, maxLines int) (string, error) {
	rendered, err := r.Render(markdown)
	if err != nil {
		return "", err
	}

	lines := strings.Split(rendered, "\n")
	if len(lines) <= maxLines {
		return rendered, nil
	}

	// Take first maxLines and add truncation indicator
	preview := strings.Join(lines[:maxLines], "\n")
	preview += "\n\n... (truncated)"

	return preview, nil
}

// ValidateMarkdown checks if markdown content is valid
func (r *MarkdownRenderer) ValidateMarkdown(markdown string) error {
	_, err := r.renderer.Render(markdown)
	return err
}

// RenderTable specifically handles table rendering with better formatting
func (r *MarkdownRenderer) RenderTable(tableMarkdown string) (string, error) {
	// Ensure proper table formatting
	lines := strings.Split(tableMarkdown, "\n")
	var formattedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Ensure table rows start and end with |
		if strings.Contains(line, "|") && !strings.HasPrefix(line, "|") {
			line = "|" + line
		}
		if strings.Contains(line, "|") && !strings.HasSuffix(line, "|") {
			line = line + "|"
		}

		formattedLines = append(formattedLines, line)
	}

	formattedTable := strings.Join(formattedLines, "\n")
	return r.Render(formattedTable)
}

// RenderCodeBlock renders content as a code block with optional language
func (r *MarkdownRenderer) RenderCodeBlock(content, language string) (string, error) {
	codeBlock := fmt.Sprintf("```%s\n%s\n```", language, content)
	return r.Render(codeBlock)
}

// RenderList renders content as a markdown list
func (r *MarkdownRenderer) RenderList(items []string, ordered bool) (string, error) {
	var listMarkdown strings.Builder

	for i, item := range items {
		if ordered {
			listMarkdown.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		} else {
			listMarkdown.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}

	return r.Render(listMarkdown.String())
}

// GetSupportedStyles returns a list of supported glamour styles
func GetSupportedStyles() []string {
	return []string{
		"auto",
		"dark",
		"light",
		"notty",
		"ascii",
		"base16",
		"dracula",
		"github",
		"monokai",
		"paraiso",
		"solarized-dark",
		"solarized-light",
	}
}

// GetStyleDescription returns a description of what each style looks like
func GetStyleDescription(style string) string {
	descriptions := map[string]string{
		"auto":            "Automatically detect terminal capabilities",
		"dark":            "Dark theme optimized for dark terminals",
		"light":           "Light theme optimized for light terminals",
		"notty":           "Plain text output without colors",
		"ascii":           "ASCII-only output for maximum compatibility",
		"base16":          "Base16 color scheme",
		"dracula":         "Dracula theme with purple accents",
		"github":          "GitHub-style markdown rendering",
		"monokai":         "Monokai theme with vibrant colors",
		"paraiso":         "Paraiso theme with warm colors",
		"solarized-dark":  "Solarized dark theme",
		"solarized-light": "Solarized light theme",
	}

	if desc, exists := descriptions[style]; exists {
		return desc
	}
	return "Custom or unknown style"
}
