package llm

import (
	"fmt"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

type OpenRouterProvider struct {
	openAICompatibleProvider
}

func NewOpenRouterAdapter(cfg ProviderConfig, l golog.MyLogger) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("openrouter: missing model")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("openrouter: missing baseURl")
	}
	return NewOpenAICompatAdapter(cfg, ProviderOpenRouter, cfg.BaseURL, l)
}
