package llm

import "testing"

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
