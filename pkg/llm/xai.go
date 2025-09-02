package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const XaiUrl = "https://api.x.ai/v1/chat/completions"
const xaiChatCompletionsPath = "/chat/completions"

type XaiProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
	// ExtraHeaders merged on each request
	ExtraHeaders map[string]string
}

func newXaiAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("xai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("xai: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.x.ai/v1"
	}
	return &XaiProvider{
		BaseURL:      base,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Client:       &http.Client{},
		ExtraHeaders: cfg.ExtraHeaders,
	}, nil
}

func (x *XaiProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// Map LLMRequest -> OpenAI-style payload
	payload := map[string]any{
		"model":       firstNonEmpty(req.Model, x.Model),
		"messages":    toOpenAIChatMessages(req.Messages),
		"temperature": req.Temperature,
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

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("xai: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", x.BaseURL+xaiChatCompletionsPath, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("xai: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+x.APIKey)
	for k, v := range x.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := x.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xai: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("xai: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xai: http %d: %s", resp.StatusCode, string(respBody))
	}

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
		return nil, fmt.Errorf("xai: unmarshal: %w", err)
	}
	out := &LLMResponse{Usage: wire.Usage, Raw: json.RawMessage(respBody)}
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

func (x *XaiProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// TODO: implement SSE streaming if/when x.ai exposes OpenAI-compatible streaming.
	return nil, fmt.Errorf("xai: streaming not implemented")
}

func (x *XaiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// TODO: implement when x.ai model listing endpoint is available.
	return nil, fmt.Errorf("xai: ListModels not implemented")
}
