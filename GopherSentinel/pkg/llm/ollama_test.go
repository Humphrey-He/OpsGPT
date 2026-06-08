package llm

import (
	"testing"
	"time"
)

func TestOllamaClient_DefaultConfig(t *testing.T) {
	config := DefaultOllamaConfig()

	if config.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %v, want http://localhost:11434", config.BaseURL)
	}
	if config.Model != "llama3" {
		t.Errorf("Model = %v, want llama3", config.Model)
	}
	if config.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("EmbeddingModel = %v, want nomic-embed-text", config.EmbeddingModel)
	}
	if config.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want 120s", config.Timeout)
	}
}

func TestNewOllamaClient(t *testing.T) {
	config := OllamaConfig{
		BaseURL:        "http://localhost:11434",
		Model:         "llama3",
		EmbeddingModel: "nomic-embed-text",
		Timeout:       60 * time.Second,
	}

	client := NewOllamaClient(config)

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %v, want http://localhost:11434", client.baseURL)
	}
	if client.model != "llama3" {
		t.Errorf("model = %v, want llama3", client.model)
	}
	if client.embeddingModel != "nomic-embed-text" {
		t.Errorf("embeddingModel = %v, want nomic-embed-text", client.embeddingModel)
	}
}

func TestNewOllamaClient_EmptyConfig(t *testing.T) {
	// Should use defaults for empty config
	client := NewOllamaClient(OllamaConfig{})

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %v, want http://localhost:11434", client.baseURL)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if id1 == "" {
		t.Error("GenerateID() returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateID() returned same ID twice")
	}
}

func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	if msg.Role != "user" {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Content = %v, want Hello, world!", msg.Content)
	}
}
