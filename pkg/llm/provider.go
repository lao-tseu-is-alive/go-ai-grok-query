package llm

import (
	"context"
	"errors"
	"fmt"

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

func NewProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Kind {
	case ProviderOpenAI:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.openai.com/v1"
		}
		return newOpenAIAdapter(cfg)
	case ProviderOpenRouter:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://openrouter.ai/api/v1"
		}
		// OpenRouter is OpenAI-compatible chat/completions; reuse adapter
		return newOpenRouterAdapter(cfg)

	case ProviderGemini:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://generativelanguage.googleapis.com"
		}
		if cfg.APIKey == "" {
			key, err := config.GetGeminiApiKeyFromEnv()
			if err != nil {
				return nil, err
			}
			cfg.APIKey = key
		}
		return newGeminiAdapter(cfg)
	case ProviderXAI:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.x.ai/v1"
		}
		if cfg.APIKey == "" {
			key, err := config.GetXaiApiKeyFromEnv()
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
		return newOllamaAdapter(cfg)
	default:
		return nil, errors.New(fmt.Sprintf("unsupported provider kind: %s", cfg.Kind))
	}
}
