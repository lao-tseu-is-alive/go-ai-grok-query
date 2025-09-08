package llm

import "encoding/json"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type LLMMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	// Optional: name (assistant tool name), tool call id when returning tool output
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// JSON Schema for arguments
	Parameters map[string]any `json:"parameters"`
}

type Tool struct {
	Type     string   `json:"type"` // must be "function" for OpenAI schema
	Function ToolSpec `json:"function"`
}

type ToolChoice struct {
	Type string `json:"type"` // "auto", "none", "required", or "function"
	// When Type == "function"
	Function struct {
		Name string `json:"name"`
	} `json:"function,omitempty"`
}

type ResponseFormat struct {
	// "json_object" for JSON mode; or "json_schema" for structured outputs (where supported)
	Type       string         `json:"type,omitempty"`
	JSONSchema map[string]any `json:"json_schema,omitempty"`
}

type LLMRequest struct {
	Model          string          `json:"model"`
	Messages       []LLMMessage    `json:"messages"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     any             `json:"tool_choice,omitempty"` // ToolChoice or string
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Stream      bool    `json:"stream,omitempty"`

	// ProviderExtras allows per-provider flags without polluting the core schema
	ProviderExtras map[string]any `json:"-"`
	// ExtraHeaders (per-request) merged with ProviderConfig.ExtraHeaders
	ExtraHeaders map[string]string `json:"-"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type LLMResponse struct {
	Text         string     `json:"text"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	// Raw provider response for debugging
	Raw json.RawMessage `json:"raw,omitempty"`
}

type Delta struct {
	// Text delta for streaming
	Text string `json:"text,omitempty"`
	// ToolCall deltas when tools are emitted
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// Whether this is the final chunk
	Done bool `json:"done,omitempty"`
	// Optional reason on done
	FinishReason string `json:"finish_reason,omitempty"`
}

type ModelInfo struct {
	Name          string `json:"name"`
	Family        string `json:"family,omitempty"`
	Size          int64  `json:"size,omitempty"`
	ParameterSize string `json:"parameter_size,omitempty"`
	ContextSize   int    `json:"context_size,omitempty"`
	// Feature flags
	SupportsTools      bool `json:"supports_tools,omitempty"`
	SupportsThinking   bool `json:"supports_thinking,omitempty"`
	SupportsInputImage bool `json:"supports_input_image,omitempty"`
	SupportsStreaming  bool `json:"supports_streaming,omitempty"`
	SupportsJSONMode   bool `json:"supports_json_mode,omitempty"`
	SupportsStructured bool `json:"supports_structured,omitempty"`
}

//To calculate how fast the response is generated in tokens per second
// (token/s), divide eval_count / eval_duration * 10^9.
