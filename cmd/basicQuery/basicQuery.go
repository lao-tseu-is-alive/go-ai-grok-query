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
	APP                = "basicQuery"
	defaultRole        = "You are a helpful bash shell assistant.Your output should be concise, efficient and easy to read in a bash Linux console."
	defaultTemperature = 0.2
	defaultTimeout     = 120 * time.Second
)

type argumentsToBasicQuery struct {
	Provider     string
	Model        string
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	Streaming    bool
}

// usage provides a more detailed help message for the CLI tool.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -provider=<provider> [options]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "A powerful and flexible CLI to query various Large Language Models.")
	fmt.Fprintln(os.Stderr, "\nRequired Flags:")
	fmt.Fprintf(os.Stderr, "  -provider\tProvider to use (ollama, gemini, xai, openai, openrouter)\n")
	fmt.Fprintln(os.Stderr, "\nOptions for querying:")
	fmt.Fprintf(os.Stderr, "  -model\tModel to use. If blank, a default for the provider is chosen.\n")
	fmt.Fprintf(os.Stderr, "  -prompt\tThe prompt to send to the LLM. Required for querying.\n")
	fmt.Fprintf(os.Stderr, "  -system\tThe system role for the assistant.\n")
	fmt.Fprintf(os.Stderr, "  -temperature\tThe temperature of the model. Increasing the temperature will make the model answer more creatively(value range 0.0 - 2.0).\n")
	fmt.Fprintf(os.Stderr, "  -stream\tEnable streaming the response.\n") // Add this line
	fmt.Fprintln(os.Stderr, "\nOptions for listing models:")
	fmt.Fprintf(os.Stderr, "  -list-models\tLists available models for the specified provider and exits.\n")
	fmt.Fprintf(os.Stderr, "  -json-output\tUse with -list-models to output in JSON format.\n\n")
}

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic("stderr"),
		config.GetLogLevelFromEnvOrPanic(golog.InfoLevel),
		APP,
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
	temperatureFlag := flag.Float64("temperature", defaultTemperature, fmt.Sprintf("The temperature for the LLM response (0.0 - 2.0) default value is : %f", defaultTemperature))
	streamFlag := flag.Bool("stream", false, "Enable streaming the response")
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
		flag.Usage()
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
	l.Info("the system prompt is : %s", *systemPromptFlag)

	params := argumentsToBasicQuery{
		Provider:     *providerFlag,
		Model:        *modelFlag,
		SystemPrompt: *systemPromptFlag,
		UserPrompt:   *userPromptFlag,
		Temperature:  *temperatureFlag,
		Streaming:    *streamFlag,
	}

	if err := run(l, params, os.Stdout); err != nil {
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
func run(l golog.MyLogger, params argumentsToBasicQuery, out io.Writer) error {
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, git: %s", APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	if params.UserPrompt == "" {
		return fmt.Errorf("💥💥 provider : %s,  error user prompt cannot be empty ", params.Provider)
	}

	kind, defaultModel, err := llm.GetProviderKindAndDefaultModel(params.Provider)
	if err != nil {
		return fmt.Errorf("💥💥  error getting provider %s kind :%v", params.Provider, err)
	}
	modelToUse := defaultModel
	if params.Model != "" {
		modelToUse = params.Model
		l.Info("using model override from flag: %s", modelToUse)
	} else {
		l.Info("using default model for provider: %s", modelToUse)
	}

	provider, err := llm.NewProvider(kind, modelToUse, l)
	if err != nil {
		return fmt.Errorf("💥💥 error creating provider '%s': %v", params.Provider, err)
	}

	// 4. Add model validation logic before querying
	l.Info("Validating model '%s' with provider...", modelToUse)
	modelsList, err := llm.GetModelsList(l, provider, defaultTimeout)
	if err != nil {
		return fmt.Errorf("error getting list of models for provider %s. err: %w", params.Provider, err)
	}
	// Check if the desired model exists in the list returned by the provider.
	if !slices.Contains(modelsList, modelToUse) {
		return fmt.Errorf("model '%s' is not available for this provider. Use -list-models to see valid options", modelToUse)
	}
	temperature := llm.Clamp(params.Temperature, 0.0, 2.0)

	req := &llm.LLMRequest{
		Model: modelToUse, // Use the validated or default model
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: params.SystemPrompt},
			{Role: llm.RoleUser, Content: params.UserPrompt},
		},
		Temperature: temperature,
		Stream:      params.Streaming,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if params.Streaming {
		l.Info("Sending prompt to %s LLM (streaming)...\n", params.Provider)
		fmt.Fprintln(out, "\nLLM Response (Streaming):")

		// onDelta callback prints text chunks as they arrive
		onDelta := func(delta llm.Delta) {
			fmt.Fprint(out, delta.Text)
		}

		_, err := provider.Stream(ctx, req, onDelta)
		if err != nil {
			return fmt.Errorf("error querying LLM via stream: %w", err)
		}
		fmt.Fprintln(out) // Add a final newline for clean output
	} else {

		l.Info("Sending prompt to %s LLM...\n", params.Provider)
		resp, err := provider.Query(ctx, req)
		if err != nil {
			return fmt.Errorf("error querying LLM: %w", err)
		}

		fmt.Fprintln(out, "\nLLM Response:")
		fmt.Fprintln(out, resp.Text)
	}

	return nil
}
