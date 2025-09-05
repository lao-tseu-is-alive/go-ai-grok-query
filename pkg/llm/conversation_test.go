package llm

import (
	"testing"
)

func TestConversation(t *testing.T) {
	systemPrompt := "You are a test assistant."

	t.Run("NewConversation", func(t *testing.T) {
		convo, err := NewConversation(systemPrompt)
		if err != nil {
			t.Fatalf("NewConversation failed: %v", err)
		}
		if convo.SystemPrompt != systemPrompt {
			t.Errorf("Expected system prompt '%s', got '%s'", systemPrompt, convo.SystemPrompt)
		}
		if len(convo.Messages) != 1 || convo.Messages[0].Role != RoleSystem {
			t.Errorf("Expected 1 system message, got %d", len(convo.Messages))
		}

		// Test empty system prompt
		_, err = NewConversation("")
		if err == nil {
			t.Error("Expected error for empty system prompt, got nil")
		}
	})

	t.Run("AddMessages", func(t *testing.T) {
		convo, _ := NewConversation(systemPrompt)

		// Add User Message
		userContent := "Hello, world!"
		err := convo.AddUserMessage(userContent)
		if err != nil {
			t.Fatalf("AddUserMessage failed: %v", err)
		}
		if len(convo.Messages) != 2 || convo.Messages[1].Content != userContent {
			t.Error("User message not added correctly")
		}

		// Add Assistant Response
		assistantResp := &LLMResponse{Text: "Hi there!"}
		convo.AddAssistantResponse(assistantResp)
		if len(convo.Messages) != 3 || convo.Messages[2].Content != assistantResp.Text {
			t.Error("Assistant response not added correctly")
		}

		// Add Tool Result
		toolCallID := "tool-123"
		toolResult := `{"status": "ok"}`
		convo.AddToolResultMessage(toolCallID, toolResult)
		if len(convo.Messages) != 4 || convo.Messages[3].ToolCallID != toolCallID {
			t.Error("Tool result message not added correctly")
		}
	})

	t.Run("MessagesCopy", func(t *testing.T) {
		convo, _ := NewConversation(systemPrompt)
		convo.AddUserMessage("test")

		originalMessages := convo.Messages
		copiedMessages := convo.MessagesCopy()

		if len(originalMessages) != len(copiedMessages) {
			t.Fatalf("Copied slice has different length")
		}

		// Modify the copy and check if the original is unchanged
		copiedMessages[0].Content = "modified"
		if originalMessages[0].Content == copiedMessages[0].Content {
			t.Error("Original slice was modified when the copy changed, MessagesCopy is not returning a true copy.")
		}
	})
}
