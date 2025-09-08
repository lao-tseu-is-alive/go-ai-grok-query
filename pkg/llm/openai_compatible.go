package llm

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// openAICompatibleProvider provides a base for providers that use an OpenAI-compatible API.
type openAICompatibleProvider struct {
	BaseURL      string
	Kind         ProviderKind
	APIKey       string
	Model        string
	Client       *http.Client
	ExtraHeaders map[string]string
	Endpoint     string
	l            golog.MyLogger
}

// NewOpenAICompatAdapter is a shared constructor for OpenAI-like providers.
func NewOpenAICompatAdapter(cfg ProviderConfig, kind ProviderKind, defaultBaseURL string, l golog.MyLogger) (Provider, error) {
	baseURL := FirstNonEmpty(cfg.BaseURL, defaultBaseURL)
	return &openAICompatibleProvider{
		BaseURL:      baseURL,
		Kind:         kind,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Client:       &http.Client{},
		ExtraHeaders: maps.Clone(cfg.ExtraHeaders), // Go 1.21+
		Endpoint:     "/chat/completions",
		l:            l,
	}, nil
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
	p.l.Debug("about to send request to %s", p.BaseURL+p.Endpoint)
	_, rawBody, err := HttpRequest[map[string]any, any](
		ctx, p.Client, p.BaseURL+p.Endpoint, headers, payload, p.l,
	)
	if err != nil {
		p.l.Warn("got error during HttpRequest: %q", err)
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	p.l.Debug("successful HttpRequest, rawbody: %s", string(rawBody))
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
		"messages": ToOpenAIChatMessages(req.Messages),
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

// ListModels fetches the list of available models from an OpenAI-compatible API.
func (p *openAICompatibleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := p.BaseURL + "/models"
	headers := http.Header{
		"Authorization": []string{"Bearer " + p.APIKey},
	}
	for key, value := range p.ExtraHeaders {
		headers.Set(key, value)
	}

	type modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	resp, err := httpGetRequest[modelsResponse](ctx, p.Client, url, headers, p.l)
	if err != nil {
		return nil, fmt.Errorf("failed to list models from %s: %w", p.BaseURL, err)
	}

	modelInfos := make([]ModelInfo, 0, len(resp.Data))
	for _, model := range resp.Data {
		var tempModelInfo ModelInfo
		tempModelInfo.SupportsStreaming = true
		switch p.Kind {
		case ProviderXAI:
			switch model.ID {
			case "grok-code-fast-1", "grok-4-0709":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 256000
				tempModelInfo.SupportsTools = true
				tempModelInfo.SupportsStructured = true
				tempModelInfo.SupportsThinking = true
			case "grok-3", "grok-3-mini":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 131072
				tempModelInfo.SupportsTools = true
				tempModelInfo.SupportsStructured = true
				tempModelInfo.SupportsThinking = true
			default:
				// do not list deprecated models or image generation models
			}
		case ProviderOpenAI:
			switch model.ID {
			case "gpt-5", "gpt-5-mini", "gpt-5-nano":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 400000
				tempModelInfo.SupportsTools = true
				tempModelInfo.SupportsStructured = true
				tempModelInfo.SupportsInputImage = true
				tempModelInfo.SupportsThinking = true

			case "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 1000000     // mini also documented with long context in 4.1 guides[2][1]
				tempModelInfo.SupportsTools = true      // function calling / tools[1][3]
				tempModelInfo.SupportsStructured = true // structured outputs supported in newer models[3][4]
				tempModelInfo.SupportsInputImage = true
				tempModelInfo.SupportsThinking = false // not in reasoning series[5][1]

				// GPT‑4o family (omni; widely used for chat; text-to-text supported)
			case "gpt-4o", "gpt-4o-mini":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 128000 // typical large context for 4o in platform docs[6][3]
				tempModelInfo.SupportsTools = true // tools/function calling supported[7][3]
				tempModelInfo.SupportsStructured = true
				tempModelInfo.SupportsInputImage = true
				tempModelInfo.SupportsThinking = false // not a reasoning-series model[7][5]

				// Reasoning series (o3 / o3-mini / o4-mini) — used for tougher chat tasks
			case "o3", "o3-mini":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 200000      // per o3 model page[8]
				tempModelInfo.SupportsTools = true      // reasoning models support tools[9][8]
				tempModelInfo.SupportsStructured = true // reasoning models in 2025 support structured outputs[10][11]
				tempModelInfo.SupportsThinking = true   // reasoning models (internal deliberate thinking)[12][8]

			case "o4", "o4-mini":
				tempModelInfo.Name = model.ID
				tempModelInfo.ContextSize = 200000      // successor small reasoning model; large context typical[13][7]
				tempModelInfo.SupportsTools = true      // reasoning models support tools[14][13]
				tempModelInfo.SupportsStructured = true // modern structured output support[14][10]
				tempModelInfo.SupportsThinking = true   // reasoning-focused model[13][7]

			default:
				// Ignore deprecated and old very expensive ones, 3.5 series, embeddings, audio/realtime/search/transcribe previews, image-only, nanos
			}

		default:
			tempModelInfo.Name = model.ID
		}
		if tempModelInfo.Name != "" {
			modelInfos = append(modelInfos, tempModelInfo)
		}
	}

	slices.SortStableFunc(modelInfos, func(i, j ModelInfo) int {
		return cmp.Compare(i.Name, j.Name)
	})

	return modelInfos, nil
}
