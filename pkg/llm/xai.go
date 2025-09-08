package llm

import (
	"fmt"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

type XaiProvider struct {
	openAICompatibleProvider
}

func newXaiAdapter(cfg ProviderConfig, l golog.MyLogger) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("xai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("xai: missing model")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("xai: missing baseURl")
	}
	return NewOpenAICompatAdapter(cfg, ProviderXAI, cfg.BaseURL, l)
}
