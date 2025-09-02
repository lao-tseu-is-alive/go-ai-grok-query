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

	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:   llm.ProviderOpenAI,
		Model:  "gpt-4.1-mini",
		APIKey: apiKey,
	})
	llm.Check(err, "creating OpenAI provider")

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

	// 1. Start a new conversation
	convo := llm.NewConversation("You are a helpful weather assistant.")
	convo.AddUserMessage("What's the weather right now in San Francisco, CA?")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. First API Call
	fmt.Println("Calling GPT for tool decision...")
	req := &llm.LLMRequest{
		Model:      "gpt-4.1-mini",
		Messages:   convo.Messages,
		Tools:      []llm.Tool{weatherTool},
		ToolChoice: "auto",
	}
	resp, err := provider.Query(ctx, req)
	llm.Check(err, "first query")

	// Add the assistant's turn to the conversation
	convo.AddAssistantResponse(resp)

	// 3. Check for tool calls and execute them
	if len(resp.ToolCalls) > 0 {
		for _, tc := range resp.ToolCalls {
			fmt.Printf("Model requested tool call: %s(%s)\n", tc.Name, string(tc.Arguments))
			result, err := getCurrentWeather(tc.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error":%q}`, err.Error())
			}
			fmt.Printf("result of calling : %s(%s)  : %#v\n", tc.Name, string(tc.Arguments), result)
			// Add the tool result back to the conversation
			convo.AddToolResultMessage(tc.ID, result)
		}

		// 4. Second API Call (with the tool result in the history)
		fmt.Println("Calling GPT for final answer...")
		req2 := &llm.LLMRequest{
			Model:    "gpt-4.1-mini",
			Messages: convo.Messages,
			Tools:    []llm.Tool{weatherTool},
		}
		resp2, err := provider.Query(ctx, req2)
		llm.Check(err, "second query")

		fmt.Println("\nAssistant:")
		fmt.Println(resp2.Text)
	} else {
		// No tool call was made, just print the response
		fmt.Println("\nAssistant:")
		fmt.Println(resp.Text)
	}
}
