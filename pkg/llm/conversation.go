package llm

import (
	"errors"
	"log/slog"
	"slices"
	"sync"
)

// Conversation manages a thread-safe history of LLM messages.
// It supports system prompts, user/assistant turns, and tool results.
type Conversation struct {
	mu           sync.RWMutex
	Messages     []LLMMessage
	SystemPrompt string // Cache for easy access
}

// NewConversation creates a new conversation with the given system prompt.
// Returns an error if the prompt is empty for safety.
func NewConversation(systemPrompt string) (*Conversation, error) {
	if systemPrompt == "" {
		return nil, errors.New("system prompt cannot be empty")
	}
	return &Conversation{
		Messages: []LLMMessage{
			{Role: RoleSystem, Content: systemPrompt},
		},
		SystemPrompt: systemPrompt,
	}, nil
}

// AddUserMessage appends a user message.
// Returns an error if content is empty.
func (c *Conversation) AddUserMessage(content string) error {
	if content == "" {
		return errors.New("user message content cannot be empty")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, LLMMessage{Role: RoleUser, Content: content})
	return nil
}

// AddAssistantResponse appends an assistant response, including tool calls.
// Extracts text and tool calls from the response.
func (c *Conversation) AddAssistantResponse(resp *LLMResponse) {
	if resp == nil {
		slog.Warn("Nil response passed to AddAssistantResponse")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, LLMMessage{
		Role:      RoleAssistant,
		Content:   resp.Text,
		ToolCalls: resp.ToolCalls,
	})
}

// AddToolResultMessage appends a tool's result by its call ID.
func (c *Conversation) AddToolResultMessage(toolCallID, result string) {
	if toolCallID == "" || result == "" {
		slog.Warn("Empty tool call ID or result", "id", toolCallID)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, LLMMessage{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Content:    result,
	})
}

// MessagesCopy returns a thread-safe copy of messages for querying.
func (c *Conversation) MessagesCopy() []LLMMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slices.Clone(c.Messages) // Go 1.21+ for immutability
}
