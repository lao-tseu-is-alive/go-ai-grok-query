#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

echo "## 🚀 Starting End-to-End test for basicQuery..."

# 1. Build the binaries
echo "## 🔨 Building basicQuery and mock_server..."
go build -o basicQuery ./cmd/basicQuery
go build -o mock_server ./test/mock_server

# 2. Start the mock server in the background
echo "## 🌐 Starting mock server..."
./mock_server &
# Capture the Process ID (PID) of the server
SERVER_PID=$!
# Ensure the server is killed when the script exits, even on error
trap 'echo "## 🛑 Stopping mock server..."; kill $SERVER_PID' EXIT

# Give the server a moment to start up
sleep 1

# 3. Set environment variables to redirect API calls to our mock server
export OLLAMA_API_BASE="http://localhost:8181"
export GEMINI_API_BASE="http://localhost:8181"
export XAI_API_BASE="http://localhost:8181"
export OPENAI_API_BASE="http://localhost:8181"
export OPENROUTER_API_BASE="http://localhost:8181"

# Set dummy API keys to pass startup checks
export GEMINI_API_KEY="dummy-key-for-testing-gemini-with-sufficient-length"
export XAI_API_KEY="dummy-key-for-testing-xai-with-sufficient-length"
export OPENAI_API_KEY="dummy-key-for-testing-openai-with-sufficient-length"
export OPENROUTER_API_KEY="dummy-key-for-testing-openrouter-with-sufficient-length"

# 4. Run tests for each provider
providers=("ollama" "gemini" "xai" "openai" "openrouter")
all_passed=true

for provider in "${providers[@]}"; do
    echo "## 🧪 Testing provider: $provider..."

    # Execute the command and capture its output
    # We use a test model name for Gemini to match the mock server path
    model_arg=""
    if [[ "$provider" == "gemini" ]]; then
        model_arg="-model gemini-test"
    fi

    output=$(./basicQuery -provider="$provider" -prompt="test" $model_arg)

    # Check if the output contains the expected mock response
    expected_response=""
    case "$provider" in
        "openai"|"openrouter"|"xai")
            expected_response="Mock response for OpenAI-compatible API"
            ;;
        "ollama")
            expected_response="Mock response for Ollama"
            ;;
        "gemini")
            expected_response="Mock response for Gemini"
            ;;
    esac

    if echo "$output" | grep -q "$expected_response"; then
        echo "## ✅ PASSED: $provider"
    else
        echo "## ❌ FAILED: $provider"
        echo "## Expected to find: '$expected_response'"
        echo "## Got output:"
        echo "$output"
        all_passed=false
    fi
done

# 5. Final result
if [ "$all_passed" = true ]; then
    echo "## 🎉 All provider tests passed!"
    exit 0
else
    echo "## 🔥 Some provider tests failed."
    exit 1
fi