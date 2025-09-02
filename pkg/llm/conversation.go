package llm

// Conversation manages the history of a chat.
type Conversation struct {
	Messages []LLMMessage
}

func NewConversation(systemPrompt string) *Conversation {
	return &Conversation{
		Messages: []LLMMessage{
			{Role: RoleSystem, Content: systemPrompt},
		},
	}
}

// AddUserMessage adds a user message to the conversation.
func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, LLMMessage{Role: RoleUser, Content: content})
}

// AddAssistantResponse adds the assistant's response, including any tool calls.
func (c *Conversation) AddAssistantResponse(resp *LLMResponse) {
	// For OpenAI compatibility, we need to create a message with tool_calls.
	// We'll add this to the types later. For now, let's keep it simple.
	c.Messages = append(c.Messages, LLMMessage{
		Role:      RoleAssistant,
		Content:   resp.Text,
		ToolCalls: resp.ToolCalls,
	})
}

// AddToolResultMessage adds the result of a tool call to the conversation.
func (c *Conversation) AddToolResultMessage(toolCallID, result string) {
	c.Messages = append(c.Messages, LLMMessage{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Content:    result,
	})
}
