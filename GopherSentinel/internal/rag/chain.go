package rag

import (
	"context"
	"fmt"
	"strings"

	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/parser"
	"GopherSentinel/pkg/vector"
)

// RAGConfig holds configuration for the RAG chain
type RAGConfig struct {
	ChunkSize      int
	ChunkOverlap   int
	TopK           int
	ScoreThreshold float32
	MaxContextLen  int
}

// DefaultRAGConfig returns default RAG configuration
func DefaultRAGConfig() RAGConfig {
	return RAGConfig{
		ChunkSize:      512,
		ChunkOverlap:   50,
		TopK:           5,
		ScoreThreshold: 0.7,
		MaxContextLen:  4096,
	}
}

// RAGChain implements the Retrieval-Augmented Generation chain
type RAGChain struct {
	llmClient   *llm.OllamaClient
	vectorStore *vector.QdrantClient
	config      RAGConfig
	chunker     *parser.SlidingWindowChunker
}

// NewRAGChain creates a new RAG chain
func NewRAGChain(llmClient *llm.OllamaClient, vectorStore *vector.QdrantClient, config RAGConfig) *RAGChain {
	chunkerConfig := parser.ChunkerConfig{
		ChunkSize:    config.ChunkSize,
		ChunkOverlap: config.ChunkOverlap,
		MinChunkSize: 100,
		MaxChunkSize: 1024,
	}

	return &RAGChain{
		llmClient:   llmClient,
		vectorStore: vectorStore,
		config:      config,
		chunker:     parser.NewSlidingWindowChunker(chunkerConfig),
	}
}

// IngestDocuments parses and indexes documents
func (r *RAGChain) IngestDocuments(ctx context.Context, documents []*parser.Document) error {
	for _, doc := range documents {
		if err := r.IngestDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to ingest document %s: %w", doc.Metadata.Source, err)
		}
	}
	return nil
}

// IngestDocument parses and indexes a single document
func (r *RAGChain) IngestDocument(ctx context.Context, doc *parser.Document) error {
	// Chunk the document
	chunks, err := r.chunker.Chunkize(doc.Content, parser.ChunkMetadata{
		Source:   doc.Metadata.Source,
		DocTitle: doc.Metadata.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	// Index each chunk
	for i, chunk := range chunks {
		// Get embedding for the chunk
		embedding, err := r.llmClient.GetEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to get embedding for chunk %d: %w", i, err)
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"source":       chunk.Metadata.Source,
			"title":        chunk.Metadata.DocTitle,
			"chunk_index":  chunk.Index,
			"total_chunks": chunk.Metadata.TotalChunks,
			"start_char":   chunk.StartChar,
			"end_char":     chunk.EndChar,
		}

		// Index the chunk
		if err := r.vectorStore.IndexDocument(ctx, chunk.Content, embedding, metadata); err != nil {
			return fmt.Errorf("failed to index chunk %d: %w", i, err)
		}
	}

	return nil
}

// IngestDirectory parses and indexes all documents in a directory
func (r *RAGChain) IngestDirectory(ctx context.Context, dirPath string) (int, error) {
	// Parse all documents
	documents, err := parser.ParseDirectory(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse directory: %w", err)
	}

	if len(documents) == 0 {
		return 0, fmt.Errorf("no documents found in %s", dirPath)
	}

	// Count total chunks
	totalChunks := 0
	for _, doc := range documents {
		chunks, _ := r.chunker.Chunkize(doc.Content, parser.ChunkMetadata{
			Source:   doc.Metadata.Source,
			DocTitle: doc.Metadata.Title,
		})
		totalChunks += len(chunks)
	}

	// Index documents
	if err := r.IngestDocuments(ctx, documents); err != nil {
		return 0, err
	}

	return totalChunks, nil
}

// Query represents a query to the RAG chain
type Query struct {
	Question string
	TopK     int
	Stream   bool
}

// Answer represents an answer from the RAG chain
type Answer struct {
	Content    string
	Sources    []Source
	Generation string
}

// Source represents a source document used in the answer
type Source struct {
	Content  string
	Score    float32
	Title    string
	Source   string
	ChunkIdx int
}

// Ask generates an answer to a question using RAG
func (r *RAGChain) Ask(ctx context.Context, query Query) (*Answer, error) {
	// Retrieve relevant chunks
	results, err := r.vectorStore.SearchWithScoreThreshold(
		ctx,
		nil, // Will get embedding from query
		query.TopK,
		r.config.ScoreThreshold,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Get query embedding
	queryEmbedding, err := r.llmClient.GetEmbedding(ctx, query.Question)
	if err != nil {
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}

	// Re-search with the correct query vector
	results, err = r.vectorStore.SearchWithScoreThreshold(
		ctx,
		queryEmbedding,
		query.TopK,
		r.config.ScoreThreshold,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Build context from results
	context, sources := r.buildContext(results)

	// Generate answer
	messages := r.buildPrompt(query.Question, context)
	response, err := r.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &Answer{
		Content:    response.Message.Content,
		Sources:    sources,
		Generation: response.Message.Content,
	}, nil
}

// AskStream generates an answer with streaming output
func (r *RAGChain) AskStream(ctx context.Context, query Query) (<-chan StreamAnswer, error) {
	// Get query embedding
	queryEmbedding, err := r.llmClient.GetEmbedding(ctx, query.Question)
	if err != nil {
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}

	// Search for relevant chunks
	results, err := r.vectorStore.SearchWithScoreThreshold(
		ctx,
		queryEmbedding,
		query.TopK,
		r.config.ScoreThreshold,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Build context from results
	context, sources := r.buildContext(results)

	// Generate answer with streaming
	messages := r.buildPrompt(query.Question, context)
	streamChan, err := r.llmClient.ChatStream(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	// Convert to StreamAnswer channel
	answerChan := make(chan StreamAnswer)

	go func() {
		defer close(answerChan)

		// Send sources first
		answerChan <- StreamAnswer{
			Type:    "sources",
			Sources: sources,
		}

		// Stream the response
		for resp := range streamChan {
			if resp.Error != nil {
				answerChan <- StreamAnswer{
					Type:  "error",
					Error: resp.Error.Error(),
				}
				return
			}

			answerChan <- StreamAnswer{
				Type:    "content",
				Content: resp.Message.Content,
			}
		}

		answerChan <- StreamAnswer{
			Type: "done",
		}
	}()

	return answerChan, nil
}

// StreamAnswer represents a streaming answer
type StreamAnswer struct {
	Type    string
	Content string
	Sources []Source
	Error   string
	Done    bool
}

// buildContext builds the context string from search results
func (r *RAGChain) buildContext(results []vector.SearchResult) (string, []Source) {
	if len(results) == 0 {
		return "", nil
	}

	var contextParts []string
	var sources []Source

	for i, result := range results {
		// Format the context entry
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s", i+1, result.Content, ""))

		// Extract source info
		source := Source{
			Content: result.Content,
			Score:   result.Score,
		}

		if result.Payload != nil {
			if title, ok := result.Payload["title"].(string); ok {
				source.Title = title
			}
			if src, ok := result.Payload["source"].(string); ok {
				source.Source = src
			}
			if idx, ok := result.Payload["chunk_index"].(int64); ok {
				source.ChunkIdx = int(idx)
			}
		}

		sources = append(sources, source)
	}

	return strings.Join(contextParts, "\n---\n"), sources
}

// buildPrompt builds the prompt for the LLM
func (r *RAGChain) buildPrompt(question, context string) []llm.Message {
	systemPrompt := `You are a helpful AI assistant that answers questions based on the provided context.

Guidelines:
1. Only answer based on the provided context. If the answer cannot be found in the context, say "I cannot find the answer in the provided documents."
2. Be concise and helpful in your responses.
3. If you use information from the context, briefly reference it.
4. Format your answers clearly using markdown when appropriate.`

	userPrompt := fmt.Sprintf("Context:\n%s\n\nQuestion: %s\n\nAnswer:", context, question)

	return []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// GetCollectionStats returns statistics about the indexed collection
func (r *RAGChain) GetCollectionStats(ctx context.Context) (*vector.CollectionInfo, error) {
	return r.vectorStore.GetCollectionInfo(ctx)
}

// ClearCollection removes all indexed documents
func (r *RAGChain) ClearCollection(ctx context.Context) error {
	return r.vectorStore.DeleteCollection(ctx)
}
