package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// INFO: https://github.com/ollama/ollama/blob/main/docs/api.md

const OllamaUrl = "http://localhost:11434/api/chat"
const ollamaChatPath = "/api/chat"

// OllamaProvider implements the Provider interface for local Ollama models.
type OllamaProvider struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func newOllamaAdapter(cfg ProviderConfig) (Provider, error) {
	base := cfg.BaseURL
	if base == "" {
		base = "http://localhost:11434"
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("ollama: missing model")
	}
	return &OllamaProvider{
		BaseURL: base,
		Model:   cfg.Model,
		Client:  &http.Client{Timeout: 0},
	}, nil
}

func (o *OllamaProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	payload := map[string]any{
		"model":    firstNonEmpty(req.Model, o.Model),
		"messages": toOpenAIChatMessages(req.Messages),
		"stream":   false,
	}
	if req.Temperature > 0 {
		payload["options"] = map[string]any{"temperature": req.Temperature}
	}
	if len(req.Tools) > 0 {
		// Ollama mirrors OpenAI-style tools schema
		payload["tools"] = req.Tools
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+ollamaChatPath, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("ollama: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama: http %d: %s", resp.StatusCode, string(respBody))
	}

	// Newer Ollama returns OpenAI-like single object for non-stream
	var wire struct {
		Model   string `json:"model"`
		Message struct {
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
		Done  bool   `json:"done"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal: %w", err)
	}
	if wire.Error != "" {
		return nil, fmt.Errorf("ollama: error: %s", wire.Error)
	}

	out := &LLMResponse{Text: wire.Message.Content, Raw: json.RawMessage(respBody)}
	for _, tc := range wire.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}
func (o *OllamaProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// TODO: implement streaming with chunked JSON lines; Ollama streams sequence of objects.
	return nil, fmt.Errorf("ollama: streaming not implemented")
}
func (o *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Optional: GET /api/tags to list models
	return nil, fmt.Errorf("ollama: ListModels not implemented")
}
