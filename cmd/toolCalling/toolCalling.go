package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

// A simple local function that emulates a backend tool.
func getCurrentWeather(args json.RawMessage) (string, error) {
	var p struct {
		Location string `json:"location"`
		Format   string `json:"format,omitempty"` // "celsius" | "fahrenheit"
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if p.Location == "" {
		return "", fmt.Errorf("missing location")
	}
	if p.Format == "" {
		p.Format = "celsius"
	}
	// Stub a fake reading
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

	// 1) Create GPT provider (OpenAI)
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:   llm.ProviderOpenAI,
		Model:  "gpt-4.1-mini",
		APIKey: apiKey,
	})
	check(err, "creating OpenAI provider")

	// 2) Declare tools (OpenAI tools schema)
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

	// 3) Initialize conversation
	messages := []llm.LLMMessage{
		{Role: llm.RoleSystem, Content: "You are a helpful weather assistant. Use tools when needed and ask for clarification if inputs are ambiguous."},
		{Role: llm.RoleUser, Content: "What's the weather right now in San Francisco, CA?"},
	}

	// 4) First call with tools, expecting a tool call back
	req := &llm.LLMRequest{
		Model:      "gpt-4.1-mini",
		Messages:   messages,
		Tools:      []llm.Tool{weatherTool},
		ToolChoice: "auto", // let the model decide
		Stream:     false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Calling GPT for tool decision...")
	resp, err := provider.Query(ctx, req)
	check(err, "first query")
	// If the model returns text AND tool calls, the text can be treated as thoughts/explanation; the tool call drives execution. [16]

	// 5) Handle a single tool call (extend for multiple as needed)
	if len(resp.ToolCalls) == 0 {
		// No tool call: print the text answer and exit
		fmt.Println("\nAssistant:", resp.Text)
		return
	}
	// 5a) Re-insert the assistant message that requested the tools.
	// Many APIs accept tool calls in assistant additional fields even if content is empty.
	assistantToolMsg := llm.LLMMessage{
		Role:    llm.RoleAssistant,
		Content: resp.Text, // optional; can be empty if model returned only tool calls
	}
	// Attach tool_calls in ProviderExtras for adapter to forward, OR if your adapter
	// already stores tool calls only in response, you can keep content and just proceed.
	// For OpenAI-compatible schemas, you must serialize tool_calls into the assistant message.
	// If your adapter doesn't support that directly, you can skip setting tool_calls here
	// and rely on the prior response; BUT you must still add an assistant turn to maintain order.
	messages = append(messages, assistantToolMsg)

	// 5b) Append one tool result per tool_call id, in order
	for _, call := range resp.ToolCalls {
		fmt.Printf("Model requested tool call: %s(%s)\n", call.Name, string(call.Arguments))

		// Execute the tool based on call.Name
		var result string
		var err error
		switch call.Name {
		case "get_current_weather":
			result, err = getCurrentWeather(call.Arguments)
		// add more tools here...
		default:
			err = fmt.Errorf("unknown tool: %s", call.Name)
		}
		if err != nil {
			// decide how to surface tool error; often returned as tool content too
			result = fmt.Sprintf(`{"error":%q}`, err.Error())
		}

		// Append tool result message with matching tool_call_id for each call
		messages = append(messages, llm.LLMMessage{
			Role:       llm.RoleTool,
			Name:       call.Name,
			ToolCallID: call.ID,
			Content:    result, // should be a JSON string
		})
	}

	// 7) Second call with the tool result included to get the final natural language answer
	req2 := &llm.LLMRequest{
		Model:    "gpt-4.1-mini",
		Messages: messages,
		Stream:   false,
		// Tools can be included again to allow follow-up calls; or omitted if not needed.
		Tools:      []llm.Tool{weatherTool},
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
