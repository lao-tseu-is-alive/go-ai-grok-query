package llm

import (
	"encoding/json"
	"fmt"
	"math"
)

// ToOpenAIChatMessages converts internal messages to OpenAI API format.
// It handles optional fields like tool_calls and ensures compatibility.
func ToOpenAIChatMessages(msgs []LLMMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		item := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Name != "" {
			item["name"] = msg.Name
		}
		if msg.ToolCallID != "" {
			item["tool_call_id"] = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			apiToolCalls := make([]map[string]any, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				apiToolCalls[i] = map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": string(tc.Arguments),
					},
				}
			}
			item["tool_calls"] = apiToolCalls
			item["content"] = nil // OpenAI spec requires null when tool_calls present
		}
		out = append(out, item)
	}
	return out
}

// FirstNonEmpty returns the first non-empty string, falling back to the second.
func FirstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ToolExecutor defines an interface for executing tools.
// This makes tools pluggable and testable.
type ToolExecutor interface {
	Execute(args json.RawMessage) (string, error)
}

// ExampleToolRegistry is a simple map of tool names to executors (for demonstration).
type ExampleToolRegistry map[string]ToolExecutor

// Execute looks up and runs a tool by name.
func (r ExampleToolRegistry) Execute(name string, args json.RawMessage) (string, error) {
	exec, exists := r[name]
	if !exists {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return exec.Execute(args)
}

func Clamp(val, min, max float64) float64 {
	return math.Min(max, math.Max(min, val))
}
