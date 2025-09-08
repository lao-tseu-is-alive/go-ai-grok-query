package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
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
func NewProvider(kind ProviderKind, model string, l golog.MyLogger) (Provider, error) {
	if kind == "" {
		return nil, errors.New("provider kind cannot be empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model required for provider %q", kind)
	}
	cfg := ProviderConfig{
		Kind:         kind,
		BaseURL:      "",
		APIKey:       "",
		Model:        model,
		ExtraHeaders: nil,
		Extras:       nil,
	}

	switch cfg.Kind {
	case ProviderOpenAI:
		if cfg.APIKey == "" {
			key, err := config.GetOpenAIApiKey()
			if err != nil {
				return nil, err
			}
			l.Info("success retrieving OpenAI ApiKey")
			cfg.APIKey = key
		}
		cfg.BaseURL = config.GetApiBase("OPENAI_API_BASE", "https://api.openai.com/v1", l)
		return NewOpenAIAdapter(cfg, l)
	case ProviderOpenRouter:
		if cfg.APIKey == "" {
			key, err := config.GetOpenRouterApiKey()
			if err != nil {
				return nil, err
			}
			l.Info("success retrieving OpenRouter ApiKey")
			cfg.APIKey = key
		}
		cfg.BaseURL = config.GetApiBase("OPENROUTER_API_BASE", "https://openrouter.ai/api/v1", l)
		return NewOpenRouterAdapter(cfg, l)

	case ProviderGemini:
		if cfg.APIKey == "" {
			key, err := config.GetGeminiApiKey()
			if err != nil {
				return nil, err
			}
			l.Info("success retrieving Gemini ApiKey")
			cfg.APIKey = key
		}
		cfg.BaseURL = config.GetApiBase("GEMINI_API_BASE", "https://generativelanguage.googleapis.com", l)
		return NewGeminiAdapter(cfg, l)
	case ProviderXAI:
		if cfg.APIKey == "" {
			key, err := config.GetXaiApiKey()
			if err != nil {
				return nil, err
			}
			l.Info("success retrieving XAI ApiKey")
			cfg.APIKey = key
		}
		cfg.BaseURL = config.GetApiBase("XAI_API_BASE", "https://api.x.ai/v1", l)
		return newXaiAdapter(cfg, l) // if using OpenAI-compatible chat/completions semantics
	case ProviderOllama:
		cfg.BaseURL = config.GetApiBase("OLLAMA_API_BASE", "http://localhost:11434", l)
		return NewOllamaAdapter(cfg, l)

	default:
		return nil, fmt.Errorf("unsupported provider: %q", cfg.Kind)
	}
}

// IsLocalProvider checks if a provider doesn't need an explicit API key.
func IsLocalProvider(kind ProviderKind) bool {
	return kind == ProviderOllama
}

func GetProviderKindAndDefaultModel(kind string) (p ProviderKind, defaultModel string, err error) {
	switch kind {
	case "ollama":
		return ProviderOllama, "qwen3:latest", nil
	case "gemini":
		return ProviderGemini, "gemini-2.5-flash", nil
	case "xai":
		//standard price per 1M tokens [2025/09/08] grok3-3-mini input:$0.30, cached-input:$0.075,	output:$0.50, Live Search :$25.00/ 1K sources
		return ProviderXAI, "grok-3-mini", nil
	case "openai":
		//standard price per 1M tokens [2025/09/08] gpt-4o-mini	input:$0.15, cached-input:$0.075,	output:$0.60
		return ProviderOpenAI, "gpt-4o-mini", nil
	case "openrouter":
		return ProviderOpenRouter, "qwen/qwen3-4b:free", nil

	default:
		return "", "", fmt.Errorf("provider kind %s is not available", kind)

	}
}
