# Go AI LLM Query

[![Go version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

`go-ai-llm-query` is a powerful and flexible command-line interface (CLI) and Go library designed to provide a unified API for interacting with various Large Language Models (LLMs). It simplifies the process of sending queries and handling responses, with built-in support for advanced features like tool calling (function calling).

The project is architected with a clean separation of concerns, making it easy to use as a standalone tool or to integrate its `pkg/llm` library into your own Go applications.

---

## ✨ Features

* **Multi-Provider Support**: A single, unified interface to query different LLM providers.
    * OpenAI (`gpt-4o-mini`, etc.)
    * OpenRouter (Access a wide range of models)
    * Gemini (Google's models)
    * XAI (`grok-3-mini`, etc.)
    * Ollama (For local models like Llama3, Qwen, etc.)
* **Advanced Tool Calling**: A full implementation of the tool-calling workflow, allowing models to request the execution of functions (e.g., `get_current_weather`) and receive the results to formulate a final answer.
* **Unified API**: Abstracted `LLMRequest` and `LLMResponse` structs provide a consistent experience, regardless of the backend provider.
* **Extensible by Design**: The `Provider` interface makes it simple to add new LLM providers in the future.
* **Configurable**: Easily configure API keys and logging settings via environment variables.

## ⚙️ Installation

1.  **Clone the repository:**
    ```sh
    git clone [https://github.com/lao-tseu-is-alive/go-ai-llm-query.git](https://github.com/lao-tseu-is-alive/go-ai-llm-query.git)
    cd go-ai-llm-query
    ```

2.  **Build the executables:**
    The project contains two main applications. You can build them using the standard Go toolchain.

    ```sh
    # Build the basic query tool
    go build -o basicQuery ./cmd/basicQuery

    # Build the tool-calling example
    go build -o toolCalling ./cmd/toolCalling
    ```

## 🔑 Configuration

The application uses environment variables to manage API keys. Create a `.env` file in the root of the project or export the variables directly into your shell.

**`.env` file example:**
```env
# For OpenAI
OPENAI_API_KEY="sk-..."

# For OpenRouter.ai
OPEN_ROUTER_API_KEY="sk-or-..."

# For Google Gemini
GEMINI_API_KEY="..."

# For XAI (Grok)
XAI_API_KEY="..."

# --- Log Configuration (Optional) ---
# LOG_LEVEL can be: debug, info, warn, error
LOG_LEVEL="info" 
# LOG_FILE can be: stderr, stdout, or a filename
LOG_FILE="stderr"
```

The application will automatically load variables from a `.env` file if it exists when using the provided helper scripts.

> **Note**: Ollama runs locally and does not require an API key.

## 🚀 Usage

### 1. Basic Queries (`basicQuery`)

The `basicQuery` tool sends a single prompt to a specified provider and prints the response.

**Syntax:**
```sh
./basicQuery -provider=<provider> -prompt="Your question here"
```

**Examples:**

* **Query OpenAI:**
    ```sh
    export OPENAI_API_KEY="sk-..."
    ./basicQuery -provider=openai -prompt="What are the main principles of Go programming?"
    ```

* **Query a local Ollama model:**
    ```sh
    ./basicQuery -provider=ollama -prompt="Write a simple bash script to list all files in a directory."
    ```

* **Query Grok via XAI:**
    ```sh
    export XAI_API_KEY="..."
    ./basicQuery -provider=xai -prompt="Explain the Fermi Paradox."
    ```

### 2. Tool Calling (`toolCalling`)

The `toolCalling` tool demonstrates how an LLM can use tools to answer a question. The example tool can "get the current weather".

**Example:**

Run the tool with the default prompt ("What's the weather right now in Lausanne in Switzerland?"). The LLM will first decide to call the `get_current_weather` function, the application will "execute" it, and then the LLM will use the function's output to give you a final, user-friendly answer.

```sh
export OPENAI_API_KEY="sk-..."
./toolCalling -provider=openai
```

**Expected output flow:**
1.  The application logs that it's asking the LLM to decide on a tool.
2.  The LLM responds, requesting to call `get_current_weather` with the argument `{"location": "Lausanne, Switzerland"}`.
3.  The application executes the tool and gets a JSON result like `{"summary": "You better look out your window", ...}`.
4.  The application sends this result back to the LLM.
5.  The LLM generates the final response: "You better look out your window to check the weather in Lausanne."

### 3. Helper Scripts

The `scripts/` directory contains convenient wrappers for common tasks.

* **Run a query with an environment file:**
    ```sh
    # This script runs the toolCalling app using variables from .env
    ./scripts/01_go_run.sh
    ```

* **Quickly query XAI or Ollama:**
    ```sh
    ./scripts/runXaiQuery.sh "Your question for Grok"
    ./scripts/runOllamaQuery.sh "Your question for your local model"
    ```

## 📦 Using as a Library

The real power of this project is its `pkg/llm` library. You can easily integrate it into your own applications.

**Example Snippet:**
```go
package main

import (
	"context"
	"fmt"
	"log"

	"[github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm](https://github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/llm)"
	"[github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog](https://github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog)"
)

func main() {
	// Use your preferred logger
	l, _ := golog.NewLogger("simple", "stderr", golog.InfoLevel, "my-app")

	// 1. Create a provider
	// The NewProvider function handles fetching the API key from the environment.
	provider, err := llm.NewProvider(llm.ProviderOpenAI, "gpt-4o-mini", l)
	if err != nil {
		log.Fatalf("Error creating provider: %v", err)
	}

	// 2. Build your request
	req := &llm.LLMRequest{
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant."},
			{Role: llm.RoleUser, Content: "What is the capital of Switzerland?"},
		},
	}

	// 3. Query the LLM
	resp, err := provider.Query(context.Background(), req)
	if err != nil {
		log.Fatalf("Error querying LLM: %v", err)
	}

	// 4. Use the response
	fmt.Println("LLM Response:")
	fmt.Println(resp.Text)
}
```

## 📜 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.