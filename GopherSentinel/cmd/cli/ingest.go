package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"GopherSentinel/internal/rag"
	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/parser"
	"GopherSentinel/pkg/vector"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest <path>",
	Short: "Ingest and index documents",
	Long: `Parse and index documents into the vector database.

Supports:
- Markdown (.md, .markdown)
- PDF (.pdf)
- Plain text (.txt)

Example:
  GopherSentinel ingest ./docs
  GopherSentinel ingest ./docs --collection my_collection`,
	Args: cobra.ExactArgs(1),
	RunE: runIngest,
}

var ingestFlags = struct {
	collection    string
	force         bool
	showProgress  bool
}{}

func init() {
	ingestCmd.Flags().StringVar(&ingestFlags.collection, "collection", "", "collection name (default: from config)")
	ingestCmd.Flags().BoolVarP(&ingestFlags.force, "force", "f", false, "force re-indexing (clears existing data)")
	ingestCmd.Flags().BoolVar(&ingestFlags.showProgress, "progress", true, "show progress during indexing")
}

func runIngest(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	path := args[0]

	// Initialize configuration
	config := initConfig()

	// Initialize LLM client
	llmClient := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:        config.Ollama.BaseURL,
		Model:          config.Ollama.Model,
		EmbeddingModel: config.Ollama.EmbeddingModel,
		Timeout:        time.Duration(config.Ollama.Timeout) * time.Second,
	})

	// Check if Ollama is available
	if !llmClient.IsAvailable(ctx) {
		fmt.Printf("⚠️  Warning: Ollama is not available at %s\n", config.Ollama.BaseURL)
		fmt.Println("   Please ensure Ollama is running.")
	}

	// Initialize vector store
	collectionName := ingestFlags.collection
	if collectionName == "" {
		collectionName = config.Qdrant.Collection
	}

	vectorStore, err := vector.NewQdrantClient(vector.QdrantConfig{
		URL:        config.Qdrant.URL,
		Collection: collectionName,
		VectorSize: config.Qdrant.VectorSize,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	// Initialize RAG chain
	ragConfig := rag.RAGConfig{
		ChunkSize:      config.RAG.ChunkSize,
		ChunkOverlap:   config.RAG.ChunkOverlap,
		TopK:           config.RAG.TopK,
		ScoreThreshold: 0.7,
		MaxContextLen:  config.RAG.MaxContextTokens,
	}
	ragChain := rag.NewRAGChain(llmClient, vectorStore, ragConfig)

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	// Handle force re-indexing
	if ingestFlags.force {
		fmt.Println("🗑️  Clearing existing collection...")
		if err := ragChain.ClearCollection(ctx); err != nil {
			fmt.Printf("⚠️  Warning: failed to clear collection: %v\n", err)
		}
	}

	// Determine if path is file or directory
	var chunkCount int
	if info.IsDir() {
		fmt.Printf("📁 Indexing directory: %s\n", path)
		chunkCount, err = ragChain.IngestDirectory(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to ingest directory: %w", err)
		}
	} else {
		fmt.Printf("📄 Indexing file: %s\n", path)

		// Parse the document
		doc, err := parser.ParseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse document: %w", err)
		}

		if err := ragChain.IngestDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to ingest document: %w", err)
		}

		// Count chunks (simplified - actual count would come from chunker)
		chunkCount = 1
	}

	fmt.Printf("\n✅ Successfully indexed %d chunks\n", chunkCount)
	return nil
}
