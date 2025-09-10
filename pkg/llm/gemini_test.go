package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// TestGeminiProvider_Stream verifies that the Gemini provider can correctly
// parse a streaming JSON array response.
func TestGeminiProvider_Stream(t *testing.T) {
	// 1. Define the mock server response.
	// This is a valid JSON array streamed piece by piece.
	mockStreamChunks := []string{
		`[`,
		`{
			"candidates": [{
				"content": {"parts": [{"text": "Hello,"}]},
				"finishReason": "RECITATION",
				"index": 0
			}]
		}`,
		`,`,
		`{
			"candidates": [{
				"content": {"parts": [{"text": " world!"}]},
				"finishReason": "STOP",
				"index": 0
			}],
			"usageMetadata": {"promptTokenCount": 5, "candidatesTokenCount": 10, "totalTokenCount": 15}
		}`,
		`]`,
	}
	expectedFullText := "Hello, world!"

	// 2. Create a mock HTTP server that simulates streaming.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Stream the chunks with a small delay to mimic a real network response.
		for _, chunk := range mockStreamChunks {
			_, err := w.Write([]byte(chunk))
			if err != nil {
				return
			}
			// Flush the writer to ensure the client receives the data immediately.
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(10 * time.Millisecond) // Small delay
		}
	}))
	defer server.Close()

	// 3. Set up the GeminiProvider to use the mock server.
	l, _ := golog.NewLogger("simple", io.Discard, golog.FatalLevel, "test")
	provider := &GeminiProvider{
		BaseURL: server.URL,
		APIKey:  "test-api-key", // Not used by the mock server, but required by the struct.
		Model:   "gemini-test",
		Client:  server.Client(),
		l:       l,
	}

	// 4. Prepare the request and the onDelta callback to collect results.
	req := &LLMRequest{
		Messages: []LLMMessage{{Role: RoleUser, Content: "Hi"}},
	}
	var receivedDeltas []string
	var finalDelta Delta
	onDelta := func(delta Delta) {
		if delta.Text != "" {
			receivedDeltas = append(receivedDeltas, delta.Text)
		}
		if delta.Done {
			finalDelta = delta
		}
	}

	// 5. Execute the Stream method.
	finalResponse, err := provider.Stream(context.Background(), req, onDelta)

	// 6. Assert the results.
	if err != nil {
		t.Fatalf("Stream() returned an unexpected error: %v", err)
	}

	// Check the collected deltas.
	receivedText := strings.Join(receivedDeltas, "")
	if receivedText != expectedFullText {
		t.Errorf("Expected concatenated deltas to be '%s', but got '%s'", expectedFullText, receivedText)
	}

	// Check the final aggregated response.
	if finalResponse.Text != expectedFullText {
		t.Errorf("Expected final response text to be '%s', but got '%s'", expectedFullText, finalResponse.Text)
	}

	// Check the finish reason from the final delta and the response.
	expectedFinishReason := "STOP"
	if finalDelta.FinishReason != expectedFinishReason {
		t.Errorf("Expected final delta finish reason to be '%s', but got '%s'", expectedFinishReason, finalDelta.FinishReason)
	}
	if finalResponse.FinishReason != expectedFinishReason {
		t.Errorf("Expected final response finish reason to be '%s', but got '%s'", expectedFinishReason, finalResponse.FinishReason)
	}

	// Check usage data.
	if finalResponse.Usage.TotalTokens != 15 {
		t.Errorf("Expected final usage total tokens to be 15, but got %d", finalResponse.Usage.TotalTokens)
	}

	fmt.Println("✅ Gemini stream test passed successfully.")
}
