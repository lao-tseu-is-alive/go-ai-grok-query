package llm

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// openAICompatibleProvider provides a base for providers that use an OpenAI-compatible API.
type openAICompatibleProvider struct {
	BaseURL                string
	Kind                   ProviderKind
	APIKey                 string
	Model                  string
	CatalogProvidersModels *ModelCatalog
	Client                 *http.Client
	ExtraHeaders           map[string]string
	Endpoint               string
	l                      golog.MyLogger
}

// NewOpenAICompatAdapter is a shared constructor for OpenAI-like providers.
func NewOpenAICompatAdapter(cfg ProviderConfig, kind ProviderKind, defaultBaseURL string, l golog.MyLogger) (Provider, error) {
	baseURL := FirstNonEmpty(cfg.BaseURL, defaultBaseURL)

	filepath := config.GetProviderInfoFilePathFromEnv(defaultModelInfoFilePath)
	// Load only once the external model configuration
	catalog, err := LoadModelCatalog(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model catalog: %w", err)
	}

	return &openAICompatibleProvider{
		BaseURL:                baseURL,
		Kind:                   kind,
		APIKey:                 cfg.APIKey,
		Model:                  cfg.Model,
		CatalogProvidersModels: catalog,
		Client:                 &http.Client{},
		ExtraHeaders:           maps.Clone(cfg.ExtraHeaders), // Go 1.21+
		Endpoint:               "/chat/completions",
		l:                      l,
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
		providerConfig, ok := p.CatalogProvidersModels.Providers[string(p.Kind)]
		if !ok {
			return nil, errors.New("provider configuration not found in models.json")
		}
		tempModelInfo = providerConfig.Defaults
		// Apply model-specific overrides from the config
		if specificOverrides, exists := providerConfig.Models[model.ID]; exists {
			tempModelInfo = MergeModelInfo(providerConfig.Defaults, specificOverrides)
			p.l.Debug("model info %s after merge: %#v", model.ID, tempModelInfo)
			tempModelInfo.Name = model.ID
		} else {
			// decide if you wanna keep this model or not
			// for now by design decision we decide to discard it if not present in the  models.json config
			// as a way to filter models that should not be use because too old, or just not ok for the task

			// and let's say for now that we take all openrouter models
			if p.Kind == ProviderOpenRouter {
				tempModelInfo.Name = model.ID
			}
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

// Stream sends a streaming request to an OpenAI-compatible API.
// Deltas are sent to the onDelta callback as they arrive.
func (p *openAICompatibleProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if onDelta == nil {
		return nil, errors.New("onDelta callback cannot be nil for streaming")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("request must have at least one message")
	}

	req.Stream = true // Ensure stream is enabled
	payload := buildPayload(req, p.Model)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + p.APIKey},
		"Accept":        []string{"text/event-stream"}, // Important for SSE
		"Connection":    []string{"keep-alive"},
	}
	for key, value := range p.ExtraHeaders {
		headers[key] = []string{value}
	}

	// Create request
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request payload: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+p.Endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header = headers

	// Execute request
	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-2xx status code %d: %s", resp.StatusCode, string(body))
	}

	// Process the SSE stream
	scanner := bufio.NewScanner(resp.Body)
	finalResponse := &LLMResponse{}
	fullText := &strings.Builder{}

	// SSE wire format for deltas
	type streamChoice struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	}
	type streamChunk struct {
		Choices []streamChoice `json:"choices"`
		Usage   *Usage         `json:"usage"` // Sometimes usage is in the last chunk
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			p.l.Warn("failed to unmarshal stream chunk: %v. data: %s", err, data)
			continue
		}

		if len(chunk.Choices) > 0 {
			// Send text delta
			textDelta := chunk.Choices[0].Delta.Content
			if textDelta != "" {
				fullText.WriteString(textDelta)
				onDelta(Delta{Text: textDelta})
			}

			// Capture finish reason
			if chunk.Choices[0].FinishReason != "" {
				finalResponse.FinishReason = chunk.Choices[0].FinishReason
			}
		}

		// Capture usage stats if present in the final chunk
		if chunk.Usage != nil {
			finalResponse.Usage = chunk.Usage
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	onDelta(Delta{Done: true, FinishReason: finalResponse.FinishReason})
	finalResponse.Text = fullText.String()
	return finalResponse, nil
}
