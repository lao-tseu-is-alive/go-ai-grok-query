package config

import (
	"fmt"
	"log/slog" // Use slog for optional, structured logging
	"net/url"
	"os"
	"unicode/utf8"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
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
	return getApiKey("OPENROUTER_API_KEY", "OpenRouter")
}

// GetApiBase retrieves a base URL from a given environment variable.
// It validates that the URL is well-formed. If the environment variable is not set,
// is empty, or contains an invalid URL, it logs a warning and returns the
// provided defaultURL as a safe fallback.
func GetApiBase(envVar, defaultURL string, l golog.MyLogger) string {
	// 1. Get the URL from the environment variable.
	envURL := os.Getenv(envVar)
	// 2. If the environment variable is not set, use the default.
	if envURL == "" {
		return defaultURL
	}
	// 3. If it is set, validate it.
	_, err := url.ParseRequestURI(envURL)
	if err != nil {
		// If the URL is invalid, log a warning and use the default as a safe fallback.
		l.Warn("in env %s, got invalid URL %s; falling back to default: %s, err: %s", envVar, envURL, defaultURL, err)
		return defaultURL
	}
	// 4. If it's valid, return the user-provided URL.
	l.Info("Using custom API base URL from env %s : %s", envVar, envURL)
	return envURL
}
