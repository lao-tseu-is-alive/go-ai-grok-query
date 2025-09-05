package config

import (
	"fmt"
	"log/slog" // Use slog for optional, structured logging
	"os"
	"unicode/utf8"
)

const minKeyLength = 35

func getApiKey(envVar, providerName string) (string, error) {
	apiKey, exists := os.LookupEnv(envVar)
	if !exists {
		slog.Error(fmt.Sprintf("%s API key not set", providerName), "env_var", envVar)
		return "", fmt.Errorf("%s API key not set", providerName)
	}
	if utf8.RuneCountInString(apiKey) < minKeyLength {
		slog.Error(fmt.Sprintf("%s API key too short", providerName), "required", minKeyLength, "got", utf8.RuneCountInString(apiKey))
		return "", fmt.Errorf("%s API key must be at least %d characters (got %d)", providerName, minKeyLength, utf8.RuneCountInString(apiKey))
	}
	return apiKey, nil
}

// GetXaiApiKey returns the XAI API key from the environment.
func GetXaiApiKey() (string, error) {
	return getApiKey("XAI_API_KEY", "XAI")
}

// GetGeminiApiKey returns the Gemini API key from the environment.
func GetGeminiApiKey() (string, error) {
	return getApiKey("GEMINI_API_KEY", "Gemini")
}

// GetOpenAIApiKey returns the Gemini API key from the environment.
func GetOpenAIApiKey() (string, error) {
	return getApiKey("OPENAI_API_KEY", "OpenAI")
}

// GetOpenRouterApiKey returns the Gemini API key from the environment.
func GetOpenRouterApiKey() (string, error) {
	return getApiKey("OPEN_ROUTER_API_KEY", "OpenRouter")
}
