package llm

import (
	"errors"
	"fmt"
)

type Provider interface {
	Query(systemPrompt, UserPrompt string) (string, error)
	ListModels()
}

func GetInstance(brand, model string) (Provider, error) {
	var (
		pr  Provider
		err error
	)

	switch brand {
	case "XAI":
		pr, err = newXaiProvider(model)
		if err != nil {
			return nil, fmt.Errorf("error opening new Xai provider: %s", err)
		}
	case "Gemini":
		pr, err = newGeminiProvider(model)
		if err != nil {
			return nil, fmt.Errorf("error opening new Gemini provider: %s", err)
		}
	case "Ollama":
		pr, err = newOllamaProvider(model)
		if err != nil {
			return nil, fmt.Errorf("error opening new Ollama provider: %s", err)
		}
	default:
		return nil, errors.New("unsupported Provider ")
	}
	return pr, nil

}
