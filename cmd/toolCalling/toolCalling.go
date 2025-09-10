package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/version"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

const (
	defaultSystemPrompt = "You are a helpful weather assistant. Use tools when asked for weather data."
	defaultPrompt       = "What's the weather right now in Lausanne in Switzerland?"
	defaultLogName      = "stderr"
)

func check(err error, msg string, l golog.MyLogger) {
	if err != nil {
		l.Error("💥💥 error %s: %v\n", msg, err)
		os.Exit(1)
	}
}

// WeatherTool Dummy weather tool with proper implementation.
type WeatherTool struct {
	l golog.MyLogger
}

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

	w.l.Info("Executing tool get_current_weather location %s", params.Location)

	// Simulate a service call with mock data (replace with real API).
	result := map[string]any{
		"location": params.Location,
		"temp":     22.5,
		"unit":     map[string]string{"celsius": "C", "fahrenheit": "F"}[params.Format],
		"summary":  "Partly cloudy with light breeze",
	}

	if strings.Contains(params.Location, "Lausanne") {
		result["summary"] = "You better look out your window"
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err) // Unlikely, but safe.
	}
	return string(out), nil
}

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic(defaultLogName),
		config.GetLogLevelFromEnvOrPanic(golog.InfoLevel),
		version.APP,
	)
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	// Define command-line flags for provider selection and prompt
	providerFlag := flag.String("provider", "openai", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	systemRoleFlag := flag.String("system", defaultSystemPrompt, "The system prompt, it default here to a weather assistant")
	promptFlag := flag.String("prompt", defaultPrompt, "The prompt to send to the LLM")
	flag.Parse()

	if *promptFlag == "" {
		fmt.Println("Usage: go run basicQuery.go -provider=<provider> -prompt='your prompt'")
		fmt.Println("Available providers: ollama, gemini, xai, openai, openrouter")
		os.Exit(1)
	}

	kind, model, err := llm.GetProviderKindAndDefaultModel(*providerFlag)
	if err != nil {
		fmt.Printf("## 💥💥 Error: Unknown provider '%s'. Available: ollama, gemini, xai, openai, openrouter\n", *providerFlag)
		os.Exit(1)
	}
	l.Info("will create provider llm.NewProvider(kind:%s, model:%s)", kind, model)
	provider, err := llm.NewProvider(kind, model, l)
	if err != nil {
		l.Error("## 💥💥 Error creating provider %s: %v", *providerFlag, err)
		os.Exit(1)
	}

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

	l.Info("🚀 step 1: Creating a new conversation with a system prompt : ")
	l.Info("🚀 system prompt : %s", *systemRoleFlag)
	convo, err := llm.NewConversation(*systemRoleFlag)
	check(err, "starting conversation", l)

	l.Info("Adding the user's prompt : %s", *promptFlag)
	err = convo.AddUserMessage(*promptFlag)
	check(err, "adding user message", l)

	l.Info("step 2: First API call to let the model decide on tools.")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	l.Info("calling LLM for tool decision", "model", model)
	req := &llm.LLMRequest{
		Messages:   convo.MessagesCopy(), // Use thread-safe copy from Conversation.
		Tools:      []llm.Tool{weatherTool},
		ToolChoice: "auto",
	}
	resp, err := provider.Query(ctx, req)
	check(err, "first query", l)
	l.Info("LLM returned first response: %s, finishReason: %s, toolsCalls: %#v", resp.Text, resp.FinishReason, resp.ToolCalls)
	// Add the assistant's response (could include tool calls).
	convo.AddAssistantResponse(resp)

	// Step 3: If tool calls were made, execute them and collect results.
	if len(resp.ToolCalls) > 0 {
		for _, tc := range resp.ToolCalls {
			l.Info("LLM returned requested tool call: %s(%s)\n", tc.Name, string(tc.Arguments))

			// Execute the tool: Use the WeatherTool struct (no more undefined function).
			tool := WeatherTool{l: l} // Instantiate the tool.
			result, err := tool.Execute(tc.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error": %q}`, err.Error()) // JSON-safe error response.
				l.Warn("Tool execution failed", "tool", tc.Name, "error", err)
			}
			l.Info("Result of tool call %s(%s): %v\n", tc.Name, string(tc.Arguments), result)

			// Add the tool's result back to the conversation.
			convo.AddToolResultMessage(tc.ID, result)
		}

		l.Info("step 4: Second API call with tool results for the final response.")
		l.Info("Calling LLM for final answer with tool results")
		req2 := &llm.LLMRequest{
			Model:    "gpt-4o-mini",
			Messages: convo.MessagesCopy(),    // Safe copy again.
			Tools:    []llm.Tool{weatherTool}, // Include tools if needed for consistency.
		}
		resp2, err := provider.Query(ctx, req2)
		check(err, "second query", l)

		l.Info("\nAssistant's Final Response:")
		fmt.Println(resp2.Text)
	} else {
		// No tool calls: Just print the direct response.
		l.Info("\nAssistant's Direct Response (no tool calls):")
		fmt.Println(resp.Text)
	}

	l.Info("Tool calling example completed successfully")
}
