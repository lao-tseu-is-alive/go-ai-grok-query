package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

// Ollama-specific request payload structure
type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []map[string]any `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []Tool           `json:"tools,omitempty"`
	Options  map[string]any   `json:"options,omitempty"`
}

// Ollama-specific response payload structure
type ollamaResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
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
	// 1. Build the Ollama-specific request payload
	payload := ollamaRequest{
		Model:    firstNonEmpty(req.Model, o.Model),
		Messages: toOpenAIChatMessages(req.Messages),
		Stream:   false,
	}
	if req.Temperature > 0 {
		payload.Options = map[string]any{"temperature": req.Temperature}
	}
	if len(req.Tools) > 0 {
		payload.Tools = req.Tools
	}

	// 2. Prepare headers and call the generic helper
	headers := http.Header{"Content-Type": []string{"application/json"}}
	url := o.BaseURL + ollamaChatPath

	wire, rawResp, err := httpRequest[ollamaRequest, ollamaResponse](ctx, o.Client, url, headers, payload)
	if err != nil {
		return nil, fmt.Errorf("ollama: http error: %w body: %s", err, string(rawResp))
	}
	if wire.Error != "" {
		return nil, fmt.Errorf("ollama: provider error: %s", wire.Error)
	}

	// 3. Map the Ollama response to our standard LLMResponse
	out := &LLMResponse{
		Text: wire.Message.Content,
		Raw:  json.RawMessage(rawResp),
	}
	for _, tc := range wire.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			// Note: Ollama doesn't provide a tool call ID, so we might need
			// to generate one if downstream logic depends on it.
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
