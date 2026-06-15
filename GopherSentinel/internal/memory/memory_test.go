package memory

import (
	"testing"

	"GopherSentinel/pkg/llm"
)

func TestNewConversationHistory(t *testing.T) {
	history := NewConversationHistory(10)

	if history.maxHistory != 10 {
		t.Errorf("Expected maxHistory 10, got %d", history.maxHistory)
	}
	if history.maxTokens != 4096 {
		t.Errorf("Expected maxTokens 4096, got %d", history.maxTokens)
	}
	if len(history.messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(history.messages))
	}
}

func TestConversationHistory_AddUserMessage(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("Hello")

	if len(history.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history.messages))
	}
	if history.messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", history.messages[0].Role)
	}
	if history.messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", history.messages[0].Content)
	}
}

func TestConversationHistory_AddAssistantMessage(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddAssistantMessage("Hi there")

	if len(history.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history.messages))
	}
	if history.messages[0].Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", history.messages[0].Role)
	}
}

func TestConversationHistory_AddMessage(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddMessage("system", "You are helpful")

	if len(history.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history.messages))
	}
	if history.messages[0].Role != "system" {
		t.Errorf("Expected role 'system', got '%s'", history.messages[0].Role)
	}
}

func TestConversationHistory_MaxHistory(t *testing.T) {
	history := NewConversationHistory(3)

	// Add 5 messages
	for i := 0; i < 5; i++ {
		history.AddUserMessage("Message")
	}

	// Should only keep last 3
	if len(history.messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(history.messages))
	}
}

func TestConversationHistory_GetMessages(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("Hello")
	history.AddAssistantMessage("Hi")

	messages := history.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestConversationHistory_GetLastNMessages(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("1")
	history.AddUserMessage("2")
	history.AddUserMessage("3")
	history.AddUserMessage("4")
	history.AddUserMessage("5")

	last3 := history.GetLastNMessages(3)
	if len(last3) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(last3))
	}
}

func TestConversationHistory_GetLastNMessages_LessThanAvailable(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("1")
	history.AddUserMessage("2")

	// Request 10 but only 2 available
	last10 := history.GetLastNMessages(10)
	if len(last10) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(last10))
	}
}

func TestConversationHistory_GetLastNMessages_InvalidN(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("1")
	history.AddUserMessage("2")

	// Request 0 or negative
	last0 := history.GetLastNMessages(0)
	if len(last0) != 2 {
		t.Errorf("Expected all 2 messages for n=0, got %d", len(last0))
	}

	lastNeg := history.GetLastNMessages(-1)
	if len(lastNeg) != 2 {
		t.Errorf("Expected all 2 messages for n=-1, got %d", len(lastNeg))
	}
}

func TestConversationHistory_Clear(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("Hello")
	history.Clear()

	if len(history.messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(history.messages))
	}
}

func TestConversationHistory_Size(t *testing.T) {
	history := NewConversationHistory(10)
	if history.Size() != 0 {
		t.Errorf("Expected size 0, got %d", history.Size())
	}

	history.AddUserMessage("Hello")
	if history.Size() != 1 {
		t.Errorf("Expected size 1, got %d", history.Size())
	}
}

func TestConversationHistory_FormatHistory(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("Hello")
	history.AddAssistantMessage("Hi there")

	formatted := history.FormatHistory()
	if formatted == "" {
		t.Error("Expected non-empty formatted history")
	}
}

func TestConversationHistory_FormatHistory_Empty(t *testing.T) {
	history := NewConversationHistory(10)
	formatted := history.FormatHistory()
	if formatted != "" {
		t.Errorf("Expected empty string, got '%s'", formatted)
	}
}

func TestNewMessageWindow(t *testing.T) {
	window := NewMessageWindow(5, 2)

	if window.maxWindow != 5 {
		t.Errorf("Expected maxWindow 5, got %d", window.maxWindow)
	}
	if window.overlap != 2 {
		t.Errorf("Expected overlap 2, got %d", window.overlap)
	}
}

func TestMessageWindow_Add(t *testing.T) {
	window := NewMessageWindow(5, 2)
	window.Add(llm.Message{Role: "user", Content: "Hello"})

	if len(window.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(window.messages))
	}
}

func TestMessageWindow_GetMessages(t *testing.T) {
	window := NewMessageWindow(5, 2)
	window.Add(llm.Message{Role: "user", Content: "1"})
	window.Add(llm.Message{Role: "user", Content: "2"})

	messages := window.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestMessageWindow_MaxSize(t *testing.T) {
	window := NewMessageWindow(3, 1)

	// Add 5 messages (exceeds max of 3)
	for i := 0; i < 5; i++ {
		window.Add(llm.Message{Role: "user", Content: "Message"})
	}

	// Should be trimmed but keep overlap
	if len(window.messages) > 3 {
		t.Errorf("Expected at most 3 messages, got %d", len(window.messages))
	}
}

func TestConversationHistory_GetSystemContext(t *testing.T) {
	history := NewConversationHistory(10)
	history.AddUserMessage("First question")
	history.AddAssistantMessage("First answer")
	history.AddUserMessage("Second question")

	context := history.GetSystemContext()
	if context == "" {
		t.Error("Expected non-empty system context")
	}
}

func TestConversationHistory_GetSystemContext_Empty(t *testing.T) {
	history := NewConversationHistory(10)
	context := history.GetSystemContext()

	if context != "" {
		t.Errorf("Expected empty context, got '%s'", context)
	}
}

func TestMessageWindow_Overlap(t *testing.T) {
	window := NewMessageWindow(3, 1)

	// Add messages to test overlap behavior
	window.Add(llm.Message{Role: "user", Content: "A"})
	window.Add(llm.Message{Role: "user", Content: "B"})
	window.Add(llm.Message{Role: "user", Content: "C"})

	// After 3 messages, should be at capacity
	if len(window.messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(window.messages))
	}

	// Add one more
	window.Add(llm.Message{Role: "user", Content: "D"})

	// Should have some overlap (at most 3 messages)
	if len(window.messages) > 3 {
		t.Errorf("Expected at most 3 messages, got %d", len(window.messages))
	}
}

func TestMessageWindow_Empty(t *testing.T) {
	window := NewMessageWindow(5, 2)

	messages := window.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

func TestMessageWindow_GetSystemContext(t *testing.T) {
	// Test that MessageWindow can be used similarly to ConversationHistory
	window := NewMessageWindow(10, 2)
	window.Add(llm.Message{Role: "user", Content: "Question 1"})
	window.Add(llm.Message{Role: "assistant", Content: "Answer 1"})
	window.Add(llm.Message{Role: "user", Content: "Question 2"})

	messages := window.GetMessages()
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}
}
