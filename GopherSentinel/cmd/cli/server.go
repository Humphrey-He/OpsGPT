package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"GopherSentinel/internal/rag"
	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/vector"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the GopherSentinel API server",
	Long: `Start the GopherSentinel API server.

The server provides REST API endpoints for:
- Document ingestion
- Question answering
- Collection management

Example:
  GopherSentinel server --port 8080`,
	RunE: runServer,
}

var serverFlags = struct {
	port    int
	host    string
	timeout int
}{}

func init() {
	serverCmd.Flags().IntVar(&serverFlags.port, "port", 8080, "server port")
	serverCmd.Flags().StringVar(&serverFlags.host, "host", "0.0.0.0", "server host")
	serverCmd.Flags().IntVar(&serverFlags.timeout, "timeout", 120, "request timeout in seconds")
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize configuration
	config := initConfig()

	// Initialize LLM client
	llmClient := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:        config.Ollama.BaseURL,
		Model:         config.Ollama.Model,
		EmbeddingModel: config.Ollama.EmbeddingModel,
		Timeout:       time.Duration(config.Ollama.Timeout) * time.Second,
	})

	// Initialize vector store
	vectorStore, err := vector.NewQdrantClient(vector.QdrantConfig{
		URL:        config.Qdrant.URL,
		Collection: config.Qdrant.Collection,
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
	_ = rag.NewRAGChain(llmClient, vectorStore, ragConfig)

	// Setup handlers
	mux := http.NewServeMux()
	setupHandlers(mux, config)

	// Create server
	addr := fmt.Sprintf("%s:%d", serverFlags.host, serverFlags.port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(serverFlags.timeout) * time.Second,
		WriteTimeout: time.Duration(serverFlags.timeout) * time.Second,
	}

	fmt.Printf("🚀 GopherSentinel server starting on %s\n", addr)
	fmt.Println("   Press Ctrl+C to stop")

	// Start server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown
	fmt.Println("\n👋 Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	return nil
}

func setupHandlers(mux *http.ServeMux, config *Config) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/api/ask", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Not implemented"}`))
	})

	mux.HandleFunc("/api/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Not implemented"}`))
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Not implemented"}`))
	})
}
