package llm

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

var l golog.MyLogger

func TestGetProviderKindAndDefaultModel(t *testing.T) {
	testCases := []struct {
		name          string
		kindInput     string
		expectedKind  ProviderKind
		expectedModel string
		expectError   bool
	}{
		{"Ollama", "ollama", ProviderOllama, "qwen3:latest", false},
		{"Gemini", "gemini", ProviderGemini, "gemini-2.5-flash", false},
		{"XAI", "xai", ProviderXAI, "grok-3-mini", false},
		{"OpenAI", "openai", ProviderOpenAI, "gpt-4o-mini", false},
		{"OpenRouter", "openrouter", ProviderOpenRouter, "qwen/qwen3-4b:free", false},
		{"Invalid", "invalid-provider", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kind, model, err := GetProviderKindAndDefaultModel(tc.kindInput)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error for kind '%s', but got nil", tc.kindInput)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if kind != tc.expectedKind {
					t.Errorf("Expected kind '%s', but got '%s'", tc.expectedKind, kind)
				}
				if model != tc.expectedModel {
					t.Errorf("Expected model '%s', but got '%s'", tc.expectedModel, model)
				}
			}
		})
	}
}

// TestNewProvider verifies that the provider factory function correctly creates
// provider instances or returns appropriate errors.
func TestNewProvider(t *testing.T) {
	// Use a null logger for all test cases to keep output clean.
	l, _ := golog.NewLogger("simple", io.Discard, golog.FatalLevel, "test")

	// Define a dummy API key that is long enough to pass validation.
	dummyApiKey := "a_sufficiently_long_dummy_api_key_for_testing_purposes"

	testCases := []struct {
		name          string
		kind          ProviderKind
		model         string
		setupEnv      func(t *testing.T) // Function to set up environment variables for the test
		expectedType  reflect.Type
		expectError   bool
		errorContains string
	}{
		{
			name:         "Success: Create Ollama Provider",
			kind:         ProviderOllama,
			model:        "llama3",
			setupEnv:     func(t *testing.T) {}, // No API key needed for Ollama
			expectedType: reflect.TypeOf(&OllamaProvider{}),
			expectError:  false,
		},
		{
			name:  "Success: Create Gemini Provider",
			kind:  ProviderGemini,
			model: "gemini-pro",
			setupEnv: func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", dummyApiKey)
			},
			expectedType: reflect.TypeOf(&GeminiProvider{}),
			expectError:  false,
		},
		{
			name:  "Success: Create OpenAI Provider",
			kind:  ProviderOpenAI,
			model: "gpt-4",
			setupEnv: func(t *testing.T) {
				t.Setenv("OPENAI_API_KEY", dummyApiKey)
			},
			expectedType: reflect.TypeOf(&openAICompatibleProvider{}), // It's an openAICompatibleProvider underneath
			expectError:  false,
		},
		{
			name:  "Success: Create XAI Provider",
			kind:  ProviderXAI,
			model: "grok-1",
			setupEnv: func(t *testing.T) {
				t.Setenv("XAI_API_KEY", dummyApiKey)
			},
			expectedType: reflect.TypeOf(&openAICompatibleProvider{}),
			expectError:  false,
		},
		{
			name:  "Success: Create OpenRouter Provider",
			kind:  ProviderOpenRouter,
			model: "mistral-7b",
			setupEnv: func(t *testing.T) {
				t.Setenv("OPENROUTER_API_KEY", dummyApiKey)
			},
			expectedType: reflect.TypeOf(&openAICompatibleProvider{}),
			expectError:  false,
		},
		{
			name:          "Failure: Unsupported Provider Kind",
			kind:          "UnsupportedProvider",
			model:         "any-model",
			setupEnv:      func(t *testing.T) {},
			expectError:   true,
			errorContains: "unsupported provider",
		},
		{
			name:          "Failure: Empty Model Name",
			kind:          ProviderOpenAI,
			model:         "", // Model name is required
			setupEnv:      func(t *testing.T) {},
			expectError:   true,
			errorContains: "model required",
		},
		{
			name:  "Failure: Missing API Key for OpenAI",
			kind:  ProviderOpenAI,
			model: "gpt-4",
			setupEnv: func(t *testing.T) {
				t.Setenv("OPENAI_API_KEY", "") // Unset the key
			},
			expectError:   true,
			errorContains: "API key not set",
		},
		{
			name:  "Failure: Missing API Key for Gemini",
			kind:  ProviderGemini,
			model: "gemini-pro",
			setupEnv: func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", "") // Unset the key
			},
			expectError:   true,
			errorContains: "API key not set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the environment for this specific test case.
			tc.setupEnv(t)

			// Call the function we are testing.
			provider, err := NewProvider(tc.kind, tc.model, l)

			// Assert whether an error was expected.
			if tc.expectError {
				if err == nil {
					t.Fatal("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error, but got: %v", err)
				}
				// Assert that the returned provider is of the correct type.
				if reflect.TypeOf(provider) != tc.expectedType {
					t.Errorf("Expected provider of type %v, but got %v", tc.expectedType, reflect.TypeOf(provider))
				}
			}
		})
	}
}

func init() {
	var err error
	const (
		APP   = "provider"
		DEBUG = false
	)
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
