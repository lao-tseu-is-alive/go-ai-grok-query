package llm

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
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

// OllamaModelDetails provides details about a model.
type OllamaModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// OllamaListModelResponse is a single model description in [ListResponse].
type OllamaListModelResponse struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	ModifiedAt time.Time          `json:"modified_at"`
	Size       int64              `json:"size"`
	Digest     string             `json:"digest"`
	Details    OllamaModelDetails `json:"details,omitempty"`
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
	url := o.BaseURL + "/api/tags"
	headers := http.Header{} // Ollama doesn't require auth headers

	type ollamaTagsResponse struct {
		Models []OllamaListModelResponse
	}

	resp, err := httpGetRequest[ollamaTagsResponse](ctx, o.Client, url, headers, o.l)
	if err != nil {
		return nil, fmt.Errorf("failed to list ollama models: %w", err)
	}

	// remove from our list the embedding

	modelInfos := make([]ModelInfo, len(resp.Models))
	for i, model := range resp.Models {
		// o.l.Info("ollama model info %s: %#v", model.Name, model)
		if !strings.Contains(model.Name, "embed") && !strings.Contains(model.Name, "paraphrase") {
			modelInfos[i] = ModelInfo{
				Name:               model.Name,
				Family:             model.Details.Family,
				Size:               model.Size,
				ParameterSize:      model.Details.ParameterSize,
				ContextSize:        8192, // safe default
				SupportsTools:      false,
				SupportsThinking:   false,
				SupportsInputImage: false,
				SupportsStreaming:  false,
				SupportsJSONMode:   false,
				SupportsStructured: false,
			}
			// adjust context size and other specifics based on documentation
			switch model.Name {
			case "deepcoder:1.5b", "deepcoder:latest":
				modelInfos[i].ContextSize = 131072
			case "deepseek-r1:latest", "deepseek-r1:1.5b", "deepseek-r1:7b,", "deepseek-r1:8b", "deepseek-r1:14b", "deepseek-r1:32b", "deepseek-r1:70b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsThinking = true
			case "devstral:latest", "devstral:24b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
			case "dolphin3:latest", "dolphin3:8b":
				modelInfos[i].ContextSize = 131072
			case "exaone-deep:latest", "exaone-deep:2.4b", "exaone-deep:7.8b", "exaone-deep:32b":
				modelInfos[i].ContextSize = 32768
			case "falcon3:latest", "falcon3:3b", "falcon3:7b", "falcon3:10b":
				modelInfos[i].ContextSize = 32768
			case "gemma3:270m", "gemma3:1b":
				modelInfos[i].ContextSize = 32768
			case "gemma3:latest", "gemma3:4b", "gemma3:12b", "gemma3:27b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsInputImage = true
			case "gpt-oss:latest", "gpt-oss:20b", "gpt-oss:120b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsThinking = true
			case "llama3.1:8b-instruct-q8_0", "llama3.1:latest", "llama3.1:8b", "llama3.1:70b", "llama3.1:405b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
			case "llama3.2:latest", "llama3.2:1b", "llama3.2:3b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
			case "llama3.2-vision:latest", "llama3.2-vision:11b", "llama3.2-vision:90b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsInputImage = true
			case "llava:latest", "llava:7b":
				modelInfos[i].ContextSize = 32768
				modelInfos[i].SupportsInputImage = true
			case "llava:13b", "llava:34b":
				modelInfos[i].ContextSize = 4096
				modelInfos[i].SupportsInputImage = true
			case "magistral:latest", "magistral:24b":
				modelInfos[i].ContextSize = 40000
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsThinking = true
			case "mistral-nemo:latest", "mistral-nemo:12b":
				modelInfos[i].ContextSize = 1024000
				modelInfos[i].SupportsTools = true
			case "mistral-small3.1:latest", "mistral-small3.1:24b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsInputImage = true
			case "mistral-small3.2:latest", "mistral-small3.2:24b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsInputImage = true
			case "mistral-small:latest", "mistral-small:24b":
				modelInfos[i].ContextSize = 32768
				modelInfos[i].SupportsTools = true
			case "mistral-small:22b":
				modelInfos[i].ContextSize = 131072
				modelInfos[i].SupportsTools = true
			case "mistral:latest", "mistral:7b":
				modelInfos[i].ContextSize = 32768
				modelInfos[i].SupportsTools = true
			case "mixtral:latest", "mixtral:8x7b":
				modelInfos[i].ContextSize = 32768
				modelInfos[i].SupportsTools = true
			case "mixtral:8x22b":
				modelInfos[i].ContextSize = 65536
				modelInfos[i].SupportsTools = true
			case "openthinker:latest", "openthinker:7b", "openthinker:32b":
				modelInfos[i].ContextSize = 32768
			case "phi4:latest", "phi4:14b":
				modelInfos[i].ContextSize = 16384
			case "qwen3:latest", "qwen3:0.6b", "qwen3:1.7b", "qwen3:8b", "qwen3:14b", "qwen3:32b":
				modelInfos[i].ContextSize = 40960
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsThinking = true
			case "qwen3-coder:latest", "qwen3-coder:30b", "qwen3-coder:480b":
				modelInfos[i].ContextSize = 262144
			case "qwen3:4b", "qwen3:30b", "qwen3:235b":
				modelInfos[i].ContextSize = 262144
				modelInfos[i].SupportsTools = true
				modelInfos[i].SupportsThinking = true
			}
		} else {
			o.l.Warn("ollama model embedding %s discarded: %#v", model.Name, model)
		}
	}

	slices.SortStableFunc(modelInfos, func(i, j ModelInfo) int {
		return cmp.Compare(i.Name, j.Name)
	})

	return modelInfos, nil
}
