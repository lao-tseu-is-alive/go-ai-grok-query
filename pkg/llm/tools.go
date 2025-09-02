package llm

import (
	"fmt"
	"os"
)

// Reuse helpers from other adapters

func toOpenAIChatMessages(msgs []LLMMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		// The content can be an empty string, but for some models (like early OpenAI ones)
		// it was better to send `null`. Sending an empty string is broadly compatible.
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

		// ✨ NEW: Handle serializing the assistant's tool calls ✨
		if len(m.ToolCalls) > 0 {
			// The API expects a specific structure for tool_calls
			apiToolCalls := make([]map[string]any, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				apiToolCalls[i] = map[string]any{
					"id":   tc.ID,
					"type": "function", // Currently, only "function" is supported
					"function": map[string]any{
						"name": tc.Name,
						// The API expects the arguments to be a string
						"arguments": string(tc.Arguments),
					},
				}
			}
			item["tool_calls"] = apiToolCalls

			// Per OpenAI spec, if tool_calls is present, content must be null.
			// While many models handle an empty string, `nil` is more correct.
			item["content"] = nil
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

func Check(err error, msg string) {
	if err != nil {
		fmt.Printf("Error %s: %v\n", msg, err)
		os.Exit(1)
	}
}
