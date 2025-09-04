package config

import (
	"errors"
	"fmt"
	"log/slog" // Use slog for optional, structured logging
	"os"
	"unicode/utf8"
)

const minKeyLength = 35

// GetXaiApiKey returns the XAI API key from the environment.
func GetXaiApiKey() (string, error) {
	apiKey, exists := os.LookupEnv("XAI_API_KEY")
	if !exists {
		slog.Error("XAI API key not set", "env_var", "XAI_API_KEY", "help", "set via export XAI_API_KEY or https://console.x.ai")
		return "", errors.New("XAI API key not set")
	}
	if utf8.RuneCountInString(apiKey) < minKeyLength {
		slog.Error("XAI API key too short", "required", minKeyLength, "got", utf8.RuneCountInString(apiKey))
		return "", fmt.Errorf("XAI API key must be at least %d characters (got %d)", minKeyLength, utf8.RuneCountInString(apiKey))
	}
	return apiKey, nil
}

// GetGeminiApiKey returns the Gemini API key from the environment.
func GetGeminiApiKey() (string, error) {
	apiKey, exists := os.LookupEnv("GEMINI_API_KEY")
	if !exists {
		slog.Error("Gemini API key not set", "env_var", "GEMINI_API_KEY", "help", "set via export GEMINI_API_KEY")
		return "", errors.New("gemini API key not set")
	}
	if utf8.RuneCountInString(apiKey) < minKeyLength {
		slog.Error("Gemini API key too short", "required", minKeyLength, "got", utf8.RuneCountInString(apiKey))
		return "", fmt.Errorf("gemini API key must be at least %d characters (got %d)", minKeyLength, utf8.RuneCountInString(apiKey))
	}
	return apiKey, nil
}

// GetOpenAIApiKey returns the Gemini API key from the environment.
func GetOpenAIApiKey() (string, error) {
	apiKey, exists := os.LookupEnv("OPENAI_API_KEY")
	if !exists {
		slog.Error("OpenAI API key not set", "env_var", "OPENAI_API_KEY", "help", "set via export OPENAI_API_KEY")
		return "", errors.New("OpenAI API key not set")
	}
	if utf8.RuneCountInString(apiKey) < minKeyLength {
		slog.Error("OpenAI API key too short", "required", minKeyLength, "got", utf8.RuneCountInString(apiKey))
		return "", fmt.Errorf("OpenAI API key must be at least %d characters (got %d)", minKeyLength, utf8.RuneCountInString(apiKey))
	}
	return apiKey, nil
}

// GetOpenRouterApiKey returns the Gemini API key from the environment.
func GetOpenRouterApiKey() (string, error) {
	apiKey, exists := os.LookupEnv("OPEN_ROUTER_API_KEY")
	if !exists {
		slog.Error("OpenRouter API key not set", "env_var", "OPEN_ROUTER_API_KEY", "help", "set via export OPEN_ROUTER_API_KEY")
		return "", errors.New("OpenRouter API key not set")
	}
	if utf8.RuneCountInString(apiKey) < minKeyLength {
		slog.Error("OpenRouter API key too short", "required", minKeyLength, "got", utf8.RuneCountInString(apiKey))
		return "", fmt.Errorf("OpenRouter API key must be at least %d characters (got %d)", minKeyLength, utf8.RuneCountInString(apiKey))
	}
	return apiKey, nil
}
