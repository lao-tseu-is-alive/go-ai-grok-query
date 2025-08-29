// main.go
package main

import (
	"fmt"
	"os"

	"github.com/lao-tseu-is-alive/go-ai-grok-query/pkg/llm"
)

const (
	defaultRole = "You are an helpfully bash shell assistant."
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go 'your prompt'")
		os.Exit(1)
	}
	prompt := os.Args[1]
	myChat, err := llm.GetInstance("XAI", "grok-3-mini")
	if err != nil {
		fmt.Printf("Error getting XAI LLM: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sending prompt to LLM...")
	response, err := myChat.Query(defaultRole, prompt)
	if err != nil {
		fmt.Printf("Error querying LLM: %v\n", err)
		os.Exit(1)
	}

	// 4. Print the response from the LLM.
	fmt.Println("\nLLM Response:")
	fmt.Println(response)
}
