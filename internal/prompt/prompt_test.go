package prompt

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestContextCollector(t *testing.T) {
	collector := NewContextCollector()

	// Test default settings
	if collector.maxFiles != 20 {
		t.Errorf("Expected max files 20, got %d", collector.maxFiles)
	}

	if collector.includeHidden {
		t.Error("Expected includeHidden to be false by default")
	}

	// Test configuration
	collector.SetMaxFiles(10).SetIncludeHidden(true).SetIncludeEnvVars(true)

	if collector.maxFiles != 10 {
		t.Errorf("Expected max files 10, got %d", collector.maxFiles)
	}

	if !collector.includeHidden {
		t.Error("Expected includeHidden to be true")
	}

	if !collector.includeEnvVars {
		t.Error("Expected includeEnvVars to be true")
	}
}

func TestContextCollection(t *testing.T) {
	collector := NewContextCollector()

	ctx, err := collector.Collect()
	if err != nil {
		t.Fatalf("Context collection failed: %v", err)
	}

	// Test basic system information
	if ctx.OS == "" {
		t.Error("Expected OS to be set")
	}

	if ctx.Arch == "" {
		t.Error("Expected Arch to be set")
	}

	if ctx.WorkingDir == "" {
		t.Error("Expected WorkingDir to be set")
	}

	// Test shell detection
	if ctx.Shell == "" {
		t.Error("Expected Shell to be detected")
	}
}

func TestContextFormatting(t *testing.T) {
	ctx := &Context{
		OS:             "linux",
		Arch:           "amd64",
		Shell:          "bash",
		WorkingDir:     "/home/user",
		Files:          []string{"file1.txt", "file2.py", "dir1/"},
		DirectoryCount: 1,
		FileCount:      2,
	}

	formatted := ctx.FormatForPrompt()

	expectedParts := []string{
		"Operating System: linux (amd64)",
		"Shell: bash",
		"Current Directory: /home/user",
		"Directory Contents: file1.txt, file2.py, dir1/",
		"(1 directories, 2 files)",
	}

	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("Expected formatted context to contain '%s', got:\n%s", part, formatted)
		}
	}
}

func TestContextMethods(t *testing.T) {
	ctx := &Context{
		WorkingDir: "/test/dir",
		Files:      []string{"test.py", "script.sh", "README.md", "data.json"},
	}

	// Test GetWorkingDirectory
	if ctx.GetWorkingDirectory() != "/test/dir" {
		t.Errorf("Expected working dir '/test/dir', got '%s'", ctx.GetWorkingDirectory())
	}

	// Test GetFilesByExtension
	pyFiles := ctx.GetFilesByExtension(".py")
	if len(pyFiles) != 1 || pyFiles[0] != "test.py" {
		t.Errorf("Expected 1 Python file, got %v", pyFiles)
	}

	jsonFiles := ctx.GetFilesByExtension("json")
	if len(jsonFiles) != 1 || jsonFiles[0] != "data.json" {
		t.Errorf("Expected 1 JSON file, got %v", jsonFiles)
	}

	// Test HasFiles
	if !ctx.HasFiles("py") {
		t.Error("Expected to find Python files")
	}

	if !ctx.HasFiles("README") {
		t.Error("Expected to find README file")
	}

	if ctx.HasFiles("nonexistent") {
		t.Error("Expected not to find nonexistent files")
	}
}

func TestCommandPromptTemplate(t *testing.T) {
	ctx := &Context{
		OS:         "linux",
		Shell:      "bash",
		WorkingDir: "/home/user",
		Files:      []string{"test.txt", "script.py"},
		FileCount:  2,
	}

	template := NewCommandPromptTemplate("list all files", ctx)
	prompt := template.Build()

	// Check that prompt contains key elements
	expectedElements := []string{
		"command line expert assistant",
		"list all files",
		"Operating System: linux",
		"Shell: bash",
		"test.txt, script.py",
		"JSON only",
	}

	for _, element := range expectedElements {
		if !strings.Contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s', got:\n%s", element, prompt)
		}
	}
}

func TestCommandPromptTemplateWithExamples(t *testing.T) {
	ctx := &Context{
		OS:        "linux",
		Files:     []string{"script.py", "test.txt"},
		FileCount: 2,
	}

	template := NewCommandPromptTemplate("run python", ctx).
		EnhanceWithExamples()

	prompt := template.Build()

	// Should contain examples section for Python files
	if !strings.Contains(prompt, "EXAMPLES:") {
		t.Error("Expected examples section in enhanced prompt")
	}

	if !strings.Contains(prompt, "python script.py") {
		t.Error("Expected Python example in enhanced prompt")
	}
}

func TestPromptBuilder(t *testing.T) {
	builder := NewPromptBuilder()

	// Test basic prompt building
	prompt, err := builder.BuildCommandPrompt(context.Background(), "list files")
	if err != nil {
		t.Errorf("BuildCommandPrompt failed: %v", err)
	}

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !strings.Contains(prompt, "list files") {
		t.Error("Expected prompt to contain user input")
	}
}

func TestPromptBuilderQuick(t *testing.T) {
	builder := NewPromptBuilder()

	prompt := builder.BuildQuickPrompt("show directory")

	if prompt == "" {
		t.Error("Expected non-empty quick prompt")
	}

	if !strings.Contains(prompt, "show directory") {
		t.Error("Expected prompt to contain user input")
	}

	if !strings.Contains(prompt, "JSON format") {
		t.Error("Expected prompt to specify JSON format")
	}
}

func TestPromptBuilderCustom(t *testing.T) {
	builder := NewPromptBuilder()

	template := "Help with: {user_input} on {os} system"
	variables := map[string]string{
		"custom_var": "test_value",
	}

	prompt, err := builder.BuildCustomPrompt(template, "test task", variables)
	if err != nil {
		t.Errorf("BuildCustomPrompt failed: %v", err)
	}

	if !strings.Contains(prompt, "test task") {
		t.Error("Expected prompt to contain user input")
	}

	// Should contain OS information from context
	if !strings.Contains(prompt, "system") {
		t.Error("Expected prompt to contain system information")
	}
}

func TestPromptBuilderValidation(t *testing.T) {
	builder := NewPromptBuilder()

	// Test empty prompt
	err := builder.ValidatePrompt("")
	if err == nil {
		t.Error("Expected validation error for empty prompt")
	}

	// Test valid prompt
	err = builder.ValidatePrompt("This is a valid prompt")
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	// Test very long prompt
	longPrompt := strings.Repeat("a", 10000)
	err = builder.ValidatePrompt(longPrompt)
	if err == nil {
		t.Error("Expected validation error for very long prompt")
	}
}

func TestPromptBuilderWithOptions(t *testing.T) {
	builder := NewPromptBuilder().
		WithContextOptions(5, true, true)

	collector := builder.GetContextCollector()

	if collector.maxFiles != 5 {
		t.Errorf("Expected max files 5, got %d", collector.maxFiles)
	}

	if !collector.includeHidden {
		t.Error("Expected includeHidden to be true")
	}

	if !collector.includeEnvVars {
		t.Error("Expected includeEnvVars to be true")
	}
}

func TestShellDetection(t *testing.T) {
	collector := NewContextCollector()

	// Test with SHELL environment variable
	originalShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", originalShell)

	os.Setenv("SHELL", "/bin/zsh")
	shell := collector.detectShell()
	if shell != "zsh" {
		t.Errorf("Expected shell 'zsh', got '%s'", shell)
	}

	// Test without SHELL variable
	os.Unsetenv("SHELL")
	shell = collector.detectShell()
	if shell == "" {
		t.Error("Expected default shell to be detected")
	}
}

func TestQuickCommandPrompt(t *testing.T) {
	prompt := QuickCommandPrompt("test command", "linux", "bash")

	expectedElements := []string{
		"test command",
		"linux",
		"bash",
		"JSON format",
	}

	for _, element := range expectedElements {
		if !strings.Contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s', got:\n%s", element, prompt)
		}
	}
}
