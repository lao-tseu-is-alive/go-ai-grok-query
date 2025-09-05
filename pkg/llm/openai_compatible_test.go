package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// TestOpenAICompatProviderQuery tests the full Query flow against a mock server.
func TestOpenAICompatProviderQuery(t *testing.T) {
	// Simple text response from the mock API
	mockTextResponse := `{
		"choices": [
			{
				"finish_reason": "stop",
				"message": {
					"role": "assistant",
					"content": "This is a test response."
				}
			}
		],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
	}`

	// Tool call response from the mock API
	mockToolCallResponse := `{
		"choices": [
			{
				"finish_reason": "tool_calls",
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [
						{
							"id": "call_abc123",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\": \"Lausanne\"}"
							}
						}
					]
				}
			}
		]
	}`

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request path is correct
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		// Check for Authorization header
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode the request body to see what kind of response to send
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		w.Header().Set("Content-Type", "application/json")
		if _, ok := reqBody["tools"]; ok {
			// If the request includes tools, send the tool call response
			w.Write([]byte(mockToolCallResponse))
		} else {
			// Otherwise, send the simple text response
			w.Write([]byte(mockTextResponse))
		}
	}))
	defer server.Close()

	// Use a null logger for tests
	l, _ := golog.NewLogger("simple", io.Discard, golog.FatalLevel, "test")

	// Create the provider pointing to our mock server
	provider := &openAICompatibleProvider{
		BaseURL:  server.URL,
		APIKey:   "test-api-key",
		Model:    "test-model",
		Client:   server.Client(),
		Endpoint: "/chat/completions",
		l:        l,
	}

	// === Test Case 1: Simple Text Query ===
	t.Run("SimpleTextQuery", func(t *testing.T) {
		req := &LLMRequest{
			Messages: []LLMMessage{{Role: RoleUser, Content: "Hello"}},
		}

		resp, err := provider.Query(context.Background(), req)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if resp.Text != "This is a test response." {
			t.Errorf("Expected text 'This is a test response.', got '%s'", resp.Text)
		}
		if len(resp.ToolCalls) != 0 {
			t.Errorf("Expected 0 tool calls, got %d", len(resp.ToolCalls))
		}
		if resp.Usage.TotalTokens != 15 {
			t.Errorf("Expected usage.total_tokens to be 15, got %d", resp.Usage.TotalTokens)
		}
	})

	// === Test Case 2: Tool Call Query ===
	t.Run("ToolCallQuery", func(t *testing.T) {
		req := &LLMRequest{
			Messages: []LLMMessage{{Role: RoleUser, Content: "What's the weather?"}},
			Tools:    []Tool{{Type: "function", Function: ToolSpec{Name: "get_weather"}}},
		}

		resp, err := provider.Query(context.Background(), req)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if resp.Text != "" {
			t.Errorf("Expected empty text for tool call response, got '%s'", resp.Text)
		}
		if len(resp.ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
		}
		toolCall := resp.ToolCalls[0]
		if toolCall.ID != "call_abc123" {
			t.Errorf("Expected tool ID 'call_abc123', got '%s'", toolCall.ID)
		}
		if toolCall.Name != "get_weather" {
			t.Errorf("Expected tool name 'get_weather', got '%s'", toolCall.Name)
		}
	})
}
