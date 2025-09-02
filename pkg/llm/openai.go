package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const openAIChatCompletionsPath = "/chat/completions"

type OpenAIProvider struct {
	BaseURL      string
	APIKey       string
	Model        string
	Client       *http.Client
	ExtraHeaders map[string]string
}

func newOpenAIAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("openai: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	client := &http.Client{}
	return &OpenAIProvider{
		BaseURL:      base,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Client:       client,
		ExtraHeaders: cfg.ExtraHeaders,
	}, nil
}

func (o *OpenAIProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	payload := map[string]any{
		"model":    firstNonEmpty(req.Model, o.Model),
		"messages": toOpenAIChatMessages(req.Messages),
		"stream":   false,
	}

	// Optional controls
	if req.Temperature != 0 {
		payload["temperature"] = req.Temperature
	}
	if req.TopP != 0 {
		payload["top_p"] = req.TopP
	}
	if req.MaxTokens > 0 {
		// Some models use max_completion_tokens; chat completions harmonizes as max_tokens
		payload["max_tokens"] = req.MaxTokens
	}

	// Tools and tool choice
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		// Accept string values: "auto" | "none" | "required"; or structured object {type:function,name:...}
		payload["tool_choice"] = req.ToolChoice
	}

	// JSON mode / structured outputs
	if req.ResponseFormat != nil {
		// Typical: { "type": "json_object" } or { "type": "json_schema", "json_schema": {...} }
		payload["response_format"] = req.ResponseFormat
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+openAIChatCompletionsPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("openai: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	for k, v := range o.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := o.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Chat Completions response
	var wire struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int    `json:"index"`
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
		return nil, fmt.Errorf("openai: unmarshal: %w", err)
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

func (o *OpenAIProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// TODO: Implement SSE streaming parsing:
	// - Send "stream": true
	// - Read event-stream lines and parse each JSON data chunk
	// - Emit onDelta with delta.content and delta.tool_calls as present
	// Reference behavior: choices[].delta.content, choices[].delta.tool_calls[*].function.{name,arguments}
	return nil, fmt.Errorf("openai: streaming not implemented")
}

func (o *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Optional: GET /v1/models to enumerate models and set feature flags
	// Many orgs restrict visibility; consider deferring implementation or making it configurable.
	return nil, fmt.Errorf("openai: ListModels not implemented")
}

// Reuse helpers from other adapters

func toOpenAIChatMessages(msgs []LLMMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		item := map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
		if m.Name != "" {
			item["name"] = m.Name
		}
		if m.ToolCallID != "" {
			item["tool_call_id"] = m.ToolCallID
		}
		out = append(out, item)
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
