package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OllamaClient is a client for the Ollama API
type OllamaClient struct {
	baseURL    string
	model      string
	embeddingModel string
	httpClient *http.Client
	timeout    time.Duration
}

// OllamaConfig holds configuration for the Ollama client
type OllamaConfig struct {
	BaseURL        string
	Model         string
	EmbeddingModel string
	Timeout       time.Duration
}

// DefaultOllamaConfig returns default configuration
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL:        "http://localhost:11434",
		Model:         "llama3",
		EmbeddingModel: "nomic-embed-text",
		Timeout:       120 * time.Second,
	}
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	if config.BaseURL == "" {
		config = DefaultOllamaConfig()
	}

	return &OllamaClient{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		model:      config.Model,
		embeddingModel: config.EmbeddingModel,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		timeout: config.Timeout,
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Model       string    `json:"model"`
	CreatedAt   time.Time `json:"created_at"`
	Message     Message   `json:"message"`
	Done        bool      `json:"done"`
	TotalDuration int64   `json:"total_duration,omitempty"`
	EvalCount   int       `json:"eval_count,omitempty"`
}

// Chat sends a chat completion request
func (c *OllamaClient) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// ChatStream sends a streaming chat request and returns a channel of responses
func (c *OllamaClient) ChatStream(ctx context.Context, messages []Message) (<-chan StreamResponse, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	streamChan := make(chan StreamResponse)

	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var streamResp StreamResponse
				if err := decoder.Decode(&streamResp); err != nil {
					if err == io.EOF {
						return
					}
					streamChan <- StreamResponse{Error: err}
					return
				}
				streamChan <- streamResp
				if streamResp.Done {
					return
				}
			}
		}
	}()

	return streamChan, nil
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	Model    string    `json:"model"`
	Message  Message   `json:"message"`
	Done     bool      `json:"done"`
	Error    error     `json:"-"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Model      string    `json:"model"`
	Embedding  []float32 `json:"embedding"`
	TotalDuration int64   `json:"total_duration,omitempty"`
}

// GetEmbedding gets embeddings for a text
func (c *OllamaClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := EmbeddingRequest{
		Model:  c.embeddingModel,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embResp.Embedding, nil
}

// GetEmbeddings gets embeddings for multiple texts
func (c *OllamaClient) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := c.GetEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedding for text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// GenerateRequest represents a text generation request
type GenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream,omitempty"`
	Context []int  `json:"context,omitempty"`
}

// GenerateResponse represents a text generation response
type GenerateResponse struct {
	Model        string `json:"model"`
	Response     string `json:"response"`
	Done         bool   `json:"done"`
	Context      []int  `json:"context,omitempty"`
	TotalDuration int64 `json:"total_duration,omitempty"`
	EvalCount    int    `json:"eval_count,omitempty"`
}

// Generate generates text from a prompt
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (*GenerateResponse, error) {
	req := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &genResp, nil
}

// GenerateStream generates text with streaming
func (c *OllamaClient) GenerateStream(ctx context.Context, prompt string) (<-chan StreamResponse, error) {
	req := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	streamChan := make(chan StreamResponse)

	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var streamResp StreamResponse
				if err := decoder.Decode(&streamResp); err != nil {
					if err == io.EOF {
						return
					}
					streamChan <- StreamResponse{Error: err}
					return
				}
				// For generate, we put the content in Message for consistency
				streamChan <- StreamResponse{
					Model: streamResp.Model,
					Message: Message{
						Role:    "assistant",
						Content: streamResp.Message.Content,
					},
					Done: streamResp.Done,
				}
				if streamResp.Done {
					return
				}
			}
		}
	}()

	return streamChan, nil
}

// CreateModelRequest represents a request to create a model
type CreateModelRequest struct {
	Name      string `json:"name"`
	Modelfile string `json:"modelfile"`
}

// IsAvailable checks if Ollama is available
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ListModels returns a list of available models
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d)", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// GenerateID generates a unique ID
func GenerateID() string {
	return uuid.New().String()
}
