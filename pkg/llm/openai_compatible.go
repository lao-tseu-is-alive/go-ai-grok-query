package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// openAICompatibleProvider provides a base for providers that use an OpenAI-compatible API.
type openAICompatibleProvider struct {
	BaseURL      string
	APIKey       string
	Model        string
	Client       *http.Client
	ExtraHeaders map[string]string
	Endpoint     string
}

func (p *openAICompatibleProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 1. Build the payload
	payload := buildPayload(req, p.Model)

	// 2. Marshal the payload
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// 3. Create and send the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+p.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	for k, v := range p.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	// 4. Execute the request
	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 5. Read and check the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-2xx status code %d: %s", resp.StatusCode, string(respBody))
	}

	// 6. Unmarshal and format the response
	return unmarshalResponse(respBody)
}

// buildPayload creates the request payload for an OpenAI-compatible API.
func buildPayload(req *LLMRequest, defaultModel string) map[string]any {
	payload := map[string]any{
		"model":    firstNonEmpty(req.Model, defaultModel),
		"messages": toOpenAIChatMessages(req.Messages),
		"stream":   req.Stream,
	}
	// Add optional parameters...
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		payload["top_p"] = req.TopP
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		payload["tool_choice"] = req.ToolChoice
	}
	if req.ResponseFormat != nil {
		payload["response_format"] = req.ResponseFormat
	}
	if req.ProviderExtras != nil {
		if mos, ok := req.ProviderExtras["messages_override"].([]map[string]any); ok && len(mos) > 0 {
			payload["messages"] = mos
		}
	}
	return payload
}

// unmarshalResponse parses the response from an OpenAI-compatible API.
func unmarshalResponse(respBody []byte) (*LLMResponse, error) {
	var wire struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message"`
		} `json:"choices"`
		Usage *Usage `json:"usage,omitempty"`
	}

	if err := json.Unmarshal(respBody, &wire); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	out := &LLMResponse{
		Raw:   json.RawMessage(respBody),
		Usage: wire.Usage,
	}
	if len(wire.Choices) > 0 {
		first := wire.Choices[0]
		out.Text = first.Message.Content
		out.FinishReason = first.FinishReason
		for _, tc := range first.Message.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}
	return out, nil
}

// Stream and ListModels would also be part of this struct
func (p *openAICompatibleProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	return nil, fmt.Errorf("streaming not implemented")
}

func (p *openAICompatibleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Optional: GET /v1/models to enumerate models and set feature flags
	// Many orgs restrict visibility; consider deferring implementation or making it configurable.

	return nil, fmt.Errorf("ListModels not implemented")
}
