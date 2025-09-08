package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
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

// usage provides a more detailed help message for the CLI tool.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -provider=<provider> [options]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "A powerful and flexible CLI to query various Large Language Models.")
	fmt.Fprintln(os.Stderr, "\nRequired Flags:")
	fmt.Fprintf(os.Stderr, "  -provider\tProvider to use (ollama, gemini, xai, openai, openrouter)\n")
	fmt.Fprintln(os.Stderr, "\nOptions for querying:")
	fmt.Fprintf(os.Stderr, "  -prompt\tThe prompt to send to the LLM. Required for querying.\n")
	fmt.Fprintf(os.Stderr, "  -model\tModel to use. If blank, a default for the provider is chosen.\n")
	fmt.Fprintf(os.Stderr, "  -system\tThe system role for the assistant.\n")
	fmt.Fprintln(os.Stderr, "\nOptions for listing models:")
	fmt.Fprintf(os.Stderr, "  -list-models\tLists available models for the specified provider and exits.\n")
	fmt.Fprintf(os.Stderr, "  -json-output\tUse with -list-models to output in JSON format.\n\n")
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

	// 1. Update flag definitions and set custom usage function
	flag.Usage = usage
	providerFlag := flag.String("provider", "", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	modelFlag := flag.String("model", "", "Model to use, depends on chosen provider, leave blank for a default valid choice")
	systemPromptFlag := flag.String("system", defaultRole, "The system role for your assistant, it default to an helpful shell assistant")
	userPromptFlag := flag.String("prompt", "", "The prompt to send to the LLM")
	listModelsFlag := flag.Bool("list-models", false, "List available models for the provider and exit")
	jsonOutputFlag := flag.Bool("json-output", false, "Use with -list-models for JSON output")
	flag.Parse()

	// 2. Make the -provider flag mandatory
	if *providerFlag == "" {
		l.Error("💥💥 Error: -provider flag is required.")
		flag.Usage()
		os.Exit(1)
	}
	l.Info("you asked for provider: %s", *providerFlag)

	// Create the provider instance early to use it for listing or querying
	kind, _, err := llm.GetProviderKindAndDefaultModel(*providerFlag)
	if err != nil {
		l.Error("💥💥 %v", err)
		os.Exit(1)
	}
	// The model will be set properly in the run() function, we can use a dummy value here.
	provider, err := llm.NewProvider(kind, "default", l)
	if err != nil {
		l.Error("💥💥 Error creating provider '%s': %v", *providerFlag, err)
		os.Exit(1)
	}

	// 3. Handle the -list-models functionality
	if *listModelsFlag {
		if err := handleListModels(l, provider, *jsonOutputFlag); err != nil {
			l.Error("💥💥 Could not list models: %v", err)
			os.Exit(1)
		}
		return // Exit successfully after listing models
	}

	// For querying, a prompt is now mandatory
	if *userPromptFlag == "" {
		l.Error("💥💥 Error: -prompt flag is required for querying.")
		flag.Usage()
		os.Exit(1)
	}

	params := argumentsToBasicQuery{
		Provider:     *providerFlag,
		Model:        *modelFlag,
		SystemPrompt: *systemPromptFlag,
		UserPrompt:   *userPromptFlag,
	}

	if err := run(l, provider, params, os.Stdout); err != nil {
		l.Error("💥💥 application error: %v\n", err)
		os.Exit(1)
	}
}

// handleListModels fetches and displays the models from a provider.
func handleListModels(l golog.MyLogger, provider llm.Provider, jsonOutput bool) error {
	l.Info("Fetching available models...")
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	models, err := provider.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("error fetching models from provider: %w", err)
	}

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(models, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal models to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Available models:")
		for _, m := range models {
			fmt.Printf("- %s\n", m.Name)
		}
	}
	return nil
}

// run is now responsible for validating the model and executing the query.
func run(l golog.MyLogger, provider llm.Provider, params argumentsToBasicQuery, out io.Writer) error {
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	_, defaultModel, _ := llm.GetProviderKindAndDefaultModel(params.Provider)
	modelToUse := defaultModel
	if params.Model != "" {
		modelToUse = params.Model
		l.Info("using model override from flag: %s", modelToUse)
	} else {
		l.Info("using default model for provider: %s", modelToUse)
	}

	// 4. Add model validation logic before querying
	l.Info("Validating model '%s' with provider...", modelToUse)
	ctxValidate, cancelValidate := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelValidate()

	availableModels, err := provider.ListModels(ctxValidate)
	if err != nil {
		// If the list endpoint fails, we log a warning but proceed cautiously.
		l.Warn("Could not validate model with provider (ListModels failed: %v). Proceeding with query anyway.", err)
	} else {
		// Check if the desired model exists in the list returned by the provider.
		isValid := slices.ContainsFunc(availableModels, func(m llm.ModelInfo) bool {
			// Some providers prefix with "models/", so we check for both formats.
			return m.Name == modelToUse || m.Name == "models/"+modelToUse
		})

		if !isValid {
			return fmt.Errorf("model '%s' is not available for this provider. Use -list-models to see valid options", modelToUse)
		}
		l.Info("✅ Model '%s' is valid.", modelToUse)
	}

	req := &llm.LLMRequest{
		Model: modelToUse, // Use the validated or default model
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: params.SystemPrompt},
			{Role: llm.RoleUser, Content: params.UserPrompt},
		},
		Temperature: defaultTemperature,
		Stream:      false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	l.Info("Sending prompt to %s LLM...\n", params.Provider)
	resp, err := provider.Query(ctx, req)
	if err != nil {
		return fmt.Errorf("error querying LLM: %w", err)
	}

	fmt.Fprintln(out, "\nLLM Response:")
	fmt.Fprintln(out, resp.Text)

	return nil
}
