package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// mockApiServer remains the same, it correctly handles the different paths.
func mockApiServer() *httptest.Server {
	handler := http.NewServeMux()

	handler.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		response := `{"choices": [{"message": {"role": "assistant", "content": "Mock response for OpenAI-compatible API"}}], "usage": {"total_tokens": 7}}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	})

	handler.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		response := `{"message": {"role": "assistant", "content": "Mock response for Ollama"}, "done": true}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	})

	// The model name is part of the URL, so we handle a generic path for any test model.
	handler.HandleFunc("/v1beta/models/", func(w http.ResponseWriter, r *http.Request) {
		response := `{"candidates": [{"content": {"parts": [{"text": "Mock response for Gemini"}]}}], "usageMetadata": {"totalTokenCount": 8}}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	})

	return httptest.NewServer(handler)
}

func TestAllProvidersIntegration(t *testing.T) {
	server := mockApiServer()
	defer server.Close()

	l, _ := golog.NewLogger("simple", io.Discard, golog.FatalLevel, "test")

	// Set dummy API keys to pass the initial configuration checks inside NewProvider
	t.Setenv("OPENAI_API_KEY", "dummy-key-for-testing-openai-with-sufficient-length")
	t.Setenv("OPENROUTER_API_KEY", "dummy-key-for-testing-openrouter-with-sufficient-length")
	t.Setenv("XAI_API_KEY", "dummy-key-for-testing-xai-with-sufficient-length")
	t.Setenv("GEMINI_API_KEY", "dummy-key-for-testing-gemini-with-sufficient-length")

	testCases := []struct {
		providerKind     string
		model            string
		expectedResponse string
	}{
		{"openai", "gpt-test", "Mock response for OpenAI-compatible API"},
		{"openrouter", "router-test", "Mock response for OpenAI-compatible API"},
		{"xai", "grok-test", "Mock response for OpenAI-compatible API"},
		{"ollama", "ollama-test", "Mock response for Ollama"},
		{"gemini", "gemini-test", "Mock response for Gemini"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Provider_%s", tc.providerKind), func(t *testing.T) {
			kind, _, err := GetProviderKindAndDefaultModel(tc.providerKind)
			if err != nil {
				t.Fatalf("GetProviderKindAndDefaultModel failed: %v", err)
			}

			// *** THIS IS THE NEW, SIMPLER, AND CORRECT LOGIC ***
			// We no longer rely on reflection. We create a config and pass it to the
			// specific adapter, which is a much cleaner way to control the BaseURL.
			var provider Provider
			cfg := ProviderConfig{
				Kind:    kind,
				Model:   tc.model,
				BaseURL: server.URL, // Set the mock server URL directly
				APIKey:  "dummy-key-for-testing-with-sufficient-length",
			}

			switch kind {
			case ProviderOpenAI, ProviderOpenRouter:
				provider, err = NewOpenAICompatAdapter(cfg, server.URL, l)
			case ProviderXAI:
				provider, err = newXaiAdapter(cfg, l)
			case ProviderOllama:
				provider, err = NewOllamaAdapter(cfg, l)
			case ProviderGemini:
				provider, err = NewGeminiAdapter(cfg, l)
			default:
				t.Fatalf("unhandled provider kind: %s", kind)
			}

			if err != nil {
				t.Fatalf("Adapter creation failed for kind '%s': %v", kind, err)
			}
			// *** END OF CORRECTION ***

			req := &LLMRequest{
				Messages: []LLMMessage{{Role: RoleUser, Content: "Ping"}},
			}

			resp, err := provider.Query(context.Background(), req)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}

			if resp.Text != tc.expectedResponse {
				t.Errorf("Expected response '%s', but got '%s'", tc.expectedResponse, resp.Text)
			}
		})
	}
}
