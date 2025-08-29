package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// INFO: https://github.com/ollama/ollama/blob/main/docs/api.md

// OllamaProvider implements the Provider interface for local Ollama models.
type OllamaProvider struct {
	BaseUrl string
	Model   string
}

const OllamaUrl = "http://localhost:11434/api/chat"

func newOllamaProvider(model string) (Provider, error) {
	return &OllamaProvider{
		BaseUrl: OllamaUrl,
		Model:   model,
	}, nil
}

// Query prepares and sends the HTTP request to the Ollama API.
func (o *OllamaProvider) Query(systemPrompt, UserPrompt string) (string, error) {
	requestPayload := APIRequest{
		Model: o.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: UserPrompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", o.BaseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResponse OllamaAPIResponse
	if err := json.Unmarshal(body, &ollamaResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	if len(ollamaResponse.Message.Content) > 0 {
		return ollamaResponse.Message.Content, nil
	}

	fmt.Printf("### something went wrong, here is the received body:\n %#v\n###", body)

	return "No response content received.", nil
}

func (o *OllamaProvider) ListModels() {
	//TODO implement me
	panic("implement me")
}
