package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const openRouterChatCompletionsPath = "/chat/completions"

type OpenRouterProvider struct {
	BaseURL      string
	APIKey       string
	Model        string
	Client       *http.Client
	ExtraHeaders map[string]string
}

func newOpenRouterAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("openrouter: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}
	client := &http.Client{}
	return &OpenRouterProvider{
		BaseURL:      base,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Client:       client,
		ExtraHeaders: cfg.ExtraHeaders,
	}, nil
}

func (o *OpenRouterProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	payload := map[string]any{
		"model":    firstNonEmpty(req.Model, o.Model),
		"messages": toOpenAIChatMessages(req.Messages),
		"stream":   false,
	}
	// Controls
	if req.Temperature != 0 {
		payload["temperature"] = req.Temperature
	}
	if req.TopP != 0 {
		payload["top_p"] = req.TopP
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	// Tools
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		payload["tool_choice"] = req.ToolChoice
	}
	// JSON mode / structured outputs
	if req.ResponseFormat != nil {
		payload["response_format"] = req.ResponseFormat
	}
	// OpenRouter-specific optional routing controls can be added via ProviderExtras, e.g.:
	// payload["provider"] = map[string]any{"order": [...], "allow_fallbacks": false, "require_parameters": true}
	if req.ProviderExtras != nil {
		for k, v := range req.ProviderExtras {
			payload[k] = v
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("openrouter: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+openRouterChatCompletionsPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("openrouter: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	// Optional attribution headers
	for k, v := range o.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := o.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openrouter: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter: http %d: %s", resp.StatusCode, string(respBody))
	}

	// The response schema mirrors OpenAI Chat Completions
	var wire struct {
		ID      string `json:"id"`
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
		return nil, fmt.Errorf("openrouter: unmarshal: %w", err)
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

func (o *OpenRouterProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// TODO: implement SSE streaming:
	// - payload["stream"] = true
	// - parse event-stream lines with choices[].delta.content and choices[].delta.tool_calls
	// OpenRouter streams with OpenAI-compatible deltas.
	return nil, fmt.Errorf("openrouter: streaming not implemented")
}

func (o *OpenRouterProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Optional: GET /api/v1/models isn’t standardized; many apps enumerate from website docs or keep a static list.
	// You can implement listing by calling OpenRouter’s model directory endpoint if/when available.
	return nil, fmt.Errorf("openrouter: ListModels not implemented")
}
