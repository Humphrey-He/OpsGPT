package memory

import (
	"fmt"
	"strings"
	"time"

	"GopherSentinel/pkg/llm"
)

// ConversationHistory manages conversation history for context
type ConversationHistory struct {
	messages   []llm.Message
	maxTokens  int
	maxHistory int
}

// NewConversationHistory creates a new conversation history manager
func NewConversationHistory(maxHistory int) *ConversationHistory {
	return &ConversationHistory{
		maxHistory: maxHistory,
		maxTokens:  4096,
		messages:   make([]llm.Message, 0),
	}
}

// AddUserMessage adds a user message to history
func (h *ConversationHistory) AddUserMessage(content string) {
	h.messages = append(h.messages, llm.Message{
		Role:    "user",
		Content: content,
	})
	h.trim()
}

// AddAssistantMessage adds an assistant message to history
func (h *ConversationHistory) AddAssistantMessage(content string) {
	h.messages = append(h.messages, llm.Message{
		Role:    "assistant",
		Content: content,
	})
	h.trim()
}

// AddMessage adds a message to history
func (h *ConversationHistory) AddMessage(role, content string) {
	h.messages = append(h.messages, llm.Message{
		Role:    role,
		Content: content,
	})
	h.trim()
}

// GetMessages returns the conversation history
func (h *ConversationHistory) GetMessages() []llm.Message {
	return h.messages
}

// GetLastNMessages returns the last N messages
func (h *ConversationHistory) GetLastNMessages(n int) []llm.Message {
	if n <= 0 || n > len(h.messages) {
		return h.messages
	}
	start := len(h.messages) - n
	return h.messages[start:]
}

// Clear clears the conversation history
func (h *ConversationHistory) Clear() {
	h.messages = make([]llm.Message, 0)
}

// Size returns the number of messages in history
func (h *ConversationHistory) Size() int {
	return len(h.messages)
}

// trim trims history to stay within limits
func (h *ConversationHistory) trim() {
	// Trim to max history size
	if len(h.messages) > h.maxHistory {
		h.messages = h.messages[len(h.messages)-h.maxHistory:]
	}
}

// FormatHistory formats the history as a string
func (h *ConversationHistory) FormatHistory() string {
	var parts []string
	for _, msg := range h.messages {
		role := msg.Role
		if role == "user" {
			role = "You"
		} else if role == "assistant" {
			role = "Assistant"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", role, msg.Content))
	}
	return strings.Join(parts, "\n")
}

// Session represents a conversation session
type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	History   *ConversationHistory
	Metadata  map[string]interface{}
}

// NewSession creates a new conversation session
func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		History:   NewConversationHistory(50),
		Metadata:  make(map[string]interface{}),
	}
}

// Update updates the session timestamp
func (s *Session) Update() {
	s.UpdatedAt = time.Now()
}

// MessageWindow represents a sliding window of messages
type MessageWindow struct {
	messages   []llm.Message
	maxWindow  int
	overlap    int
}

// NewMessageWindow creates a new message window
func NewMessageWindow(maxWindow, overlap int) *MessageWindow {
	return &MessageWindow{
		maxWindow: maxWindow,
		overlap:   overlap,
		messages:  make([]llm.Message, 0),
	}
}

// Add adds a message to the window
func (w *MessageWindow) Add(msg llm.Message) {
	w.messages = append(w.messages, msg)
	w.trim()
}

// trim trims messages to stay within window size
func (w *MessageWindow) trim() {
	if len(w.messages) > w.maxWindow {
		// Keep overlapping messages
		w.messages = w.messages[len(w.messages)-w.maxWindow+w.overlap:]
	}
}

// GetMessages returns all messages in the window
func (w *MessageWindow) GetMessages() []llm.Message {
	return w.messages
}

// GetSystemContext builds a system context from recent messages
func (h *ConversationHistory) GetSystemContext() string {
	if len(h.messages) == 0 {
		return ""
	}

	var context []string
	// Get last few messages for context
	lastMessages := h.GetLastNMessages(6)
	for _, msg := range lastMessages {
		if msg.Role == "user" {
			context = append(context, fmt.Sprintf("Previous question: %s", msg.Content))
		}
	}

	return strings.Join(context, "\n")
}
