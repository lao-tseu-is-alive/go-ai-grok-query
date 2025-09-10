package main

import (
	"context"
	"encoding/json"
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
	APP                = "askToAllModels"
	defaultTemperature = 0.2
	defaultTimeout     = 90 * time.Second
)

type argumentsToAskToAll struct {
	Provider     string
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
}

type llmResult struct {
	Provider     string `json:"provider,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	UserPrompt   string `json:"user_prompt,omitempty"`
	Response     string `json:"response,omitempty"`
}

// usage provides a more detailed help message for the CLI tool.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -provider=<provider> [options]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "A powerful and flexible CLI to query all models from a provider ans save the result.")
	fmt.Fprintln(os.Stderr, "\nRequired Flags:")
	fmt.Fprintf(os.Stderr, "  -provider\tProvider to use (ollama, gemini, xai, openai, openrouter)\n")
	fmt.Fprintf(os.Stderr, "  -prompt\tThe prompt to send to LLM model.\n")
	fmt.Fprintf(os.Stderr, "  -system\tThe system role for the assistant.\n")
	fmt.Fprintln(os.Stderr, "\nOptional Flags:")
	fmt.Fprintf(os.Stderr, "  -temperature\tThe temperature of the model. Increasing the temperature will make the model answer more creatively(value range 0.0 - 2.0).\n")
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

	flag.Usage = usage
	providerFlag := flag.String("provider", "", "Provider to use (ollama, gemini, xai, openai, openrouter)")
	systemPromptFlag := flag.String("system", "", "The system role for your assistant, it default to an helpful shell assistant")
	userPromptFlag := flag.String("prompt", "", "The prompt to send to the LLM")
	temperatureFlag := flag.Float64("temperature", defaultTemperature, fmt.Sprintf("The temperature for the LLM response (0.0 - 2.0) default value is : %f", defaultTemperature))

	flag.Parse()

	if *providerFlag == "" {
		l.Error("💥💥 Error: -provider flag is required.")
		flag.Usage()
		os.Exit(1)
	}
	l.Info("you asked for provider: %s", *providerFlag)

	if *userPromptFlag == "" {
		l.Error("💥💥 Error: user  -prompt flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	if *systemPromptFlag == "" {
		l.Error("💥💥 Error:  -system flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	params := argumentsToAskToAll{
		Provider:     *providerFlag,
		SystemPrompt: *systemPromptFlag,
		UserPrompt:   *userPromptFlag,
		Temperature:  *temperatureFlag,
	}

	if err := run(l, params); err != nil {
		l.Error("💥💥 application error: %v\n", err)
		os.Exit(1)
	}
}

func getModelsName(l golog.MyLogger, provider llm.Provider) ([]string, error) {
	l.Info("Fetching available models...")
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	models, err := provider.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching models from provider: %w", err)
	}
	modelNames := make([]string, 0, len(models))

	for _, m := range models {
		modelNames = append(modelNames, m.Name)
	}
	return modelNames, nil
}

func run(l golog.MyLogger, params argumentsToAskToAll) error {
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, git: %s", APP, version.VERSION, version.BuildStamp, version.REPOSITORY)
	kind, defModel, err := llm.GetProviderKindAndDefaultModel(params.Provider)
	if err != nil {
		return fmt.Errorf("💥💥  error getting provider %s kind :%v", params.Provider, err)
	}

	provider, err := llm.NewProvider(kind, defModel, l)
	if err != nil {
		return fmt.Errorf("💥💥 error creating provider '%s': %v", params.Provider, err)
	}
	modelsList, err := getModelsName(l, provider)
	if err != nil {
		return fmt.Errorf("error getting list of models for provider %s. err: %w", params.Provider, err)
	}
	temperature := llm.Clamp(params.Temperature, 0.0, 2.0)
	allResults := make([]llmResult, 0, len(modelsList))
	// Loop through each model and query it
	for i, currentModel := range modelsList {
		req := &llm.LLMRequest{
			Model: currentModel, // Use the validated or default model
			Messages: []llm.LLMMessage{
				{Role: llm.RoleSystem, Content: params.SystemPrompt},
				{Role: llm.RoleUser, Content: params.UserPrompt},
			},
			Temperature: temperature,
			Stream:      false,
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		l.Info("Sending prompt to %s LLM, model: %s (%d of %d)...\n", params.Provider, currentModel, i, len(modelsList))
		resp, err := provider.Query(ctx, req)
		if err != nil {
			l.Warn("error querying model %s LLM: %w", currentModel, err)
			continue // let's skip this one
		}
		currentResult := llmResult{
			Provider:     params.Provider,
			ModelName:    currentModel,
			SystemPrompt: params.SystemPrompt,
			UserPrompt:   params.UserPrompt,
			Response:     resp.Text,
		}
		allResults = append(allResults, currentResult)

		l.Info("\nLLM Response: \n%s", resp.Text)
	}
	// Save the allResults to a file (e.g., JSON)
	jsonData, err := json.MarshalIndent(allResults, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal allResults: %v", err)
	}

	err = os.WriteFile("model_comparison_results.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("Failed to write allResults file: %v", err)
	}

	fmt.Println("Comparison completed. Results saved to model_comparison_results.json")
	return nil
}
