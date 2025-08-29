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

type XaiProvider struct {
	BaseUrl string
	APiKey  string
	Model   string
}

const XaiUrl = "https://api.x.ai/v1/chat/completions"

func newXaiProvider(model string) (Provider, error) {
	var (
		xai XaiProvider
		err error
	)
	//TODO: define a list of valid model for XAI
	if model != "grok-3-mini" {
		return nil, fmt.Errorf("xai does not have a modl named : %s ", model)
	}
	key, err := config.GetXaiApiKeyFromEnv()
	if err != nil {
		return nil, err
	}
	xai.BaseUrl = XaiUrl
	xai.APiKey = key
	xai.Model = model
	return &xai, nil
}

// Query prepares and sends the HTTP request to the LLM API.
func (x *XaiProvider) Query(systemPrompt, UserPrompt string) (string, error) {
	// Create the request payload using the structs defined earlier.
	// This now includes a system message, the correct model, and other parameters.
	requestPayload := APIRequest{
		Model: x.Model,
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
		Stream:      false,
		Temperature: 0,
	}

	// Marshal the Go struct into a JSON byte slice.
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %w", err)
	}

	// Create a new HTTP request. We use a context with a timeout to prevent
	// the program from hanging indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", x.BaseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set the required HTTP headers.
	// The Content-Type tells the server we're sending JSON.
	// The Authorization header carries our API key.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+x.APiKey)

	// Execute the request using the default HTTP client.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to API: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Check for non-200 status codes which indicate an API error.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the JSON response body into our APIResponse struct.
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	// Extract the content from the first choice.
	// Check if there are any choices to avoid a panic.
	if len(apiResponse.Choices) > 0 {
		fmt.Printf("We received %d\n choices", len(apiResponse.Choices))
		return apiResponse.Choices[0].Message.Content, nil
	}

	return "No response content received.", nil
}

func (x *XaiProvider) ListModels() {
	//TODO implement me
	panic("implement me")
}
