package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PathCompletionContext represents the context for path completion
type PathCompletionContext struct {
	Directory string // Directory to scan for completions
	Prefix    string // File/directory name prefix to match
	StartPos  int    // Start position in the command string
	EndPos    int    // End position in the command string
}

// ExtractPathContext extracts path completion context from command and cursor position
func ExtractPathContext(command string, cursorPos int) (*PathCompletionContext, error) {
	if cursorPos < 0 || cursorPos > len(command) {
		cursorPos = len(command)
	}

	// Find the start of the current word (path segment)
	startPos := cursorPos
	for startPos > 0 {
		char := command[startPos-1]
		// Stop at whitespace or command separators
		if char == ' ' || char == '\t' || char == '|' || char == '&' || char == ';' {
			break
		}
		startPos--
	}

	// Extract the path segment from start to cursor
	pathSegment := command[startPos:cursorPos]

	// Skip if this doesn't look like a path
	if pathSegment == "" || (!strings.Contains(pathSegment, "/") && !strings.HasPrefix(pathSegment, "~")) {
		return nil, nil
	}

	// Expand the path to get directory and prefix
	dir, prefix := filepath.Split(pathSegment)

	// Handle special cases
	if dir == "" {
		dir = "."
	}

	// Expand ~ and relative paths
	expandedDir, err := ExpandPath(dir)
	if err != nil {
		return nil, err
	}

	return &PathCompletionContext{
		Directory: expandedDir,
		Prefix:    prefix,
		StartPos:  startPos,
		EndPos:    cursorPos,
	}, nil
}

// ExpandPath expands paths with ~ and resolves relative paths
func ExpandPath(path string) (string, error) {
	if path == "" {
		return ".", nil
	}

	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") || path == "~" {
		homeDir, err := GetHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, path[2:]), nil
	}

	// Handle relative paths
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	return path, nil
}

// ScanDirectoryForCompletion scans directory and returns matching files/directories
func ScanDirectoryForCompletion(dir, prefix string) ([]string, error) {
	// Check if directory exists and is readable
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	if !dirInfo.IsDir() {
		return nil, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var candidates []string
	maxCandidates := 50 // Limit to avoid performance issues

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless prefix starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		// Check if name matches prefix
		if prefix == "" || strings.HasPrefix(name, prefix) {
			// Add / suffix for directories
			if entry.IsDir() {
				name += "/"
			}
			candidates = append(candidates, name)

			// Limit results for performance
			if len(candidates) >= maxCandidates {
				break
			}
		}
	}

	// Sort candidates
	sort.Strings(candidates)

	return candidates, nil
}

// ApplyCompletion applies the completion to the command string
func ApplyCompletion(command, completion string, startPos, endPos int) (string, int) {
	if startPos < 0 || endPos < startPos || endPos > len(command) {
		return command, len(command)
	}

	// Replace the segment from startPos to endPos with completion
	before := command[:startPos]
	after := command[endPos:]

	// Find the directory part of the original path
	originalSegment := command[startPos:endPos]
	dir, _ := filepath.Split(originalSegment)

	// Build the new path by combining directory with completion
	var newPath string
	if dir == "" || dir == "." {
		// No directory part, just use completion
		newPath = completion
	} else {
		// Combine directory with completion, ensuring proper path format
		if strings.HasSuffix(dir, "/") {
			newPath = dir + completion
		} else {
			// This handles cases like "~/doc" where dir becomes "~/" and we want "~/documents/"
			newPath = dir + completion
		}
	}

	// Build the new command
	newCommand := before + newPath + after
	newCursorPos := len(before + newPath)

	return newCommand, newCursorPos
}

// GetCompletionDisplayName formats a completion candidate for display
func GetCompletionDisplayName(candidate string) string {
	// Add visual indicators
	if strings.HasSuffix(candidate, "/") {
		return candidate // Directory already has /
	}
	return candidate
}

// IsPathLikeArgument checks if the cursor position is likely in a file path argument
func IsPathLikeArgument(command string, cursorPos int) bool {
	if cursorPos <= 0 {
		return false
	}

	// Look backwards for path-like patterns
	for i := cursorPos - 1; i >= 0; i-- {
		char := command[i]

		// Stop at whitespace
		if char == ' ' || char == '\t' {
			break
		}

		// If we find path separators, this is likely a path
		if char == '/' || (i > 0 && command[i-1:i+1] == "~/") {
			return true
		}
	}

	return false
}

// CompletePartialPath handles completion for partial paths like "some" -> matching files starting with "some"
func CompletePartialPath(dir, prefix string) ([]string, error) {
	candidates, err := ScanDirectoryForCompletion(dir, prefix)
	if err != nil {
		return nil, err
	}

	// Filter out exact matches when we have a prefix
	if prefix != "" {
		var filtered []string
		for _, candidate := range candidates {
			// Remove trailing / for comparison if it's a directory
			candidateName := strings.TrimSuffix(candidate, "/")
			if candidateName != prefix {
				filtered = append(filtered, candidate)
			}
		}
		return filtered, nil
	}

	return candidates, nil
}
