package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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

type argumentsToBasicQuery struct {
	Provider     string
	Model        string
	SystemPrompt string
	UserPrompt   string
}

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic("stderr"),
		config.GetLogLevelFromEnvOrPanic(golog.DebugLevel),
		"basicQuery",
	)
	if err != nil {
		log.Fatalf("💥💥 error creating logger: %v\n", err)
	}

	// Define command-line flags for provider selection and prompt
	providerFlag := flag.String("provider", "ollama", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	modelFlag := flag.String("model", "", "Model to use, depends on chosen provider, leave blank for a default valid choice")
	systemPromptFlag := flag.String("system", defaultRole, "The system role for your assistant, it default to an helpful shell assistant")
	userPromptFlag := flag.String("prompt", "", "The prompt to send to the LLM")
	flag.Parse()

	if *userPromptFlag == "" {
		fmt.Println("Usage: go run basicQuery.go -provider=<provider> -prompt='your prompt'")
		fmt.Println("Available providers: ollama, gemini, xai, openai, openrouter")
		os.Exit(1)
	}
	l.Info("you asked for provider: %s", *providerFlag)

	params := argumentsToBasicQuery{
		Provider:     *providerFlag,
		Model:        *modelFlag,
		SystemPrompt: *systemPromptFlag,
		UserPrompt:   *userPromptFlag,
	}

	if err := run(l, params, os.Stdout); err != nil {
		l.Error("💥💥 application error: %v\n", err)
		os.Exit(1)
	}
}

// run now relies on the flags having been parsed by main.
func run(l golog.MyLogger, params argumentsToBasicQuery, out io.Writer) error {
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	if params.UserPrompt == "" {
		return fmt.Errorf("prompt flag cannot be empty. Usage: -prompt='your question'")
	}

	l.Info("you asked for provider: %s", params.Provider)
	kind, defaultModel, err := llm.GetProviderKindAndDefaultModel(params.Provider)
	if err != nil {
		return fmt.Errorf("unknown provider '%s': %w", params.Provider, err)
	}

	model := defaultModel
	if params.Model != "" {
		model = params.Model
		l.Info("using model override from flag: %s", model)
	}

	l.Info("will call llm.NewProvider(kind:%s, model:%s)", kind, model)
	provider, err := llm.NewProvider(kind, model, l)
	if err != nil {
		return fmt.Errorf("error creating provider '%s': %w", params.Provider, err)
	}

	req := &llm.LLMRequest{
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: params.SystemPrompt},
			{Role: llm.RoleUser, Content: params.UserPrompt},
		},
		Temperature: defaultTemperature,
		Stream:      false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	l.Info("Sending prompt to %s LLM...\n", kind)
	resp, err := provider.Query(ctx, req)
	if err != nil {
		return fmt.Errorf("error querying LLM: %w", err)
	}

	fmt.Fprintln(out, "\nLLM Response:")
	fmt.Fprintln(out, resp.Text)

	return nil
}
