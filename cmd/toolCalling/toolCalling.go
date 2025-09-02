package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

// Tool implementation
func getCurrentWeather(args json.RawMessage) (string, error) {
	// OpenAI tool_calls function.arguments is a JSON-encoded string.
	// Step 1: decode into a Go string
	var argStr string
	if err := json.Unmarshal(args, &argStr); err != nil {
		// If the provider ever returns raw object (some servers do), fall back to object unmarshal
		var tmp map[string]any
		if err2 := json.Unmarshal(args, &tmp); err2 != nil {
			return "", fmt.Errorf("decode arguments: as string: %v; as object: %v", err, err2)
		}
		// Re-encode normalized object as string and continue
		b, _ := json.Marshal(tmp)
		argStr = string(b)
	}

	// Step 2: unmarshal the inner JSON into the expected struct
	var p struct {
		Location string `json:"location"`
		Format   string `json:"format,omitempty"`
	}
	if err := json.Unmarshal([]byte(argStr), &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if p.Location == "" {
		return "", fmt.Errorf("missing location")
	}
	if p.Format == "" {
		p.Format = "celsius"
	}

	// Produce a compact JSON result
	result := map[string]any{
		"location": p.Location,
		"temp":     22.5,
		"unit":     map[string]string{"celsius": "C", "fahrenheit": "F"}[p.Format],
		"summary":  "Partly cloudy with light breeze",
	}
	out, _ := json.Marshal(result)
	return string(out), nil
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY")
		os.Exit(1)
	}

	// Provider
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:   llm.ProviderOpenAI,
		Model:  "gpt-4.1-mini",
		APIKey: apiKey,
	})
	check(err, "creating OpenAI provider")

	// Tool schema
	weatherTool := llm.Tool{
		Type: "function",
		Function: llm.ToolSpec{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city and state, e.g., Portland, OR",
					},
					"format": map[string]any{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
					},
				},
				"required": []string{"location"},
			},
		},
	}

	// Turn 1: system + user
	sys := llm.LLMMessage{Role: llm.RoleSystem, Content: "You are a helpful weather assistant. Use tools when needed and ask for clarification if inputs are ambiguous."}
	usr := llm.LLMMessage{Role: llm.RoleUser, Content: "What's the weather right now in San Francisco, CA?"}
	messages := []llm.LLMMessage{sys, usr}

	req := &llm.LLMRequest{
		Model:      "gpt-4.1-mini",
		Messages:   messages,
		Tools:      []llm.Tool{weatherTool},
		ToolChoice: "auto",
		Stream:     false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Calling GPT for tool decision...")
	resp, err := provider.Query(ctx, req)
	check(err, "first query")

	if len(resp.ToolCalls) == 0 {
		fmt.Println("\nAssistant:", resp.Text)
		return
	}

	// Build exact OpenAI-shaped messages for Turn 2
	// 1) copy prior messages
	messagesOpenAI := []map[string]any{
		{"role": "system", "content": sys.Content},
		{"role": "user", "content": usr.Content},
	}

	// 2) assistant with tool_calls (from resp.ToolCalls)
	toolCallsWire := make([]map[string]any, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
		toolCallsWire = append(toolCallsWire, map[string]any{
			"id":   tc.ID,
			"type": "function",
			"function": map[string]any{
				"name":      tc.Name,
				"arguments": string(tc.Arguments),
			},
		})
	}
	messagesOpenAI = append(messagesOpenAI, map[string]any{
		"role":       "assistant",
		"content":    resp.Text, // may be empty
		"tool_calls": toolCallsWire,
	})

	// 3) tool results (one per tool_call), matching tool_call_id in the same order
	for _, tc := range resp.ToolCalls {
		fmt.Printf("Model requested tool call: %s(%s)\n", tc.Name, string(tc.Arguments))
		result, err := getCurrentWeather(tc.Arguments)
		if err != nil {
			result = fmt.Sprintf(`{"error":%q}`, err.Error())
		}
		fmt.Printf("result of calling : %s(%s)  : %#v\n", tc.Name, string(tc.Arguments), result)
		messagesOpenAI = append(messagesOpenAI, map[string]any{
			"role":         "tool",
			"tool_call_id": tc.ID,
			"name":         tc.Name,
			"content":      result,
		})
	}

	// After appending tool messages, add a brief user instruction prompting the final answer.
	// This helps models that otherwise might wait for a natural language cue.
	messagesOpenAI = append(messagesOpenAI, map[string]any{
		"role":    "user",
		"content": "Please use the tool result above to answer my original question clearly.",
	})

	// Turn 2 request: ensure payload contains a non-empty "messages"
	req2 := &llm.LLMRequest{
		Model:  "gpt-4.1-mini",
		Stream: false,
		Tools:  []llm.Tool{weatherTool},
		ProviderExtras: map[string]any{
			"messages_override": messagesOpenAI,
		},
		ToolChoice: "auto",
	}

	fmt.Println("Calling GPT for final answer...")
	resp2, err := provider.Query(ctx, req2)
	check(err, "second query")

	fmt.Println("\nAssistant:")
	fmt.Println(resp2.Text)
}

func check(err error, msg string) {
	if err != nil {
		fmt.Printf("Error %s: %v\n", msg, err)
		os.Exit(1)
	}
}
