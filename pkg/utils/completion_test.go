package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPathContext(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		cursorPos int
		want      *PathCompletionContext
		wantErr   bool
	}{
		{
			name:      "simple path at end",
			command:   "ls ~/doc",
			cursorPos: 8,
			want: &PathCompletionContext{
				Directory: expandTilde(t, "~/"),
				Prefix:    "doc",
				StartPos:  3,
				EndPos:    8,
			},
		},
		{
			name:      "absolute path",
			command:   "tar -xzf /usr/local/",
			cursorPos: 20,
			want: &PathCompletionContext{
				Directory: "/usr/local/",
				Prefix:    "",
				StartPos:  9,
				EndPos:    20,
			},
		},
		{
			name:      "no path context",
			command:   "echo hello",
			cursorPos: 10,
			want:      nil,
		},
		{
			name:      "path with prefix",
			command:   "ls /data/some",
			cursorPos: 13,
			want: &PathCompletionContext{
				Directory: "/data/",
				Prefix:    "some",
				StartPos:  3,
				EndPos:    13,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractPathContext(tt.command, tt.cursorPos)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPathContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil && got == nil {
				return // Both nil, success
			}

			if tt.want == nil || got == nil {
				t.Errorf("ExtractPathContext() got = %v, want %v", got, tt.want)
				return
			}

			if got.Directory != tt.want.Directory {
				t.Errorf("ExtractPathContext() Directory = %v, want %v", got.Directory, tt.want.Directory)
			}

			if got.Prefix != tt.want.Prefix {
				t.Errorf("ExtractPathContext() Prefix = %v, want %v", got.Prefix, tt.want.Prefix)
			}

			if got.StartPos != tt.want.StartPos {
				t.Errorf("ExtractPathContext() StartPos = %v, want %v", got.StartPos, tt.want.StartPos)
			}

			if got.EndPos != tt.want.EndPos {
				t.Errorf("ExtractPathContext() EndPos = %v, want %v", got.EndPos, tt.want.EndPos)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "tilde expansion",
			path:    "~/documents",
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/usr/local",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "./local",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Basic validation - should be non-empty for valid paths
			if !tt.wantErr && tt.path != "" && got == "" {
				t.Errorf("ExpandPath() returned empty path for input %v", tt.path)
			}

			// Tilde should be expanded
			if strings.HasPrefix(tt.path, "~/") && strings.HasPrefix(got, "~/") {
				t.Errorf("ExpandPath() failed to expand tilde in %v, got %v", tt.path, got)
			}
		})
	}
}

func TestScanDirectoryForCompletion(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create test files and directories
	testFiles := []string{
		"document.txt",
		"data.json",
		"script.sh",
		"readme.md",
	}

	testDirs := []string{
		"documents",
		"downloads",
		"src",
	}

	// Create test files
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create test directories
	for _, dir := range testDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	tests := []struct {
		name       string
		dir        string
		prefix     string
		wantCount  int
		wantPrefix string
	}{
		{
			name:      "all files and dirs",
			dir:       tmpDir,
			prefix:    "",
			wantCount: 7, // 4 files + 3 directories
		},
		{
			name:      "files with 'd' prefix",
			dir:       tmpDir,
			prefix:    "d",
			wantCount: 4, // document.txt, data.json, downloads/, documents/
		},
		{
			name:      "files with 'doc' prefix",
			dir:       tmpDir,
			prefix:    "doc",
			wantCount: 2, // document.txt, documents/
		},
		{
			name:      "no matches",
			dir:       tmpDir,
			prefix:    "xyz",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates, err := ScanDirectoryForCompletion(tt.dir, tt.prefix)
			if err != nil {
				t.Errorf("ScanDirectoryForCompletion() error = %v", err)
				return
			}

			if len(candidates) != tt.wantCount {
				t.Errorf("ScanDirectoryForCompletion() candidate count = %v, want %v, candidates: %v",
					len(candidates), tt.wantCount, candidates)
			}

			// Verify all candidates start with the prefix
			for _, candidate := range candidates {
				actualName := strings.TrimSuffix(candidate, "/")
				if tt.prefix != "" && !strings.HasPrefix(actualName, tt.prefix) {
					t.Errorf("Candidate %v does not start with prefix %v", candidate, tt.prefix)
				}
			}

			// Verify directories have '/' suffix
			for _, candidate := range candidates {
				if strings.HasSuffix(candidate, "/") {
					// Check if it's actually a directory
					candidatePath := filepath.Join(tt.dir, strings.TrimSuffix(candidate, "/"))
					if info, err := os.Stat(candidatePath); err != nil || !info.IsDir() {
						t.Errorf("Candidate %v has '/' suffix but is not a directory", candidate)
					}
				}
			}
		})
	}
}

func TestApplyCompletion(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		completion    string
		startPos      int
		endPos        int
		wantCommand   string
		wantCursorPos int
	}{
		{
			name:          "complete filename",
			command:       "ls ~/doc",
			completion:    "documents/",
			startPos:      3,
			endPos:        8,
			wantCommand:   "ls ~/documents/",
			wantCursorPos: 15,
		},
		{
			name:          "complete in middle",
			command:       "tar -xzf ~/doc more_args",
			completion:    "documents/",
			startPos:      9,
			endPos:        14,
			wantCommand:   "tar -xzf ~/documents/ more_args",
			wantCursorPos: 21,
		},
		{
			name:          "complete absolute path",
			command:       "cat /usr/loc",
			completion:    "local/",
			startPos:      4,
			endPos:        12,
			wantCommand:   "cat /usr/local/",
			wantCursorPos: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommand, gotCursorPos := ApplyCompletion(tt.command, tt.completion, tt.startPos, tt.endPos)

			if gotCommand != tt.wantCommand {
				t.Errorf("ApplyCompletion() command = %v, want %v", gotCommand, tt.wantCommand)
			}

			if gotCursorPos != tt.wantCursorPos {
				t.Errorf("ApplyCompletion() cursor position = %v, want %v", gotCursorPos, tt.wantCursorPos)
			}
		})
	}
}

func TestIsPathLikeArgument(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		cursorPos int
		want      bool
	}{
		{
			name:      "absolute path",
			command:   "ls /usr/local",
			cursorPos: 13,
			want:      true,
		},
		{
			name:      "tilde path",
			command:   "cat ~/document.txt",
			cursorPos: 17,
			want:      true,
		},
		{
			name:      "relative path",
			command:   "ls ./src/main.go",
			cursorPos: 15,
			want:      true,
		},
		{
			name:      "not a path",
			command:   "echo hello",
			cursorPos: 10,
			want:      false,
		},
		{
			name:      "command name",
			command:   "ls",
			cursorPos: 2,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathLikeArgument(tt.command, tt.cursorPos)
			if got != tt.want {
				t.Errorf("IsPathLikeArgument() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to expand tilde for testing
func expandTilde(t *testing.T, path string) string {
	expanded, err := ExpandPath(path)
	if err != nil {
		t.Fatalf("Failed to expand path %s: %v", path, err)
	}
	return expanded
}
