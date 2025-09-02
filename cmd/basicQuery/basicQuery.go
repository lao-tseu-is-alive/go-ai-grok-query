// main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

const defaultRole = "You are a helpful bash shell assistant."

func checkErr(err error, msg string) {
	if err != nil {
		fmt.Printf("## 💥💥 Error %s: %v\n", msg, err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go 'your prompt'")
		os.Exit(1)
	}
	prompt := os.Args[1]
	/*
			// Create provider (Ollama)
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Kind:  llm.ProviderOllama,
				Model: "deepseek-r1:latest",
				// BaseURL defaults to http://localhost:11434; set explicitly if needed:
				// BaseURL: "http://localhost:11434",
			})
			checkErr(err, "creating Ollama provider")
			// Build the chat request
			req := &llm.LLMRequest{
				Model: "deepseek-r1:latest",
				Messages: []llm.LLMMessage{
					{Role: llm.RoleSystem, Content: defaultRole},
					{Role: llm.RoleUser, Content: prompt},
				},
				// Optional controls
				Temperature: 0.0,
				Stream:      false,
			}
			// Create Gemini provider
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Kind:   llm.ProviderGemini,
				Model:  "gemini-2.5-flash", // pick the desired Gemini model
				APIKey: os.Getenv("GEMINI_API_KEY"),
				// BaseURL left default: https://generativelanguage.googleapis.com
			})
			checkErr(err, "creating Gemini provider")

			// Build the chat request
			req := &llm.LLMRequest{
				Model: "gemini-2.5-flash",
				Messages: []llm.LLMMessage{
					{Role: llm.RoleSystem, Content: defaultRole},
					{Role: llm.RoleUser, Content: prompt},
				},
				Temperature: 0.2,
				Stream:      false,
			}

			key, err := config.GetXaiApiKeyFromEnv()
			if err != nil {
				panic(fmt.Sprintf("need to get the api key : %v", err))
			}

			// Create Xai Groq provider
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Kind:   llm.ProviderXAI,
				Model:  "grok-3-mini",
				APIKey: key,
			})
			checkErr(err, "creating XAI provider")

			// Build the chat request
			req := &llm.LLMRequest{
				Model: "grok-3-mini",
				Messages: []llm.LLMMessage{
					{Role: llm.RoleSystem, Content: defaultRole},
					{Role: llm.RoleUser, Content: prompt},
				},
				Temperature: 0.2,
				Stream:      false,
			}
		provider, err := llm.NewProvider(llm.ProviderConfig{
			Kind:   llm.ProviderOpenAI,
			Model:  "gpt-4.1-mini", // choose an available OpenAI chat model
			APIKey: os.Getenv("OPENAI_API_KEY"),
			// BaseURL defaults to https://api.openai.com/v1
			// ExtraHeaders can be added if needed
		})
		checkErr(err, "creating OpenAI provider")

		req := &llm.LLMRequest{
			Model: "gpt-4.1-mini",
			Messages: []llm.LLMMessage{
				{Role: llm.RoleSystem, Content: defaultRole},
				{Role: llm.RoleUser, Content: prompt},
			},
			Temperature: 0.2,
			Stream:      false,
		}

	*/
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Kind:   llm.ProviderOpenRouter,
		Model:  "deepseek/deepseek-chat-v3.1:free", // choose an available OpenAI chat model
		APIKey: os.Getenv("OPEN_ROUTER_API_KEY"),
		// BaseURL defaults to https://api.openai.com/v1
		// ExtraHeaders can be added if needed
	})
	checkErr(err, "creating OpenAI provider")

	req := &llm.LLMRequest{
		Model: "deepseek/deepseek-chat-v3.1:free",
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: defaultRole},
			{Role: llm.RoleUser, Content: prompt},
		},
		Temperature: 0.2,
		Stream:      false,
	}

	// Apply a timeout to the request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Sending prompt to LLM...")
	resp, err := provider.Query(ctx, req)
	checkErr(err, "querying LLM")

	fmt.Println("\nLLM Response:")
	fmt.Println(resp.Text)
}
