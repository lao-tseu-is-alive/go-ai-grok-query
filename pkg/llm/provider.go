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

	if brand == "XAI" {
		pr, err = newXaiProvider(model)
		if err != nil {
			return nil, fmt.Errorf("error opening new Xai provider: %s", err)
		}
	} else {
		return nil, errors.New("unsupported Provider ")
	}
	return pr, nil

}
