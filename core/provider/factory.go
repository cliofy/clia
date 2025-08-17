package provider

import (
	"context"
	"fmt"

	"github.com/yourusername/clia/core/config"
	"github.com/yourusername/clia/core/provider/ollama"
	"github.com/yourusername/clia/core/provider/openai"
	"github.com/yourusername/clia/core/provider/openrouter"
)

// Factory is responsible for creating provider instances
type Factory struct{}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateProvider creates a provider instance based on configuration
func (f *Factory) CreateProvider(name string, cfg *config.ProviderConfig) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("provider config is nil")
	}

	switch cfg.Type {
	case string(ProviderTypeOpenAI):
		return f.createOpenAIProvider(cfg.Config)
	case string(ProviderTypeOpenRouter):
		return f.createOpenRouterProvider(cfg.Config)
	case string(ProviderTypeOllama):
		return f.createOllamaProvider(cfg.Config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
}

// createOpenAIProvider creates an OpenAI provider
func (f *Factory) createOpenAIProvider(configMap map[string]interface{}) (Provider, error) {
	openaiProvider, err := openai.NewProvider("openai", configMap)
	if err != nil {
		return nil, err
	}
	return &openAIProviderWrapper{provider: openaiProvider}, nil
}

// createOpenRouterProvider creates an OpenRouter provider
func (f *Factory) createOpenRouterProvider(configMap map[string]interface{}) (Provider, error) {
	openrouterProvider, err := openrouter.NewProvider("openrouter", configMap)
	if err != nil {
		return nil, err
	}
	return &openRouterProviderWrapper{provider: openrouterProvider}, nil
}

// createOllamaProvider creates an Ollama provider
func (f *Factory) createOllamaProvider(configMap map[string]interface{}) (Provider, error) {
	ollamaProvider, err := ollama.NewProvider("ollama", configMap)
	if err != nil {
		return nil, err
	}
	return &ollamaProviderWrapper{provider: ollamaProvider}, nil
}

// Manager manages provider instances and configuration
type Manager struct {
	config   *config.Config
	factory  *Factory
	providers map[string]Provider
}

// NewManager creates a new provider manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		factory:  NewFactory(),
		providers: make(map[string]Provider),
	}
}

// GetProvider returns a provider instance, creating it if necessary
func (m *Manager) GetProvider(name string) (Provider, error) {
	if name == "" {
		name = m.config.ActiveProvider
	}

	// Check if provider is already created
	if provider, exists := m.providers[name]; exists {
		return provider, nil
	}

	// Get provider config
	providerConfig, err := m.config.GetProvider(name)
	if err != nil {
		return nil, err
	}

	// Create provider instance
	provider, err := m.factory.CreateProvider(name, providerConfig)
	if err != nil {
		return nil, err
	}

	// Cache the provider
	m.providers[name] = provider

	return provider, nil
}

// GetActiveProvider returns the currently active provider
func (m *Manager) GetActiveProvider() (Provider, error) {
	return m.GetProvider("")
}

// SetActiveProvider sets the active provider
func (m *Manager) SetActiveProvider(name string) error {
	// Verify provider exists and can be created
	_, err := m.GetProvider(name)
	if err != nil {
		return err
	}

	// Update config
	err = m.config.SetActiveProvider(name)
	if err != nil {
		return err
	}

	return nil
}

// ListProviders returns a list of available provider names
func (m *Manager) ListProviders() []string {
	return m.config.ListProviders()
}

// RefreshProvider removes a provider from cache, forcing recreation on next access
func (m *Manager) RefreshProvider(name string) {
	delete(m.providers, name)
}

// RefreshAll removes all providers from cache
func (m *Manager) RefreshAll() {
	m.providers = make(map[string]Provider)
}

// UpdateConfig updates the manager's configuration
func (m *Manager) UpdateConfig(cfg *config.Config) {
	m.config = cfg
	// Clear cache to force recreation with new config
	m.RefreshAll()
}

// openAIProviderWrapper wraps the OpenAI provider to implement our generic interface
type openAIProviderWrapper struct {
	provider *openai.Provider
}

func (w *openAIProviderWrapper) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to OpenAI types
	openaiReq := &openai.ChatRequest{
		Model:    req.Model,
		Messages: make([]openai.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		openaiReq.Options = &openai.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call OpenAI provider
	resp, err := w.provider.Chat(ctx, openaiReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert response
	return &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: resp.Content,
		Created: resp.Created,
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (w *openAIProviderWrapper) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Convert to OpenAI types
	openaiReq := &openai.ChatRequest{
		Model:    req.Model,
		Messages: make([]openai.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		openaiReq.Options = &openai.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call OpenAI provider
	openaiStream, err := w.provider.ChatStream(ctx, openaiReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert stream
	genericStream := make(chan *ChatChunk)
	go func() {
		defer close(genericStream)
		for chunk := range openaiStream {
			genericChunk := &ChatChunk{
				ID:      chunk.ID,
				Model:   chunk.Model,
				Content: chunk.Content,
				Done:    chunk.Done,
			}

			select {
			case <-ctx.Done():
				return
			case genericStream <- genericChunk:
			}
		}
	}()

	return genericStream, nil
}

func (w *openAIProviderWrapper) Name() string {
	return w.provider.Name()
}

func (w *openAIProviderWrapper) Available() bool {
	return w.provider.Available()
}

func (w *openAIProviderWrapper) adaptError(err error) error {
	switch err {
	case openai.ErrAPIKeyMissing:
		return ErrAPIKeyMissing
	case openai.ErrModelNotSupported:
		return ErrModelNotSupported
	case openai.ErrRateLimitExceeded:
		return ErrRateLimitExceeded
	case openai.ErrTimeout:
		return ErrTimeout
	case openai.ErrProviderNotAvailable:
		return ErrProviderNotAvailable
	default:
		return err
	}
}

// openRouterProviderWrapper wraps the OpenRouter provider to implement our generic interface
type openRouterProviderWrapper struct {
	provider *openrouter.Provider
}

func (w *openRouterProviderWrapper) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to OpenRouter types
	openrouterReq := &openrouter.ChatRequest{
		Model:    req.Model,
		Messages: make([]openrouter.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		openrouterReq.Messages[i] = openrouter.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		openrouterReq.Options = &openrouter.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call OpenRouter provider
	resp, err := w.provider.Chat(ctx, openrouterReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert response
	return &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: resp.Content,
		Created: resp.Created,
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (w *openRouterProviderWrapper) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Convert to OpenRouter types
	openrouterReq := &openrouter.ChatRequest{
		Model:    req.Model,
		Messages: make([]openrouter.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		openrouterReq.Messages[i] = openrouter.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		openrouterReq.Options = &openrouter.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call OpenRouter provider
	openrouterStream, err := w.provider.ChatStream(ctx, openrouterReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert stream
	genericStream := make(chan *ChatChunk)
	go func() {
		defer close(genericStream)
		for chunk := range openrouterStream {
			genericChunk := &ChatChunk{
				ID:      chunk.ID,
				Model:   chunk.Model,
				Content: chunk.Content,
				Done:    chunk.Done,
			}

			select {
			case <-ctx.Done():
				return
			case genericStream <- genericChunk:
			}
		}
	}()

	return genericStream, nil
}

func (w *openRouterProviderWrapper) Name() string {
	return w.provider.Name()
}

func (w *openRouterProviderWrapper) Available() bool {
	return w.provider.Available()
}

func (w *openRouterProviderWrapper) adaptError(err error) error {
	switch err {
	case openrouter.ErrAPIKeyMissing:
		return ErrAPIKeyMissing
	case openrouter.ErrModelNotSupported:
		return ErrModelNotSupported
	case openrouter.ErrRateLimitExceeded:
		return ErrRateLimitExceeded
	case openrouter.ErrTimeout:
		return ErrTimeout
	case openrouter.ErrProviderNotAvailable:
		return ErrProviderNotAvailable
	default:
		return err
	}
}

// ollamaProviderWrapper wraps the Ollama provider to implement our generic interface
type ollamaProviderWrapper struct {
	provider *ollama.Provider
}

func (w *ollamaProviderWrapper) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to Ollama types
	ollamaReq := &ollama.ChatRequest{
		Model:    req.Model,
		Messages: make([]ollama.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		ollamaReq.Messages[i] = ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		ollamaReq.Options = &ollama.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call Ollama provider
	resp, err := w.provider.Chat(ctx, ollamaReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert response
	return &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: resp.Content,
		Created: resp.Created,
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (w *ollamaProviderWrapper) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// Convert to Ollama types
	ollamaReq := &ollama.ChatRequest{
		Model:    req.Model,
		Messages: make([]ollama.Message, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		ollamaReq.Messages[i] = ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	if req.Options != nil {
		ollamaReq.Options = &ollama.ChatOptions{
			Temperature: req.Options.Temperature,
			MaxTokens:   req.Options.MaxTokens,
			TopP:        req.Options.TopP,
			Stream:      req.Options.Stream,
		}
	}

	// Call Ollama provider
	ollamaStream, err := w.provider.ChatStream(ctx, ollamaReq)
	if err != nil {
		return nil, w.adaptError(err)
	}

	// Convert stream
	genericStream := make(chan *ChatChunk)
	go func() {
		defer close(genericStream)
		for chunk := range ollamaStream {
			genericChunk := &ChatChunk{
				ID:      chunk.ID,
				Model:   chunk.Model,
				Content: chunk.Content,
				Done:    chunk.Done,
			}

			select {
			case <-ctx.Done():
				return
			case genericStream <- genericChunk:
			}
		}
	}()

	return genericStream, nil
}

func (w *ollamaProviderWrapper) Name() string {
	return w.provider.Name()
}

func (w *ollamaProviderWrapper) Available() bool {
	return w.provider.Available()
}

func (w *ollamaProviderWrapper) adaptError(err error) error {
	switch err {
	case ollama.ErrAPIKeyMissing:
		return ErrAPIKeyMissing
	case ollama.ErrModelNotSupported:
		return ErrModelNotSupported
	case ollama.ErrRateLimitExceeded:
		return ErrRateLimitExceeded
	case ollama.ErrTimeout:
		return ErrTimeout
	case ollama.ErrProviderNotAvailable:
		return ErrProviderNotAvailable
	case ollama.ErrOllamaNotRunning:
		return ErrProviderNotAvailable
	default:
		return err
	}
}