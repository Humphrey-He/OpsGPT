package rag

import (
	"context"
	"testing"

	"GopherSentinel/pkg/parser"
	"GopherSentinel/pkg/vector"
)

// MockLLMClient for testing
type MockLLMClient struct{}

func (m *MockLLMClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	dim := 768
	vec := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vec[i] = float32(len(text)%10) * 0.1
	}
	return vec, nil
}

func (m *MockLLMClient) Chat(ctx context.Context, messages []interface{}) (string, error) {
	return "This is a mock response based on context.", nil
}

// MockVectorStore for RAG testing
type MockVectorStore struct {
	docs     map[string]string
	metadata map[string]interface{}
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		docs:     make(map[string]string),
		metadata: make(map[string]interface{}),
	}
}

func (m *MockVectorStore) IndexDocument(ctx context.Context, content string, vec []float32, meta map[string]interface{}) error {
	id := meta["source"].(string)
	m.docs[id] = content
	m.metadata[id] = meta
	return nil
}

func (m *MockVectorStore) Search(ctx context.Context, queryVector []float32, limit int, filter *vector.Filter) ([]vector.SearchResult, error) {
	results := make([]vector.SearchResult, 0)
	for id, content := range m.docs {
		payload := make(map[string]interface{})
		if m.metadata[id] != nil {
			if meta, ok := m.metadata[id].(map[string]interface{}); ok {
				payload = meta
			}
		}
		results = append(results, vector.SearchResult{
			ID:      id,
			Content: content,
			Score:   0.85,
			Payload: payload,
		})
	}
	if len(results) > limit {
		return results[:limit], nil
	}
	return results, nil
}

func TestRAGConfig_Defaults(t *testing.T) {
	config := DefaultRAGConfig()

	if config.ChunkSize != 512 {
		t.Errorf("Expected ChunkSize 512, got %d", config.ChunkSize)
	}
	if config.ChunkOverlap != 50 {
		t.Errorf("Expected ChunkOverlap 50, got %d", config.ChunkOverlap)
	}
	if config.TopK != 5 {
		t.Errorf("Expected TopK 5, got %d", config.TopK)
	}
	if config.ScoreThreshold != 0.7 {
		t.Errorf("Expected ScoreThreshold 0.7, got %f", config.ScoreThreshold)
	}
	if config.MaxContextLen != 4096 {
		t.Errorf("Expected MaxContextLen 4096, got %d", config.MaxContextLen)
	}
}

func TestQuery_Structure(t *testing.T) {
	query := Query{
		Question: "What is RAG?",
		TopK:     3,
		Stream:   false,
	}

	if query.Question != "What is RAG?" {
		t.Errorf("Expected question 'What is RAG?', got '%s'", query.Question)
	}
	if query.TopK != 3 {
		t.Errorf("Expected TopK 3, got %d", query.TopK)
	}
	if query.Stream != false {
		t.Errorf("Expected Stream false, got %v", query.Stream)
	}
}

func TestAnswer_Structure(t *testing.T) {
	answer := Answer{
		Content:    "RAG is Retrieval-Augmented Generation",
		Sources:    []Source{},
		Generation: "RAG is Retrieval-Augmented Generation",
	}

	if answer.Content != "RAG is Retrieval-Augmented Generation" {
		t.Errorf("Unexpected content: %s", answer.Content)
	}
}

func TestSource_Structure(t *testing.T) {
	source := Source{
		Content:  "RAG content here",
		Score:    0.95,
		Title:    "RAG Guide",
		Source:   "docs/rag.md",
		ChunkIdx: 0,
	}

	if source.Title != "RAG Guide" {
		t.Errorf("Expected title 'RAG Guide', got '%s'", source.Title)
	}
	if source.Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", source.Score)
	}
}

func TestStreamAnswer_Structure(t *testing.T) {
	streamAnswer := StreamAnswer{
		Type:    "content",
		Content: "Streaming content",
		Sources: []Source{},
		Error:   "",
		Done:    false,
	}

	if streamAnswer.Type != "content" {
		t.Errorf("Expected type 'content', got '%s'", streamAnswer.Type)
	}
	if streamAnswer.Content != "Streaming content" {
		t.Errorf("Unexpected content: %s", streamAnswer.Content)
	}
}

func TestBuildContext_EmptyResults(t *testing.T) {
	ragChain := &RAGChain{
		config: DefaultRAGConfig(),
	}

	context, sources := ragChain.buildContext([]vector.SearchResult{})

	if context != "" {
		t.Errorf("Expected empty context, got '%s'", context)
	}
	if len(sources) != 0 {
		t.Errorf("Expected 0 sources, got %d", len(sources))
	}
}

func TestBuildContext_WithResults(t *testing.T) {
	ragChain := &RAGChain{
		config: DefaultRAGConfig(),
	}

	results := []vector.SearchResult{
		{
			ID:      "doc1",
			Content: "First document content",
			Score:   0.9,
			Payload: map[string]interface{}{
				"title":  "Doc 1",
				"source": "doc1.md",
			},
		},
		{
			ID:      "doc2",
			Content: "Second document content",
			Score:   0.8,
			Payload: map[string]interface{}{
				"title":  "Doc 2",
				"source": "doc2.md",
			},
		},
	}

	context, sources := ragChain.buildContext(results)

	if context == "" {
		t.Error("Expected non-empty context")
	}
	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}
	if sources[0].Title != "Doc 1" {
		t.Errorf("Expected title 'Doc 1', got '%s'", sources[0].Title)
	}
}

func TestBuildPrompt(t *testing.T) {
	ragChain := &RAGChain{
		config: DefaultRAGConfig(),
	}

	question := "What is RAG?"
	context := "RAG is a technique that combines retrieval and generation."

	messages := ragChain.buildPrompt(question, context)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("Expected system role, got '%s'", messages[0].Role)
	}

	if messages[1].Role != "user" {
		t.Errorf("Expected user role, got '%s'", messages[1].Role)
	}

	if messages[1].Content == "" {
		t.Error("Expected non-empty user content")
	}
}

func TestIngestDocument_Chunking(t *testing.T) {
	chunkerConfig := parser.ChunkerConfig{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MinChunkSize: 20,
		MaxChunkSize: 200,
	}

	chunker := parser.NewSlidingWindowChunker(chunkerConfig)

	doc := &parser.Document{
		Content: "This is a test document with enough content to be chunked into multiple pieces for testing purposes.",
		Metadata: parser.DocumentMetadata{
			Source: "test.md",
			Title:  "Test Document",
		},
	}

	chunks, err := chunker.Chunkize(doc.Content, parser.ChunkMetadata{
		Source:   doc.Metadata.Source,
		DocTitle: doc.Metadata.Title,
	})

	if err != nil {
		t.Fatalf("Chunkize failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}
}

func TestNewRAGChain(t *testing.T) {
	ragChain := NewRAGChain(nil, nil, DefaultRAGConfig())

	if ragChain == nil {
		t.Fatal("Expected non-nil RAGChain")
	}

	if ragChain.config.ChunkSize != 512 {
		t.Errorf("Expected default ChunkSize 512, got %d", ragChain.config.ChunkSize)
	}
}

func TestRAGChain_BuildContext_HandlesNilPayload(t *testing.T) {
	ragChain := &RAGChain{
		config: DefaultRAGConfig(),
	}

	results := []vector.SearchResult{
		{
			ID:      "doc1",
			Content: "Content without metadata",
			Score:   0.9,
			Payload: nil,
		},
	}

	context, sources := ragChain.buildContext(results)

	if context == "" {
		t.Error("Expected non-empty context")
	}
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}
}

func TestStreamAnswer_ErrorType(t *testing.T) {
	streamAnswer := StreamAnswer{
		Type:  "error",
		Error: "Something went wrong",
	}

	if streamAnswer.Type != "error" {
		t.Errorf("Expected type 'error', got '%s'", streamAnswer.Type)
	}
	if streamAnswer.Error != "Something went wrong" {
		t.Errorf("Unexpected error: %s", streamAnswer.Error)
	}
}

func TestStreamAnswer_DoneType(t *testing.T) {
	streamAnswer := StreamAnswer{
		Type: "done",
		Done: true,
	}

	if streamAnswer.Type != "done" {
		t.Errorf("Expected type 'done', got '%s'", streamAnswer.Type)
	}
	if !streamAnswer.Done {
		t.Error("Expected Done to be true")
	}
}

func TestBuildContext_IntPayload(t *testing.T) {
	ragChain := &RAGChain{
		config: DefaultRAGConfig(),
	}

	results := []vector.SearchResult{
		{
			ID:      "doc1",
			Content: "Content with int chunk_index",
			Score:   0.9,
			Payload: map[string]interface{}{
				"chunk_index": 1, // int, not int64
			},
		},
	}

	context, sources := ragChain.buildContext(results)

	if context == "" {
		t.Error("Expected non-empty context")
	}
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}
}
