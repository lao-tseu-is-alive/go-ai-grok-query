package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Query sends a request to an OpenAI-compatible API.
// It validates inputs and handles responses robustly.
func (p *openAICompatibleProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	// Validate required fields
	if len(req.Messages) == 0 {
		return nil, errors.New("request must have at least one message")
	}
	// ... (additional validations if needed)

	payload := buildPayload(req, p.Model)
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + p.APIKey},
	}
	// Merge extra headers (p.ExtraHeaders and req.ExtraHeaders are map[string]string, so convert to []string)
	for key, value := range p.ExtraHeaders {
		headers[key] = []string{value}
	}
	for key, value := range req.ExtraHeaders {
		headers[key] = []string{value}
	}
	/*
		var wireResponse struct {
			Choices []struct {
				FinishReason string    `json:"finish_reason"`
				Message      *struct { // Pointer to detect nil
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string          `json:"id"`
						Type     string          `json:"type"`
						Function json.RawMessage `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"message"`
			} `json:"choices"`
			Usage *Usage `json:"usage,omitempty"`
		}
	*/

	_, rawBody, err := httpRequest[map[string]any, any](
		ctx, p.Client, p.BaseURL+p.Endpoint, headers, payload,
	)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	// Use dedicated unmarshal for better control
	resp, err := unmarshalResponse(rawBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return resp, nil
}

// unmarshalResponse parses wire data into LLMResponse.
// Handles common API edge cases.
func unmarshalResponse(rawResp json.RawMessage) (*LLMResponse, error) {
	var wire struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      *struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string          `json:"id"`
					Type     string          `json:"type"`
					Function json.RawMessage `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message,omitempty"`
		} `json:"choices,omitempty"`
		Usage *Usage `json:"usage,omitempty"`
	}

	if err := json.Unmarshal(rawResp, &wire); err != nil {
		return nil, fmt.Errorf("unmarshal wire response: %w", err)
	}
	if len(wire.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	firstMsg := wire.Choices[0].Message
	if firstMsg == nil {
		return nil, errors.New("first choice has nil message")
	}

	resp := &LLMResponse{
		Text:         firstMsg.Content,
		FinishReason: wire.Choices[0].FinishReason,
		Usage:        wire.Usage,
		Raw:          rawResp,
	}

	for _, tc := range firstMsg.ToolCalls {
		var fn struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(tc.Function, &fn); err != nil {
			return nil, fmt.Errorf("unmarshal tool function: %w", err)
		}
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      fn.Name,
			Arguments: fn.Arguments,
		})
	}
	return resp, nil
}

// buildPayload creates the request payload for an OpenAI-compatible API.
func buildPayload(req *LLMRequest, defaultModel string) map[string]any {
	payload := map[string]any{
		"model":    FirstNonEmpty(req.Model, defaultModel),
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

// Stream and ListModels would also be part of this struct
func (p *openAICompatibleProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	return nil, fmt.Errorf("streaming not implemented")
}

func (p *openAICompatibleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Optional: GET /v1/models to enumerate models and set feature flags
	// Many orgs restrict visibility; consider deferring implementation or making it configurable.

	return nil, fmt.Errorf("ListModels not implemented")
}
