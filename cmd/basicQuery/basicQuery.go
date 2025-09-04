// basicQuery.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

// Constants for common defaults
const (
	defaultRole        = "You are a helpful bash shell assistant."
	defaultTemperature = 0.2
	defaultTimeout     = 30 * time.Second
)

// providerConfig holds config values for each provider
type providerConfig struct {
	Kind    llm.ProviderKind
	Model   string
	APIKey  string
	BaseURL string
}

// getProviderConfigs returns a map of available provider configs for easy selection
func getProviderConfigs() map[string]providerConfig {
	return map[string]providerConfig{
		"ollama": {
			Kind:  llm.ProviderOllama,
			Model: "qwen3:latest",
			// Ollama uses local setup, no API key needed
		},
		"gemini": {
			Kind:   llm.ProviderGemini,
			Model:  "gemini-2.5-flash",
			APIKey: os.Getenv("GEMINI_API_KEY"),
		},
		"xai": {
			Kind:   llm.ProviderXAI,
			Model:  "grok-3-mini",
			APIKey: os.Getenv("XAI_API_KEY"),
		},
		"openai": {
			Kind:   llm.ProviderOpenAI,
			Model:  "gpt-4o-mini", // Fixed model name for OpenAI compatibility
			APIKey: os.Getenv("OPENAI_API_KEY"),
		},
		"openrouter": {
			Kind:   llm.ProviderOpenRouter,
			Model:  "deepseek/deepseek-chat-v3.1:free",
			APIKey: os.Getenv("OPEN_ROUTER_API_KEY"),
		},
	}
}

func main() {
	// Define command-line flags for provider selection and prompt
	providerFlag := flag.String("provider", "openai", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	promptFlag := flag.String("prompt", "", "The prompt to send to the LLM")
	flag.Parse()

	if *promptFlag == "" {
		fmt.Println("Usage: go run basicQuery.go -provider=<provider> -prompt='your prompt'")
		fmt.Println("Available providers: ollama, gemini, xai, openai, openrouter")
		os.Exit(1)
	}

	// Fetch provider config
	configs := getProviderConfigs()
	cfg, exists := configs[*providerFlag]
	if !exists {
		fmt.Printf("## 💥💥 Error: Unknown provider '%s'. Available: ollama, gemini, xai, openai, openrouter\n", *providerFlag)
		os.Exit(1)
	}

	// Validate API key if required (Ollama is exempt)
	if cfg.Kind != llm.ProviderOllama && cfg.APIKey == "" {
		fmt.Printf("## 💥💥 Error: API key for provider '%s' not found in environment variables\n", *providerFlag)
		os.Exit(1)
	}

	// Create provider
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:    cfg.Kind,
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		// ExtraHeaders can be added here if needed per provider
	})
	if err != nil {
		log.Fatalf("## 💥💥 Error creating provider %s: %v", *providerFlag, err)
	}

	// Build the request
	req := &llm.LLMRequest{
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: defaultRole},
			{Role: llm.RoleUser, Content: *promptFlag},
		},
		Temperature: defaultTemperature,
		Stream:      false,
	}

	// Apply timeout and query
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	fmt.Printf("Sending prompt to %s LLM...\n", *providerFlag)
	resp, err := provider.Query(ctx, req)
	if err != nil {
		log.Fatalf("## 💥💥 Error querying LLM: %v", err)
	}

	fmt.Println("\nLLM Response:")
	fmt.Println(resp.Text)
}
