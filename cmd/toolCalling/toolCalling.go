package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

func check(err error, msg string) {
	if err != nil {
		fmt.Printf("Error %s: %v\n", msg, err)
		os.Exit(1)
	}
}

// WeatherTool Dummy weather tool with proper implementation.
type WeatherTool struct{}

// Execute Tool implementation
func (w WeatherTool) Execute(args json.RawMessage) (string, error) {
	// Step 1: Unmarshal into string first (as per your original logic)
	var argStr string
	if err := json.Unmarshal(args, &argStr); err != nil {
		// Fallback to object
		var tmp map[string]any
		if err2 := json.Unmarshal(args, &tmp); err2 != nil {
			return "", fmt.Errorf("decode arguments: as string: %v; as object: %v", err, err2)
		}
		data, _ := json.Marshal(tmp)
		argStr = string(data)
	}

	// Step 2: Parse JSON
	var params struct {
		Location string `json:"location"`
		Format   string `json:"format,omitempty"`
	}
	if err := json.Unmarshal([]byte(argStr), &params); err != nil {
		return "", fmt.Errorf("parse params: %w", err)
	}
	if params.Location == "" {
		return "", fmt.Errorf("missing location parameter")
	}
	params.Format = llm.FirstNonEmpty(params.Format, "celsius")

	// Log for debugging (easy to toggle via slog level).
	slog.Info("Executing weather tool",
		"location", params.Location,
		"format", params.Format,
		"tool", "get_current_weather")

	// Simulate a service call with mock data (replace with real API).
	// For demo, always returns the same weather.
	result := map[string]any{
		"location": params.Location,
		"temp":     22.5,
		"unit":     map[string]string{"celsius": "C", "fahrenheit": "F"}[params.Format],
		"summary":  "Partly cloudy with light breeze",
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err) // Unlikely, but safe.
	}
	return string(out), nil
}

func main() {
	// Validate environment (keeps secure, no secrets in code).
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable.")
		os.Exit(1)
	}

	// Initialize provider (with improved error handling).
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:   llm.ProviderOpenAI,
		Model:  "gpt-4o-mini", // Fixed: Assuming this is what you meant (common OpenAI model). Change if incorrect.
		APIKey: apiKey,
	})
	check(err, "creating OpenAI provider")

	// Define the tool schema (following OpenAI Function Calling spec).
	weatherTool := llm.Tool{
		Type: "function",
		Function: llm.ToolSpec{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "City and state, e.g., San Francisco, CA.",
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

	// Step 1: Create a new conversation with a system prompt.
	convo, err := llm.NewConversation("You are a helpful weather assistant. Use tools when asked for weather data.")
	check(err, "starting conversation") // Now properly handles the error.

	// Add the user's query.
	err = convo.AddUserMessage("What's the weather right now in San Francisco, CA?")
	check(err, "adding user message")

	// Step 2: First API call to let the model decide on tools.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Info("Calling LLM for tool decision", "model", "gpt-4o-mini")
	req := &llm.LLMRequest{
		Model:      "gpt-4o-mini",        // Consistent with provider config.
		Messages:   convo.MessagesCopy(), // Use thread-safe copy from Conversation.
		Tools:      []llm.Tool{weatherTool},
		ToolChoice: "auto",
	}
	resp, err := provider.Query(ctx, req)
	check(err, "first query")

	// Add the assistant's response (could include tool calls).
	convo.AddAssistantResponse(resp)

	// Step 3: If tool calls were made, execute them and collect results.
	if len(resp.ToolCalls) > 0 {
		for _, tc := range resp.ToolCalls {
			fmt.Printf("Model requested tool call: %s(%s)\n", tc.Name, string(tc.Arguments))

			// Execute the tool: Use the WeatherTool struct (no more undefined function).
			tool := WeatherTool{} // Instantiate the tool.
			result, err := tool.Execute(tc.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error": %q}`, err.Error()) // JSON-safe error response.
				slog.Warn("Tool execution failed", "tool", tc.Name, "error", err)
			}
			fmt.Printf("Result of tool call %s(%s): %v\n", tc.Name, string(tc.Arguments), result)

			// Add the tool's result back to the conversation.
			convo.AddToolResultMessage(tc.ID, result)
		}

		// Step 4: Second API call with tool results for the final response.
		slog.Info("Calling LLM for final answer with tool results")
		req2 := &llm.LLMRequest{
			Model:    "gpt-4o-mini",
			Messages: convo.MessagesCopy(),    // Safe copy again.
			Tools:    []llm.Tool{weatherTool}, // Include tools if needed for consistency.
		}
		resp2, err := provider.Query(ctx, req2)
		check(err, "second query")

		fmt.Println("\nAssistant's Final Response:")
		fmt.Println(resp2.Text)
	} else {
		// No tool calls: Just print the direct response.
		fmt.Println("\nAssistant's Direct Response (no tool calls):")
		fmt.Println(resp.Text)
	}

	slog.Info("Tool calling example completed successfully")
}
