// basicQuery.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/config"
	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/version"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// Constants for common defaults
const (
	defaultRole        = "You are a helpful bash shell assistant.Your output should be concise, efficient and easy to read in a bash Linux console."
	defaultTemperature = 0.2
	defaultTimeout     = 30 * time.Second
)

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic("stderr"),
		config.GetLogLevelFromEnvOrPanic(golog.DebugLevel),
		"basicQuery",
	)
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	// Define command-line flags for provider selection and prompt
	providerFlag := flag.String("provider", "openai", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	systemRoleFlag := flag.String("system role", defaultRole, "The system role for your assistant, it default to an helpful shell assistant")
	promptFlag := flag.String("prompt", "", "The prompt to send to the LLM")
	flag.Parse()

	if *promptFlag == "" {
		fmt.Println("Usage: go run basicQuery.go -provider=<provider> -prompt='your prompt'")
		fmt.Println("Available providers: ollama, gemini, xai, openai, openrouter")
		os.Exit(1)
	}
	l.Info("you asked for provider: %s", *providerFlag)
	kind, model, err := llm.GetProviderKindAndDefaultModel(*providerFlag)
	if err != nil {
		fmt.Printf("## 💥💥 Error: Unknown provider '%s'. Available: ollama, gemini, xai, openai, openrouter\n", *providerFlag)
		os.Exit(1)
	}

	// Create provider
	l.Info("will call llm.NewProvider(kind:%s, model:%s)", kind, model)
	provider, err := llm.NewProvider(kind, model, l)
	if err != nil {
		log.Fatalf("## 💥💥 Error creating provider %s: %v", *providerFlag, err)
	}

	// Build the request
	req := &llm.LLMRequest{
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: *systemRoleFlag},
			{Role: llm.RoleUser, Content: *promptFlag},
		},
		Temperature: defaultTemperature,
		Stream:      false,
	}

	// Apply timeout and query
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	l.Info("Sending prompt to %s LLM...\n", kind)
	resp, err := provider.Query(ctx, req)
	if err != nil {
		log.Fatalf("## 💥💥 Error querying LLM: %v", err)
	}

	fmt.Println("\nLLM Response:")
	fmt.Println(resp.Text)
}
