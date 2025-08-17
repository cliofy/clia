package provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider implements Provider interface for testing
type MockProvider struct {
	name          string
	available     bool
	chatResponse  *ChatResponse
	chatError     error
	streamChunks  []*ChatChunk
	streamError   error
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name:      name,
		available: true,
	}
}

func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.chatError != nil {
		return nil, m.chatError
	}
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	// Default response
	return &ChatResponse{
		ID:      "test-id",
		Model:   req.Model,
		Content: "Test response",
		Created: time.Now(),
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *MockProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	
	ch := make(chan *ChatChunk)
	go func() {
		defer close(ch)
		
		if m.streamChunks != nil {
			for _, chunk := range m.streamChunks {
				select {
				case <-ctx.Done():
					return
				case ch <- chunk:
				}
			}
		} else {
			// Default stream
			chunks := []string{"Hello", " ", "world", "!"}
			for i, content := range chunks {
				select {
				case <-ctx.Done():
					return
				case ch <- &ChatChunk{
					ID:      "test-stream",
					Model:   req.Model,
					Content: content,
					Done:    i == len(chunks)-1,
				}:
				}
			}
		}
	}()
	
	return ch, nil
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Available() bool {
	return m.available
}

// Test Case 1: Basic chat functionality
func TestProvider_Chat(t *testing.T) {
	tests := []struct {
		name     string
		request  *ChatRequest
		wantErr  bool
		validate func(t *testing.T, resp *ChatResponse)
	}{
		{
			name: "simple chat request",
			request: &ChatRequest{
				Model: "test-model",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp *ChatResponse) {
				assert.NotEmpty(t, resp.ID)
				assert.Equal(t, "test-model", resp.Model)
				assert.NotEmpty(t, resp.Content)
				assert.NotNil(t, resp.Usage)
			},
		},
		{
			name: "chat with options",
			request: &ChatRequest{
				Model: "test-model",
				Messages: []Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
				Options: &ChatOptions{
					Temperature: floatPtr(0.7),
					MaxTokens:   intPtr(100),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp *ChatResponse) {
				assert.NotEmpty(t, resp.Content)
				assert.NotNil(t, resp.Usage)
				assert.Greater(t, resp.Usage.TotalTokens, 0)
			},
		},
		{
			name: "empty messages",
			request: &ChatRequest{
				Model:    "test-model",
				Messages: []Message{},
			},
			wantErr: false, // Provider should handle gracefully
		},
	}

	provider := NewMockProvider("test-provider")
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := provider.Chat(ctx, tt.request)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, resp)
			
			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

// Test Case 2: Streaming response
func TestProvider_ChatStream(t *testing.T) {
	tests := []struct {
		name         string
		request      *ChatRequest
		streamChunks []*ChatChunk
		wantErr      bool
		validate     func(t *testing.T, chunks []*ChatChunk)
	}{
		{
			name: "basic streaming",
			request: &ChatRequest{
				Model: "test-model",
				Messages: []Message{
					{Role: "user", Content: "Tell me a story"},
				},
				Options: &ChatOptions{
					Stream: true,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, chunks []*ChatChunk) {
				assert.Greater(t, len(chunks), 0)
				// Last chunk should have Done=true
				lastChunk := chunks[len(chunks)-1]
				assert.True(t, lastChunk.Done)
				// Combine all chunks
				var combined string
				for _, chunk := range chunks {
					combined += chunk.Content
				}
				assert.NotEmpty(t, combined)
			},
		},
		{
			name: "streaming with context cancellation",
			request: &ChatRequest{
				Model: "test-model",
				Messages: []Message{
					{Role: "user", Content: "Long response"},
				},
			},
			streamChunks: []*ChatChunk{
				{Content: "Part1", Done: false},
				{Content: "Part2", Done: false},
				{Content: "Part3", Done: false},
				{Content: "Part4", Done: true},
			},
			wantErr: false,
			validate: func(t *testing.T, chunks []*ChatChunk) {
				// Should receive at least some chunks before cancellation
				assert.Greater(t, len(chunks), 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMockProvider("test-provider")
			if tt.streamChunks != nil {
				provider.streamChunks = tt.streamChunks
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			stream, err := provider.ChatStream(ctx, tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, stream)

			// Collect chunks
			var chunks []*ChatChunk
			for chunk := range stream {
				chunks = append(chunks, chunk)
				// Optionally cancel early for testing
				if tt.name == "streaming with context cancellation" && len(chunks) >= 2 {
					cancel()
					break
				}
			}

			if tt.validate != nil {
				tt.validate(t, chunks)
			}
		})
	}
}

// Test Case 3: Error handling and retry mechanism
func TestProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockProvider)
		request     *ChatRequest
		expectedErr error
	}{
		{
			name: "API key missing",
			setupMock: func(m *MockProvider) {
				m.chatError = ErrAPIKeyMissing
			},
			request: &ChatRequest{
				Model:    "test-model",
				Messages: []Message{{Role: "user", Content: "test"}},
			},
			expectedErr: ErrAPIKeyMissing,
		},
		{
			name: "model not supported",
			setupMock: func(m *MockProvider) {
				m.chatError = ErrModelNotSupported
			},
			request: &ChatRequest{
				Model:    "unsupported-model",
				Messages: []Message{{Role: "user", Content: "test"}},
			},
			expectedErr: ErrModelNotSupported,
		},
		{
			name: "rate limit exceeded",
			setupMock: func(m *MockProvider) {
				m.chatError = ErrRateLimitExceeded
			},
			request: &ChatRequest{
				Model:    "test-model",
				Messages: []Message{{Role: "user", Content: "test"}},
			},
			expectedErr: ErrRateLimitExceeded,
		},
		{
			name: "provider not available",
			setupMock: func(m *MockProvider) {
				m.available = false
				m.chatError = ErrProviderNotAvailable
			},
			request: &ChatRequest{
				Model:    "test-model",
				Messages: []Message{{Role: "user", Content: "test"}},
			},
			expectedErr: ErrProviderNotAvailable,
		},
		{
			name: "context timeout",
			setupMock: func(m *MockProvider) {
				m.chatError = context.DeadlineExceeded
			},
			request: &ChatRequest{
				Model:    "test-model",
				Messages: []Message{{Role: "user", Content: "test"}},
			},
			expectedErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMockProvider("test-provider")
			
			if tt.setupMock != nil {
				tt.setupMock(provider)
			}

			ctx := context.Background()
			resp, err := provider.Chat(ctx, tt.request)

			assert.Error(t, err)
			assert.Nil(t, resp)
			
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
			
			// Verify provider availability
			if tt.name == "provider not available" {
				assert.False(t, provider.Available())
			}
		})
	}
}

// Test helper: Provider availability check
func TestProvider_Available(t *testing.T) {
	t.Run("provider available", func(t *testing.T) {
		provider := NewMockProvider("test-provider")
		assert.True(t, provider.Available())
		assert.Equal(t, "test-provider", provider.Name())
	})

	t.Run("provider not available", func(t *testing.T) {
		provider := NewMockProvider("test-provider")
		provider.available = false
		assert.False(t, provider.Available())
	})
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}