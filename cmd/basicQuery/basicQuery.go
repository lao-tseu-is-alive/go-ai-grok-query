// main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// XaiUrl The base URL for the x.ai (Grok) API.
const (
	XaiUrl      = "https://api.x.ai/v1/chat/completions"
	defaultRole = "You are an helpfully bash shell assistant."
)

// APIRequest Define the structure for the request payload sent to the LLM API.
// This has been updated to include Stream and Temperature fields.
type APIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

// Message Define the structure for a single message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// APIResponse Define the structure for the expected API response.
// We are interested in the 'choices' array, which contains the model's output.
type APIResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice Define the structure for a single choice in the response.
type Choice struct {
	Message Message `json:"message"`
}

// queryLLM prepares and sends the HTTP request to the LLM API.
func queryLLM(role, prompt, apiURL, apiKey string) (string, error) {
	// Create the request payload using the structs defined earlier.
	// This now includes a system message, the correct model, and other parameters.
	requestPayload := APIRequest{
		Model: "grok-3-mini",
		Messages: []Message{
			{
				Role:    "system",
				Content: role,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream:      false,
		Temperature: 0,
	}

	// Marshal the Go struct into a JSON byte slice.
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %w", err)
	}

	// Create a new HTTP request. We use a context with a timeout to prevent
	// the program from hanging indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set the required HTTP headers.
	// The Content-Type tells the server we're sending JSON.
	// The Authorization header carries our API key.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute the request using the default HTTP client.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to API: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Check for non-200 status codes which indicate an API error.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the JSON response body into our APIResponse struct.
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	// Extract the content from the first choice.
	// Check if there are any choices to avoid a panic.
	if len(apiResponse.Choices) > 0 {
		fmt.Printf("We received %d\n choices", len(apiResponse.Choices))
		return apiResponse.Choices[0].Message.Content, nil
	}

	return "No response content received.", nil
}

func main() {
	// 1. Get the prompt from command-line arguments.
	// The program expects the prompt to be the first argument after the program name.
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go \"<your prompt>\"")
		os.Exit(1)
	}
	prompt := os.Args[1]

	// 2. Get the API key from an environment variable for security.
	// Updated to use XAI_API_KEY.
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: XAI_API_KEY environment variable not set.")
		fmt.Println("Please set it before running the program:")
		fmt.Println("export XAI_API_KEY='your_api_key_here'")
		os.Exit(1)
	}

	// 3. Call the queryLLM function and handle any potential errors.
	fmt.Println("Sending prompt to LLM...")
	response, err := queryLLM(defaultRole, prompt, XaiUrl, apiKey)
	if err != nil {
		fmt.Printf("Error querying LLM: %v\n", err)
		os.Exit(1)
	}

	// 4. Print the response from the LLM.
	fmt.Println("\nLLM Response:")
	fmt.Println(response)
}
