package llm

import (
	"fmt"
	"net/http"
)

type OpenRouterProvider struct {
	openAICompatibleProvider
}

func newOpenRouterAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("openai: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}

	return &OpenAIProvider{
		openAICompatibleProvider: openAICompatibleProvider{
			BaseURL:      base,
			APIKey:       cfg.APIKey,
			Model:        cfg.Model,
			Client:       &http.Client{},
			ExtraHeaders: cfg.ExtraHeaders,
			Endpoint:     "/chat/completions",
		},
	}, nil
}
