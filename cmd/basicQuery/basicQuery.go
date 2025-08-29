// main.go
package main

import (
	"fmt"
	"os"

	"github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm"
)

const (
	defaultRole = "You are an helpfully bash shell assistant."
)

// checkErr is a helper function to handle errors
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
	myChat, err := llm.GetInstance("Ollama", "deepseek-r1:latest")
	checkErr(err, "getting ollama LLM")

	fmt.Println("Sending prompt to LLM...")
	response, err := myChat.Query(defaultRole, prompt)
	checkErr(err, "querying LLM")

	fmt.Println("\nLLM Response:")
	fmt.Println(response)
}
