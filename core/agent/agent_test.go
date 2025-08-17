package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Case 1: Natural language to command conversion
func TestAgent_NaturalLanguageToCommand(t *testing.T) {
	agent := NewDefaultAgent()
	ctx := context.Background()

	tests := []struct {
		name             string
		query            string
		expectedCommands []string // Any of these commands would be acceptable
		minConfidence    float64
	}{
		{
			name:             "list files in current directory",
			query:            "显示当前目录下所有文件",
			expectedCommands: []string{"ls -la", "ls -al", "ls -l", "ls"},
			minConfidence:    0.7,
		},
		{
			name:             "show processes using port",
			query:            "显示占用3000端口的程序",
			expectedCommands: []string{"lsof -i :3000", "netstat -tulpn | grep 3000", "ss -tulpn | grep 3000"},
			minConfidence:    0.7,
		},
		{
			name:             "find large files",
			query:            "找出当前目录下大于100M的文件",
			expectedCommands: []string{"find . -type f -size +100M", "find . -size +100M -type f"},
			minConfidence:    0.7,
		},
		{
			name:             "check disk usage",
			query:            "查看磁盘使用情况",
			expectedCommands: []string{"df -h", "df -H", "df"},
			minConfidence:    0.8,
		},
		{
			name:             "kill process by name",
			query:            "终止所有Chrome进程",
			expectedCommands: []string{"pkill Chrome", "killall Chrome", "pkill -9 Chrome"},
			minConfidence:    0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion, err := agent.ProcessQuery(ctx, tt.query)
			require.NoError(t, err)
			require.NotNil(t, suggestion)

			// Check if the suggested command matches any expected command
			found := false
			for _, expected := range tt.expectedCommands {
				if strings.Contains(suggestion.Command, expected) || expected == suggestion.Command {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected one of %v, got %s", tt.expectedCommands, suggestion.Command)
			
			// Check confidence
			assert.GreaterOrEqual(t, suggestion.Confidence, tt.minConfidence)
			
			// Should have an explanation
			assert.NotEmpty(t, suggestion.Explanation)
		})
	}
}

// Test Case 2: Context memory and conversation continuity
func TestAgent_ContextMemory(t *testing.T) {
	agent := NewDefaultAgent()
	ctx := context.Background()

	// First query - ask about files
	suggestion1, err := agent.ProcessQuery(ctx, "列出当前目录的文件")
	require.NoError(t, err)
	require.NotNil(t, suggestion1)

	// Simulate execution result
	agent.AddExecutionResult(suggestion1.Command, "file1.txt\nfile2.log\ndata.json", 0)

	// Second query - reference previous result
	suggestion2, err := agent.ProcessQuery(ctx, "删除刚才列出的log文件")
	require.NoError(t, err)
	require.NotNil(t, suggestion2)

	// Should suggest removing file2.log based on context
	assert.Contains(t, suggestion2.Command, "file2.log")
	assert.True(t, strings.Contains(suggestion2.Command, "rm") || 
	              strings.Contains(suggestion2.Command, "delete"))

	// Check context history
	context := agent.GetContext()
	assert.GreaterOrEqual(t, len(context.History), 2)
	assert.GreaterOrEqual(t, len(context.Executions), 1)

	// Third query - ask about previous commands
	suggestion3, err := agent.ProcessQuery(ctx, "我刚才执行了什么命令")
	require.NoError(t, err)
	require.NotNil(t, suggestion3)
	
	// Should reference the history
	assert.NotEmpty(t, suggestion3.Explanation)
	
	// Test context clearing
	agent.ClearContext()
	clearedContext := agent.GetContext()
	assert.Empty(t, clearedContext.History)
	assert.Empty(t, clearedContext.Executions)
}

// Test Case 3: Safety checks for dangerous commands
func TestAgent_SafetyChecks(t *testing.T) {
	agent := NewDefaultAgent()
	ctx := context.Background()

	dangerousQueries := []struct {
		name         string
		query        string
		expectedRisk RiskLevel
		shouldWarn   bool
	}{
		{
			name:         "delete root directory",
			query:        "删除系统所有文件",
			expectedRisk: RiskCritical,
			shouldWarn:   true,
		},
		{
			name:         "remove home directory",
			query:        "删除用户主目录",
			expectedRisk: RiskHigh,
			shouldWarn:   true,
		},
		{
			name:         "format disk",
			query:        "格式化硬盘",
			expectedRisk: RiskCritical,
			shouldWarn:   true,
		},
		{
			name:         "modify system permissions",
			query:        "给所有文件777权限",
			expectedRisk: RiskHigh,
			shouldWarn:   true,
		},
		{
			name:         "overwrite important config",
			query:        "清空bashrc文件",
			expectedRisk: RiskHigh,
			shouldWarn:   true,
		},
	}

	for _, tt := range dangerousQueries {
		t.Run(tt.name, func(t *testing.T) {
			suggestion, err := agent.ProcessQuery(ctx, tt.query)
			require.NoError(t, err)
			require.NotNil(t, suggestion)

			// Should have security risks identified
			if tt.shouldWarn {
				assert.NotEmpty(t, suggestion.Risks, "Expected security risks for query: %s", tt.query)
				
				// Check risk level
				hasExpectedRisk := false
				for _, risk := range suggestion.Risks {
					if risk.Level == tt.expectedRisk || risk.Level == RiskCritical {
						hasExpectedRisk = true
						assert.NotEmpty(t, risk.Description)
						assert.NotEmpty(t, risk.Mitigation)
						break
					}
				}
				assert.True(t, hasExpectedRisk, "Expected risk level %s for query: %s", tt.expectedRisk, tt.query)
			}

			// For critical risks, confidence should be lower
			if tt.expectedRisk == RiskCritical {
				assert.LessOrEqual(t, suggestion.Confidence, 0.5)
			}
		})
	}

	// Test safe commands should not have warnings
	safeQueries := []struct {
		name  string
		query string
	}{
		{
			name:  "list files",
			query: "显示文件列表",
		},
		{
			name:  "show date",
			query: "显示当前日期",
		},
		{
			name:  "check network",
			query: "测试网络连接",
		},
	}

	for _, tt := range safeQueries {
		t.Run(tt.name+"_safe", func(t *testing.T) {
			suggestion, err := agent.ProcessQuery(ctx, tt.query)
			require.NoError(t, err)
			require.NotNil(t, suggestion)

			// Should not have high or critical risks
			for _, risk := range suggestion.Risks {
				assert.NotEqual(t, RiskCritical, risk.Level)
				assert.NotEqual(t, RiskHigh, risk.Level)
			}
		})
	}
}

// Test Agent initialization and configuration
func TestAgent_Configuration(t *testing.T) {
	config := &Config{
		Provider:      "openai",
		Model:         "gpt-3.5-turbo",
		Temperature:   0.7,
		MaxTokens:     1000,
		SafetyEnabled: true,
		ContextSize:   10,
	}

	agent := NewAgent(config)
	require.NotNil(t, agent)

	// Test context configuration
	ctx := agent.GetContext()
	assert.Equal(t, 10, ctx.MaxHistorySize)
	
	// Test system info population
	assert.NotEmpty(t, ctx.SystemInfo.OS)
	assert.NotEmpty(t, ctx.SystemInfo.Shell)
	assert.NotEmpty(t, ctx.SystemInfo.Username)
}

// Test execution result tracking
func TestAgent_ExecutionTracking(t *testing.T) {
	agent := NewDefaultAgent()

	// Add multiple execution results
	agent.AddExecutionResult("ls -la", "total 48\ndrwxr-xr-x  10 user  staff   320 Jan 15 10:00 .", 0)
	agent.AddExecutionResult("pwd", "/Users/test/project", 0)
	agent.AddExecutionResult("git status", "fatal: not a git repository", 128)

	ctx := agent.GetContext()
	assert.Len(t, ctx.Executions, 3)

	// Check execution records
	assert.Equal(t, "ls -la", ctx.Executions[0].Command)
	assert.Equal(t, 0, ctx.Executions[0].ExitCode)
	
	assert.Equal(t, "git status", ctx.Executions[2].Command)
	assert.Equal(t, 128, ctx.Executions[2].ExitCode)

	// Verify timestamps are set
	for _, exec := range ctx.Executions {
		assert.False(t, exec.Timestamp.IsZero())
	}
}

// Test context size limits
func TestAgent_ContextLimits(t *testing.T) {
	config := &Config{
		ContextSize: 3, // Limit to 3 history items
	}
	agent := NewAgent(config)
	ctx := context.Background()

	// Add more queries than the limit
	queries := []string{
		"显示文件",
		"查看进程",
		"检查内存",
		"查看网络",
		"显示时间",
	}

	for _, query := range queries {
		_, err := agent.ProcessQuery(ctx, query)
		require.NoError(t, err)
	}

	// Context should only keep the last 3
	agentCtx := agent.GetContext()
	assert.LessOrEqual(t, len(agentCtx.History), 3)
	
	// Should have the most recent queries
	if len(agentCtx.History) > 0 {
		lastQuery := agentCtx.History[len(agentCtx.History)-1].Query
		assert.Equal(t, "显示时间", lastQuery)
	}
}

// Helper function to create default agent (will be implemented)
func NewDefaultAgent() Agent {
	return NewAgent(&Config{
		Provider:      "openai", 
		Model:         "gpt-3.5-turbo",
		Temperature:   0.7,
		MaxTokens:     1000,
		SafetyEnabled: true,
		ContextSize:   10,
	})
}

// Helper function to create agent with config (will be implemented)
func NewAgent(config *Config) Agent {
	// Create a mock provider for testing
	mockProvider := NewMockProvider()
	
	return &defaultAgent{
		config:   config,
		context:  newContext(config),
		provider: mockProvider,
		prompt:   NewPromptBuilder(config),
		safety:   NewSafetyChecker(config),
		parser:   NewResponseParser(),
	}
}