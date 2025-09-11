package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

const (
	DEBUG                           = true
	assertCorrectStatusCodeExpected = "expected status code should be returned"
	fmtErrNewRequest                = "### ERROR http.NewRequest %s on [%s] error is :%v\n"
	fmtTraceInfo                    = "### %s : %s on %s\n"
	fmtErr                          = "### GOT ERROR : %s\n%s"
	msgRespNotExpected              = "Response should contain what was expected."
)

var l golog.MyLogger

// mockApiServer is a helper that creates and returns a new httptest.Server
// that mimics all the LLM provider backends.
func mockApiServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"choices": [{"message": {"content": "Mock response for OpenAI-compatible API"}}]}`)
	})
	handler.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"message": {"content": "Mock response for Ollama"}}`)
	})
	handler.HandleFunc("/v1beta/models/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ":generateContent") {
			fmt.Fprintln(w, `{"candidates": [{"content": {"parts": [{"text": "Mock response for Gemini"}]}}]}`)
		} else {
			http.NotFound(w, r)
		}
	})
	//list models for openai compatible
	handler.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"object": "list","data": [{"id":"gpt-4o-mini"},{"id":"qwen/qwen3-4b:free"},{"id":"grok-3-mini"},{"id":"grok-3-mini"}]}`)
	})
	//list models for ollama
	handler.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"qwen3:latest"}]}`)
	})
	handler.HandleFunc("/v1beta/models", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"models/gemini-2.5-flash"}]}`)
	})
	return httptest.NewServer(handler)
}

func Test_run(t *testing.T) {
	// Start the mock server once for all tests
	server := mockApiServer()
	defer server.Close()

	// Define test cases for each provider
	tests := []struct {
		name    string
		p       argumentsToBasicQuery
		wantOut string // The substring we expect in the output
		wantErr bool
	}{
		{
			name: "ollama provider",
			p: argumentsToBasicQuery{
				Provider:     "ollama",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "Mock response for Ollama",
			wantErr: false,
		},
		{
			name: "openai provider",
			p: argumentsToBasicQuery{
				Provider:     "openai",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "Mock response for OpenAI-compatible API",
			wantErr: false,
		},
		{
			name: "xai provider",
			p: argumentsToBasicQuery{
				Provider:     "xai",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "Mock response for OpenAI-compatible API",
			wantErr: false,
		},
		{
			name: "openrouter provider",
			p: argumentsToBasicQuery{
				Provider:     "openrouter",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "Mock response for OpenAI-compatible API",
			wantErr: false,
		},
		{
			name: "gemini provider",
			p: argumentsToBasicQuery{
				Provider:     "gemini",
				Model:        "models/gemini-2.5-flash",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "Mock response for Gemini",
			wantErr: false,
		},
		{
			name: "no prompt error",
			p: argumentsToBasicQuery{
				Provider:     "ollama",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "",
			},
			wantOut: "prompt cannot be empty",
			wantErr: true,
		},
		{
			name: "invalid provider error",
			p: argumentsToBasicQuery{
				Provider:     "invalid",
				Model:        "",
				SystemPrompt: "",
				UserPrompt:   "test",
			},
			wantOut: "provider kind invalid is not available",
			wantErr: true,
		},
	}

	// Set environment variables for the test
	t.Setenv("OLLAMA_API_BASE", server.URL)
	t.Setenv("GEMINI_API_BASE", server.URL)
	t.Setenv("XAI_API_BASE", server.URL)
	t.Setenv("OPENAI_API_BASE", server.URL)
	t.Setenv("OPENROUTER_API_BASE", server.URL)
	t.Setenv("GEMINI_API_KEY", "dummy-key-for-testing-gemini-with-sufficient-length")
	t.Setenv("XAI_API_KEY", "dummy-key-for-testing-xai-with-sufficient-length")
	t.Setenv("OPENAI_API_KEY", "dummy-key-for-testing-openai-with-sufficient-length")
	t.Setenv("OPENROUTER_API_KEY", "dummy-key-for-testing-openrouter-with-sufficient-length")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a bytes.Buffer to capture the output of the run function.
			out := &bytes.Buffer{}
			err := run(l, tt.p, out)

			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For successful runs, check if the output contains the expected response.
			// For error runs, check if the error message contains the expected text.
			if !tt.wantErr {
				if gotOut := out.String(); !strings.Contains(gotOut, tt.wantOut) {
					t.Errorf("run() gotOut = %v, want to contain %v", gotOut, tt.wantOut)
				}
			} else if err != nil {
				if !strings.Contains(err.Error(), tt.wantOut) {
					t.Errorf("run() error = %v, want to contain %v", err.Error(), tt.wantOut)
				}
			}
		})
	}
}
func init() {
	var err error
	if DEBUG {
		l, err = golog.NewLogger("simple", os.Stdout, golog.DebugLevel, fmt.Sprintf("testing_%s ", APP))
		if err != nil {
			log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
		}
	} else {
		l, err = golog.NewLogger("simple", os.Stdout, golog.ErrorLevel, fmt.Sprintf("testing_%s ", APP))
		if err != nil {
			log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
		}
	}
}
