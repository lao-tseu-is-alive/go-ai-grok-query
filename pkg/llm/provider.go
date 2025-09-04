package llm

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
)

type ProviderKind string

const (
	ProviderOpenAI     ProviderKind = "OpenAI"
	ProviderOpenRouter ProviderKind = "OpenRouter"
	ProviderGemini     ProviderKind = "Gemini"
	ProviderXAI        ProviderKind = "XAI"
	ProviderOllama     ProviderKind = "Ollama"
)

type Provider interface {
	// Query performs a single non-streaming request.
	Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	// Stream performs a streaming request and emits deltas via the callback.
	Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error)
	// ListModels returns available models (optional to implement initially).
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

type ProviderConfig struct {
	Kind    ProviderKind
	BaseURL string
	APIKey  string
	Model   string
	// Optional headers (e.g., OpenRouter: HTTP-Referer, X-Title)
	ExtraHeaders map[string]string
	// ProviderExtras for feature flags, timeouts, etc.
	Extras map[string]any
}

// NewProvider creates a new provider based on kind.
// Validates config and applies defaults.
func NewProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.Kind == "" {
		return nil, errors.New("provider kind cannot be empty")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model required for provider %q", cfg.Kind)
	}

	switch cfg.Kind {
	case ProviderOpenAI:
		if cfg.APIKey == "" {
			key, err := config.GetOpenAIApiKey()
			if err != nil {
				return nil, err
			}
			cfg.APIKey = key
		}
		return NewOpenAICompatAdapter(cfg, "https://api.openai.com/v1")
	case ProviderOpenRouter:
		if cfg.APIKey == "" {
			key, err := config.GetOpenRouterApiKey()
			if err != nil {
				return nil, err
			}
			cfg.APIKey = key
		}
		return NewOpenAICompatAdapter(cfg, "https://openrouter.ai/api/v1")

	case ProviderGemini:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://generativelanguage.googleapis.com"
		}
		if cfg.APIKey == "" {
			key, err := config.GetGeminiApiKey()
			if err != nil {
				return nil, err
			}
			cfg.APIKey = key
		}
		return NewGeminiAdapter(cfg)
	case ProviderXAI:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.x.ai/v1"
		}
		if cfg.APIKey == "" {
			key, err := config.GetXaiApiKey()
			if err != nil {
				return nil, err
			}
			cfg.APIKey = key
		}
		return newXaiAdapter(cfg) // if using OpenAI-compatible chat/completions semantics
	case ProviderOllama:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://localhost:11434"
		}
		return NewOllamaAdapter(cfg)

	default:
		return nil, fmt.Errorf("unsupported provider: %q", cfg.Kind)
	}
}

// isLocalProvider checks if a provider doesn't need an explicit API key.
func isLocalProvider(kind ProviderKind) bool {
	return kind == ProviderOllama
}

// NewOpenAICompatAdapter is a shared constructor for OpenAI-like providers.
func NewOpenAICompatAdapter(cfg ProviderConfig, defaultBaseURL string) (Provider, error) {
	baseURL := FirstNonEmpty(cfg.BaseURL, defaultBaseURL)
	return &openAICompatibleProvider{
		BaseURL:      baseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Client:       &http.Client{},
		ExtraHeaders: maps.Clone(cfg.ExtraHeaders), // Go 1.21+
		Endpoint:     "/chat/completions",
	}, nil
}
