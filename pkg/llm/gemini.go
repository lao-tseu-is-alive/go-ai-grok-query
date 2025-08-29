package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
)

// GeminiProvider implements the Provider interface for Google's Gemini models.
type GeminiProvider struct {
	BaseUrl string
	APiKey  string
	Model   string
}

const GeminiUrl = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

func newGeminiProvider(model string) (Provider, error) {
	var (
		gemini GeminiProvider
		err    error
	)
	key, err := config.GetGeminiApiKeyFromEnv()
	if err != nil {
		return nil, err
	}

	gemini.APiKey = key
	gemini.Model = model
	gemini.BaseUrl = fmt.Sprintf(GeminiUrl, model)

	return &gemini, nil
}

// Query prepares and sends the HTTP request to the Gemini API.
func (g *GeminiProvider) Query(systemPrompt, UserPrompt string) (string, error) {
	// The Gemini API does not use a "system" role in the messages array for the 'generateContent' endpoint.
	// Instead, a `system_instruction` field is used. Since your `APIRequest` doesn't have this field,
	// we will include the system prompt with the user prompt to maintain the chat flow.
	// This is a common pattern for APIs that do not support a dedicated system message role.
	requestPayload := APIRequest{
		Model: g.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("%s\n\n%s", systemPrompt, UserPrompt),
			},
		},
		Temperature: 0,
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", g.BaseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.APiKey)

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

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	}

	return "No response content received.", nil
}

func (g *GeminiProvider) ListModels() {
	//TODO implement me
	panic("implement me")
}
