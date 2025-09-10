# Go AI LLM Query

[![Go version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/lao-tseu-is-alive/go-ai-llm-query)](https://github.com/lao-tseu-is-alive/go-ai-llm-query/releases/latest)

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
* **Customizable System Prompt**: Tailor the assistant's personality and instructions using the `-system.role` flag.
* **Unified API**: Abstracted `LLMRequest` and `LLMResponse` structs provide a consistent experience, regardless of the backend provider.
* **Automated Releases**: Binaries for multiple platforms are automatically built and published via GitHub Actions.

## ⚙️ Installation

### From Pre-compiled Binaries (Recommended)

You can download ready-to-use executables for Linux, macOS, and Windows directly from our GitHub Releases page.

1.  Navigate to the **[Latest Release](https://github.com/lao-tseu-is-alive/go-ai-llm-query/releases/latest)**.
2.  In the **Assets** section, download the archive corresponding to your operating system and architecture (e.g., `basicQuery-linux-amd64.tar.gz` or `basicQuery-windows-amd64.zip`).
3.  Extract the archive, and you're ready to run the `basicQuery` executable.

### From Source (for Developers)

1.  **Clone the repository:**
    ```sh
    git clone [https://github.com/lao-tseu-is-alive/go-ai-llm-query.git](https://github.com/lao-tseu-is-alive/go-ai-llm-query.git)
    cd go-ai-llm-query
    ```

2.  **Build the executables:**
    ```sh
    # Build the basic query tool
    go build -o basicQuery ./cmd/basicQuery

    # Build the tool-calling example
    go build -o toolCalling ./cmd/toolCalling
    
    # Build the new model comparison tool
    go build -o askToAllModels ./cmd/askToAllModels
    ```

## 🔑 Configuration

The application uses environment variables to manage API keys. Create a `.env` file in the root of the project or export the variables directly into your shell.

**`.env` file example:**
```env
# For OpenAI
OPENAI_API_KEY="sk-..."

# For OpenRouter.ai
OPENROUTER_API_KEY="sk-or-..."

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

> **Note**: Ollama runs locally and does not require an API key.

## 🚀 Usage

### 1. Basic Queries (`basicQuery`)

The `basicQuery` tool sends a single prompt to a specified provider and prints the response.

**Syntax:**
```sh
./basicQuery -provider=<provider> -prompt="Your question" [-system.role="Custom instructions"]

## usage 
Usage: ./basicQuery -provider=<provider> [options]

A powerful and flexible CLI to query various Large Language Models.

Required Flags:
  -provider	Provider to use (ollama, gemini, xai, openai, openrouter)

Options for querying:
  -prompt	The prompt to send to the LLM. Required for querying.
  -model	Model to use. If blank, a default for the provider is chosen.
  -system	The system role for the assistant.
  -temperature	The temperature of the model. Increasing the temperature will make the model answer more creatively(value range 0.0 - 2.0).
  -stream	Enable streaming the response.

Options for listing models:
  -list-models	Lists available models for the specified provider and exits.
  -json-output	Use with -list-models to output in JSON format.


```

**Examples:**

* **Query a local Ollama model with default settings:**
    ```sh
    ./basicQuery -provider=ollama -prompt="Write a simple bash script to list all files in a directory."
    ```

* **Query OpenAI with a custom system prompt:**
    ```sh
    export OPENAI_API_KEY="sk-..."
    ./basicQuery -provider=openai \
                 -prompt="How do I create a virtual environment?" \
                 -system.role="You are a senior Python developer. Your answers are clear, concise, and always include best practices."
    ```

* **Query Grok via XAI:**
    ```sh
    export XAI_API_KEY="..."
    ./basicQuery -provider=xai -prompt="Explain the Fermi Paradox in simple terms."
    ```

### 2. Tool Calling (`toolCalling`)

The `toolCalling` tool demonstrates how an LLM can use tools to answer a question.

**Example:**
This command asks the LLM about the weather. The model will decide to call the `get_current_weather` function, the application will execute it, and the LLM will use the function's result to formulate a final answer.

```sh
export OPENAI_API_KEY="sk-..."
./toolCalling -provider=openai -prompt="What's the weather like in Lausanne, Switzerland?"
```

**Expected output flow:**
1.  The application logs that it's asking the LLM to decide on a tool.
2.  The LLM responds, requesting a call to `get_current_weather` with `{"location": "Lausanne, Switzerland"}`.
3.  The application executes the tool and gets a JSON result like `{"summary": "You better look out your window", ...}`.
4.  This result is sent back to the LLM.
5.  The LLM generates the final, user-friendly response.


### 3. Model Comparison  (`askToAllModels`)

The `askToAllModels` is a new tool sends a single query to all available models of a given provider and saves the full responses to a JSON file for easy comparison.

**Syntax:**
```sh
./askToAllModels -provider=<provider> -prompt="Your question" [-system="Custom instructions"] [-temperature=0.2]
```


**Example:**
This command will query all ollama models and save the results to model_comparison_results.json.

```sh
./askToAllModels -provider=ollama -system='you are an honest and helpful assistant' -prompt='Tell me about your strengths and weaknesses' -temperature=0.2
```


### 4. Helper Scripts

The `scripts/` directory contains convenient wrappers for common tasks.

* **Run a query using an environment file:**
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

The `pkg/llm` package is designed to be used in other Go applications.

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
	l, _ := golog.NewLogger("simple", "stderr", golog.InfoLevel, "my-app")

	// 1. Create a provider (API keys are read from the environment automatically)
	provider, err := llm.NewProvider(llm.ProviderOpenAI, "gpt-4o-mini", l)
	if err != nil {
		log.Fatalf("Error creating provider: %v", err)
	}

	// 2. Build your request
	req := &llm.LLMRequest{
		Messages: []llm.LLMMessage{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant that always replies in Markdown."},
			{Role: ll.RoleUser, Content: "What are the three largest cities in Switzerland?"},
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