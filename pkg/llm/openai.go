package llm

import (
	"fmt"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

type OpenAIProvider struct {
	openAICompatibleProvider
}

func NewOpenAIAdapter(cfg ProviderConfig, l golog.MyLogger) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("openai: missing model")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("openai: missing baseUrl")
	}
	return NewOpenAICompatAdapter(cfg, cfg.BaseURL, l)
}
