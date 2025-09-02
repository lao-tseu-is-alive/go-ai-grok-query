package llm

import (
	"fmt"
	"net/http"
)

type XaiProvider struct {
	openAICompatibleProvider
}

func newXaiAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("xai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("xai: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.x.ai/v1"
	}
	return &XaiProvider{
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
