package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourusername/clia/core/provider"
)

// MockProvider is a mock implementation of the Provider interface for testing
type MockProvider struct {
	responses map[string]string
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	mp := &MockProvider{
		responses: make(map[string]string),
	}
	mp.initResponses()
	return mp
}

// initResponses initializes mock responses
func (mp *MockProvider) initResponses() {
	// Add mock responses for common queries
	mp.responses["显示当前目录下所有文件"] = `{
		"command": "ls -la",
		"explanation": "Lists all files including hidden ones with detailed information",
		"confidence": 0.9,
		"risks": [],
		"alternatives": ["ls -al", "ls -l", "ls"]
	}`

	mp.responses["显示占用3000端口的程序"] = `{
		"command": "lsof -i :3000",
		"explanation": "Shows processes using port 3000",
		"confidence": 0.85,
		"risks": [],
		"alternatives": ["netstat -tulpn | grep 3000", "ss -tulpn | grep 3000"]
	}`

	mp.responses["找出当前目录下大于100M的文件"] = `{
		"command": "find . -type f -size +100M",
		"explanation": "Finds files larger than 100MB in current directory",
		"confidence": 0.8,
		"risks": [],
		"alternatives": ["find . -size +100M -type f"]
	}`

	mp.responses["查看磁盘使用情况"] = `{
		"command": "df -h",
		"explanation": "Shows disk usage in human-readable format",
		"confidence": 0.95,
		"risks": [],
		"alternatives": ["df -H", "df"]
	}`

	mp.responses["终止所有Chrome进程"] = `{
		"command": "pkill Chrome",
		"explanation": "Terminates all Chrome processes",
		"confidence": 0.7,
		"risks": [{"level": "low", "description": "Will close all Chrome windows", "mitigation": "Save your work first"}],
		"alternatives": ["killall Chrome", "pkill -9 Chrome"]
	}`

	// Dangerous commands
	mp.responses["删除系统所有文件"] = `{
		"command": "rm -rf /",
		"explanation": "WARNING: This would delete all system files",
		"confidence": 0.1,
		"risks": [{"level": "critical", "description": "This will destroy your system", "mitigation": "Never run this command"}],
		"alternatives": []
	}`

	mp.responses["删除用户主目录"] = `{
		"command": "rm -rf ~",
		"explanation": "WARNING: This would delete your home directory",
		"confidence": 0.2,
		"risks": [{"level": "high", "description": "This will delete all your personal files", "mitigation": "Make backups first"}],
		"alternatives": []
	}`

	mp.responses["格式化硬盘"] = `{
		"command": "mkfs.ext4 /dev/sda",
		"explanation": "WARNING: This would format the hard disk",
		"confidence": 0.1,
		"risks": [{"level": "critical", "description": "This will erase all data on the disk", "mitigation": "Never run unless you want to wipe the disk"}],
		"alternatives": []
	}`

	mp.responses["给所有文件777权限"] = `{
		"command": "chmod -R 777 /",
		"explanation": "WARNING: This gives everyone full permissions",
		"confidence": 0.3,
		"risks": [{"level": "high", "description": "This compromises system security", "mitigation": "Use specific permissions and paths"}],
		"alternatives": []
	}`

	mp.responses["清空bashrc文件"] = `{
		"command": "> ~/.bashrc",
		"explanation": "WARNING: This clears your bash configuration",
		"confidence": 0.4,
		"risks": [{"level": "high", "description": "This will reset your shell configuration", "mitigation": "Backup the file first"}],
		"alternatives": []
	}`

	// Safe commands
	mp.responses["显示文件列表"] = `{
		"command": "ls",
		"explanation": "Lists files in current directory",
		"confidence": 0.95,
		"risks": [],
		"alternatives": ["ls -l", "ls -la"]
	}`

	mp.responses["显示当前日期"] = `{
		"command": "date",
		"explanation": "Shows current date and time",
		"confidence": 0.99,
		"risks": [],
		"alternatives": ["date +%Y-%m-%d", "date -u"]
	}`

	mp.responses["测试网络连接"] = `{
		"command": "ping -c 4 google.com",
		"explanation": "Tests network connectivity by pinging Google",
		"confidence": 0.85,
		"risks": [],
		"alternatives": ["ping -c 4 8.8.8.8", "curl -I https://google.com"]
	}`
}

// Chat implements the Provider interface
func (mp *MockProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	// Extract query from the last user message
	query := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			query = req.Messages[i].Content
			break
		}
	}

	// Find matching response
	response := mp.findResponse(query)
	
	return &provider.ChatResponse{
		ID:      "mock-response-1",
		Model:   "mock-model",
		Content: response,
		Usage: &provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

// findResponse finds a matching response for the query
func (mp *MockProvider) findResponse(query string) string {
	// Extract the actual user query from the formatted prompt
	// The prompt might contain system info and other context
	actualQuery := mp.extractUserQuery(query)
	
	// Try exact match first
	if response, ok := mp.responses[actualQuery]; ok {
		return response
	}

	// Try to find partial match
	queryLower := strings.ToLower(actualQuery)
	for key, response := range mp.responses {
		if strings.Contains(queryLower, strings.ToLower(key)) || 
		   strings.Contains(strings.ToLower(key), queryLower) {
			return response
		}
	}

	// Check for context-based queries
	if strings.Contains(query, "删除") && strings.Contains(query, "log文件") {
		return `{
			"command": "rm file2.log",
			"explanation": "Removes the log file from previous listing",
			"confidence": 0.8,
			"risks": [{"level": "low", "description": "File will be permanently deleted", "mitigation": "Use rm -i for confirmation"}],
			"alternatives": ["rm -i file2.log", "mv file2.log /tmp/"]
		}`
	}

	if strings.Contains(query, "列出") && strings.Contains(query, "文件") {
		return `{
			"command": "ls -la",
			"explanation": "Lists all files with details",
			"confidence": 0.9,
			"risks": [],
			"alternatives": ["ls", "ls -l"]
		}`
	}

	if strings.Contains(query, "执行了什么命令") {
		return `{
			"command": "history",
			"explanation": "You previously executed ls and rm commands based on the context",
			"confidence": 0.7,
			"risks": [],
			"alternatives": ["history | tail -10"]
		}`
	}

	// Default response for unknown queries
	return fmt.Sprintf(`{
		"command": "echo 'Processing: %s'",
		"explanation": "Mock response for query",
		"confidence": 0.5,
		"risks": [],
		"alternatives": []
	}`, query)
}

// ChatStream implements the Provider interface
func (mp *MockProvider) ChatStream(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.ChatChunk, error) {
	// For testing, we'll just send the full response as a single chunk
	ch := make(chan *provider.ChatChunk, 1)
	
	resp, _ := mp.Chat(ctx, req)
	ch <- &provider.ChatChunk{
		ID:      "mock-stream-1",
		Model:   "mock-model",
		Content: resp.Content,
		Done:    true,
	}
	close(ch)
	
	return ch, nil
}

// Name implements the Provider interface
func (mp *MockProvider) Name() string {
	return "mock"
}

// Available implements the Provider interface
func (mp *MockProvider) Available() bool {
	return true
}

// AddResponse adds a custom response for testing
func (mp *MockProvider) AddResponse(query, response string) {
	if mp.responses == nil {
		mp.responses = make(map[string]string)
	}
	mp.responses[query] = response
}

// AddJSONResponse adds a JSON response for testing
func (mp *MockProvider) AddJSONResponse(query string, suggestion CommandSuggestion) {
	data, _ := json.Marshal(suggestion)
	mp.AddResponse(query, string(data))
}

// extractUserQuery extracts the actual user query from a formatted prompt
func (mp *MockProvider) extractUserQuery(prompt string) string {
	// Look for "User Query:" marker
	if idx := strings.Index(prompt, "User Query:"); idx >= 0 {
		query := prompt[idx+len("User Query:"):]
		query = strings.TrimSpace(query)
		// Take until the next section or end
		if endIdx := strings.Index(query, "\n\n"); endIdx > 0 {
			query = query[:endIdx]
		}
		return strings.TrimSpace(query)
	}
	
	// Fallback: return the original prompt
	return prompt
}