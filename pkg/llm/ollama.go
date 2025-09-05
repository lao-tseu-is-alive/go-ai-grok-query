package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid" // Add this import for tool ID generation
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// OllamaProvider implements the Provider interface for local Ollama models.
// It handles tool calls, though Ollama's API lacks tool call IDs (we generate UUIDs to maintain compatibility).
type OllamaProvider struct {
	BaseURL string
	Model   string
	Client  *http.Client
	l       golog.MyLogger
}

// ollamaRequest represents the request payload for Ollama's chat API.
type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []map[string]any `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []Tool           `json:"tools,omitempty"`
	Options  map[string]any   `json:"options,omitempty"`
}

// ollamaResponse represents the response payload from Ollama's chat API.
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

// NewOllamaAdapter creates a new OllamaProvider from config.
func NewOllamaAdapter(cfg ProviderConfig, l golog.MyLogger) (Provider, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("ollama: missing model")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("ollama: missing baseURl")
	}
	return &OllamaProvider{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Client:  &http.Client{Timeout: 30 * time.Second}, // Add timeout to prevent hangs
		l:       l,
	}, nil
}

func (o *OllamaProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("request must have messages")
	}

	// Build payload
	payload := ollamaRequest{
		Model:    FirstNonEmpty(req.Model, o.Model),
		Messages: ToOpenAIChatMessages(req.Messages), // Exported version
		Stream:   false,
	}
	if req.Temperature > 0 {
		payload.Options = map[string]any{"temperature": req.Temperature}
	}
	if len(req.Tools) > 0 {
		payload.Tools = req.Tools
	}

	headers := http.Header{"Content-Type": []string{"application/json"}}
	url := o.BaseURL + "/api/chat"

	responseData, rawResp, err := HttpRequest[ollamaRequest, ollamaResponse](ctx, o.Client, url, headers, payload, o.l)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w (raw body: %s)", err, string(rawResp))
	}
	if responseData.Error != "" {
		return nil, fmt.Errorf("ollama API error: %s", responseData.Error)
	}

	// Map to LLMResponse
	llmResp := &LLMResponse{
		Text: responseData.Message.Content,
		Raw:  json.RawMessage(rawResp),
	}
	for _, tc := range responseData.Message.ToolCalls {
		toolCall := ToolCall{
			ID:        uuid.NewString(), // Generate ID to avoid nil/blank values
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
		llmResp.ToolCalls = append(llmResp.ToolCalls, toolCall)
	}

	return llmResp, nil
}

func (o *OllamaProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	return nil, errors.New("ollama streaming not implemented") // Stub with proper error
}

func (o *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return nil, errors.New("ollama list models not implemented") // Stub
}
