package llm

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
