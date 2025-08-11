package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Context represents the current environment context
type Context struct {
	// System information
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Shell    string `json:"shell"`
	User     string `json:"user"`
	Hostname string `json:"hostname"`
	
	// Directory information
	WorkingDir     string   `json:"working_dir"`
	Files          []string `json:"files"`
	DirectoryCount int      `json:"directory_count"`
	FileCount      int      `json:"file_count"`
	
	// Environment
	EnvVars map[string]string `json:"env_vars,omitempty"`
}

// ContextCollector collects environment context information
type ContextCollector struct {
	maxFiles      int
	includeHidden bool
	maxPathDepth  int
	includeEnvVars bool
}

// NewContextCollector creates a new context collector
func NewContextCollector() *ContextCollector {
	return &ContextCollector{
		maxFiles:      20,      // Limit file list to avoid huge prompts
		includeHidden: false,   // Skip hidden files by default
		maxPathDepth:  3,       // Limit subdirectory depth
		includeEnvVars: false,  // Don't include env vars by default for privacy
	}
}

// SetMaxFiles sets the maximum number of files to include
func (c *ContextCollector) SetMaxFiles(max int) *ContextCollector {
	c.maxFiles = max
	return c
}

// SetIncludeHidden sets whether to include hidden files
func (c *ContextCollector) SetIncludeHidden(include bool) *ContextCollector {
	c.includeHidden = include
	return c
}

// SetIncludeEnvVars sets whether to include environment variables
func (c *ContextCollector) SetIncludeEnvVars(include bool) *ContextCollector {
	c.includeEnvVars = include
	return c
}

// Collect gathers current environment context
func (c *ContextCollector) Collect() (*Context, error) {
	ctx := &Context{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	ctx.WorkingDir = wd
	
	// Get user information
	if user := os.Getenv("USER"); user != "" {
		ctx.User = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		ctx.User = user
	}
	
	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		ctx.Hostname = hostname
	}
	
	// Get shell information
	ctx.Shell = c.detectShell()
	
	// Collect files in current directory
	if files, dirCount, fileCount, err := c.collectFiles(wd); err == nil {
		ctx.Files = files
		ctx.DirectoryCount = dirCount
		ctx.FileCount = fileCount
	}
	
	// Collect relevant environment variables (if enabled)
	if c.includeEnvVars {
		ctx.EnvVars = c.collectRelevantEnvVars()
	}
	
	return ctx, nil
}

// detectShell attempts to detect the current shell
func (c *ContextCollector) detectShell() string {
	// Check SHELL environment variable
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}
	
	// Check common shell indicators
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}
	if os.Getenv("FISH_VERSION") != "" {
		return "fish"
	}
	
	// Default based on OS
	switch runtime.GOOS {
	case "windows":
		return "powershell"
	default:
		return "bash"
	}
}

// collectFiles collects files in the specified directory
func (c *ContextCollector) collectFiles(dir string) ([]string, int, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, 0, err
	}
	
	var files []string
	var dirCount, fileCount int
	
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files unless explicitly requested
		if !c.includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		
		if entry.IsDir() {
			dirCount++
			files = append(files, name+"/")
		} else {
			fileCount++
			files = append(files, name)
		}
		
		// Limit the number of files to avoid overwhelming the prompt
		if len(files) >= c.maxFiles {
			break
		}
	}
	
	return files, dirCount, fileCount, nil
}

// collectRelevantEnvVars collects environment variables that might be relevant for command suggestions
func (c *ContextCollector) collectRelevantEnvVars() map[string]string {
	relevantVars := []string{
		"PATH", "HOME", "USER", "USERNAME", "SHELL",
		"LANG", "LC_ALL", "TERM", "EDITOR", "PAGER",
		"GOPATH", "GOROOT", "JAVA_HOME", "NODE_ENV",
		"VIRTUAL_ENV", "CONDA_DEFAULT_ENV",
	}
	
	envVars := make(map[string]string)
	for _, varName := range relevantVars {
		if value := os.Getenv(varName); value != "" {
			envVars[varName] = value
		}
	}
	
	return envVars
}

// FormatForPrompt formats the context for inclusion in a prompt
func (ctx *Context) FormatForPrompt() string {
	var parts []string
	
	// System information
	parts = append(parts, fmt.Sprintf("Operating System: %s (%s)", ctx.OS, ctx.Arch))
	if ctx.Shell != "" {
		parts = append(parts, fmt.Sprintf("Shell: %s", ctx.Shell))
	}
	
	// Directory information
	parts = append(parts, fmt.Sprintf("Current Directory: %s", ctx.WorkingDir))
	
	// File listing
	if len(ctx.Files) > 0 {
		fileList := strings.Join(ctx.Files, ", ")
		parts = append(parts, fmt.Sprintf("Directory Contents: %s", fileList))
		
		if ctx.DirectoryCount > 0 || ctx.FileCount > 0 {
			parts = append(parts, fmt.Sprintf("(%d directories, %d files)", ctx.DirectoryCount, ctx.FileCount))
		}
	} else {
		parts = append(parts, "Directory Contents: (empty or unreadable)")
	}
	
	// Environment variables (if any)
	if len(ctx.EnvVars) > 0 {
		var envList []string
		for key, value := range ctx.EnvVars {
			// Truncate very long values
			if len(value) > 50 {
				value = value[:47] + "..."
			}
			envList = append(envList, fmt.Sprintf("%s=%s", key, value))
		}
		parts = append(parts, fmt.Sprintf("Environment: %s", strings.Join(envList, ", ")))
	}
	
	return strings.Join(parts, "\n")
}

// GetWorkingDirectory returns the current working directory
func (ctx *Context) GetWorkingDirectory() string {
	return ctx.WorkingDir
}

// GetFilesByExtension returns files with the specified extension
func (ctx *Context) GetFilesByExtension(ext string) []string {
	var matchingFiles []string
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	
	for _, file := range ctx.Files {
		if strings.HasSuffix(strings.ToLower(file), ext) {
			matchingFiles = append(matchingFiles, file)
		}
	}
	
	return matchingFiles
}

// HasFiles returns true if the directory contains any files matching the pattern
func (ctx *Context) HasFiles(pattern string) bool {
	pattern = strings.ToLower(pattern)
	for _, file := range ctx.Files {
		if strings.Contains(strings.ToLower(file), pattern) {
			return true
		}
	}
	return false
}